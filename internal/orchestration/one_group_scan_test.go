package orchestration

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestRunOneGroupScanSuccessUsesExactCollectorRequestAndPipeline(t *testing.T) {
	request := validOneGroupScanRequest(t)
	posts := []domain.RawPost{
		oneGroupPost(t, "post-001", request.WatchedGroup.ID(), "can mua MacBook HCM"),
		oneGroupPost(t, "post-002", request.WatchedGroup.ID(), "Bán MacBook Pro HCM"),
	}
	collector := &recordingGroupPostCollector{result: GroupCollectionResult{
		WatchedGroupID: request.WatchedGroup.ID(),
		Posts:          posts,
	}}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if err != nil {
		t.Fatalf("RunOneGroupScan() error = %v", err)
	}
	if collector.calls != 1 {
		t.Fatalf("collector calls = %d, want 1", collector.calls)
	}
	if len(collector.requests) != 1 || !reflect.DeepEqual(collector.requests[0].WatchedGroup, request.WatchedGroup) {
		t.Fatalf("collector group = %#v, want %#v", collector.requests, request.WatchedGroup)
	}
	if !reflect.DeepEqual(collector.requests[0].ScanWindow, request.ScanWindow) {
		t.Fatalf("collector ScanWindow = %#v, want %#v", collector.requests[0].ScanWindow, request.ScanWindow)
	}

	attempt := result.Attempt()
	if attempt.Status() != application.GroupAttemptStatusSucceeded {
		t.Fatalf("attempt status = %q, want %q", attempt.Status(), application.GroupAttemptStatusSucceeded)
	}
	if attempt.BatchID() != request.ScanID || attempt.AttemptID() != request.AttemptID || attempt.WatchedGroupID() != request.WatchedGroup.ID() {
		t.Fatalf("attempt identity = batch %q attempt %q group %q", attempt.BatchID(), attempt.AttemptID(), attempt.WatchedGroupID())
	}
	if startedAt, ok := attempt.StartedAt(); !ok || !startedAt.Equal(request.StartedAt) {
		t.Fatalf("attempt StartedAt = %v, %v; want %v, true", startedAt, ok, request.StartedAt)
	}
	if completedAt, ok := attempt.CompletedAt(); !ok || !completedAt.Equal(request.CompletedAt) {
		t.Fatalf("attempt CompletedAt = %v, %v; want %v, true", completedAt, ok, request.CompletedAt)
	}
	if result.CollectedPostCount() != 2 || result.FailureCode() != "" {
		t.Fatalf("result count/code = %d/%q, want 2/empty", result.CollectedPostCount(), result.FailureCode())
	}

	applicationResult, ok := result.ApplicationResult()
	if !ok {
		t.Fatal("ApplicationResult() missing after success")
	}
	if got := oneGroupPostIDs(applicationResult.FlattenedPosts()); !reflect.DeepEqual(got, []string{"post-001", "post-002"}) {
		t.Fatalf("flattened post order = %#v", got)
	}
	if got := applicationResult.Summary(); got.GroupCount != 1 || got.InputPostCount != 2 || got.EvaluatedPostCount != 2 {
		t.Fatalf("application summary = %#v", got)
	}
}

func TestRunOneGroupScanRejectsInactiveGroupBeforeCollection(t *testing.T) {
	request := validOneGroupScanRequest(t)
	request.WatchedGroup = request.WatchedGroup.WithActive(false)
	collector := &recordingGroupPostCollector{}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if !errors.Is(err, ErrOneGroupScanInactiveGroup) {
		t.Fatalf("RunOneGroupScan() error = %v, want %v", err, ErrOneGroupScanInactiveGroup)
	}
	if !reflect.DeepEqual(result, OneGroupScanResult{}) {
		t.Fatalf("result = %#v, want zero", result)
	}
	if collector.calls != 0 {
		t.Fatalf("collector calls = %d, want 0", collector.calls)
	}
}

func TestRunOneGroupScanCollectorFailureIsT19StyleFailedWithoutRetry(t *testing.T) {
	request := validOneGroupScanRequest(t)
	wantErr := errors.New("fixture collection failed")
	collector := &recordingGroupPostCollector{err: wantErr}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if !errors.Is(err, ErrOneGroupScanCollectionFailed) || !errors.Is(err, wantErr) {
		t.Fatalf("RunOneGroupScan() error = %v, want collection and underlying errors", err)
	}
	if collector.calls != 1 {
		t.Fatalf("collector calls = %d, want exactly 1", collector.calls)
	}
	if result.Attempt().Status() != application.GroupAttemptStatusFailed {
		t.Fatalf("attempt status = %q, want failed", result.Attempt().Status())
	}
	if result.Attempt().Status() == application.GroupAttemptStatusSucceeded {
		t.Fatal("failed attempt appeared succeeded")
	}
	if result.FailureCode() != OneGroupScanFailureCollection || result.CollectedPostCount() != 0 {
		t.Fatalf("failure code/count = %q/%d", result.FailureCode(), result.CollectedPostCount())
	}
	if _, ok := result.ApplicationResult(); ok {
		t.Fatal("collector failure fabricated an application result")
	}
}

func TestRunOneGroupScanRejectsCollectionGroupIdentityMismatch(t *testing.T) {
	request := validOneGroupScanRequest(t)
	collector := &recordingGroupPostCollector{result: GroupCollectionResult{
		WatchedGroupID: "another-group",
		Posts:          []domain.RawPost{oneGroupPost(t, "post-001", request.WatchedGroup.ID(), "can mua MacBook HCM")},
	}}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if !errors.Is(err, ErrOneGroupScanGroupIdentityMismatch) {
		t.Fatalf("RunOneGroupScan() error = %v, want %v", err, ErrOneGroupScanGroupIdentityMismatch)
	}
	assertFailedOneGroupResult(t, result, OneGroupScanFailureGroupIdentity)
}

func TestRunOneGroupScanReusesBatchPostGroupConsistency(t *testing.T) {
	request := validOneGroupScanRequest(t)
	collector := &recordingGroupPostCollector{result: GroupCollectionResult{
		WatchedGroupID: request.WatchedGroup.ID(),
		Posts:          []domain.RawPost{oneGroupPost(t, "post-001", "another-group", "can mua MacBook HCM")},
	}}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if !errors.Is(err, ErrOneGroupScanGroupIdentityMismatch) || !errors.Is(err, application.ErrScanBatchPostGroupIDMismatch) {
		t.Fatalf("RunOneGroupScan() error = %v, want identity and batch mismatch errors", err)
	}
	assertFailedOneGroupResult(t, result, OneGroupScanFailureGroupIdentity)
}

func TestRunOneGroupScanApplicationFailureLeavesAttemptFailed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*OneGroupScanRequest)
		want   error
	}{
		{name: "invalid profile", mutate: func(request *OneGroupScanRequest) { request.SearchProfile = domain.SearchProfile{} }, want: application.ErrInvalidPipelineSearchProfile},
		{name: "invalid mode", mutate: func(request *OneGroupScanRequest) { request.GeographicMode = domain.GeographicMode("invalid") }, want: application.ErrInvalidPipelineGeographicMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validOneGroupScanRequest(t)
			tt.mutate(&request)
			collector := &recordingGroupPostCollector{result: GroupCollectionResult{
				WatchedGroupID: request.WatchedGroup.ID(),
				Posts:          []domain.RawPost{oneGroupPost(t, "post-001", request.WatchedGroup.ID(), "can mua MacBook HCM")},
			}}

			result, err := RunOneGroupScan(context.Background(), request, collector)
			if !errors.Is(err, ErrOneGroupScanApplicationFailed) || !errors.Is(err, tt.want) {
				t.Fatalf("RunOneGroupScan() error = %v, want application and %v", err, tt.want)
			}
			if collector.calls != 1 {
				t.Fatalf("collector calls = %d, want 1", collector.calls)
			}
			assertFailedOneGroupResult(t, result, OneGroupScanFailureApplication)
		})
	}
}

func TestRunOneGroupScanZeroPostsSucceedsExplicitly(t *testing.T) {
	request := validOneGroupScanRequest(t)
	collector := &recordingGroupPostCollector{result: GroupCollectionResult{WatchedGroupID: request.WatchedGroup.ID()}}

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if err != nil {
		t.Fatalf("RunOneGroupScan() error = %v", err)
	}
	if result.Attempt().Status() != application.GroupAttemptStatusSucceeded || result.CollectedPostCount() != 0 {
		t.Fatalf("zero-post result status/count = %q/%d", result.Attempt().Status(), result.CollectedPostCount())
	}
	applicationResult, ok := result.ApplicationResult()
	if !ok || applicationResult.Summary().InputPostCount != 0 || applicationResult.Summary().EvaluatedPostCount != 0 {
		t.Fatalf("zero-post application result = %#v, present %v", applicationResult.Summary(), ok)
	}
}

func TestRunOneGroupScanInvalidRequestFailsBeforeCollection(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*OneGroupScanRequest)
		want   error
	}{
		{name: "empty scan id", mutate: func(request *OneGroupScanRequest) { request.ScanID = " " }, want: ErrOneGroupScanInvalidRequest},
		{name: "empty attempt id", mutate: func(request *OneGroupScanRequest) { request.AttemptID = "\t" }, want: ErrOneGroupScanInvalidRequest},
		{name: "invalid window", mutate: func(request *OneGroupScanRequest) { request.ScanWindow = domain.ScanWindow{} }, want: application.ErrScanBatchLifecycleInvalidScanWindow},
		{name: "zero started time", mutate: func(request *OneGroupScanRequest) { request.StartedAt = time.Time{} }, want: ErrOneGroupScanInvalidRequest},
		{name: "completion before start", mutate: func(request *OneGroupScanRequest) { request.CompletedAt = request.StartedAt.Add(-time.Second) }, want: ErrOneGroupScanInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validOneGroupScanRequest(t)
			tt.mutate(&request)
			collector := &recordingGroupPostCollector{}
			result, err := RunOneGroupScan(context.Background(), request, collector)
			if !errors.Is(err, tt.want) {
				t.Fatalf("RunOneGroupScan() error = %v, want %v", err, tt.want)
			}
			if !reflect.DeepEqual(result, OneGroupScanResult{}) || collector.calls != 0 {
				t.Fatalf("invalid request result/calls = %#v/%d", result, collector.calls)
			}
		})
	}
}

func TestRunOneGroupScanDayBoundaryUsesExistingExpirationSemantics(t *testing.T) {
	t.Run("pending expires before collector", func(t *testing.T) {
		request := validOneGroupScanRequest(t)
		request.StartedAt = oneGroupTime(t, 2026, 8, 13, 0, 1)
		request.CompletedAt = request.StartedAt
		collector := &recordingGroupPostCollector{}

		result, err := RunOneGroupScan(context.Background(), request, collector)
		if !errors.Is(err, ErrOneGroupScanLifecycleFailed) || !errors.Is(err, application.ErrScanBatchLifecycleDayBoundaryReached) {
			t.Fatalf("RunOneGroupScan() error = %v", err)
		}
		if collector.calls != 0 || result.Attempt().Status() != application.GroupAttemptStatusExpiredAtDayBoundary {
			t.Fatalf("pending boundary calls/status = %d/%q", collector.calls, result.Attempt().Status())
		}
	})

	t.Run("running expires after one collection", func(t *testing.T) {
		request := validOneGroupScanRequest(t)
		request.CompletedAt = oneGroupTime(t, 2026, 8, 13, 0, 1)
		collector := &recordingGroupPostCollector{result: GroupCollectionResult{WatchedGroupID: request.WatchedGroup.ID()}}

		result, err := RunOneGroupScan(context.Background(), request, collector)
		if !errors.Is(err, ErrOneGroupScanLifecycleFailed) || !errors.Is(err, application.ErrScanBatchLifecycleDayBoundaryReached) {
			t.Fatalf("RunOneGroupScan() error = %v", err)
		}
		if collector.calls != 1 || result.Attempt().Status() != application.GroupAttemptStatusExpiredAtDayBoundary {
			t.Fatalf("running boundary calls/status = %d/%q", collector.calls, result.Attempt().Status())
		}
		if _, ok := result.ApplicationResult(); ok {
			t.Fatal("day-boundary expiration fabricated application result")
		}
	})
}

func TestRunOneGroupScanCancellationIsExplicitAndNeverRetries(t *testing.T) {
	t.Run("already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		collector := &recordingGroupPostCollector{}

		result, err := RunOneGroupScan(ctx, validOneGroupScanRequest(t), collector)
		if !errors.Is(err, ErrOneGroupScanCanceled) || !errors.Is(err, context.Canceled) {
			t.Fatalf("RunOneGroupScan() error = %v", err)
		}
		if collector.calls != 0 || !reflect.DeepEqual(result, OneGroupScanResult{}) {
			t.Fatalf("pre-canceled calls/result = %d/%#v", collector.calls, result)
		}
	})

	t.Run("canceled by collector", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		collector := &recordingGroupPostCollector{collect: func(ctx context.Context, _ GroupCollectionRequest) (GroupCollectionResult, error) {
			cancel()
			return GroupCollectionResult{}, ctx.Err()
		}}

		result, err := RunOneGroupScan(ctx, validOneGroupScanRequest(t), collector)
		if !errors.Is(err, ErrOneGroupScanCanceled) || !errors.Is(err, context.Canceled) {
			t.Fatalf("RunOneGroupScan() error = %v", err)
		}
		if collector.calls != 1 {
			t.Fatalf("collector calls = %d, want 1", collector.calls)
		}
		assertFailedOneGroupResult(t, result, OneGroupScanFailureCanceled)
	})
}

func TestRunOneGroupScanRejectsNilContextAndCollector(t *testing.T) {
	request := validOneGroupScanRequest(t)
	collector := &recordingGroupPostCollector{}
	if _, err := RunOneGroupScan(nil, request, collector); !errors.Is(err, ErrOneGroupScanInvalidContext) {
		t.Fatalf("nil context error = %v", err)
	}
	if _, err := RunOneGroupScan(context.Background(), request, nil); !errors.Is(err, ErrNilGroupPostCollector) {
		t.Fatalf("nil collector error = %v", err)
	}
	var typedNil *recordingGroupPostCollector
	if _, err := RunOneGroupScan(context.Background(), request, typedNil); !errors.Is(err, ErrNilGroupPostCollector) {
		t.Fatalf("typed nil collector error = %v", err)
	}
}

func TestRunOneGroupScanIsDeterministicAndDefensive(t *testing.T) {
	request := validOneGroupScanRequest(t)
	posts := []domain.RawPost{oneGroupPost(t, "post-001", request.WatchedGroup.ID(), "can mua MacBook HCM")}
	run := func() (OneGroupScanResult, error) {
		collector := &recordingGroupPostCollector{result: GroupCollectionResult{WatchedGroupID: request.WatchedGroup.ID(), Posts: posts}}
		return RunOneGroupScan(context.Background(), request, collector)
	}

	first, firstErr := run()
	second, secondErr := run()
	if firstErr != nil || secondErr != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated result differs: first %#v/%v second %#v/%v", first, firstErr, second, secondErr)
	}

	posts[0].Body = "mutated after collection"
	applicationResult, ok := first.ApplicationResult()
	if !ok || applicationResult.FlattenedPosts()[0].Body != "can mua MacBook HCM" {
		t.Fatalf("collector input mutation leaked into result: %#v", applicationResult.FlattenedPosts())
	}
	returned := applicationResult.FlattenedPosts()
	returned[0].Body = "mutated returned slice"
	applicationResultAgain, _ := first.ApplicationResult()
	if applicationResultAgain.FlattenedPosts()[0].Body != "can mua MacBook HCM" {
		t.Fatal("returned slice mutation leaked into stored result")
	}
}

func TestOneGroupScanSourceHasNoForbiddenRuntimeEdges(t *testing.T) {
	source, err := os.ReadFile("one_group_scan.go")
	if err != nil {
		t.Fatalf("ReadFile(one_group_scan.go) error = %v", err)
	}
	text := string(source)
	for _, forbidden := range []string{
		"time.Now(", "go func", "internal/facebook", "internal/persistence",
		"net/http", "SelectNextFive", "WatchedGroupSelectionCursor",
		"RunAndSaveScanBatch", "SaveBatch(", "AcquireSafari", "uuid.", "rand.",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("one_group_scan.go contains forbidden runtime edge %q", forbidden)
		}
	}
}

type recordingGroupPostCollector struct {
	calls    int
	requests []GroupCollectionRequest
	result   GroupCollectionResult
	err      error
	collect  func(context.Context, GroupCollectionRequest) (GroupCollectionResult, error)
}

func (collector *recordingGroupPostCollector) CollectGroupPosts(ctx context.Context, request GroupCollectionRequest) (GroupCollectionResult, error) {
	collector.calls++
	collector.requests = append(collector.requests, request)
	if collector.collect != nil {
		return collector.collect(ctx, request)
	}
	return collector.result, collector.err
}

func validOneGroupScanRequest(t *testing.T) OneGroupScanRequest {
	t.Helper()
	createdAt := oneGroupTime(t, 2026, 8, 12, 8, 0)
	group, err := domain.NewWatchedGroup(
		"group-001",
		"facebook-group-001",
		"https://www.facebook.com/groups/group-001",
		"Group One",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	scanStarted := oneGroupTime(t, 2026, 8, 12, 10, 0)
	startOfDay := oneGroupTime(t, 2026, 8, 12, 0, 0)
	window, err := domain.NewScanWindow(startOfDay, startOfDay, scanStarted)
	if err != nil {
		t.Fatalf("NewScanWindow() error = %v", err)
	}
	return OneGroupScanRequest{
		ScanID:         "scan-001",
		AttemptID:      "attempt-001",
		WatchedGroup:   group,
		ScanWindow:     window,
		SearchProfile:  domain.MacBookSearchProfile(),
		GeographicMode: domain.GeographicModeAllVietnam,
		Blocklist:      blocklist.NewList(nil),
		StartedAt:      scanStarted,
		CompletedAt:    scanStarted.Add(5 * time.Minute),
	}
}

func oneGroupPost(t *testing.T, postID string, groupID string, body string) domain.RawPost {
	t.Helper()
	return domain.RawPost{
		PostID:    postID,
		GroupID:   groupID,
		GroupName: "Group One",
		PostURL:   "https://facebook.example/posts/" + postID,
		Author: domain.AuthorIdentity{
			FacebookUserID: "buyer-001",
			DisplayName:    "Buyer One",
		},
		Body:       body,
		CreatedAt:  oneGroupTime(t, 2026, 8, 12, 9, 0),
		CapturedAt: oneGroupTime(t, 2026, 8, 12, 10, 0),
	}
}

func oneGroupTime(t *testing.T, year int, month time.Month, day int, hour int, minute int) time.Time {
	t.Helper()
	location, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}

func oneGroupPostIDs(posts []domain.RawPost) []string {
	ids := make([]string, len(posts))
	for i, post := range posts {
		ids[i] = post.PostID
	}
	return ids
}

func assertFailedOneGroupResult(t *testing.T, result OneGroupScanResult, wantCode OneGroupScanFailureCode) {
	t.Helper()
	if result.Attempt().Status() != application.GroupAttemptStatusFailed {
		t.Fatalf("attempt status = %q, want failed", result.Attempt().Status())
	}
	if result.FailureCode() != wantCode {
		t.Fatalf("FailureCode() = %q, want %q", result.FailureCode(), wantCode)
	}
	if _, ok := result.ApplicationResult(); ok {
		t.Fatal("failed result unexpectedly contains application result")
	}
}
