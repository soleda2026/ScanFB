package application

import (
	"errors"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

var (
	ErrScanBatchLifecycleInvalidBatchID     = errors.New("application: scan batch lifecycle batch id is invalid")
	ErrScanBatchLifecycleInvalidScanWindow  = errors.New("application: scan batch lifecycle scan window is invalid")
	ErrScanBatchLifecycleInvalidGroupCount  = errors.New("application: scan batch lifecycle requires exactly five groups")
	ErrScanBatchLifecycleEmptyGroupID       = errors.New("application: scan batch lifecycle group id is empty")
	ErrScanBatchLifecycleDuplicateGroupID   = errors.New("application: scan batch lifecycle group id is duplicate")
	ErrScanBatchLifecycleInvalidAttemptID   = errors.New("application: scan batch lifecycle attempt id is invalid")
	ErrScanBatchLifecycleDuplicateAttemptID = errors.New("application: scan batch lifecycle attempt id is duplicate")
	ErrScanBatchLifecycleInvalidTime        = errors.New("application: scan batch lifecycle supplied time is invalid")
	ErrScanBatchLifecycleTimeBeforeScan     = errors.New("application: scan batch lifecycle time before scan start")
	ErrScanBatchLifecycleCompleteBeforeRun  = errors.New("application: scan batch lifecycle completion before attempt start")
	ErrScanBatchLifecycleAttemptNotFound    = errors.New("application: scan batch lifecycle attempt not found")
	ErrScanBatchLifecycleAttemptNotPending  = errors.New("application: scan batch lifecycle attempt is not pending")
	ErrScanBatchLifecycleAttemptNotRunning  = errors.New("application: scan batch lifecycle attempt is not running")
	ErrScanBatchLifecycleAttemptTerminal    = errors.New("application: scan batch lifecycle attempt is terminal")
	ErrScanBatchLifecycleAnotherRunning     = errors.New("application: scan batch lifecycle another attempt is running")
	ErrScanBatchLifecycleBatchTerminal      = errors.New("application: scan batch lifecycle batch is terminal")
	ErrScanBatchLifecycleDayBoundaryReached = errors.New("application: scan batch lifecycle day boundary reached")
	ErrScanBatchLifecycleDayBoundaryPending = errors.New("application: scan batch lifecycle day boundary not reached")
	ErrScanBatchLifecycleOutOfOrderAttempt  = errors.New("application: scan batch lifecycle attempt is out of order")
)

// GroupAttemptStatus is the finite lifecycle state for one watched group attempt.
type GroupAttemptStatus string

const (
	GroupAttemptStatusPending              GroupAttemptStatus = "pending"
	GroupAttemptStatusRunning              GroupAttemptStatus = "running"
	GroupAttemptStatusSucceeded            GroupAttemptStatus = "succeeded"
	GroupAttemptStatusFailed               GroupAttemptStatus = "failed"
	GroupAttemptStatusSkipped              GroupAttemptStatus = "skipped"
	GroupAttemptStatusExpiredAtDayBoundary GroupAttemptStatus = "expired_at_day_boundary"
)

// GroupScanAttemptInput contains caller-supplied identity for one planned group attempt.
type GroupScanAttemptInput struct {
	AttemptID      string
	WatchedGroupID string
}

// GroupScanAttempt records lifecycle state for one watched group inside a scan batch.
type GroupScanAttempt struct {
	attemptID      string
	batchID        string
	watchedGroupID string
	status         GroupAttemptStatus
	startedAt      time.Time
	completedAt    time.Time
}

// ScanBatchLifecycle models the ordered lifecycle of one production-shaped batch.
type ScanBatchLifecycle struct {
	batchID  string
	window   domain.ScanWindow
	attempts []GroupScanAttempt
}

// ScanBatchLifecycleSummary is count-only lifecycle accounting for one batch.
type ScanBatchLifecycleSummary struct {
	Total                int
	Pending              int
	Running              int
	Succeeded            int
	Failed               int
	Skipped              int
	ExpiredAtDayBoundary int
	Terminal             int
}

// NewScanBatchLifecycle creates one deterministic five-group lifecycle batch.
func NewScanBatchLifecycle(batchID string, window domain.ScanWindow, inputs []GroupScanAttemptInput) (ScanBatchLifecycle, error) {
	if len(inputs) != domain.MaxScanRequestGroups {
		return ScanBatchLifecycle{}, ErrScanBatchLifecycleInvalidGroupCount
	}
	return newScanLifecycle(batchID, window, inputs)
}

// NewOneGroupScanLifecycle creates one lifecycle containing exactly one caller-selected group.
func NewOneGroupScanLifecycle(scanID string, window domain.ScanWindow, input GroupScanAttemptInput) (ScanBatchLifecycle, error) {
	return newScanLifecycle(scanID, window, []GroupScanAttemptInput{input})
}

func newScanLifecycle(batchID string, window domain.ScanWindow, inputs []GroupScanAttemptInput) (ScanBatchLifecycle, error) {
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return ScanBatchLifecycle{}, ErrScanBatchLifecycleInvalidBatchID
	}
	if !validLifecycleScanWindow(window) {
		return ScanBatchLifecycle{}, ErrScanBatchLifecycleInvalidScanWindow
	}

	attempts := make([]GroupScanAttempt, len(inputs))
	seenAttemptIDs := make(map[string]struct{}, len(inputs))
	seenGroupIDs := make(map[string]struct{}, len(inputs))
	for i, input := range inputs {
		attemptID := strings.TrimSpace(input.AttemptID)
		if attemptID == "" {
			return ScanBatchLifecycle{}, ErrScanBatchLifecycleInvalidAttemptID
		}
		if _, exists := seenAttemptIDs[attemptID]; exists {
			return ScanBatchLifecycle{}, ErrScanBatchLifecycleDuplicateAttemptID
		}
		seenAttemptIDs[attemptID] = struct{}{}

		groupID := strings.TrimSpace(input.WatchedGroupID)
		if groupID == "" {
			return ScanBatchLifecycle{}, ErrScanBatchLifecycleEmptyGroupID
		}
		if _, exists := seenGroupIDs[groupID]; exists {
			return ScanBatchLifecycle{}, ErrScanBatchLifecycleDuplicateGroupID
		}
		seenGroupIDs[groupID] = struct{}{}

		attempts[i] = GroupScanAttempt{
			attemptID:      attemptID,
			batchID:        batchID,
			watchedGroupID: groupID,
			status:         GroupAttemptStatusPending,
		}
	}

	return ScanBatchLifecycle{
		batchID:  batchID,
		window:   window,
		attempts: copyGroupScanAttempts(attempts),
	}, nil
}

func (b ScanBatchLifecycle) BatchID() string {
	return b.batchID
}

func (b ScanBatchLifecycle) ScanWindow() domain.ScanWindow {
	return b.window
}

func (b ScanBatchLifecycle) Attempts() []GroupScanAttempt {
	return copyGroupScanAttempts(b.attempts)
}

func (b ScanBatchLifecycle) Attempt(attemptID string) (GroupScanAttempt, bool) {
	for _, attempt := range b.attempts {
		if attempt.attemptID == attemptID {
			return attempt, true
		}
	}
	return GroupScanAttempt{}, false
}

func (b ScanBatchLifecycle) Summary() ScanBatchLifecycleSummary {
	summary := ScanBatchLifecycleSummary{Total: len(b.attempts)}
	for _, attempt := range b.attempts {
		switch attempt.status {
		case GroupAttemptStatusPending:
			summary.Pending++
		case GroupAttemptStatusRunning:
			summary.Running++
		case GroupAttemptStatusSucceeded:
			summary.Succeeded++
			summary.Terminal++
		case GroupAttemptStatusFailed:
			summary.Failed++
			summary.Terminal++
		case GroupAttemptStatusSkipped:
			summary.Skipped++
			summary.Terminal++
		case GroupAttemptStatusExpiredAtDayBoundary:
			summary.ExpiredAtDayBoundary++
			summary.Terminal++
		}
	}
	return summary
}

func (b ScanBatchLifecycle) IsTerminal() bool {
	summary := b.Summary()
	return summary.Total > 0 && summary.Terminal == summary.Total
}

func (b ScanBatchLifecycle) ActiveAttempt() (GroupScanAttempt, bool) {
	for _, attempt := range b.attempts {
		if attempt.status == GroupAttemptStatusRunning {
			return attempt, true
		}
	}
	return GroupScanAttempt{}, false
}

func (b *ScanBatchLifecycle) StartNextPending(startedAt time.Time) (GroupScanAttempt, error) {
	if err := b.validateRegularTransitionTime(startedAt); err != nil {
		return GroupScanAttempt{}, err
	}

	index := b.firstPendingIndex()
	if index == -1 {
		return GroupScanAttempt{}, ErrScanBatchLifecycleBatchTerminal
	}
	return b.StartAttempt(b.attempts[index].attemptID, startedAt)
}

func (b *ScanBatchLifecycle) StartAttempt(attemptID string, startedAt time.Time) (GroupScanAttempt, error) {
	if err := b.validateRegularTransitionTime(startedAt); err != nil {
		return GroupScanAttempt{}, err
	}
	if b.runningIndex() != -1 {
		return GroupScanAttempt{}, ErrScanBatchLifecycleAnotherRunning
	}

	index := b.indexOfAttempt(attemptID)
	if index == -1 {
		return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptNotFound
	}
	if b.attempts[index].status != GroupAttemptStatusPending {
		if isTerminalAttemptStatus(b.attempts[index].status) {
			return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptTerminal
		}
		return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptNotPending
	}
	if index != b.firstPendingIndex() {
		return GroupScanAttempt{}, ErrScanBatchLifecycleOutOfOrderAttempt
	}

	b.attempts[index].status = GroupAttemptStatusRunning
	b.attempts[index].startedAt = startedAt
	return b.attempts[index], nil
}

func (b *ScanBatchLifecycle) SucceedAttempt(attemptID string, completedAt time.Time) (GroupScanAttempt, error) {
	index, err := b.validateRunningTransition(attemptID, completedAt)
	if err != nil {
		return GroupScanAttempt{}, err
	}

	b.attempts[index].status = GroupAttemptStatusSucceeded
	b.attempts[index].completedAt = completedAt
	return b.attempts[index], nil
}

func (b *ScanBatchLifecycle) FailAttempt(attemptID string, completedAt time.Time) (GroupScanAttempt, error) {
	index, err := b.validateRunningTransition(attemptID, completedAt)
	if err != nil {
		return GroupScanAttempt{}, err
	}

	b.attempts[index].status = GroupAttemptStatusFailed
	b.attempts[index].completedAt = completedAt
	return b.attempts[index], nil
}

func (b *ScanBatchLifecycle) SkipAttempt(attemptID string, completedAt time.Time) (GroupScanAttempt, error) {
	if err := b.validateRegularTransitionTime(completedAt); err != nil {
		return GroupScanAttempt{}, err
	}
	if b.runningIndex() != -1 {
		return GroupScanAttempt{}, ErrScanBatchLifecycleAnotherRunning
	}

	index := b.indexOfAttempt(attemptID)
	if index == -1 {
		return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptNotFound
	}
	if b.attempts[index].status != GroupAttemptStatusPending {
		if isTerminalAttemptStatus(b.attempts[index].status) {
			return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptTerminal
		}
		return GroupScanAttempt{}, ErrScanBatchLifecycleAttemptNotPending
	}
	if index != b.firstPendingIndex() {
		return GroupScanAttempt{}, ErrScanBatchLifecycleOutOfOrderAttempt
	}

	b.attempts[index].status = GroupAttemptStatusSkipped
	b.attempts[index].completedAt = completedAt
	return b.attempts[index], nil
}

func (b *ScanBatchLifecycle) ExpireAtDayBoundary(currentTime time.Time) error {
	if currentTime.IsZero() {
		return ErrScanBatchLifecycleInvalidTime
	}
	if !b.dayBoundaryReached(currentTime) {
		return ErrScanBatchLifecycleDayBoundaryPending
	}

	for i := range b.attempts {
		switch b.attempts[i].status {
		case GroupAttemptStatusPending, GroupAttemptStatusRunning:
			b.attempts[i].status = GroupAttemptStatusExpiredAtDayBoundary
			b.attempts[i].completedAt = currentTime
		}
	}
	return nil
}

func (a GroupScanAttempt) AttemptID() string {
	return a.attemptID
}

func (a GroupScanAttempt) BatchID() string {
	return a.batchID
}

func (a GroupScanAttempt) WatchedGroupID() string {
	return a.watchedGroupID
}

func (a GroupScanAttempt) Status() GroupAttemptStatus {
	return a.status
}

func (a GroupScanAttempt) StartedAt() (time.Time, bool) {
	if a.startedAt.IsZero() {
		return time.Time{}, false
	}
	return a.startedAt, true
}

func (a GroupScanAttempt) CompletedAt() (time.Time, bool) {
	if a.completedAt.IsZero() {
		return time.Time{}, false
	}
	return a.completedAt, true
}

func (b ScanBatchLifecycle) validateRegularTransitionTime(value time.Time) error {
	if value.IsZero() {
		return ErrScanBatchLifecycleInvalidTime
	}
	if value.Before(b.window.ScanStarted()) {
		return ErrScanBatchLifecycleTimeBeforeScan
	}
	if b.dayBoundaryReached(value) {
		return ErrScanBatchLifecycleDayBoundaryReached
	}
	if b.IsTerminal() {
		return ErrScanBatchLifecycleBatchTerminal
	}
	return nil
}

func (b ScanBatchLifecycle) validateRunningTransition(attemptID string, completedAt time.Time) (int, error) {
	if err := b.validateRegularTransitionTime(completedAt); err != nil {
		return -1, err
	}

	index := b.indexOfAttempt(attemptID)
	if index == -1 {
		return -1, ErrScanBatchLifecycleAttemptNotFound
	}
	if b.attempts[index].status != GroupAttemptStatusRunning {
		if isTerminalAttemptStatus(b.attempts[index].status) {
			return -1, ErrScanBatchLifecycleAttemptTerminal
		}
		return -1, ErrScanBatchLifecycleAttemptNotRunning
	}
	if completedAt.Before(b.attempts[index].startedAt) {
		return -1, ErrScanBatchLifecycleCompleteBeforeRun
	}
	return index, nil
}

func (b ScanBatchLifecycle) indexOfAttempt(attemptID string) int {
	attemptID = strings.TrimSpace(attemptID)
	if attemptID == "" {
		return -1
	}
	for i, attempt := range b.attempts {
		if attempt.attemptID == attemptID {
			return i
		}
	}
	return -1
}

func (b ScanBatchLifecycle) firstPendingIndex() int {
	for i, attempt := range b.attempts {
		if attempt.status == GroupAttemptStatusPending {
			return i
		}
	}
	return -1
}

func (b ScanBatchLifecycle) runningIndex() int {
	for i, attempt := range b.attempts {
		if attempt.status == GroupAttemptStatusRunning {
			return i
		}
	}
	return -1
}

func (b ScanBatchLifecycle) dayBoundaryReached(value time.Time) bool {
	local := value.In(b.window.ScanDate().Location())
	localYear, localMonth, localDay := local.Date()
	scanYear, scanMonth, scanDay := b.window.ScanDate().Date()
	return time.Date(localYear, localMonth, localDay, 0, 0, 0, 0, b.window.ScanDate().Location()).
		After(time.Date(scanYear, scanMonth, scanDay, 0, 0, 0, 0, b.window.ScanDate().Location()))
}

func validLifecycleScanWindow(window domain.ScanWindow) bool {
	return window.Timezone() == domain.RequiredTimezone &&
		!window.ScanDate().IsZero() &&
		!window.StartOfDay().IsZero() &&
		!window.ScanStarted().IsZero()
}

func isTerminalAttemptStatus(status GroupAttemptStatus) bool {
	switch status {
	case GroupAttemptStatusSucceeded,
		GroupAttemptStatusFailed,
		GroupAttemptStatusSkipped,
		GroupAttemptStatusExpiredAtDayBoundary:
		return true
	default:
		return false
	}
}

func copyGroupScanAttempts(attempts []GroupScanAttempt) []GroupScanAttempt {
	if len(attempts) == 0 {
		return nil
	}
	copied := make([]GroupScanAttempt, len(attempts))
	copy(copied, attempts)
	return copied
}
