package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/orchestration"
	"github.com/soleda2026/ScanFB/internal/persistence"
)

func TestPreparedGroupScanRunsOneAuthoritativeActiveGroupAndMapsSummary(t *testing.T) {
	repository := preparedGroupScanRepository(t, true)
	defer repository.Close()
	before, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() before scan error = %v", err)
	}

	invocations := 0
	runner := func(
		ctx context.Context,
		request orchestration.OneGroupScanRequest,
		collector orchestration.GroupPostCollector,
	) (orchestration.OneGroupScanResult, error) {
		invocations++
		if request.WatchedGroup.ID() != "group-authoritative" || request.WatchedGroup.Name() != "Authoritative Group" {
			t.Fatalf("runner group = %q %q", request.WatchedGroup.ID(), request.WatchedGroup.Name())
		}
		if request.SearchProfile.ID() != "macbook" {
			t.Fatalf("runner profile = %q", request.SearchProfile.ID())
		}
		if request.GeographicMode != domain.GeographicModeHoChiMinhCity {
			t.Fatalf("runner geographic mode = %q", request.GeographicMode)
		}
		return orchestration.RunOneGroupScan(ctx, request, collector)
	}

	response := HandlePreparedGroupScan(context.Background(), repository, validPreparedGroupScanRequest(), runner)
	if response.Status != PreparedGroupScanStatusOK || response.AttemptStatus != "succeeded" {
		t.Fatalf("response status = %#v", response)
	}
	if response.GroupName != "Authoritative Group" {
		t.Fatalf("group name = %q", response.GroupName)
	}
	if response.CollectedPostCount != 1 || response.EvaluatedPostCount != 1 ||
		response.IncludedPostCount != 1 || response.ReviewPostCount != 0 ||
		response.ExcludedPostCount != 0 || response.AllowedLeadCount != 1 {
		t.Fatalf("response summary = %#v", response)
	}
	if invocations != 1 {
		t.Fatalf("runner invocations = %d, want 1", invocations)
	}

	after, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() after scan error = %v", err)
	}
	if !reflect.DeepEqual(before.Groups(), after.Groups()) || before.Cursor() != after.Cursor() {
		t.Fatalf("prepared scan mutated watched group state or cursor")
	}
}

func TestPreparedGroupScanRejectsMissingAndInactiveGroupsBeforeRunner(t *testing.T) {
	tests := []struct {
		name      string
		active    bool
		groupID   string
		wantError string
	}{
		{name: "missing", active: true, groupID: "missing", wantError: PreparedGroupScanErrorGroupNotFound},
		{name: "inactive", active: false, groupID: "group-authoritative", wantError: PreparedGroupScanErrorInactiveGroup},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := preparedGroupScanRepository(t, test.active)
			defer repository.Close()
			request := validPreparedGroupScanRequest()
			request.GroupID = test.groupID
			invocations := 0
			response := HandlePreparedGroupScan(
				context.Background(),
				repository,
				request,
				func(context.Context, orchestration.OneGroupScanRequest, orchestration.GroupPostCollector) (orchestration.OneGroupScanResult, error) {
					invocations++
					return orchestration.OneGroupScanResult{}, nil
				},
			)
			if response.Status != PreparedGroupScanStatusError || response.ErrorCode != test.wantError {
				t.Fatalf("response = %#v", response)
			}
			if invocations != 0 {
				t.Fatalf("runner invoked %d times", invocations)
			}
		})
	}
}

func TestPreparedGroupScanInvalidPayloadFailsWithoutPartialSuccessOrRetry(t *testing.T) {
	repository := preparedGroupScanRepository(t, true)
	defer repository.Close()
	request := validPreparedGroupScanRequest()
	request.PreparedSnapshot = json.RawMessage(`{"schema_version":1,"posts":[]}`)
	invocations := 0
	runner := func(
		ctx context.Context,
		request orchestration.OneGroupScanRequest,
		collector orchestration.GroupPostCollector,
	) (orchestration.OneGroupScanResult, error) {
		invocations++
		return orchestration.RunOneGroupScan(ctx, request, collector)
	}

	response := HandlePreparedGroupScan(context.Background(), repository, request, runner)
	if response.Status != PreparedGroupScanStatusError || response.ErrorCode != PreparedGroupScanErrorInvalidPayload {
		t.Fatalf("response = %#v", response)
	}
	if response.AttemptStatus != "failed" || response.CollectedPostCount != 0 ||
		response.EvaluatedPostCount != 0 || response.IncludedPostCount != 0 ||
		response.ReviewPostCount != 0 || response.ExcludedPostCount != 0 ||
		response.AllowedLeadCount != 0 {
		t.Fatalf("failure exposed partial success = %#v", response)
	}
	if invocations != 1 {
		t.Fatalf("runner invocations = %d, want exactly 1", invocations)
	}
}

func TestPreparedGroupScanRequestAndResponseRemainBounded(t *testing.T) {
	request := validPreparedGroupScanRequest()
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	decoded, err := DecodePreparedGroupScanRequest(bytes.NewReader(encoded))
	if err != nil || decoded.GroupID != request.GroupID {
		t.Fatalf("DecodePreparedGroupScanRequest() = %#v, %v", decoded, err)
	}

	oversized := append(bytes.Repeat([]byte{' '}, MaxPreparedGroupScanRequestBytes), 'x')
	if _, err := DecodePreparedGroupScanRequest(bytes.NewReader(oversized)); !errors.Is(err, ErrMalformedRequest) {
		t.Fatalf("oversized request error = %v", err)
	}

	response := preparedGroupScanErrorResponse(PreparedGroupScanErrorScanFailed)
	response.ErrorMessage = strings.Repeat("x", MaxPreparedGroupScanResponseBytes)
	if _, err := EncodePreparedGroupScanResponse(response); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("oversized response error = %v", err)
	}
}

func TestPreparedGroupScanDiagnosticsDoNotContainPrivatePayload(t *testing.T) {
	privateValue := "PRIVATE-BODY-AUTHOR-URL"
	request := validPreparedGroupScanRequest()
	request.SchemaVersion = 99
	request.PreparedSnapshot = json.RawMessage(`{"private":"` + privateValue + `"}`)
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := ServePreparedGroupScan(
		bytes.NewReader(payload),
		&stdout,
		&stderr,
		func() (WatchedGroupStateRepository, error) { return nil, errors.New("must not open") },
		orchestration.RunOneGroupScan,
	)
	if exitCode != 2 || strings.Contains(stderr.String(), privateValue) || strings.Contains(stdout.String(), privateValue) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func preparedGroupScanRepository(t *testing.T, active bool) *persistence.SQLiteWatchedGroupRepository {
	t.Helper()
	repository, err := persistence.OpenSQLiteWatchedGroupRepository(filepath.Join(t.TempDir(), "watched-groups.sqlite3"))
	if err != nil {
		t.Fatalf("OpenSQLiteWatchedGroupRepository() error = %v", err)
	}
	createdAt := preparedGroupScanTime(t, 8, 0)
	group, err := domain.NewWatchedGroup(
		"group-authoritative",
		"facebook-authoritative",
		"https://www.facebook.com/groups/authoritative",
		"Authoritative Group",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	if _, err := repository.Add(group.WithActive(active)); err != nil {
		repository.Close()
		t.Fatalf("Add() error = %v", err)
	}
	return repository
}

func validPreparedGroupScanRequest() PreparedGroupScanRequest {
	return PreparedGroupScanRequest{
		SchemaVersion: PreparedGroupScanSchemaVersion,
		Operation:     OperationPreparedGroupScan,
		GroupID:       "group-authoritative",
		ScanID:        "scan-manual-001",
		AttemptID:     "attempt-manual-001",
		ActionAt:      "2026-08-13T10:00:00+07:00",
		PreparedSnapshot: json.RawMessage(`{
			"schema_version":1,
			"posts":[{
				"post_id":"post-001",
				"post_url":"https://www.facebook.com/groups/authoritative/posts/post-001",
				"author":{"facebook_user_id":"buyer-001","canonical_profile_url":"","username":"","display_name":"Buyer One"},
				"body":"Can mua MacBook tai HCM",
				"created_at":"2026-08-13T09:00:00+07:00"
			}]
		}`),
	}
}

func preparedGroupScanTime(t *testing.T, hour int, minute int) time.Time {
	t.Helper()
	location, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	return time.Date(2026, 8, 13, hour, minute, 0, 0, location)
}
