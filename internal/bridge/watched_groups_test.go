package bridge

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestWatchedGroupsListPreservesInsertionOrderAndActiveState(t *testing.T) {
	groups := []WatchedGroupBridgeValue{
		bridgeGroup("group-c", "Group C", true),
		bridgeGroup("group-a", "Group A", false),
		bridgeGroup("group-b", "Group B", true),
	}

	response := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsList,
		Groups:        groups,
	})

	if response.Status != WatchedGroupsStatusOK || !reflect.DeepEqual(response.Groups, groups) {
		t.Fatalf("list response = %#v, want groups %#v", response, groups)
	}
}

func TestWatchedGroupsAddUsesDomainValidationAndStartsActive(t *testing.T) {
	response := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsAdd,
		NewGroup: &AddWatchedGroupBridgeValue{
			ID:           "group-a",
			Name:         "Group A",
			CanonicalURL: "https://www.facebook.com/groups/group-a",
			CreatedAt:    bridgeCreatedAt(),
		},
	})

	if response.Status != WatchedGroupsStatusOK || len(response.Groups) != 1 {
		t.Fatalf("add response = %#v", response)
	}
	if !response.Groups[0].Active || response.Groups[0].Name != "Group A" {
		t.Fatalf("added group = %#v, want active Group A", response.Groups[0])
	}
}

func TestWatchedGroupsAddRejectsInvalidAndDuplicateCanonicalURL(t *testing.T) {
	invalid := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsAdd,
		NewGroup: &AddWatchedGroupBridgeValue{
			ID:           "group-a",
			Name:         "Group A",
			CanonicalURL: "http://www.facebook.com/groups/group-a",
			CreatedAt:    bridgeCreatedAt(),
		},
	})
	if invalid.Status != WatchedGroupsStatusError || invalid.ErrorCode != WatchedGroupsErrorInvalidGroup {
		t.Fatalf("invalid add response = %#v", invalid)
	}

	existing := bridgeGroup("group-a", "Group A", true)
	duplicate := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsAdd,
		Groups:        []WatchedGroupBridgeValue{existing},
		NewGroup: &AddWatchedGroupBridgeValue{
			ID:           "group-b",
			Name:         "Group B",
			CanonicalURL: existing.CanonicalURL,
			CreatedAt:    bridgeCreatedAt(),
		},
	})
	if duplicate.Status != WatchedGroupsStatusError || duplicate.ErrorCode != WatchedGroupsErrorDuplicateGroup {
		t.Fatalf("duplicate add response = %#v", duplicate)
	}
}

func TestWatchedGroupsSetActiveUsesCollectionOperation(t *testing.T) {
	active := false
	response := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsSetActive,
		Groups:        []WatchedGroupBridgeValue{bridgeGroup("group-a", "Group A", true)},
		GroupID:       "group-a",
		Active:        &active,
	})

	if response.Status != WatchedGroupsStatusOK || len(response.Groups) != 1 || response.Groups[0].Active {
		t.Fatalf("set-active response = %#v", response)
	}
}

func TestWatchedGroupsNextFiveReturnsExactGoSelectionOrder(t *testing.T) {
	groups := bridgeGroups(7)
	response := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsNextFive,
		Groups:        groups,
		Cursor:        5,
	})

	if response.Status != WatchedGroupsStatusOK || response.NextCursor == nil || *response.NextCursor != 3 {
		t.Fatalf("selection response = %#v", response)
	}
	assertBridgeGroupIDs(t, response.Selection, []string{"group-f", "group-g", "group-a", "group-b", "group-c"})
}

func TestWatchedGroupsNextFiveSkipsInactiveAndDoesNotReturnPartialSelection(t *testing.T) {
	groups := bridgeGroups(7)
	groups[1].Active = false
	selected := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsNextFive,
		Groups:        groups,
	})
	assertBridgeGroupIDs(t, selected.Selection, []string{"group-a", "group-c", "group-d", "group-e", "group-f"})

	groups[4].Active = false
	groups[5].Active = false
	insufficient := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsNextFive,
		Groups:        groups,
	})
	if insufficient.Status != WatchedGroupsStatusError || insufficient.ErrorCode != WatchedGroupsErrorInsufficientActive {
		t.Fatalf("insufficient response = %#v", insufficient)
	}
	if len(insufficient.Selection) != 0 || insufficient.NextCursor != nil {
		t.Fatalf("insufficient response contains partial selection: %#v", insufficient)
	}
}

func TestWatchedGroupsSnapshotFailureDoesNotMutateRequest(t *testing.T) {
	groups := bridgeGroups(5)
	groups[4].CanonicalURL = groups[0].CanonicalURL
	before := append([]WatchedGroupBridgeValue(nil), groups...)

	response := HandleWatchedGroups(WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsNextFive,
		Groups:        groups,
	})

	if response.Status != WatchedGroupsStatusError || !reflect.DeepEqual(groups, before) {
		t.Fatalf("malformed snapshot response=%#v mutated=%v", response, !reflect.DeepEqual(groups, before))
	}
}

func TestServeDispatchesWatchedGroupsWithoutDiagnosticsOnSuccess(t *testing.T) {
	request := WatchedGroupsRequest{
		SchemaVersion: SchemaVersion,
		Operation:     OperationWatchedGroupsList,
		Groups:        []WatchedGroupBridgeValue{},
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := Serve(bytes.NewReader(payload), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("Serve exit=%d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var response WatchedGroupsResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("stdout is not watched-groups JSON: %v", err)
	}
	if response.Status != WatchedGroupsStatusOK || response.Operation != OperationWatchedGroupsList {
		t.Fatalf("response = %#v", response)
	}
}

func bridgeGroups(count int) []WatchedGroupBridgeValue {
	groups := make([]WatchedGroupBridgeValue, count)
	for i := range groups {
		id := "group-" + string(rune('a'+i))
		groups[i] = bridgeGroup(id, "Group "+string(rune('A'+i)), true)
	}
	return groups
}

func bridgeGroup(id string, name string, active bool) WatchedGroupBridgeValue {
	return WatchedGroupBridgeValue{
		ID:           id,
		Name:         name,
		CanonicalURL: "https://www.facebook.com/groups/" + id,
		CreatedAt:    bridgeCreatedAt(),
		Active:       active,
	}
}

func bridgeCreatedAt() string {
	return time.Date(2026, time.August, 12, 9, 0, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)).Format(time.RFC3339Nano)
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
