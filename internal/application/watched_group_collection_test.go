package application

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestWatchedGroupCollectionAddAndLookup(t *testing.T) {
	collection := NewWatchedGroupCollection()
	group := phase9BGroup(t, "local-1", "facebook-1", "", "Group one")

	if err := collection.Add(group); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got, err := collection.GroupByID("local-1")
	if err != nil {
		t.Fatalf("GroupByID() error = %v", err)
	}
	if got != group || collection.Count() != 1 {
		t.Fatalf("unexpected stored group: got=%#v count=%d", got, collection.Count())
	}
}

func TestWatchedGroupCollectionZeroValueIsUsable(t *testing.T) {
	var collection WatchedGroupCollection
	if err := collection.Add(phase9BGroup(t, "local-1", "facebook-1", "", "Group one")); err != nil {
		t.Fatalf("zero-value Add() error = %v", err)
	}
	if collection.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", collection.Count())
	}
}

func TestWatchedGroupCollectionRejectsInvalidGroup(t *testing.T) {
	collection := NewWatchedGroupCollection()
	if err := collection.Add(domain.WatchedGroup{}); !errors.Is(err, domain.ErrInvalidWatchedGroupID) {
		t.Fatalf("Add() error = %v", err)
	}
	if collection.Count() != 0 {
		t.Fatalf("Count() = %d after invalid add", collection.Count())
	}
}

func TestWatchedGroupCollectionRejectsDuplicateLocalIDWithoutOverwrite(t *testing.T) {
	collection := NewWatchedGroupCollection()
	original := phase9BGroup(t, "local-1", "facebook-1", "", "Original")
	mustAddPhase9BGroup(t, collection, original)

	err := collection.Add(phase9BGroup(t, "local-1", "facebook-2", "", "Replacement"))
	if !errors.Is(err, ErrDuplicateWatchedGroupID) {
		t.Fatalf("Add() error = %v", err)
	}
	got, lookupErr := collection.GroupByID("local-1")
	if lookupErr != nil || got != original || collection.Count() != 1 {
		t.Fatalf("duplicate local ID overwrote collection: got=%#v error=%v count=%d", got, lookupErr, collection.Count())
	}
}

func TestWatchedGroupCollectionRejectsDuplicateAuthoritativeFacebookIdentity(t *testing.T) {
	collection := NewWatchedGroupCollection()
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-1", "facebook-1", "", "One"))

	err := collection.Add(phase9BGroup(t, "local-2", "facebook-1", "https://www.facebook.com/groups/two", "Two"))
	if !errors.Is(err, ErrDuplicateWatchedGroupIdentity) {
		t.Fatalf("Add() error = %v", err)
	}
	if collection.Count() != 1 {
		t.Fatalf("Count() = %d after duplicate identity", collection.Count())
	}
}

func TestWatchedGroupCollectionRejectsDuplicateCanonicalIdentityForURLOnlyGroups(t *testing.T) {
	collection := NewWatchedGroupCollection()
	canonicalURL := "https://www.facebook.com/groups/macbook-vietnam"
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-1", "", canonicalURL, "One"))

	err := collection.Add(phase9BGroup(t, "local-2", "", canonicalURL, "Two"))
	if !errors.Is(err, ErrDuplicateWatchedGroupIdentity) {
		t.Fatalf("Add() error = %v", err)
	}
	if collection.Count() != 1 {
		t.Fatalf("Count() = %d after duplicate identity", collection.Count())
	}
}

func TestWatchedGroupCollectionUsesFacebookIDAsAuthoritativeIdentity(t *testing.T) {
	collection := NewWatchedGroupCollection()
	sharedURL := "https://www.facebook.com/groups/shared-slug"
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-1", "facebook-1", sharedURL, "One"))

	if err := collection.Add(phase9BGroup(t, "local-2", "facebook-2", sharedURL, "Two")); err != nil {
		t.Fatalf("different authoritative facebook IDs should remain distinct: %v", err)
	}
	if collection.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", collection.Count())
	}
}

func TestWatchedGroupCollectionRejectsExactCrossKindCanonicalIdentity(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/shared-slug"
	tests := []struct {
		name     string
		first    domain.WatchedGroup
		incoming domain.WatchedGroup
	}{
		{
			name:     "existing URL-only and incoming facebook ID",
			first:    phase9BGroup(t, "local-1", "", canonicalURL, "One"),
			incoming: phase9BGroup(t, "local-2", "facebook-2", canonicalURL, "Two"),
		},
		{
			name:     "existing facebook ID and incoming URL-only",
			first:    phase9BGroup(t, "local-1", "facebook-1", canonicalURL, "One"),
			incoming: phase9BGroup(t, "local-2", "", canonicalURL, "Two"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collection := NewWatchedGroupCollection()
			mustAddPhase9BGroup(t, collection, test.first)
			before := collection.Groups()

			if err := collection.Add(test.incoming); !errors.Is(err, ErrDuplicateWatchedGroupIdentity) {
				t.Fatalf("Add() error = %v", err)
			}
			if after := collection.Groups(); !reflect.DeepEqual(after, before) {
				t.Fatalf("cross-kind duplicate mutated collection: before=%#v after=%#v", before, after)
			}
		})
	}
}

func TestWatchedGroupCollectionPreservesInsertionOrder(t *testing.T) {
	collection := NewWatchedGroupCollection()
	want := []string{"local-3", "local-1", "local-2"}
	for i, id := range want {
		mustAddPhase9BGroup(t, collection, phase9BGroup(t, id, "facebook-"+id, "", "Group "+id))
		if collection.Count() != i+1 {
			t.Fatalf("Count() = %d, want %d", collection.Count(), i+1)
		}
	}
	assertPhase9BGroupOrder(t, collection.Groups(), want)
}

func TestWatchedGroupCollectionHasNoFiveGroupLimit(t *testing.T) {
	collection := NewWatchedGroupCollection()
	const groupCount = 12
	for i := 0; i < groupCount; i++ {
		id := "local-" + string(rune('a'+i))
		mustAddPhase9BGroup(t, collection, phase9BGroup(t, id, "facebook-"+id, "", "Group "+id))
	}
	if collection.Count() != groupCount || len(collection.Groups()) != groupCount {
		t.Fatalf("collection size = %d/%d, want %d", collection.Count(), len(collection.Groups()), groupCount)
	}
}

func TestWatchedGroupCollectionNotFoundBehaviorIsExplicit(t *testing.T) {
	collection := NewWatchedGroupCollection()
	if _, err := collection.GroupByID("missing"); !errors.Is(err, ErrWatchedGroupNotFound) {
		t.Fatalf("GroupByID() error = %v", err)
	}
	if _, err := collection.GroupByID("   "); !errors.Is(err, ErrWatchedGroupNotFound) {
		t.Fatalf("GroupByID(empty) error = %v", err)
	}
}

func TestWatchedGroupCollectionDeactivateAndActivateAreIdempotent(t *testing.T) {
	collection := NewWatchedGroupCollection()
	original := phase9BGroup(t, "local-1", "facebook-1", "", "Group one")
	mustAddPhase9BGroup(t, collection, original)

	inactive, err := collection.Deactivate("local-1")
	if err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	inactiveAgain, err := collection.Deactivate("local-1")
	if err != nil {
		t.Fatalf("repeated Deactivate() error = %v", err)
	}
	if inactive.IsActive() || inactiveAgain.IsActive() {
		t.Fatal("deactivated group remains active")
	}

	active, err := collection.Activate("local-1")
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	activeAgain, err := collection.Activate("local-1")
	if err != nil {
		t.Fatalf("repeated Activate() error = %v", err)
	}
	if !active.IsActive() || !activeAgain.IsActive() {
		t.Fatal("activated group remains inactive")
	}
	if active.ID() != original.ID() || active.IdentityKey() != original.IdentityKey() || !active.CreatedAt().Equal(original.CreatedAt()) {
		t.Fatal("activation lifecycle changed identity or createdAt")
	}
}

func TestWatchedGroupCollectionMetadataUpdatePreservesIdentity(t *testing.T) {
	collection := NewWatchedGroupCollection()
	original := phase9BGroup(t, "local-1", "facebook-1", "", "Original")
	mustAddPhase9BGroup(t, collection, original)
	lastScan := phase9BCreatedAt().Add(time.Hour)

	updated, err := collection.UpdateMetadata("local-1", domain.WatchedGroupMetadata{
		Name:                 "Updated",
		Notes:                "Notes",
		LastSuccessfulScanAt: lastScan,
		LastError:            "Last error",
		DisplayOrder:         8,
	})
	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}
	if updated.ID() != original.ID() || updated.IdentityKey() != original.IdentityKey() || !updated.CreatedAt().Equal(original.CreatedAt()) {
		t.Fatal("metadata update changed identity or createdAt")
	}
	if updated.Name() != "Updated" || updated.Notes() != "Notes" || updated.LastError() != "Last error" || updated.DisplayOrder() != 8 {
		t.Fatalf("metadata not updated: %#v", updated.Metadata())
	}
}

func TestWatchedGroupCollectionInvalidMetadataUpdateDoesNotPartiallyMutate(t *testing.T) {
	collection := NewWatchedGroupCollection()
	original := phase9BGroup(t, "local-1", "facebook-1", "", "Original")
	mustAddPhase9BGroup(t, collection, original)
	before := collection.Groups()

	_, err := collection.UpdateMetadata("local-1", domain.WatchedGroupMetadata{
		Name:                 "Changed",
		Notes:                "Should not persist",
		LastSuccessfulScanAt: phase9BCreatedAt().Add(-time.Second),
	})
	if !errors.Is(err, domain.ErrWatchedGroupScanBeforeCreated) {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}
	if after := collection.Groups(); !reflect.DeepEqual(after, before) {
		t.Fatalf("invalid update mutated collection: before=%#v after=%#v", before, after)
	}
}

func TestWatchedGroupCollectionDisplayOrderIsMetadataOnly(t *testing.T) {
	collection := NewWatchedGroupCollection()
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-1", "facebook-1", "", "One"))
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-2", "facebook-2", "", "Two"))

	if _, err := collection.UpdateMetadata("local-1", domain.WatchedGroupMetadata{Name: "One", DisplayOrder: 100}); err != nil {
		t.Fatalf("UpdateMetadata(first) error = %v", err)
	}
	if _, err := collection.UpdateMetadata("local-2", domain.WatchedGroupMetadata{Name: "Two", DisplayOrder: -100}); err != nil {
		t.Fatalf("UpdateMetadata(second) error = %v", err)
	}
	assertPhase9BGroupOrder(t, collection.Groups(), []string{"local-1", "local-2"})
}

func TestWatchedGroupCollectionReturnsDefensiveSnapshots(t *testing.T) {
	collection := NewWatchedGroupCollection()
	original := phase9BGroup(t, "local-1", "facebook-1", "", "One")
	mustAddPhase9BGroup(t, collection, original)

	first := collection.Groups()
	first[0] = phase9BGroup(t, "replacement", "facebook-replacement", "", "Replacement")
	first = append(first, phase9BGroup(t, "extra", "facebook-extra", "", "Extra"))

	second := collection.Groups()
	if len(second) != 1 || second[0] != original {
		t.Fatalf("caller mutated collection snapshot: %#v", second)
	}
}

func TestWatchedGroupCollectionMutationNotFoundDoesNotChangeState(t *testing.T) {
	collection := NewWatchedGroupCollection()
	mustAddPhase9BGroup(t, collection, phase9BGroup(t, "local-1", "facebook-1", "", "One"))
	before := collection.Groups()

	operations := []struct {
		name string
		run  func() error
	}{
		{name: "update", run: func() error {
			_, err := collection.UpdateMetadata("missing", domain.WatchedGroupMetadata{Name: "Missing"})
			return err
		}},
		{name: "activate", run: func() error { _, err := collection.Activate("missing"); return err }},
		{name: "deactivate", run: func() error { _, err := collection.Deactivate("missing"); return err }},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, ErrWatchedGroupNotFound) {
				t.Fatalf("operation error = %v", err)
			}
			if after := collection.Groups(); !reflect.DeepEqual(after, before) {
				t.Fatalf("not-found operation mutated collection: before=%#v after=%#v", before, after)
			}
		})
	}
}

func TestPhase9BProductionSourcesExcludeDeferredBehavior(t *testing.T) {
	sources := []string{"watched_group_collection.go", "../domain/watched_group.go"}
	forbidden := []string{
		"time.Now(",
		"NewScanBatchLifecycle(",
		"NextFive",
		"BuildBatch",
		"SelectBatchGroups",
		"round-robin",
		"queue cursor",
		"facebook.com",
		"database/sql",
		"sqlite",
		"goroutine",
		"go func",
	}
	for _, sourcePath := range sources {
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read %s: %v", sourcePath, err)
		}
		for _, fragment := range forbidden {
			if strings.Contains(string(body), fragment) {
				t.Errorf("%s contains deferred behavior %q", sourcePath, fragment)
			}
		}
	}
}

func phase9BGroup(t *testing.T, id string, facebookID string, canonicalURL string, name string) domain.WatchedGroup {
	t.Helper()
	group, err := domain.NewWatchedGroup(id, facebookID, canonicalURL, name, phase9BCreatedAt())
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	return group
}

func mustAddPhase9BGroup(t *testing.T, collection *WatchedGroupCollection, group domain.WatchedGroup) {
	t.Helper()
	if err := collection.Add(group); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
}

func assertPhase9BGroupOrder(t *testing.T, groups []domain.WatchedGroup, want []string) {
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

func phase9BCreatedAt() time.Time {
	return time.Date(2026, time.August, 10, 9, 30, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
}
