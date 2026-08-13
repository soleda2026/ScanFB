package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/facebook"
	"github.com/soleda2026/ScanFB/internal/orchestration"
)

const (
	PreparedGroupScanSchemaVersion = 1
	OperationPreparedGroupScan     = "prepared_group_scan"

	PreparedGroupScanStatusOK    = "ok"
	PreparedGroupScanStatusError = "error"

	PreparedGroupScanErrorInvalidRequest = "invalid_request"
	PreparedGroupScanErrorGroupNotFound  = "group_not_found"
	PreparedGroupScanErrorInactiveGroup  = "inactive_group"
	PreparedGroupScanErrorInvalidPayload = "invalid_prepared_snapshot"
	PreparedGroupScanErrorScanFailed     = "scan_failed"
	PreparedGroupScanErrorStorage        = "storage_error"

	MaxPreparedGroupScanRequestBytes  = facebook.PreparedSnapshotMaxPayloadBytes + 64*1024
	MaxPreparedGroupScanResponseBytes = 16 * 1024
	MaxBridgeDispatchRequestBytes     = MaxPreparedGroupScanRequestBytes
)

type PreparedGroupScanRequest struct {
	SchemaVersion    int             `json:"schema_version"`
	Operation        string          `json:"operation"`
	GroupID          string          `json:"group_id"`
	ScanID           string          `json:"scan_id"`
	AttemptID        string          `json:"attempt_id"`
	ActionAt         string          `json:"action_at"`
	PreparedSnapshot json.RawMessage `json:"prepared_snapshot"`
}

type PreparedGroupScanResponse struct {
	SchemaVersion      int    `json:"schema_version"`
	Operation          string `json:"operation"`
	Status             string `json:"status"`
	GroupName          string `json:"group_name,omitempty"`
	AttemptStatus      string `json:"attempt_status"`
	CollectedPostCount int    `json:"collected_post_count"`
	EvaluatedPostCount int    `json:"evaluated_post_count"`
	IncludedPostCount  int    `json:"included_post_count"`
	ReviewPostCount    int    `json:"review_post_count"`
	ExcludedPostCount  int    `json:"excluded_post_count"`
	AllowedLeadCount   int    `json:"allowed_lead_count"`
	BlockedLeadCount   int    `json:"blocked_lead_count"`
	UnresolvedCount    int    `json:"unresolved_lead_count"`
	ErrorCode          string `json:"error_code,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
}

type PreparedGroupScanRunner func(
	context.Context,
	orchestration.OneGroupScanRequest,
	orchestration.GroupPostCollector,
) (orchestration.OneGroupScanResult, error)

func DecodePreparedGroupScanRequest(reader io.Reader) (PreparedGroupScanRequest, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, MaxPreparedGroupScanRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > MaxPreparedGroupScanRequestBytes {
		return PreparedGroupScanRequest{}, ErrMalformedRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var request PreparedGroupScanRequest
	if err := decoder.Decode(&request); err != nil {
		return PreparedGroupScanRequest{}, ErrMalformedRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return PreparedGroupScanRequest{}, ErrMalformedRequest
	}
	if request.SchemaVersion != PreparedGroupScanSchemaVersion {
		return PreparedGroupScanRequest{}, ErrUnsupportedSchemaVersion
	}
	if request.Operation != OperationPreparedGroupScan {
		return PreparedGroupScanRequest{}, ErrUnsupportedOperation
	}
	if len(request.PreparedSnapshot) > facebook.PreparedSnapshotMaxPayloadBytes {
		return PreparedGroupScanRequest{}, ErrMalformedRequest
	}
	return request, nil
}

func HandlePreparedGroupScan(
	ctx context.Context,
	repository WatchedGroupStateRepository,
	request PreparedGroupScanRequest,
	runner PreparedGroupScanRunner,
) PreparedGroupScanResponse {
	if ctx == nil || runner == nil || strings.TrimSpace(request.GroupID) == "" ||
		strings.TrimSpace(request.ScanID) == "" || strings.TrimSpace(request.AttemptID) == "" ||
		len(request.PreparedSnapshot) == 0 {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorInvalidRequest)
	}

	state, err := repository.Load()
	if err != nil {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorStorage)
	}
	group, found := preparedGroupScanGroup(state.Groups(), request.GroupID)
	if !found {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorGroupNotFound)
	}
	if !group.IsActive() {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorInactiveGroup)
	}

	actionAt, window, err := preparedGroupScanWindow(request.ActionAt)
	if err != nil {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorInvalidRequest)
	}
	oneGroupRequest := orchestration.OneGroupScanRequest{
		ScanID:         request.ScanID,
		AttemptID:      request.AttemptID,
		WatchedGroup:   group,
		ScanWindow:     window,
		SearchProfile:  domain.MacBookSearchProfile(),
		GeographicMode: domain.GeographicModeHoChiMinhCity,
		Blocklist:      blocklist.NewList(nil),
		StartedAt:      actionAt,
		CompletedAt:    actionAt,
	}
	collector := facebook.NewPreparedSnapshotCollector(request.PreparedSnapshot)
	result, err := runner(ctx, oneGroupRequest, collector)
	if err != nil {
		code := PreparedGroupScanErrorScanFailed
		if errors.Is(err, orchestration.ErrOneGroupScanCollectionFailed) {
			code = PreparedGroupScanErrorInvalidPayload
		}
		response := preparedGroupScanErrorResponse(code)
		response.GroupName = group.Name()
		response.AttemptStatus = string(result.Attempt().Status())
		if response.AttemptStatus == "" {
			response.AttemptStatus = string(application.GroupAttemptStatusFailed)
		}
		return response
	}

	applicationResult, ok := result.ApplicationResult()
	if !ok {
		return preparedGroupScanErrorResponse(PreparedGroupScanErrorScanFailed)
	}
	summary := applicationResult.Summary()
	return PreparedGroupScanResponse{
		SchemaVersion:      PreparedGroupScanSchemaVersion,
		Operation:          OperationPreparedGroupScan,
		Status:             PreparedGroupScanStatusOK,
		GroupName:          group.Name(),
		AttemptStatus:      string(result.Attempt().Status()),
		CollectedPostCount: result.CollectedPostCount(),
		EvaluatedPostCount: summary.EvaluatedPostCount,
		IncludedPostCount:  summary.IncludePostCount,
		ReviewPostCount:    summary.ReviewPostCount,
		ExcludedPostCount:  summary.ExcludedPostCount,
		AllowedLeadCount:   summary.AllowedLeadCount,
		BlockedLeadCount:   summary.BlockedLeadCount,
		UnresolvedCount:    summary.UnresolvedLeadCount,
	}
}

func ServePreparedGroupScan(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	factory WatchedGroupRepositoryFactory,
	runner PreparedGroupScanRunner,
) int {
	request, err := DecodePreparedGroupScanRequest(stdin)
	if err != nil {
		writeDiagnostic(stderr, "prepared group scan request rejected")
		return 2
	}
	repository, err := factory()
	if err != nil {
		writeDiagnostic(stderr, "watched group storage unavailable")
		return writePreparedGroupScanResponse(stdout, preparedGroupScanErrorResponse(PreparedGroupScanErrorStorage))
	}
	defer repository.Close()
	return writePreparedGroupScanResponse(stdout, HandlePreparedGroupScan(context.Background(), repository, request, runner))
}

func EncodePreparedGroupScanResponse(response PreparedGroupScanResponse) ([]byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(payload)+1 > MaxPreparedGroupScanResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return append(payload, '\n'), nil
}

func writePreparedGroupScanResponse(stdout io.Writer, response PreparedGroupScanResponse) int {
	payload, err := EncodePreparedGroupScanResponse(response)
	if err != nil {
		return 3
	}
	if _, err := stdout.Write(payload); err != nil {
		return 3
	}
	return 0
}

func preparedGroupScanGroup(groups []domain.WatchedGroup, groupID string) (domain.WatchedGroup, bool) {
	for _, group := range groups {
		if group.ID() == groupID {
			return group, true
		}
	}
	return domain.WatchedGroup{}, false
}

func preparedGroupScanWindow(value string) (time.Time, domain.ScanWindow, error) {
	if !strings.HasSuffix(value, "+07:00") {
		return time.Time{}, domain.ScanWindow{}, ErrMalformedRequest
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, domain.ScanWindow{}, ErrMalformedRequest
	}
	_, offset := parsed.Zone()
	if offset != 7*60*60 {
		return time.Time{}, domain.ScanWindow{}, ErrMalformedRequest
	}
	location, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		return time.Time{}, domain.ScanWindow{}, err
	}
	actionAt := parsed.In(location)
	year, month, day := actionAt.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, location)
	window, err := domain.NewScanWindow(startOfDay, startOfDay, actionAt)
	return actionAt, window, err
}

func preparedGroupScanErrorResponse(code string) PreparedGroupScanResponse {
	return PreparedGroupScanResponse{
		SchemaVersion: PreparedGroupScanSchemaVersion,
		Operation:     OperationPreparedGroupScan,
		Status:        PreparedGroupScanStatusError,
		AttemptStatus: string(application.GroupAttemptStatusFailed),
		ErrorCode:     code,
		ErrorMessage:  preparedGroupScanErrorMessage(code),
	}
}

func preparedGroupScanErrorMessage(code string) string {
	switch code {
	case PreparedGroupScanErrorGroupNotFound:
		return "enrolled group not found"
	case PreparedGroupScanErrorInactiveGroup:
		return "enrolled group is inactive"
	case PreparedGroupScanErrorInvalidPayload:
		return "prepared snapshot rejected"
	case PreparedGroupScanErrorStorage:
		return "watched group storage unavailable"
	case PreparedGroupScanErrorInvalidRequest:
		return "prepared scan request rejected"
	default:
		return "prepared group scan failed"
	}
}
