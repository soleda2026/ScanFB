package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/persistence"
)

func TestWatchedGroupsPersistentFlowRestoresAndAdvancesAuthoritativeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched-groups.sqlite3")
	for i := 0; i < 6; i++ {
		id := "group-" + string(rune('a'+i))
		response := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
			SchemaVersion: WatchedGroupsSchemaVersion,
			Operation:     OperationWatchedGroupsAdd,
			NewGroup: &AddWatchedGroupBridgeValue{
				ID:           id,
				Name:         "Group " + string(rune('A'+i)),
				CanonicalURL: "https://www.facebook.com/groups/" + id,
				CreatedAt:    bridgeCreatedAt(i),
			},
		})
		if response.Status != WatchedGroupsStatusOK || len(response.Groups) != i+1 {
			t.Fatalf("add %d response = %#v", i, response)
		}
	}

	listed := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsList,
	})
	assertBridgeGroupIDs(t, listed.Groups, []string{"group-a", "group-b", "group-c", "group-d", "group-e", "group-f"})
	assertBridgeGroupIDs(t, listed.Selection, []string{"group-a", "group-b", "group-c", "group-d", "group-e"})
	if listed.CurrentCursor != 0 {
		t.Fatalf("listed cursor = %d, want 0", listed.CurrentCursor)
	}

	advanced := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsNextFive,
	})
	if advanced.CurrentCursor != 5 {
		t.Fatalf("advanced cursor = %d, want 5", advanced.CurrentCursor)
	}
	assertBridgeGroupIDs(t, advanced.Selection, []string{"group-f", "group-a", "group-b", "group-c", "group-d"})

	restored := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsList,
	})
	if !reflect.DeepEqual(restored, advanced) {
		restored.Operation = advanced.Operation
		if !reflect.DeepEqual(restored, advanced) {
			t.Fatalf("restored response = %#v, want %#v", restored, advanced)
		}
	}
}

func TestWatchedGroupsSetActivePersistsAndRefreshesSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched-groups.sqlite3")
	for i := 0; i < 6; i++ {
		id := "group-" + string(rune('a'+i))
		serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
			SchemaVersion: WatchedGroupsSchemaVersion,
			Operation:     OperationWatchedGroupsAdd,
			NewGroup: &AddWatchedGroupBridgeValue{
				ID: id, Name: id, CanonicalURL: "https://www.facebook.com/groups/" + id, CreatedAt: bridgeCreatedAt(i),
			},
		})
	}
	active := false
	updated := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsSetActive,
		GroupID:       "group-b",
		Active:        &active,
	})
	if updated.Groups[1].Active {
		t.Fatal("group-b remains active")
	}
	assertBridgeGroupIDs(t, updated.Selection, []string{"group-a", "group-c", "group-d", "group-e", "group-f"})

	active = true
	reactivated := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsSetActive,
		GroupID:       "group-b",
		Active:        &active,
	})
	if !reactivated.Groups[1].Active {
		t.Fatal("group-b was not reactivated")
	}
}

func TestWatchedGroupsListPreservesFullPersistentMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched-groups.sqlite3")
	repo, err := persistence.OpenSQLiteWatchedGroupRepository(path)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	createdAt := time.Date(2026, time.August, 12, 9, 0, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
	group, err := domain.NewWatchedGroup("local-a", "facebook-a", "https://www.facebook.com/groups/a", "Group A", createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	group, err = group.WithMetadata(domain.WatchedGroupMetadata{
		Name: "Group A", Notes: "note", LastSuccessfulScanAt: createdAt.Add(time.Hour), LastError: "last error", DisplayOrder: 9,
	})
	if err != nil {
		t.Fatalf("WithMetadata() error = %v", err)
	}
	if _, err := repo.Add(group.WithActive(false)); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	response := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsList})
	if len(response.Groups) != 1 {
		t.Fatalf("groups = %#v", response.Groups)
	}
	got := response.Groups[0]
	if got.FacebookGroupID != "facebook-a" || got.Notes != "note" || got.LastSuccessfulScanAt == "" || got.LastError != "last error" || got.DisplayOrder != 9 || got.Active {
		t.Fatalf("full metadata response = %#v", got)
	}
}

func TestWatchedGroupsInsufficientActiveIsSuccessfulWithoutPartialSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched-groups.sqlite3")
	for i := 0; i < 4; i++ {
		id := "group-" + string(rune('a'+i))
		serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
			SchemaVersion: WatchedGroupsSchemaVersion,
			Operation:     OperationWatchedGroupsAdd,
			NewGroup:      &AddWatchedGroupBridgeValue{ID: id, Name: id, CanonicalURL: "https://www.facebook.com/groups/" + id, CreatedAt: bridgeCreatedAt(i)},
		})
	}
	response := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsList})
	if response.Status != WatchedGroupsStatusOK || len(response.Selection) != 0 || response.CurrentCursor != 0 {
		t.Fatalf("insufficient response = %#v", response)
	}
}

func TestWatchedGroupsAddRejectsInvalidAndDuplicateWithoutOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched-groups.sqlite3")
	invalid := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsAdd,
		NewGroup:      &AddWatchedGroupBridgeValue{ID: "a", Name: "A", CanonicalURL: "http://example.com/a", CreatedAt: bridgeCreatedAt(0)},
	})
	if invalid.ErrorCode != WatchedGroupsErrorInvalidGroup {
		t.Fatalf("invalid add = %#v", invalid)
	}
	first := AddWatchedGroupBridgeValue{ID: "a", Name: "A", CanonicalURL: "https://example.com/shared", CreatedAt: bridgeCreatedAt(0)}
	serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsAdd, NewGroup: &first})
	duplicate := AddWatchedGroupBridgeValue{ID: "b", Name: "B", CanonicalURL: first.CanonicalURL, CreatedAt: bridgeCreatedAt(1)}
	response := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsAdd, NewGroup: &duplicate})
	if response.ErrorCode != WatchedGroupsErrorDuplicateGroup {
		t.Fatalf("duplicate add = %#v", response)
	}
	listed := serveWatchedGroupsRequest(t, path, WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsList})
	if len(listed.Groups) != 1 || listed.Groups[0].ID != "a" {
		t.Fatalf("failed duplicate mutated state: %#v", listed)
	}
}

func TestWatchedGroupsRequestRejectsClientOwnedStateAndPaths(t *testing.T) {
	for _, payload := range []string{
		`{"schema_version":2,"operation":"watched_groups_list","groups":[]}`,
		`{"schema_version":2,"operation":"watched_groups_list","cursor":0}`,
		`{"schema_version":2,"operation":"watched_groups_list","database_path":"/tmp/state.sqlite3"}`,
	} {
		if _, err := DecodeWatchedGroupsRequest(strings.NewReader(payload)); !errors.Is(err, ErrMalformedRequest) {
			t.Fatalf("DecodeWatchedGroupsRequest(%s) error = %v, want malformed", payload, err)
		}
	}

	request := WatchedGroupsRequest{SchemaVersion: WatchedGroupsSchemaVersion, Operation: OperationWatchedGroupsList}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(payload, &keys); err != nil {
		t.Fatalf("Unmarshal(request keys) error = %v", err)
	}
	for _, forbidden := range []string{"groups", "cursor", "database_path"} {
		if _, exists := keys[forbidden]; exists {
			t.Fatalf("request payload contains key %q: %s", forbidden, payload)
		}
	}
}

func TestWatchedGroupsRequestAndResponseRemainBounded(t *testing.T) {
	oversizedRequest := strings.Repeat(" ", MaxWatchedGroupsRequestBytes+1)
	if _, err := DecodeWatchedGroupsRequest(strings.NewReader(oversizedRequest)); !errors.Is(err, ErrMalformedRequest) {
		t.Fatalf("oversized request error = %v, want malformed", err)
	}

	response := WatchedGroupsResponse{
		SchemaVersion: WatchedGroupsSchemaVersion,
		Operation:     OperationWatchedGroupsList,
		Status:        WatchedGroupsStatusOK,
		Groups: []WatchedGroupBridgeValue{{
			ID:    "group-a",
			Notes: strings.Repeat("x", MaxWatchedGroupsResponseBytes),
		}},
		Selection: []WatchedGroupBridgeValue{},
	}
	if _, err := EncodeWatchedGroupsResponse(response); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("oversized response error = %v, want response too large", err)
	}
}

func TestMalformedPersistentStateMapsToStorageError(t *testing.T) {
	if got := watchedGroupsErrorCode(persistence.ErrInvalidStoredWatchedGroupState); got != WatchedGroupsErrorStorage {
		t.Fatalf("error code = %q, want %q", got, WatchedGroupsErrorStorage)
	}
}

func TestWatchedGroupsStorageFailureReturnsTypedRedactedError(t *testing.T) {
	payload := []byte(`{"schema_version":2,"operation":"watched_groups_list"}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := ServeWithWatchedGroupRepositoryFactory(bytes.NewReader(payload), &stdout, &stderr, func() (WatchedGroupStateRepository, error) {
		return nil, errors.New("private /Users/name/state.sqlite3 open failed")
	})
	if exit != 0 {
		t.Fatalf("exit = %d, stderr=%q", exit, stderr.String())
	}
	var response WatchedGroupsResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ErrorCode != WatchedGroupsErrorStorage || response.Status != WatchedGroupsStatusError {
		t.Fatalf("response = %#v", response)
	}
	if strings.Contains(stdout.String()+stderr.String(), "/Users/") {
		t.Fatalf("diagnostics leaked raw path: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func serveWatchedGroupsRequest(t *testing.T, path string, request WatchedGroupsRequest) WatchedGroupsResponse {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal(request) error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := ServeWithWatchedGroupRepositoryFactory(bytes.NewReader(payload), &stdout, &stderr, func() (WatchedGroupStateRepository, error) {
		return persistence.OpenSQLiteWatchedGroupRepository(path)
	})
	if exit != 0 {
		t.Fatalf("Serve() exit=%d stdout=%q stderr=%q", exit, stdout.String(), stderr.String())
	}
	var response WatchedGroupsResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("Unmarshal(response) error = %v; stdout=%q", err, stdout.String())
	}
	return response
}

func bridgeCreatedAt(index int) string {
	return time.Date(2026, time.August, 12, 9, index, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)).Format(time.RFC3339Nano)
}

func assertBridgeGroupIDs(t *testing.T, groups []WatchedGroupBridgeValue, want []string) {
	t.Helper()
	if len(groups) != len(want) {
		t.Fatalf("group count = %d, want %d", len(groups), len(want))
	}
	for i, id := range want {
		if groups[i].ID != id {
			t.Fatalf("group[%d].ID = %q, want %q", i, groups[i].ID, id)
		}
	}
}
