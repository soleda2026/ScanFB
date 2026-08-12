package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/persistence"
)

const (
	WatchedGroupsSchemaVersion = 2

	OperationWatchedGroupsList      = "watched_groups_list"
	OperationWatchedGroupsAdd       = "watched_groups_add"
	OperationWatchedGroupsSetActive = "watched_groups_set_active"
	OperationWatchedGroupsNextFive  = "watched_groups_next_five"

	WatchedGroupsStatusOK    = "ok"
	WatchedGroupsStatusError = "error"

	WatchedGroupsErrorInvalidGroup       = "invalid_group"
	WatchedGroupsErrorDuplicateGroup     = "duplicate_group"
	WatchedGroupsErrorGroupNotFound      = "group_not_found"
	WatchedGroupsErrorInsufficientActive = "insufficient_active_groups"
	WatchedGroupsErrorInvalidCursor      = "invalid_cursor"
	WatchedGroupsErrorStorage            = "storage_error"

	MaxWatchedGroupsRequestBytes  = 64 * 1024
	MaxWatchedGroupsResponseBytes = 1024 * 1024
)

type WatchedGroupBridgeValue struct {
	ID                   string `json:"id"`
	FacebookGroupID      string `json:"facebook_group_id"`
	CanonicalURL         string `json:"canonical_url"`
	Name                 string `json:"name"`
	CreatedAt            string `json:"created_at"`
	Active               bool   `json:"active"`
	Notes                string `json:"notes"`
	LastSuccessfulScanAt string `json:"last_successful_scan_at"`
	LastError            string `json:"last_error"`
	DisplayOrder         int    `json:"display_order"`
}

type AddWatchedGroupBridgeValue struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CanonicalURL string `json:"canonical_url"`
	CreatedAt    string `json:"created_at"`
}

type WatchedGroupsRequest struct {
	SchemaVersion int                         `json:"schema_version"`
	Operation     string                      `json:"operation"`
	NewGroup      *AddWatchedGroupBridgeValue `json:"new_group,omitempty"`
	GroupID       string                      `json:"group_id,omitempty"`
	Active        *bool                       `json:"active,omitempty"`
}

type WatchedGroupsResponse struct {
	SchemaVersion int                       `json:"schema_version"`
	Operation     string                    `json:"operation"`
	Status        string                    `json:"status"`
	Groups        []WatchedGroupBridgeValue `json:"groups"`
	Selection     []WatchedGroupBridgeValue `json:"selection"`
	CurrentCursor int                       `json:"current_cursor"`
	ErrorCode     string                    `json:"error_code,omitempty"`
}

type WatchedGroupStateRepository interface {
	Load() (persistence.WatchedGroupState, error)
	Add(domain.WatchedGroup) (persistence.WatchedGroupState, error)
	SetActive(string, bool) (persistence.WatchedGroupState, error)
	AdvanceCursor() (persistence.WatchedGroupState, error)
	Close() error
}

type WatchedGroupRepositoryFactory func() (WatchedGroupStateRepository, error)

func productionWatchedGroupRepositoryFactory() (WatchedGroupStateRepository, error) {
	return persistence.OpenProductionSQLiteWatchedGroupRepository()
}

func DecodeWatchedGroupsRequest(reader io.Reader) (WatchedGroupsRequest, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, MaxWatchedGroupsRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > MaxWatchedGroupsRequestBytes {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request WatchedGroupsRequest
	if err := decoder.Decode(&request); err != nil {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return WatchedGroupsRequest{}, ErrMalformedRequest
	}
	if request.SchemaVersion != WatchedGroupsSchemaVersion {
		return WatchedGroupsRequest{}, ErrUnsupportedSchemaVersion
	}
	if !isWatchedGroupsOperation(request.Operation) {
		return WatchedGroupsRequest{}, ErrUnsupportedOperation
	}
	return request, nil
}

func HandleWatchedGroups(repository WatchedGroupStateRepository, request WatchedGroupsRequest) WatchedGroupsResponse {
	var (
		state persistence.WatchedGroupState
		err   error
	)

	switch request.Operation {
	case OperationWatchedGroupsList:
		state, err = repository.Load()
	case OperationWatchedGroupsAdd:
		if request.NewGroup == nil {
			return watchedGroupsErrorResponse(request.Operation, domain.ErrMissingWatchedGroupIdentity)
		}
		var group domain.WatchedGroup
		group, err = groupFromAddBridgeValue(*request.NewGroup)
		if err == nil {
			state, err = repository.Add(group)
		}
	case OperationWatchedGroupsSetActive:
		if request.Active == nil {
			return watchedGroupsErrorResponse(request.Operation, domain.ErrInvalidWatchedGroupID)
		}
		state, err = repository.SetActive(request.GroupID, *request.Active)
	case OperationWatchedGroupsNextFive:
		state, err = repository.AdvanceCursor()
	}
	if err != nil {
		return watchedGroupsErrorResponse(request.Operation, err)
	}
	return watchedGroupsResponseFromState(request.Operation, state)
}

func watchedGroupsResponseFromState(operation string, state persistence.WatchedGroupState) WatchedGroupsResponse {
	response := WatchedGroupsResponse{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     operation,
		Status:        WatchedGroupsStatusOK,
		Groups:        bridgeValuesFromGroups(state.Groups()),
		Selection:     []WatchedGroupBridgeValue{},
		CurrentCursor: state.Cursor().Position(),
	}
	selection, err := application.SelectNextFiveActiveGroups(state.Groups(), state.Cursor())
	if err == nil {
		response.Selection = bridgeValuesFromGroups(selection.Groups())
		return response
	}
	if errors.Is(err, application.ErrEmptyWatchedGroupSelectionCollection) || errors.Is(err, application.ErrInsufficientActiveWatchedGroups) {
		return response
	}
	return watchedGroupsErrorResponse(operation, err)
}

func EncodeWatchedGroupsResponse(response WatchedGroupsResponse) ([]byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(payload)+1 > MaxWatchedGroupsResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return append(payload, '\n'), nil
}

func ServeWatchedGroups(stdin io.Reader, stdout io.Writer, stderr io.Writer, factory WatchedGroupRepositoryFactory) int {
	request, err := DecodeWatchedGroupsRequest(stdin)
	if err != nil {
		writeDiagnostic(stderr, "watched groups request rejected")
		return 2
	}

	repository, err := factory()
	if err != nil {
		writeDiagnostic(stderr, "watched group storage unavailable")
		return writeWatchedGroupsResponse(stdout, watchedGroupsErrorResponse(request.Operation, err))
	}
	defer repository.Close()

	return writeWatchedGroupsResponse(stdout, HandleWatchedGroups(repository, request))
}

func writeWatchedGroupsResponse(stdout io.Writer, response WatchedGroupsResponse) int {
	payload, err := EncodeWatchedGroupsResponse(response)
	if err != nil {
		return 3
	}
	if _, err := stdout.Write(payload); err != nil {
		return 3
	}
	return 0
}

func isWatchedGroupsOperation(operation string) bool {
	switch operation {
	case OperationWatchedGroupsList,
		OperationWatchedGroupsAdd,
		OperationWatchedGroupsSetActive,
		OperationWatchedGroupsNextFive:
		return true
	default:
		return false
	}
}

func groupFromAddBridgeValue(value AddWatchedGroupBridgeValue) (domain.WatchedGroup, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, value.CreatedAt)
	if err != nil {
		return domain.WatchedGroup{}, domain.ErrInvalidWatchedGroupCreatedAt
	}
	return domain.NewWatchedGroup(value.ID, "", value.CanonicalURL, value.Name, createdAt)
}

func bridgeValuesFromGroups(groups []domain.WatchedGroup) []WatchedGroupBridgeValue {
	values := make([]WatchedGroupBridgeValue, len(groups))
	for i, group := range groups {
		lastSuccessful := ""
		if value, ok := group.LastSuccessfulScanAt(); ok {
			lastSuccessful = value.Format(time.RFC3339Nano)
		}
		values[i] = WatchedGroupBridgeValue{
			ID:                   group.ID(),
			FacebookGroupID:      group.FacebookGroupID(),
			CanonicalURL:         group.CanonicalURL(),
			Name:                 group.Name(),
			CreatedAt:            group.CreatedAt().Format(time.RFC3339Nano),
			Active:               group.IsActive(),
			Notes:                group.Notes(),
			LastSuccessfulScanAt: lastSuccessful,
			LastError:            group.LastError(),
			DisplayOrder:         group.DisplayOrder(),
		}
	}
	return values
}

func watchedGroupsErrorResponse(operation string, err error) WatchedGroupsResponse {
	return WatchedGroupsResponse{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     operation,
		Status:        WatchedGroupsStatusError,
		Groups:        []WatchedGroupBridgeValue{},
		Selection:     []WatchedGroupBridgeValue{},
		ErrorCode:     watchedGroupsErrorCode(err),
	}
}

func watchedGroupsErrorCode(err error) string {
	switch {
	case errors.Is(err, application.ErrDuplicateWatchedGroupID),
		errors.Is(err, application.ErrDuplicateWatchedGroupIdentity),
		errors.Is(err, application.ErrDuplicateWatchedGroupSelectionGroup):
		return WatchedGroupsErrorDuplicateGroup
	case errors.Is(err, application.ErrWatchedGroupNotFound):
		return WatchedGroupsErrorGroupNotFound
	case errors.Is(err, application.ErrInsufficientActiveWatchedGroups),
		errors.Is(err, application.ErrEmptyWatchedGroupSelectionCollection):
		return WatchedGroupsErrorInsufficientActive
	case errors.Is(err, application.ErrInvalidWatchedGroupSelectionCursor):
		return WatchedGroupsErrorInvalidCursor
	case errors.Is(err, persistence.ErrInvalidStoredWatchedGroupState):
		return WatchedGroupsErrorStorage
	case errors.Is(err, domain.ErrInvalidWatchedGroupID),
		errors.Is(err, domain.ErrInvalidWatchedGroupName),
		errors.Is(err, domain.ErrInvalidWatchedGroupCreatedAt),
		errors.Is(err, domain.ErrMissingWatchedGroupIdentity),
		errors.Is(err, domain.ErrInvalidWatchedGroupCanonicalURL),
		errors.Is(err, domain.ErrWatchedGroupScanBeforeCreated):
		return WatchedGroupsErrorInvalidGroup
	default:
		return WatchedGroupsErrorStorage
	}
}
