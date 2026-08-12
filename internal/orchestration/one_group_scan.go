package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
)

var (
	ErrNilGroupPostCollector             = errors.New("orchestration: nil group post collector")
	ErrOneGroupScanInvalidContext        = errors.New("orchestration: one-group scan context is invalid")
	ErrOneGroupScanInvalidRequest        = errors.New("orchestration: one-group scan request is invalid")
	ErrOneGroupScanInactiveGroup         = errors.New("orchestration: one-group scan watched group is inactive")
	ErrOneGroupScanCollectionFailed      = errors.New("orchestration: one-group scan collection failed")
	ErrOneGroupScanGroupIdentityMismatch = errors.New("orchestration: one-group scan collected group identity mismatch")
	ErrOneGroupScanApplicationFailed     = errors.New("orchestration: one-group scan application processing failed")
	ErrOneGroupScanLifecycleFailed       = errors.New("orchestration: one-group scan lifecycle transition failed")
	ErrOneGroupScanCanceled              = errors.New("orchestration: one-group scan canceled")
)

// OneGroupScanFailureCode identifies the failed execution stage without adding a second result model.
type OneGroupScanFailureCode string

const (
	OneGroupScanFailureCollection    OneGroupScanFailureCode = "collection_failed"
	OneGroupScanFailureGroupIdentity OneGroupScanFailureCode = "group_identity_mismatch"
	OneGroupScanFailureApplication   OneGroupScanFailureCode = "application_failed"
	OneGroupScanFailureLifecycle     OneGroupScanFailureCode = "lifecycle_failed"
	OneGroupScanFailureCanceled      OneGroupScanFailureCode = "canceled"
)

type GroupCollectionRequest = application.GroupCollectionRequest
type GroupCollectionResult = application.GroupCollectionResult
type GroupPostCollector = application.GroupPostCollector
type OneGroupScanRequest = application.OneGroupScanRequest

// OneGroupScanResult contains the final Phase 9A attempt and optional Phase 5B result.
type OneGroupScanResult struct {
	attempt            application.GroupScanAttempt
	applicationResult  application.ScanBatchResult
	hasApplicationData bool
	collectedPostCount int
	failureCode        OneGroupScanFailureCode
}

func (r OneGroupScanResult) Attempt() application.GroupScanAttempt {
	return r.attempt
}

func (r OneGroupScanResult) ApplicationResult() (application.ScanBatchResult, bool) {
	if !r.hasApplicationData {
		return application.ScanBatchResult{}, false
	}
	return r.applicationResult, true
}

func (r OneGroupScanResult) CollectedPostCount() int {
	return r.collectedPostCount
}

func (r OneGroupScanResult) FailureCode() OneGroupScanFailureCode {
	return r.failureCode
}

// RunOneGroupScan performs one synchronous collection call and processes its posts in memory.
func RunOneGroupScan(ctx context.Context, request OneGroupScanRequest, collector GroupPostCollector) (OneGroupScanResult, error) {
	if ctx == nil {
		return OneGroupScanResult{}, ErrOneGroupScanInvalidContext
	}
	if groupPostCollectorIsNil(collector) {
		return OneGroupScanResult{}, ErrNilGroupPostCollector
	}
	if err := ctx.Err(); err != nil {
		return OneGroupScanResult{}, fmt.Errorf("%w: %w", ErrOneGroupScanCanceled, err)
	}
	if err := validateOneGroupScanRequest(request); err != nil {
		return OneGroupScanResult{}, err
	}
	if !request.WatchedGroup.IsActive() {
		return OneGroupScanResult{}, ErrOneGroupScanInactiveGroup
	}

	lifecycle, err := application.NewOneGroupScanLifecycle(
		request.ScanID,
		request.ScanWindow,
		application.GroupScanAttemptInput{
			AttemptID:      request.AttemptID,
			WatchedGroupID: request.WatchedGroup.ID(),
		},
	)
	if err != nil {
		return OneGroupScanResult{}, fmt.Errorf("%w: %w", ErrOneGroupScanInvalidRequest, err)
	}

	if _, err := lifecycle.StartNextPending(request.StartedAt); err != nil {
		if errors.Is(err, application.ErrScanBatchLifecycleDayBoundaryReached) {
			return expireOneGroupScan(&lifecycle, request.AttemptID, request.StartedAt, 0, err)
		}
		return oneGroupLifecycleResult(lifecycle, request.AttemptID, 0), fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, err)
	}

	collection, collectionErr := collector.CollectGroupPosts(ctx, GroupCollectionRequest{
		WatchedGroup: request.WatchedGroup,
		ScanWindow:   request.ScanWindow,
	})
	posts := collection.OrderedPosts()

	if err := lifecycle.ExpireAtDayBoundary(request.CompletedAt); err == nil {
		return oneGroupFailureResult(lifecycle, request.AttemptID, len(posts), OneGroupScanFailureLifecycle),
			fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, application.ErrScanBatchLifecycleDayBoundaryReached)
	} else if !errors.Is(err, application.ErrScanBatchLifecycleDayBoundaryPending) {
		return oneGroupLifecycleResult(lifecycle, request.AttemptID, len(posts)), fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, err)
	}

	if err := ctx.Err(); err != nil {
		return failOneGroupScan(&lifecycle, request, len(posts), OneGroupScanFailureCanceled, fmt.Errorf("%w: %w", ErrOneGroupScanCanceled, err))
	}
	if collectionErr != nil {
		return failOneGroupScan(&lifecycle, request, 0, OneGroupScanFailureCollection, fmt.Errorf("%w: %w", ErrOneGroupScanCollectionFailed, collectionErr))
	}
	if collection.WatchedGroupID != request.WatchedGroup.ID() {
		return failOneGroupScan(&lifecycle, request, len(posts), OneGroupScanFailureGroupIdentity, ErrOneGroupScanGroupIdentityMismatch)
	}

	applicationResult, err := application.RunScanBatch(application.ScanBatchInput{
		Groups: []application.GroupBatch{{
			GroupID:   request.WatchedGroup.ID(),
			GroupName: request.WatchedGroup.Name(),
			Posts:     posts,
		}},
		ScanWindow:     request.ScanWindow,
		SearchProfile:  request.SearchProfile,
		GeographicMode: request.GeographicMode,
		Blocklist:      request.Blocklist,
	})
	if err != nil {
		if errors.Is(err, application.ErrScanBatchPostGroupIDMismatch) {
			return failOneGroupScan(
				&lifecycle,
				request,
				len(posts),
				OneGroupScanFailureGroupIdentity,
				fmt.Errorf("%w: %w", ErrOneGroupScanGroupIdentityMismatch, err),
			)
		}
		return failOneGroupScan(
			&lifecycle,
			request,
			len(posts),
			OneGroupScanFailureApplication,
			fmt.Errorf("%w: %w", ErrOneGroupScanApplicationFailed, err),
		)
	}

	attempt, err := lifecycle.SucceedAttempt(request.AttemptID, request.CompletedAt)
	if err != nil {
		return oneGroupLifecycleResult(lifecycle, request.AttemptID, len(posts)), fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, err)
	}
	return OneGroupScanResult{
		attempt:            attempt,
		applicationResult:  applicationResult,
		hasApplicationData: true,
		collectedPostCount: len(posts),
	}, nil
}

func validateOneGroupScanRequest(request OneGroupScanRequest) error {
	if request.ScanID == "" || request.ScanID != strings.TrimSpace(request.ScanID) {
		return ErrOneGroupScanInvalidRequest
	}
	if request.AttemptID == "" || request.AttemptID != strings.TrimSpace(request.AttemptID) {
		return ErrOneGroupScanInvalidRequest
	}
	if err := request.WatchedGroup.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrOneGroupScanInvalidRequest, err)
	}
	if request.StartedAt.IsZero() || request.CompletedAt.IsZero() || request.CompletedAt.Before(request.StartedAt) {
		return ErrOneGroupScanInvalidRequest
	}
	return nil
}

func failOneGroupScan(
	lifecycle *application.ScanBatchLifecycle,
	request OneGroupScanRequest,
	collectedPostCount int,
	failureCode OneGroupScanFailureCode,
	cause error,
) (OneGroupScanResult, error) {
	attempt, err := lifecycle.FailAttempt(request.AttemptID, request.CompletedAt)
	if err != nil {
		return oneGroupLifecycleResult(*lifecycle, request.AttemptID, collectedPostCount), fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, err)
	}
	return OneGroupScanResult{
		attempt:            attempt,
		collectedPostCount: collectedPostCount,
		failureCode:        failureCode,
	}, cause
}

func expireOneGroupScan(
	lifecycle *application.ScanBatchLifecycle,
	attemptID string,
	currentTime time.Time,
	collectedPostCount int,
	cause error,
) (OneGroupScanResult, error) {
	if err := lifecycle.ExpireAtDayBoundary(currentTime); err != nil {
		return oneGroupLifecycleResult(*lifecycle, attemptID, collectedPostCount), fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, err)
	}
	return oneGroupFailureResult(*lifecycle, attemptID, collectedPostCount, OneGroupScanFailureLifecycle),
		fmt.Errorf("%w: %w", ErrOneGroupScanLifecycleFailed, cause)
}

func oneGroupFailureResult(
	lifecycle application.ScanBatchLifecycle,
	attemptID string,
	collectedPostCount int,
	failureCode OneGroupScanFailureCode,
) OneGroupScanResult {
	result := oneGroupLifecycleResult(lifecycle, attemptID, collectedPostCount)
	result.failureCode = failureCode
	return result
}

func oneGroupLifecycleResult(lifecycle application.ScanBatchLifecycle, attemptID string, collectedPostCount int) OneGroupScanResult {
	attempt, _ := lifecycle.Attempt(attemptID)
	return OneGroupScanResult{
		attempt:            attempt,
		collectedPostCount: collectedPostCount,
		failureCode:        OneGroupScanFailureLifecycle,
	}
}

func groupPostCollectorIsNil(collector GroupPostCollector) bool {
	if collector == nil {
		return true
	}
	value := reflect.ValueOf(collector)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
