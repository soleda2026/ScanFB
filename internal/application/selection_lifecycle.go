package application

import (
	"errors"

	"github.com/soleda2026/ScanFB/internal/domain"
)

var (
	ErrMalformedFiveGroupSelection    = errors.New("application: five group selection is malformed")
	ErrInvalidSelectionAttemptIDCount = errors.New("application: lifecycle mapping requires exactly five attempt ids")
)

// NewScanBatchLifecycleFromSelection maps one approved exact-five selection into pending lifecycle attempts.
func NewScanBatchLifecycleFromSelection(batchID string, window domain.ScanWindow, selection FiveGroupSelection, attemptIDs []string) (ScanBatchLifecycle, error) {
	groups := selection.Groups()
	if len(groups) != domain.MaxScanRequestGroups {
		return ScanBatchLifecycle{}, ErrMalformedFiveGroupSelection
	}
	if err := validateWatchedGroupSelectionSnapshot(groups); err != nil {
		return ScanBatchLifecycle{}, ErrMalformedFiveGroupSelection
	}
	for _, group := range groups {
		if !group.IsActive() {
			return ScanBatchLifecycle{}, ErrMalformedFiveGroupSelection
		}
	}
	if len(attemptIDs) != domain.MaxScanRequestGroups {
		return ScanBatchLifecycle{}, ErrInvalidSelectionAttemptIDCount
	}

	inputs := make([]GroupScanAttemptInput, domain.MaxScanRequestGroups)
	for i, group := range groups {
		inputs[i] = GroupScanAttemptInput{
			AttemptID:      attemptIDs[i],
			WatchedGroupID: group.ID(),
		}
	}
	return NewScanBatchLifecycle(batchID, window, inputs)
}
