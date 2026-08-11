package application

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestWatchedGroupSelectionInitialCursorStartsAtCollectionBeginning(t *testing.T) {
	cursor := InitialWatchedGroupSelectionCursor()
	if cursor.Position() != 0 {
		t.Fatalf("initial cursor position = %d, want 0", cursor.Position())
	}
}

func TestNewWatchedGroupSelectionCursorRejectsNegativePosition(t *testing.T) {
	if _, err := NewWatchedGroupSelectionCursor(-1); !errors.Is(err, ErrInvalidWatchedGroupSelectionCursor) {
		t.Fatalf("NewWatchedGroupSelectionCursor() error = %v", err)
	}
}

func TestSelectNextFiveActiveGroupsSelectsExactlyFiveInInsertionOrder(t *testing.T) {
	groups := phase9CGroups(t, 5)
	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}

	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
	if selection.NextCursor().Position() != 0 {
		t.Fatalf("next cursor position = %d, want 0", selection.NextCursor().Position())
	}
}

func TestSelectNextFiveActiveGroupsFromLargerCollection(t *testing.T) {
	selection, err := SelectNextFiveActiveGroups(phase9CGroups(t, 7), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}

	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
	if selection.NextCursor().Position() != 5 {
		t.Fatalf("next cursor position = %d, want 5", selection.NextCursor().Position())
	}
}

func TestSelectNextFiveActiveGroupsSecondSelectionContinuesRoundRobin(t *testing.T) {
	groups := phase9CGroups(t, 7)
	first, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("first selection error = %v", err)
	}
	second, err := SelectNextFiveActiveGroups(groups, first.NextCursor())
	if err != nil {
		t.Fatalf("second selection error = %v", err)
	}

	assertPhase9CGroupIDs(t, second.Groups(), []string{"group-F", "group-G", "group-A", "group-B", "group-C"})
	if second.NextCursor().Position() != 3 {
		t.Fatalf("second next cursor position = %d, want 3", second.NextCursor().Position())
	}
}

func TestSelectNextFiveActiveGroupsWrapsAtCollectionEnd(t *testing.T) {
	cursor := mustPhase9CCursor(t, 4)
	selection, err := SelectNextFiveActiveGroups(phase9CGroups(t, 7), cursor)
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}

	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-E", "group-F", "group-G", "group-A", "group-B"})
	if selection.NextCursor().Position() != 2 {
		t.Fatalf("next cursor position = %d, want 2", selection.NextCursor().Position())
	}
}

func TestSelectNextFiveActiveGroupsSkipsInactiveGroups(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 7))
	if _, err := collection.Deactivate("group-B"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	selection, err := SelectNextFiveActiveGroups(collection.Groups(), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-C", "group-D", "group-E", "group-F"})
	if selection.NextCursor().Position() != 6 {
		t.Fatalf("next cursor position = %d, want 6", selection.NextCursor().Position())
	}
}

func TestSelectNextFiveActiveGroupsKeepsInactivePositionsInCursorGeometry(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 7))
	if _, err := collection.Deactivate("group-F"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	first, err := SelectNextFiveActiveGroups(collection.Groups(), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("first selection error = %v", err)
	}
	assertPhase9CGroupIDs(t, first.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
	if first.NextCursor().Position() != 5 {
		t.Fatalf("next cursor position = %d, want inactive position 5", first.NextCursor().Position())
	}

	second, err := SelectNextFiveActiveGroups(collection.Groups(), first.NextCursor())
	if err != nil {
		t.Fatalf("second selection error = %v", err)
	}
	assertPhase9CGroupIDs(t, second.Groups(), []string{"group-G", "group-A", "group-B", "group-C", "group-D"})
}

func TestSelectNextFiveActiveGroupsFailsWithoutPartialSelectionWhenInsufficient(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 7))
	for _, id := range []string{"group-E", "group-F", "group-G"} {
		if _, err := collection.Deactivate(id); err != nil {
			t.Fatalf("Deactivate(%q) error = %v", id, err)
		}
	}
	cursor := mustPhase9CCursor(t, 3)

	selection, err := SelectNextFiveActiveGroups(collection.Groups(), cursor)
	if !errors.Is(err, ErrInsufficientActiveWatchedGroups) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 {
		t.Fatalf("insufficient selection returned %d partial groups", len(selection.Groups()))
	}
	if cursor.Position() != 3 {
		t.Fatalf("caller cursor advanced to %d", cursor.Position())
	}
}

func TestSelectNextFiveActiveGroupsSucceedsWithExactlyFiveActiveAmongStoredGroups(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 8))
	for _, id := range []string{"group-B", "group-E", "group-H"} {
		if _, err := collection.Deactivate(id); err != nil {
			t.Fatalf("Deactivate(%q) error = %v", id, err)
		}
	}

	selection, err := SelectNextFiveActiveGroups(collection.Groups(), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-C", "group-D", "group-F", "group-G"})
	assertPhase9CNoDuplicateIDs(t, selection.Groups())
}

func TestSelectNextFiveActiveGroupsRejectsDuplicateLocalIDAfterSelectionBoundary(t *testing.T) {
	groups := phase9CGroups(t, 6)
	groups[5] = phase9CGroup(t, groups[0].ID(), true, phase9CCreatedAt(), "distinct-facebook-identity", "")
	before := append([]domain.WatchedGroup(nil), groups...)
	cursor := InitialWatchedGroupSelectionCursor()

	selection, err := SelectNextFiveActiveGroups(groups, cursor)
	if !errors.Is(err, ErrDuplicateWatchedGroupSelectionGroup) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 || !reflect.DeepEqual(groups, before) {
		t.Fatal("duplicate local ID returned a partial selection or mutated input")
	}
}

func TestSelectNextFiveActiveGroupsRejectsDuplicateFacebookIdentityAfterSelectionBoundary(t *testing.T) {
	groups := phase9CGroups(t, 6)
	groups[5] = phase9CGroup(t, "group-F", true, phase9CCreatedAt(), groups[0].FacebookGroupID(), "")
	before := append([]domain.WatchedGroup(nil), groups...)

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if !errors.Is(err, ErrDuplicateWatchedGroupSelectionGroup) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 || !reflect.DeepEqual(groups, before) {
		t.Fatal("duplicate Facebook identity returned a partial selection or mutated input")
	}
}

func TestSelectNextFiveActiveGroupsRejectsDuplicateURLOnlyCanonicalIdentity(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/shared-url-only"
	groups := phase9CGroups(t, 6)
	groups[0] = phase9CGroup(t, "group-A", true, phase9CCreatedAt(), "", canonicalURL)
	groups[5] = phase9CGroup(t, "group-F", true, phase9CCreatedAt(), "", canonicalURL)
	before := append([]domain.WatchedGroup(nil), groups...)

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if !errors.Is(err, ErrDuplicateWatchedGroupSelectionGroup) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 || !reflect.DeepEqual(groups, before) {
		t.Fatal("duplicate URL-only identity returned a partial selection or mutated input")
	}
}

func TestSelectNextFiveActiveGroupsRejectsCrossKindCanonicalIdentityConflict(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/shared-cross-kind"
	for _, tc := range []struct {
		name     string
		firstID  string
		secondID string
		firstFB  string
		secondFB string
	}{
		{name: "URL-only then Facebook ID", firstID: "group-A", secondID: "group-F", secondFB: "facebook-F"},
		{name: "Facebook ID then URL-only", firstID: "group-A", secondID: "group-F", firstFB: "facebook-A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groups := phase9CGroups(t, 6)
			groups[0] = phase9CGroup(t, tc.firstID, true, phase9CCreatedAt(), tc.firstFB, canonicalURL)
			groups[5] = phase9CGroup(t, tc.secondID, true, phase9CCreatedAt(), tc.secondFB, canonicalURL)
			before := append([]domain.WatchedGroup(nil), groups...)

			selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
			if !errors.Is(err, ErrDuplicateWatchedGroupSelectionGroup) {
				t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
			}
			if len(selection.Groups()) != 0 || !reflect.DeepEqual(groups, before) {
				t.Fatal("cross-kind canonical conflict returned a partial selection or mutated input")
			}
		})
	}
}

func TestSelectNextFiveActiveGroupsAllowsDifferentFacebookIdentitiesWithSameSecondaryCanonicalURL(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/shared-secondary-url"
	groups := phase9CGroups(t, 5)
	groups[0] = phase9CGroup(t, "group-A", true, phase9CCreatedAt(), "facebook-A", canonicalURL)
	groups[1] = phase9CGroup(t, "group-B", true, phase9CCreatedAt(), "facebook-B", canonicalURL)

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
}

func TestSelectNextFiveActiveGroupsAllInactiveFails(t *testing.T) {
	groups := phase9CGroups(t, 5)
	for i := range groups {
		groups[i] = groups[i].WithActive(false)
	}

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if !errors.Is(err, ErrInsufficientActiveWatchedGroups) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 {
		t.Fatalf("all-inactive selection returned %d groups", len(selection.Groups()))
	}
}

func TestSelectNextFiveActiveGroupsEmptyCollectionFails(t *testing.T) {
	selection, err := SelectNextFiveActiveGroups(nil, InitialWatchedGroupSelectionCursor())
	if !errors.Is(err, ErrEmptyWatchedGroupSelectionCollection) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 {
		t.Fatalf("empty selection returned %d groups", len(selection.Groups()))
	}
}

func TestSelectNextFiveActiveGroupsRejectsCursorPastCollectionBounds(t *testing.T) {
	groups := phase9CGroups(t, 7)
	cursor := mustPhase9CCursor(t, len(groups))
	before := append([]domain.WatchedGroup(nil), groups...)

	selection, err := SelectNextFiveActiveGroups(groups, cursor)
	if !errors.Is(err, ErrInvalidWatchedGroupSelectionCursor) {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	if len(selection.Groups()) != 0 || !reflect.DeepEqual(groups, before) {
		t.Fatal("invalid cursor returned a partial selection or mutated input")
	}
}

func TestSelectNextFiveActiveGroupsIsDeterministicAndDoesNotMutateCollection(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 7))
	if _, err := collection.Deactivate("group-C"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	before := collection.Groups()
	cursor := mustPhase9CCursor(t, 4)

	first, err := SelectNextFiveActiveGroups(collection.Groups(), cursor)
	if err != nil {
		t.Fatalf("first selection error = %v", err)
	}
	second, err := SelectNextFiveActiveGroups(collection.Groups(), cursor)
	if err != nil {
		t.Fatalf("second selection error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same input and cursor produced different results: first=%#v second=%#v", first, second)
	}
	if after := collection.Groups(); !reflect.DeepEqual(after, before) {
		t.Fatalf("selection mutated collection: before=%#v after=%#v", before, after)
	}
}

func TestFiveGroupSelectionReturnsDefensiveGroupSnapshots(t *testing.T) {
	selection, err := SelectNextFiveActiveGroups(phase9CGroups(t, 7), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	want := []string{"group-A", "group-B", "group-C", "group-D", "group-E"}

	first := selection.Groups()
	first[0] = phase9CGroup(t, "replacement", true, phase9CCreatedAt(), "facebook-replacement", "")
	first = append(first, phase9CGroup(t, "extra", true, phase9CCreatedAt(), "facebook-extra", ""))

	assertPhase9CGroupIDs(t, selection.Groups(), want)
}

func TestSelectNextFiveActiveGroupsIgnoresDisplayOrder(t *testing.T) {
	groups := phase9CGroups(t, 7)
	for i := range groups {
		updated, err := groups[i].WithMetadata(domain.WatchedGroupMetadata{
			Name:         groups[i].Name(),
			DisplayOrder: 100 - i,
		})
		if err != nil {
			t.Fatalf("WithMetadata() error = %v", err)
		}
		groups[i] = updated
	}

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
}

func TestSelectNextFiveActiveGroupsIgnoresCreatedAndLastSuccessfulTimes(t *testing.T) {
	groups := make([]domain.WatchedGroup, 7)
	for i := range groups {
		createdAt := phase9CCreatedAt().Add(time.Duration(7-i) * time.Hour)
		group := phase9CGroup(t, fmt.Sprintf("group-%c", 'A'+i), true, createdAt, fmt.Sprintf("facebook-%d", i), "")
		updated, err := group.WithMetadata(domain.WatchedGroupMetadata{
			Name:                 group.Name(),
			LastSuccessfulScanAt: createdAt.Add(time.Duration(7-i) * time.Hour),
		})
		if err != nil {
			t.Fatalf("WithMetadata() error = %v", err)
		}
		groups[i] = updated
	}

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
}

func TestSelectNextFiveActiveGroupsRespectsDeactivateAndReactivateAtInsertionPosition(t *testing.T) {
	collection := phase9CCollection(t, phase9CGroups(t, 7))
	if _, err := collection.Deactivate("group-B"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	inactiveSelection, err := SelectNextFiveActiveGroups(collection.Groups(), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("inactive selection error = %v", err)
	}
	assertPhase9CGroupIDs(t, inactiveSelection.Groups(), []string{"group-A", "group-C", "group-D", "group-E", "group-F"})

	if _, err := collection.Activate("group-B"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	reactivatedSelection, err := SelectNextFiveActiveGroups(collection.Groups(), InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("reactivated selection error = %v", err)
	}
	assertPhase9CGroupIDs(t, reactivatedSelection.Groups(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
}

func TestSelectNextFiveActiveGroupsIgnoresSourceIdentityKind(t *testing.T) {
	createdAt := phase9CCreatedAt()
	groups := []domain.WatchedGroup{
		phase9CGroup(t, "local-z", true, createdAt, "", "https://www.facebook.com/groups/a"),
		phase9CGroup(t, "local-a", true, createdAt, "facebook-b", ""),
		phase9CGroup(t, "local-m", true, createdAt, "facebook-c", "https://www.facebook.com/groups/c"),
		phase9CGroup(t, "local-b", true, createdAt, "", "https://www.facebook.com/groups/d"),
		phase9CGroup(t, "local-y", true, createdAt, "facebook-e", ""),
		phase9CGroup(t, "local-c", true, createdAt, "facebook-f", "https://www.facebook.com/groups/f"),
	}

	selection, err := SelectNextFiveActiveGroups(groups, InitialWatchedGroupSelectionCursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	assertPhase9CGroupIDs(t, selection.Groups(), []string{"local-z", "local-a", "local-m", "local-b", "local-y"})
}

func TestPhase9CSelectionSourceExcludesExecutionAndInfrastructure(t *testing.T) {
	body, err := os.ReadFile("five_group_selection.go")
	if err != nil {
		t.Fatalf("read five_group_selection.go: %v", err)
	}
	forbidden := []string{
		"NewScanBatchLifecycle(",
		"RunScanBatch(",
		"time.Now(",
		"attemptID",
		"batchID",
		"facebook.com",
		"net/http",
		"database/sql",
		"sqlite",
		"uuid",
		"rand.",
		"go func",
		"sync.",
		"scheduler",
		"retry",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 9C selection source contains deferred behavior %q", fragment)
		}
	}
}

func phase9CGroups(t *testing.T, count int) []domain.WatchedGroup {
	t.Helper()
	groups := make([]domain.WatchedGroup, count)
	for i := range groups {
		id := fmt.Sprintf("group-%c", 'A'+i)
		groups[i] = phase9CGroup(t, id, true, phase9CCreatedAt(), "facebook-"+id, "")
	}
	return groups
}

func phase9CGroup(t *testing.T, id string, active bool, createdAt time.Time, facebookID string, canonicalURL string) domain.WatchedGroup {
	t.Helper()
	group, err := domain.NewWatchedGroup(id, facebookID, canonicalURL, "Name "+id, createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	return group.WithActive(active)
}

func phase9CCollection(t *testing.T, groups []domain.WatchedGroup) *WatchedGroupCollection {
	t.Helper()
	collection := NewWatchedGroupCollection()
	for _, group := range groups {
		if err := collection.Add(group); err != nil {
			t.Fatalf("Add(%q) error = %v", group.ID(), err)
		}
	}
	return collection
}

func mustPhase9CCursor(t *testing.T, position int) WatchedGroupSelectionCursor {
	t.Helper()
	cursor, err := NewWatchedGroupSelectionCursor(position)
	if err != nil {
		t.Fatalf("NewWatchedGroupSelectionCursor(%d) error = %v", position, err)
	}
	return cursor
}

func assertPhase9CGroupIDs(t *testing.T, groups []domain.WatchedGroup, want []string) {
	t.Helper()
	if len(groups) != len(want) {
		t.Fatalf("group count = %d, want %d", len(groups), len(want))
	}
	for i, id := range want {
		if groups[i].ID() != id {
			t.Fatalf("group[%d].ID() = %q, want %q", i, groups[i].ID(), id)
		}
	}
}

func assertPhase9CNoDuplicateIDs(t *testing.T, groups []domain.WatchedGroup) {
	t.Helper()
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if _, exists := seen[group.ID()]; exists {
			t.Fatalf("duplicate selected group ID %q", group.ID())
		}
		seen[group.ID()] = struct{}{}
	}
}

func phase9CCreatedAt() time.Time {
	return time.Date(2026, time.August, 11, 9, 0, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
}
