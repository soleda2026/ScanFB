package application

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestNewScanBatchLifecycleFromSelectionMapsAuthoritativeOrderPending(t *testing.T) {
	selection := phase9DSelection(t)
	attemptIDs := []string{"attempt-101", "attempt-102", "attempt-103", "attempt-104", "attempt-105"}
	attemptIDsBefore := append([]string(nil), attemptIDs...)
	groupsBefore := selection.Groups()
	cursorBefore := selection.NextCursor()
	window := phase9AScanWindow(t, phase9AScanStart(t))

	lifecycle, err := NewScanBatchLifecycleFromSelection("batch-9d", window, selection, attemptIDs)
	if err != nil {
		t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v", err)
	}

	if lifecycle.BatchID() != "batch-9d" {
		t.Fatalf("BatchID() = %q, want batch-9d", lifecycle.BatchID())
	}
	if !reflect.DeepEqual(lifecycle.ScanWindow(), window) {
		t.Fatalf("ScanWindow() = %#v, want %#v", lifecycle.ScanWindow(), window)
	}
	attempts := lifecycle.Attempts()
	if len(attempts) != domain.MaxScanRequestGroups {
		t.Fatalf("attempt count = %d, want %d", len(attempts), domain.MaxScanRequestGroups)
	}
	wantGroupIDs := []string{"group-E", "group-F", "group-G", "group-A", "group-B"}
	for i, attempt := range attempts {
		if attempt.AttemptID() != attemptIDs[i] || attempt.WatchedGroupID() != wantGroupIDs[i] {
			t.Fatalf("attempt[%d] = (%q, %q), want (%q, %q)", i, attempt.AttemptID(), attempt.WatchedGroupID(), attemptIDs[i], wantGroupIDs[i])
		}
		if attempt.Status() != GroupAttemptStatusPending {
			t.Fatalf("attempt[%d].Status() = %q, want %q", i, attempt.Status(), GroupAttemptStatusPending)
		}
		if _, started := attempt.StartedAt(); started {
			t.Fatalf("attempt[%d] unexpectedly started", i)
		}
		if _, completed := attempt.CompletedAt(); completed {
			t.Fatalf("attempt[%d] unexpectedly completed", i)
		}
	}
	if !reflect.DeepEqual(attemptIDs, attemptIDsBefore) {
		t.Fatalf("attempt ID input mutated: got %#v want %#v", attemptIDs, attemptIDsBefore)
	}
	if !reflect.DeepEqual(selection.Groups(), groupsBefore) || selection.NextCursor() != cursorBefore {
		t.Fatal("selection groups or next cursor changed during mapping")
	}
}

func TestNewScanBatchLifecycleFromSelectionRejectsMalformedSelectionWithoutMutation(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/phase9d-shared"
	tests := []struct {
		name   string
		groups func(*testing.T) []domain.WatchedGroup
	}{
		{name: "wrong count", groups: func(t *testing.T) []domain.WatchedGroup {
			return phase9CGroups(t, 4)
		}},
		{name: "invalid group", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[2] = domain.WatchedGroup{}
			return groups
		}},
		{name: "inactive group", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[2] = groups[2].WithActive(false)
			return groups
		}},
		{name: "duplicate local ID", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[4] = phase9CGroup(t, groups[0].ID(), true, phase9CCreatedAt(), "distinct-facebook-id", "")
			return groups
		}},
		{name: "duplicate Facebook identity", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[4] = phase9CGroup(t, "group-E", true, phase9CCreatedAt(), groups[0].FacebookGroupID(), "")
			return groups
		}},
		{name: "duplicate URL-only identity", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[0] = phase9CGroup(t, "group-A", true, phase9CCreatedAt(), "", canonicalURL)
			groups[4] = phase9CGroup(t, "group-E", true, phase9CCreatedAt(), "", canonicalURL)
			return groups
		}},
		{name: "cross-kind canonical conflict", groups: func(t *testing.T) []domain.WatchedGroup {
			groups := phase9CGroups(t, 5)
			groups[0] = phase9CGroup(t, "group-A", true, phase9CCreatedAt(), "", canonicalURL)
			groups[4] = phase9CGroup(t, "group-E", true, phase9CCreatedAt(), "facebook-E", canonicalURL)
			return groups
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selection := FiveGroupSelection{
				groups:     tc.groups(t),
				nextCursor: mustPhase9CCursor(t, 2),
			}
			beforeGroups := selection.Groups()
			beforeCursor := selection.NextCursor()

			lifecycle, err := NewScanBatchLifecycleFromSelection(
				"batch-9d",
				phase9AScanWindow(t, phase9AScanStart(t)),
				selection,
				phase9DAttemptIDs(),
			)
			if !errors.Is(err, ErrMalformedFiveGroupSelection) {
				t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v, want %v", err, ErrMalformedFiveGroupSelection)
			}
			if !reflect.DeepEqual(lifecycle, ScanBatchLifecycle{}) {
				t.Fatalf("malformed selection returned lifecycle %#v", lifecycle)
			}
			if !reflect.DeepEqual(selection.Groups(), beforeGroups) || selection.NextCursor() != beforeCursor {
				t.Fatal("malformed selection changed during mapping")
			}
		})
	}
}

func TestNewScanBatchLifecycleFromSelectionRequiresExactlyFiveAttemptIDs(t *testing.T) {
	tests := []struct {
		name       string
		attemptIDs []string
	}{
		{name: "fewer", attemptIDs: phase9DAttemptIDs()[:4]},
		{name: "more", attemptIDs: append(phase9DAttemptIDs(), "attempt-106")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]string(nil), tc.attemptIDs...)
			lifecycle, err := NewScanBatchLifecycleFromSelection(
				"batch-9d",
				phase9AScanWindow(t, phase9AScanStart(t)),
				phase9DSelection(t),
				tc.attemptIDs,
			)
			if !errors.Is(err, ErrInvalidSelectionAttemptIDCount) {
				t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v, want %v", err, ErrInvalidSelectionAttemptIDCount)
			}
			if !reflect.DeepEqual(lifecycle, ScanBatchLifecycle{}) || !reflect.DeepEqual(tc.attemptIDs, before) {
				t.Fatal("invalid attempt ID count returned a lifecycle or mutated input")
			}
		})
	}
}

func TestNewScanBatchLifecycleFromSelectionPropagatesPhase9AValidation(t *testing.T) {
	validWindow := phase9AScanWindow(t, phase9AScanStart(t))
	tests := []struct {
		name       string
		batchID    string
		window     domain.ScanWindow
		attemptIDs func() []string
		wantErr    error
	}{
		{
			name:    "invalid batch ID",
			batchID: " \t ",
			window:  validWindow,
			attemptIDs: func() []string {
				return phase9DAttemptIDs()
			},
			wantErr: ErrScanBatchLifecycleInvalidBatchID,
		},
		{
			name:    "invalid scan window",
			batchID: "batch-9d",
			window:  domain.ScanWindow{},
			attemptIDs: func() []string {
				return phase9DAttemptIDs()
			},
			wantErr: ErrScanBatchLifecycleInvalidScanWindow,
		},
		{
			name:    "empty attempt ID",
			batchID: "batch-9d",
			window:  validWindow,
			attemptIDs: func() []string {
				ids := phase9DAttemptIDs()
				ids[2] = " \n\t "
				return ids
			},
			wantErr: ErrScanBatchLifecycleInvalidAttemptID,
		},
		{
			name:    "duplicate normalized attempt ID",
			batchID: "batch-9d",
			window:  validWindow,
			attemptIDs: func() []string {
				ids := phase9DAttemptIDs()
				ids[4] = " attempt-101 "
				return ids
			},
			wantErr: ErrScanBatchLifecycleDuplicateAttemptID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attemptIDs := tc.attemptIDs()
			before := append([]string(nil), attemptIDs...)
			lifecycle, err := NewScanBatchLifecycleFromSelection(tc.batchID, tc.window, phase9DSelection(t), attemptIDs)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(lifecycle, ScanBatchLifecycle{}) || !reflect.DeepEqual(attemptIDs, before) {
				t.Fatal("Phase 9A validation failure returned a lifecycle or mutated attempt IDs")
			}
		})
	}
}

func TestNewScanBatchLifecycleFromSelectionUsesPhase9AAttemptIDNormalization(t *testing.T) {
	attemptIDs := []string{" attempt-101 ", "\tattempt-102", "attempt-103\n", "attempt-104", "attempt-105"}
	before := append([]string(nil), attemptIDs...)

	lifecycle, err := NewScanBatchLifecycleFromSelection(
		"batch-9d",
		phase9AScanWindow(t, phase9AScanStart(t)),
		phase9DSelection(t),
		attemptIDs,
	)
	if err != nil {
		t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v", err)
	}
	want := phase9DAttemptIDs()
	for i, attempt := range lifecycle.Attempts() {
		if attempt.AttemptID() != want[i] {
			t.Fatalf("attempt[%d].AttemptID() = %q, want %q", i, attempt.AttemptID(), want[i])
		}
	}
	if !reflect.DeepEqual(attemptIDs, before) {
		t.Fatalf("attempt ID input mutated: got %#v want %#v", attemptIDs, before)
	}
}

func TestNewScanBatchLifecycleFromSelectionIsDeterministicAndDefensive(t *testing.T) {
	selection := phase9DSelection(t)
	window := phase9AScanWindow(t, phase9AScanStart(t))
	attemptIDs := phase9DAttemptIDs()

	first, err := NewScanBatchLifecycleFromSelection("batch-9d", window, selection, attemptIDs)
	if err != nil {
		t.Fatalf("first mapping error = %v", err)
	}
	second, err := NewScanBatchLifecycleFromSelection("batch-9d", window, selection, attemptIDs)
	if err != nil {
		t.Fatalf("second mapping error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated mapping changed: first=%#v second=%#v", first, second)
	}

	attemptIDs[0] = "mutated-after-call"
	returned := first.Attempts()
	returned[0] = GroupScanAttempt{}
	attempts := first.Attempts()
	if attempts[0].AttemptID() != "attempt-101" || attempts[0].WatchedGroupID() != "group-E" {
		t.Fatalf("lifecycle changed after caller mutation: %#v", attempts[0])
	}
}

func TestNewScanBatchLifecycleFromSelectionAllowsDistinctFacebookIDsSharingSecondaryURL(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/phase9d-shared-secondary"
	groups := phase9CGroups(t, 5)
	groups[0] = phase9CGroup(t, "group-A", true, phase9CCreatedAt(), "facebook-A", canonicalURL)
	groups[1] = phase9CGroup(t, "group-B", true, phase9CCreatedAt(), "facebook-B", canonicalURL)
	selection := FiveGroupSelection{groups: groups, nextCursor: InitialWatchedGroupSelectionCursor()}

	lifecycle, err := NewScanBatchLifecycleFromSelection(
		"batch-9d",
		phase9AScanWindow(t, phase9AScanStart(t)),
		selection,
		phase9DAttemptIDs(),
	)
	if err != nil {
		t.Fatalf("NewScanBatchLifecycleFromSelection() error = %v", err)
	}
	assertPhase9AGroupIDs(t, lifecycle.Attempts(), []string{"group-A", "group-B", "group-C", "group-D", "group-E"})
}

func TestPhase9DMapperSourceExcludesExecutionAndInfrastructure(t *testing.T) {
	body, err := os.ReadFile("selection_lifecycle.go")
	if err != nil {
		t.Fatalf("read selection_lifecycle.go: %v", err)
	}
	forbidden := []string{
		"SelectNextFiveActiveGroups(",
		"NextCursor(",
		"RunScanBatch(",
		"StartNextPending(",
		"StartAttempt(",
		"SucceedAttempt(",
		"FailAttempt(",
		"SkipAttempt(",
		"ExpireAtDayBoundary(",
		"time.Now(",
		"uuid",
		"rand.",
		"fmt.Sprintf(",
		"facebook.com",
		"net/http",
		"database/sql",
		"sqlite",
		"persistence",
		"bridge",
		"go func",
		"sync.",
		"scheduler",
		"retry",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 9D mapper source contains deferred behavior %q", fragment)
		}
	}
}

func phase9DSelection(t *testing.T) FiveGroupSelection {
	t.Helper()
	selection, err := SelectNextFiveActiveGroups(phase9CGroups(t, 7), mustPhase9CCursor(t, 4))
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() setup error = %v", err)
	}
	return selection
}

func phase9DAttemptIDs() []string {
	return []string{"attempt-101", "attempt-102", "attempt-103", "attempt-104", "attempt-105"}
}
