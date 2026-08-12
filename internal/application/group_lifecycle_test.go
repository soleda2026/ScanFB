package application

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestNewScanBatchLifecycleRequiresExactlyFiveGroups(t *testing.T) {
	tests := []struct {
		name   string
		inputs []GroupScanAttemptInput
	}{
		{name: "four groups", inputs: phase9AAttemptInputs()[:4]},
		{name: "six groups", inputs: append(phase9AAttemptInputs(), GroupScanAttemptInput{AttemptID: "attempt-006", WatchedGroupID: "group-006"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), tt.inputs); !errors.Is(err, ErrScanBatchLifecycleInvalidGroupCount) {
				t.Fatalf("NewScanBatchLifecycle error = %v, want %v", err, ErrScanBatchLifecycleInvalidGroupCount)
			}
		})
	}
}

func TestNewOneGroupScanLifecycleUsesExistingTransitionSemantics(t *testing.T) {
	lifecycle, err := NewOneGroupScanLifecycle(
		"scan-001",
		phase9AScanWindow(t, phase9AScanStart(t)),
		GroupScanAttemptInput{AttemptID: "attempt-001", WatchedGroupID: "group-001"},
	)
	if err != nil {
		t.Fatalf("NewOneGroupScanLifecycle() error = %v", err)
	}
	if got := lifecycle.Attempts(); len(got) != 1 || got[0].Status() != GroupAttemptStatusPending {
		t.Fatalf("initial attempts = %#v, want one pending attempt", got)
	}

	if _, err := lifecycle.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending() error = %v", err)
	}
	attempt, err := lifecycle.SucceedAttempt("attempt-001", phase9ATime(t, 10, 31))
	if err != nil {
		t.Fatalf("SucceedAttempt() error = %v", err)
	}
	if attempt.Status() != GroupAttemptStatusSucceeded || !lifecycle.IsTerminal() {
		t.Fatalf("completed lifecycle = %#v, want one terminal succeeded attempt", lifecycle.Attempts())
	}
}

func TestNewScanBatchLifecycleRejectsEmptyGroupID(t *testing.T) {
	inputs := phase9AAttemptInputs()
	inputs[2].WatchedGroupID = " \t "

	if _, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), inputs); !errors.Is(err, ErrScanBatchLifecycleEmptyGroupID) {
		t.Fatalf("NewScanBatchLifecycle error = %v, want %v", err, ErrScanBatchLifecycleEmptyGroupID)
	}
}

func TestNewScanBatchLifecycleRejectsDuplicateGroupID(t *testing.T) {
	inputs := phase9AAttemptInputs()
	inputs[4].WatchedGroupID = " group-001 "

	if _, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), inputs); !errors.Is(err, ErrScanBatchLifecycleDuplicateGroupID) {
		t.Fatalf("NewScanBatchLifecycle error = %v, want %v", err, ErrScanBatchLifecycleDuplicateGroupID)
	}
}

func TestNewScanBatchLifecyclePreservesCallerGroupOrder(t *testing.T) {
	batch := newPhase9ALifecycle(t)

	assertPhase9AGroupIDs(t, batch.Attempts(), []string{"group-001", "group-002", "group-003", "group-004", "group-005"})
}

func TestNewScanBatchLifecycleStartsAllAttemptsPending(t *testing.T) {
	batch := newPhase9ALifecycle(t)

	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
	})
}

func TestNewScanBatchLifecycleValidatesAttemptIDsAndBatchOwnership(t *testing.T) {
	if _, err := NewScanBatchLifecycle(" ", phase9AScanWindow(t, phase9AScanStart(t)), phase9AAttemptInputs()); !errors.Is(err, ErrScanBatchLifecycleInvalidBatchID) {
		t.Fatalf("empty batch id error = %v, want %v", err, ErrScanBatchLifecycleInvalidBatchID)
	}

	emptyAttemptID := phase9AAttemptInputs()
	emptyAttemptID[0].AttemptID = "\n\t"
	if _, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), emptyAttemptID); !errors.Is(err, ErrScanBatchLifecycleInvalidAttemptID) {
		t.Fatalf("empty attempt id error = %v, want %v", err, ErrScanBatchLifecycleInvalidAttemptID)
	}

	duplicateAttemptID := phase9AAttemptInputs()
	duplicateAttemptID[3].AttemptID = " attempt-001 "
	if _, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), duplicateAttemptID); !errors.Is(err, ErrScanBatchLifecycleDuplicateAttemptID) {
		t.Fatalf("duplicate attempt id error = %v, want %v", err, ErrScanBatchLifecycleDuplicateAttemptID)
	}

	batch := newPhase9ALifecycle(t)
	for _, attempt := range batch.Attempts() {
		if attempt.BatchID() != "batch-001" {
			t.Fatalf("Attempt %q BatchID = %q, want batch-001", attempt.AttemptID(), attempt.BatchID())
		}
	}
}

func TestScanBatchLifecycleStartNextPendingStartsFirstAttemptRunning(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	startedAt := phase9ATime(t, 10, 30)

	attempt, err := batch.StartNextPending(startedAt)
	if err != nil {
		t.Fatalf("StartNextPending returned error: %v", err)
	}
	if attempt.AttemptID() != "attempt-001" || attempt.Status() != GroupAttemptStatusRunning {
		t.Fatalf("started attempt = %#v", attempt)
	}
	if gotStartedAt, ok := attempt.StartedAt(); !ok || !gotStartedAt.Equal(startedAt) {
		t.Fatalf("StartedAt = %v ok %v, want %v true", gotStartedAt, ok, startedAt)
	}
	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusRunning,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
	})
}

func TestScanBatchLifecycleStartBeforeScanStartedRejected(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	before := batch.Attempts()

	if _, err := batch.StartNextPending(phase9ATime(t, 9, 59)); !errors.Is(err, ErrScanBatchLifecycleTimeBeforeScan) {
		t.Fatalf("StartNextPending before scan start error = %v, want %v", err, ErrScanBatchLifecycleTimeBeforeScan)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("start before scan start mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleStartExactlyAtScanStartedAccepted(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	scanStarted := batch.ScanWindow().ScanStarted()

	attempt, err := batch.StartNextPending(scanStarted)
	if err != nil {
		t.Fatalf("StartNextPending at scan start returned error: %v", err)
	}
	if attempt.Status() != GroupAttemptStatusRunning {
		t.Fatalf("Status = %q, want %q", attempt.Status(), GroupAttemptStatusRunning)
	}
	if gotStartedAt, ok := attempt.StartedAt(); !ok || !gotStartedAt.Equal(scanStarted) {
		t.Fatalf("StartedAt = %v ok %v, want %v true", gotStartedAt, ok, scanStarted)
	}
}

func TestScanBatchLifecycleAllowsOnlyOneRunningAttempt(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}

	if _, err := batch.StartAttempt("attempt-002", phase9ATime(t, 10, 31)); !errors.Is(err, ErrScanBatchLifecycleAnotherRunning) {
		t.Fatalf("StartAttempt error = %v, want %v", err, ErrScanBatchLifecycleAnotherRunning)
	}
	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusRunning,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
	})
}

func TestScanBatchLifecycleCannotStartLaterAttemptBeforeEarlierPending(t *testing.T) {
	batch := newPhase9ALifecycle(t)

	if _, err := batch.StartAttempt("attempt-003", phase9ATime(t, 10, 30)); !errors.Is(err, ErrScanBatchLifecycleOutOfOrderAttempt) {
		t.Fatalf("StartAttempt error = %v, want %v", err, ErrScanBatchLifecycleOutOfOrderAttempt)
	}
	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
	})
}

func TestScanBatchLifecycleRunningAttemptCanSucceed(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}

	completedAt := phase9ATime(t, 10, 40)
	attempt, err := batch.SucceedAttempt("attempt-001", completedAt)
	if err != nil {
		t.Fatalf("SucceedAttempt returned error: %v", err)
	}
	if attempt.Status() != GroupAttemptStatusSucceeded {
		t.Fatalf("Status = %q, want %q", attempt.Status(), GroupAttemptStatusSucceeded)
	}
	if gotCompletedAt, ok := attempt.CompletedAt(); !ok || !gotCompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v ok %v, want %v true", gotCompletedAt, ok, completedAt)
	}
}

func TestScanBatchLifecycleSucceedBeforeAttemptStartedRejected(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	before := batch.Attempts()

	if _, err := batch.SucceedAttempt("attempt-001", phase9ATime(t, 10, 29)); !errors.Is(err, ErrScanBatchLifecycleCompleteBeforeRun) {
		t.Fatalf("SucceedAttempt before startedAt error = %v, want %v", err, ErrScanBatchLifecycleCompleteBeforeRun)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("succeed before startedAt mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleRunningAttemptCanFail(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}

	completedAt := phase9ATime(t, 10, 40)
	attempt, err := batch.FailAttempt("attempt-001", completedAt)
	if err != nil {
		t.Fatalf("FailAttempt returned error: %v", err)
	}
	if attempt.Status() != GroupAttemptStatusFailed {
		t.Fatalf("Status = %q, want %q", attempt.Status(), GroupAttemptStatusFailed)
	}
	if gotCompletedAt, ok := attempt.CompletedAt(); !ok || !gotCompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v ok %v, want %v true", gotCompletedAt, ok, completedAt)
	}
}

func TestScanBatchLifecycleFailBeforeAttemptStartedRejected(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	before := batch.Attempts()

	if _, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 29)); !errors.Is(err, ErrScanBatchLifecycleCompleteBeforeRun) {
		t.Fatalf("FailAttempt before startedAt error = %v, want %v", err, ErrScanBatchLifecycleCompleteBeforeRun)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("fail before startedAt mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleCompletionExactlyAtStartedAtAccepted(t *testing.T) {
	tests := []struct {
		name     string
		complete func(*ScanBatchLifecycle, string, time.Time) (GroupScanAttempt, error)
		want     GroupAttemptStatus
	}{
		{
			name: "succeed",
			complete: func(batch *ScanBatchLifecycle, attemptID string, at time.Time) (GroupScanAttempt, error) {
				return batch.SucceedAttempt(attemptID, at)
			},
			want: GroupAttemptStatusSucceeded,
		},
		{
			name: "fail",
			complete: func(batch *ScanBatchLifecycle, attemptID string, at time.Time) (GroupScanAttempt, error) {
				return batch.FailAttempt(attemptID, at)
			},
			want: GroupAttemptStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := newPhase9ALifecycle(t)
			startedAt := phase9ATime(t, 10, 30)
			if _, err := batch.StartNextPending(startedAt); err != nil {
				t.Fatalf("StartNextPending setup returned error: %v", err)
			}

			attempt, err := tt.complete(&batch, "attempt-001", startedAt)
			if err != nil {
				t.Fatalf("completion exactly at startedAt returned error: %v", err)
			}
			if attempt.Status() != tt.want {
				t.Fatalf("Status = %q, want %q", attempt.Status(), tt.want)
			}
			if gotCompletedAt, ok := attempt.CompletedAt(); !ok || !gotCompletedAt.Equal(startedAt) {
				t.Fatalf("CompletedAt = %v ok %v, want %v true", gotCompletedAt, ok, startedAt)
			}
		})
	}
}

func TestScanBatchLifecyclePendingAttemptCanSkip(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	skippedAt := phase9ATime(t, 10, 30)

	attempt, err := batch.SkipAttempt("attempt-001", skippedAt)
	if err != nil {
		t.Fatalf("SkipAttempt returned error: %v", err)
	}
	if attempt.Status() != GroupAttemptStatusSkipped {
		t.Fatalf("Status = %q, want %q", attempt.Status(), GroupAttemptStatusSkipped)
	}
	if gotCompletedAt, ok := attempt.CompletedAt(); !ok || !gotCompletedAt.Equal(skippedAt) {
		t.Fatalf("CompletedAt = %v ok %v, want %v true", gotCompletedAt, ok, skippedAt)
	}
}

func TestScanBatchLifecycleSkipBeforeScanStartedRejected(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	before := batch.Attempts()

	if _, err := batch.SkipAttempt("attempt-001", phase9ATime(t, 9, 59)); !errors.Is(err, ErrScanBatchLifecycleTimeBeforeScan) {
		t.Fatalf("SkipAttempt before scan start error = %v, want %v", err, ErrScanBatchLifecycleTimeBeforeScan)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("skip before scan start mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleInvalidChronologyDoesNotPartiallyMutate(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	before := batch.Attempts()

	if _, err := batch.SucceedAttempt("attempt-001", batch.ScanWindow().ScanStarted()); !errors.Is(err, ErrScanBatchLifecycleCompleteBeforeRun) {
		t.Fatalf("SucceedAttempt invalid chronology error = %v, want %v", err, ErrScanBatchLifecycleCompleteBeforeRun)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("invalid chronology mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleTerminalAttemptCannotTransition(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	if _, err := batch.SucceedAttempt("attempt-001", phase9ATime(t, 10, 35)); err != nil {
		t.Fatalf("SucceedAttempt setup returned error: %v", err)
	}

	if _, err := batch.StartAttempt("attempt-001", phase9ATime(t, 10, 40)); !errors.Is(err, ErrScanBatchLifecycleAttemptTerminal) {
		t.Fatalf("StartAttempt terminal error = %v, want %v", err, ErrScanBatchLifecycleAttemptTerminal)
	}
	if _, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 40)); !errors.Is(err, ErrScanBatchLifecycleAttemptTerminal) {
		t.Fatalf("FailAttempt terminal error = %v, want %v", err, ErrScanBatchLifecycleAttemptTerminal)
	}
}

func TestScanBatchLifecycleFailedAttemptNeverAppearsSucceeded(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	if _, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 35)); err != nil {
		t.Fatalf("FailAttempt returned error: %v", err)
	}

	summary := batch.Summary()
	if summary.Failed != 1 || summary.Succeeded != 0 {
		t.Fatalf("Summary after failure = %#v, want one failed and zero succeeded", summary)
	}
	assertPhase9AStatus(t, batch, "attempt-001", GroupAttemptStatusFailed)
}

func TestScanBatchLifecycleExplicitNextActionStartsNextPendingAfterFailure(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	if _, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 35)); err != nil {
		t.Fatalf("FailAttempt setup returned error: %v", err)
	}

	attempt, err := batch.StartNextPending(phase9ATime(t, 10, 40))
	if err != nil {
		t.Fatalf("StartNextPending after failure returned error: %v", err)
	}
	if attempt.AttemptID() != "attempt-002" || attempt.Status() != GroupAttemptStatusRunning {
		t.Fatalf("next attempt = %#v", attempt)
	}
}

func TestScanBatchLifecycleFailureDoesNotAutomaticallyRetry(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	if _, err := batch.StartNextPending(phase9ATime(t, 10, 30)); err != nil {
		t.Fatalf("StartNextPending setup returned error: %v", err)
	}
	if _, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 35)); err != nil {
		t.Fatalf("FailAttempt setup returned error: %v", err)
	}

	if _, active := batch.ActiveAttempt(); active {
		t.Fatalf("failure automatically started or retried an attempt")
	}
	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusFailed,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
		GroupAttemptStatusPending,
	})
}

func TestScanBatchLifecycleSummaryReconcilesToExactlyFive(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	mustSkipPhase9A(t, &batch, "attempt-001", phase9ATime(t, 10, 10))
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 20))
	mustSucceedPhase9A(t, &batch, "attempt-002", phase9ATime(t, 10, 30))
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 40))

	summary := batch.Summary()
	gotTotal := summary.Pending + summary.Running + summary.Succeeded + summary.Failed + summary.Skipped + summary.ExpiredAtDayBoundary
	if summary.Total != domain.MaxScanRequestGroups || gotTotal != domain.MaxScanRequestGroups || summary.Terminal != 2 {
		t.Fatalf("Summary = %#v, reconciled total = %d", summary, gotTotal)
	}
}

func TestScanBatchLifecycleCompletedBatchCannotStartAnotherAttempt(t *testing.T) {
	batch := completedPhase9ALifecycle(t)

	if _, err := batch.StartNextPending(phase9ATime(t, 12, 0)); !errors.Is(err, ErrScanBatchLifecycleBatchTerminal) {
		t.Fatalf("StartNextPending terminal error = %v, want %v", err, ErrScanBatchLifecycleBatchTerminal)
	}
}

func TestScanBatchLifecycleDayBoundaryLeavesTerminalAttemptsUnchanged(t *testing.T) {
	batch := phase9ADayBoundaryLifecycle(t)
	before := map[string]GroupAttemptStatus{
		"attempt-001": GroupAttemptStatusSucceeded,
		"attempt-002": GroupAttemptStatusFailed,
		"attempt-003": GroupAttemptStatusSkipped,
	}

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	for attemptID, want := range before {
		assertPhase9AStatus(t, batch, attemptID, want)
	}
}

func TestScanBatchLifecycleDayBoundaryConvertsRunningAttempt(t *testing.T) {
	batch := phase9ADayBoundaryLifecycle(t)

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	assertPhase9AStatus(t, batch, "attempt-004", GroupAttemptStatusExpiredAtDayBoundary)
}

func TestScanBatchLifecycleDayBoundaryConvertsAllPendingAttempts(t *testing.T) {
	batch := phase9ADayBoundaryLifecycle(t)

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	assertPhase9AStatus(t, batch, "attempt-005", GroupAttemptStatusExpiredAtDayBoundary)
}

func TestScanBatchLifecycleAfterDayBoundaryBatchIsTerminal(t *testing.T) {
	batch := phase9ADayBoundaryLifecycle(t)

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	if !batch.IsTerminal() || batch.Summary().Terminal != domain.MaxScanRequestGroups {
		t.Fatalf("batch should be terminal after day boundary: %#v", batch.Summary())
	}
}

func TestScanBatchLifecycleAfterBoundaryNoAttemptCanStart(t *testing.T) {
	batch := newPhase9ALifecycle(t)

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	if _, err := batch.StartNextPending(phase9ANextDayTime(t, 0, 6)); !errors.Is(err, ErrScanBatchLifecycleDayBoundaryReached) {
		t.Fatalf("StartNextPending after boundary error = %v, want %v", err, ErrScanBatchLifecycleDayBoundaryReached)
	}
}

func TestScanBatchLifecycleUsesAsiaHoChiMinhLocalDay(t *testing.T) {
	loc := phase9ALocation(t)
	scanStarted := time.Date(2026, 8, 5, 23, 30, 0, 0, loc)
	batch, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, scanStarted), phase9AAttemptInputs())
	if err != nil {
		t.Fatalf("NewScanBatchLifecycle returned error: %v", err)
	}

	if err := batch.ExpireAtDayBoundary(time.Date(2026, 8, 5, 16, 59, 0, 0, time.UTC)); !errors.Is(err, ErrScanBatchLifecycleDayBoundaryPending) {
		t.Fatalf("same HCM day boundary error = %v, want %v", err, ErrScanBatchLifecycleDayBoundaryPending)
	}
	if err := batch.ExpireAtDayBoundary(time.Date(2026, 8, 5, 17, 5, 0, 0, time.UTC)); err != nil {
		t.Fatalf("next HCM day boundary returned error: %v", err)
	}
	assertPhase9AStatus(t, batch, "attempt-001", GroupAttemptStatusExpiredAtDayBoundary)
}

func TestScanBatchLifecycleUsesSuppliedTimeOnlyForDayBoundary(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 30))
	before := batch.Attempts()

	if err := batch.ExpireAtDayBoundary(phase9ATime(t, 23, 59)); !errors.Is(err, ErrScanBatchLifecycleDayBoundaryPending) {
		t.Fatalf("ExpireAtDayBoundary same-day error = %v, want %v", err, ErrScanBatchLifecycleDayBoundaryPending)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("same-day supplied time mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleCopiesAttemptInput(t *testing.T) {
	inputs := phase9AAttemptInputs()
	batch, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), inputs)
	if err != nil {
		t.Fatalf("NewScanBatchLifecycle returned error: %v", err)
	}
	inputs[0].AttemptID = "changed"
	inputs[0].WatchedGroupID = "changed"

	attempt, ok := batch.Attempt("attempt-001")
	if !ok {
		t.Fatalf("attempt-001 missing after input mutation")
	}
	if attempt.WatchedGroupID() != "group-001" {
		t.Fatalf("WatchedGroupID = %q, want group-001", attempt.WatchedGroupID())
	}
}

func TestScanBatchLifecycleReturnsDefensiveAttemptSnapshot(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	attempts := batch.Attempts()
	attempts[0].status = GroupAttemptStatusSucceeded
	attempts[0].watchedGroupID = "changed"

	attempt, ok := batch.Attempt("attempt-001")
	if !ok {
		t.Fatalf("attempt-001 missing")
	}
	if attempt.Status() != GroupAttemptStatusPending || attempt.WatchedGroupID() != "group-001" {
		t.Fatalf("Attempts returned mutable internal state: %#v", attempt)
	}
}

func TestScanBatchLifecycleInvalidTransitionDoesNotPartiallyMutate(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	before := batch.Attempts()

	if _, err := batch.SucceedAttempt("attempt-001", phase9ATime(t, 10, 30)); !errors.Is(err, ErrScanBatchLifecycleAttemptNotRunning) {
		t.Fatalf("SucceedAttempt pending error = %v, want %v", err, ErrScanBatchLifecycleAttemptNotRunning)
	}
	if !reflect.DeepEqual(batch.Attempts(), before) {
		t.Fatalf("invalid transition mutated lifecycle: got %#v want %#v", batch.Attempts(), before)
	}
}

func TestScanBatchLifecycleT19GroupFailureScenario(t *testing.T) {
	batch := newPhase9ALifecycle(t)
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 30))

	failed, err := batch.FailAttempt("attempt-001", phase9ATime(t, 10, 45))
	if err != nil {
		t.Fatalf("FailAttempt returned error: %v", err)
	}
	if failed.Status() != GroupAttemptStatusFailed {
		t.Fatalf("failed attempt status = %q, want %q", failed.Status(), GroupAttemptStatusFailed)
	}
	if batch.Summary().Failed != 1 || batch.Summary().Succeeded != 0 {
		t.Fatalf("failure summary was not preserved: %#v", batch.Summary())
	}
	if _, active := batch.ActiveAttempt(); active {
		t.Fatalf("failed group left an active attempt")
	}

	next, err := batch.StartNextPending(phase9ATime(t, 10, 50))
	if err != nil {
		t.Fatalf("explicit next action returned error: %v", err)
	}
	if next.AttemptID() != "attempt-002" {
		t.Fatalf("next attempt = %q, want attempt-002", next.AttemptID())
	}
}

func TestScanBatchLifecycleT20DayBoundaryScenario(t *testing.T) {
	batch := phase9ADayBoundaryLifecycle(t)

	if err := batch.ExpireAtDayBoundary(phase9ANextDayTime(t, 0, 5)); err != nil {
		t.Fatalf("ExpireAtDayBoundary returned error: %v", err)
	}
	assertPhase9AStatuses(t, batch.Attempts(), []GroupAttemptStatus{
		GroupAttemptStatusSucceeded,
		GroupAttemptStatusFailed,
		GroupAttemptStatusSkipped,
		GroupAttemptStatusExpiredAtDayBoundary,
		GroupAttemptStatusExpiredAtDayBoundary,
	})
	if !batch.IsTerminal() {
		t.Fatalf("batch not terminal after T20: %#v", batch.Summary())
	}
	if _, err := batch.StartNextPending(phase9ANextDayTime(t, 0, 6)); !errors.Is(err, ErrScanBatchLifecycleDayBoundaryReached) {
		t.Fatalf("StartNextPending after T20 error = %v, want %v", err, ErrScanBatchLifecycleDayBoundaryReached)
	}
}

func newPhase9ALifecycle(t *testing.T) ScanBatchLifecycle {
	t.Helper()

	batch, err := NewScanBatchLifecycle("batch-001", phase9AScanWindow(t, phase9AScanStart(t)), phase9AAttemptInputs())
	if err != nil {
		t.Fatalf("NewScanBatchLifecycle returned error: %v", err)
	}
	return batch
}

func completedPhase9ALifecycle(t *testing.T) ScanBatchLifecycle {
	t.Helper()

	batch := newPhase9ALifecycle(t)
	for i := 1; i <= domain.MaxScanRequestGroups; i++ {
		attempt := mustStartPhase9A(t, &batch, phase9ATime(t, 10, 10+i))
		mustSucceedPhase9A(t, &batch, attempt.AttemptID(), phase9ATime(t, 10, 20+i))
	}
	return batch
}

func phase9ADayBoundaryLifecycle(t *testing.T) ScanBatchLifecycle {
	t.Helper()

	batch := newPhase9ALifecycle(t)
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 10))
	mustSucceedPhase9A(t, &batch, "attempt-001", phase9ATime(t, 10, 20))
	mustStartPhase9A(t, &batch, phase9ATime(t, 10, 30))
	mustFailPhase9A(t, &batch, "attempt-002", phase9ATime(t, 10, 40))
	mustSkipPhase9A(t, &batch, "attempt-003", phase9ATime(t, 10, 50))
	mustStartPhase9A(t, &batch, phase9ATime(t, 23, 55))
	return batch
}

func mustStartPhase9A(t *testing.T, batch *ScanBatchLifecycle, at time.Time) GroupScanAttempt {
	t.Helper()

	attempt, err := batch.StartNextPending(at)
	if err != nil {
		t.Fatalf("StartNextPending returned error: %v", err)
	}
	return attempt
}

func mustSucceedPhase9A(t *testing.T, batch *ScanBatchLifecycle, attemptID string, at time.Time) {
	t.Helper()

	if _, err := batch.SucceedAttempt(attemptID, at); err != nil {
		t.Fatalf("SucceedAttempt(%q) returned error: %v", attemptID, err)
	}
}

func mustFailPhase9A(t *testing.T, batch *ScanBatchLifecycle, attemptID string, at time.Time) {
	t.Helper()

	if _, err := batch.FailAttempt(attemptID, at); err != nil {
		t.Fatalf("FailAttempt(%q) returned error: %v", attemptID, err)
	}
}

func mustSkipPhase9A(t *testing.T, batch *ScanBatchLifecycle, attemptID string, at time.Time) {
	t.Helper()

	if _, err := batch.SkipAttempt(attemptID, at); err != nil {
		t.Fatalf("SkipAttempt(%q) returned error: %v", attemptID, err)
	}
}

func phase9AAttemptInputs() []GroupScanAttemptInput {
	return []GroupScanAttemptInput{
		{AttemptID: "attempt-001", WatchedGroupID: "group-001"},
		{AttemptID: "attempt-002", WatchedGroupID: "group-002"},
		{AttemptID: "attempt-003", WatchedGroupID: "group-003"},
		{AttemptID: "attempt-004", WatchedGroupID: "group-004"},
		{AttemptID: "attempt-005", WatchedGroupID: "group-005"},
	}
}

func phase9AScanWindow(t *testing.T, scanStarted time.Time) domain.ScanWindow {
	t.Helper()

	year, month, day := scanStarted.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, scanStarted.Location())
	window, err := domain.NewScanWindow(startOfDay, startOfDay, scanStarted)
	if err != nil {
		t.Fatalf("NewScanWindow setup returned error: %v", err)
	}
	return window
}

func phase9AScanStart(t *testing.T) time.Time {
	t.Helper()

	return time.Date(2026, 8, 5, 10, 0, 0, 0, phase9ALocation(t))
}

func phase9ATime(t *testing.T, hour int, minute int) time.Time {
	t.Helper()

	return time.Date(2026, 8, 5, hour, minute, 0, 0, phase9ALocation(t))
}

func phase9ANextDayTime(t *testing.T, hour int, minute int) time.Time {
	t.Helper()

	return time.Date(2026, 8, 6, hour, minute, 0, 0, phase9ALocation(t))
}

func phase9ALocation(t *testing.T) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		t.Fatalf("LoadLocation(%q) returned error: %v", domain.RequiredTimezone, err)
	}
	return loc
}

func assertPhase9AStatus(t *testing.T, batch ScanBatchLifecycle, attemptID string, want GroupAttemptStatus) {
	t.Helper()

	attempt, ok := batch.Attempt(attemptID)
	if !ok {
		t.Fatalf("Attempt(%q) missing", attemptID)
	}
	if attempt.Status() != want {
		t.Fatalf("Attempt(%q).Status = %q, want %q", attemptID, attempt.Status(), want)
	}
}

func assertPhase9AStatuses(t *testing.T, attempts []GroupScanAttempt, want []GroupAttemptStatus) {
	t.Helper()

	if len(attempts) != len(want) {
		t.Fatalf("len(attempts) = %d, want %d", len(attempts), len(want))
	}
	for i, attempt := range attempts {
		if attempt.Status() != want[i] {
			t.Fatalf("attempt[%d].Status = %q, want %q", i, attempt.Status(), want[i])
		}
	}
}

func assertPhase9AGroupIDs(t *testing.T, attempts []GroupScanAttempt, want []string) {
	t.Helper()

	if len(attempts) != len(want) {
		t.Fatalf("len(attempts) = %d, want %d", len(attempts), len(want))
	}
	for i, attempt := range attempts {
		if attempt.WatchedGroupID() != want[i] {
			t.Fatalf("attempt[%d].WatchedGroupID = %q, want %q", i, attempt.WatchedGroupID(), want[i])
		}
	}
}
