package orchestration

import (
	"context"
	"errors"
	"testing"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/facebook"
)

func TestRunOneGroupScanWithPreparedSnapshotCollectorSucceeds(t *testing.T) {
	request := validOneGroupScanRequest(t)
	payload := []byte(`{
		"schema_version": 1,
		"posts": [{
			"post_id": "post-prepared-001",
			"post_url": "https://www.facebook.com/groups/source/posts/post-prepared-001",
			"author": {"display_name": "Buyer One"},
			"body": "Cần mua MacBook Pro tại HCM.",
			"created_at": "2026-08-12T09:00:00+07:00"
		}]
	}`)

	result, err := RunOneGroupScan(context.Background(), request, facebook.NewPreparedSnapshotCollector(payload))
	if err != nil {
		t.Fatalf("RunOneGroupScan() error = %v", err)
	}
	if result.Attempt().Status() != application.GroupAttemptStatusSucceeded || result.CollectedPostCount() != 1 {
		t.Fatalf("result status/count = %q/%d", result.Attempt().Status(), result.CollectedPostCount())
	}
	applicationResult, ok := result.ApplicationResult()
	if !ok {
		t.Fatal("ApplicationResult() missing")
	}
	posts := applicationResult.FlattenedPosts()
	if len(posts) != 1 || posts[0].PostID != "post-prepared-001" || posts[0].GroupID != request.WatchedGroup.ID() {
		t.Fatalf("flattened posts = %#v", posts)
	}
}

func TestRunOneGroupScanWithInvalidPreparedSnapshotRemainsFailed(t *testing.T) {
	request := validOneGroupScanRequest(t)
	collector := facebook.NewPreparedSnapshotCollector([]byte(`{"schema_version":1,"posts":[]}`))

	result, err := RunOneGroupScan(context.Background(), request, collector)
	if !errors.Is(err, ErrOneGroupScanCollectionFailed) || !errors.Is(err, facebook.ErrPreparedSnapshotInvalidPostCount) {
		t.Fatalf("RunOneGroupScan() error = %v", err)
	}
	if result.Attempt().Status() != application.GroupAttemptStatusFailed {
		t.Fatalf("attempt status = %q", result.Attempt().Status())
	}
	if result.FailureCode() != OneGroupScanFailureCollection || result.CollectedPostCount() != 0 {
		t.Fatalf("failure code/count = %q/%d", result.FailureCode(), result.CollectedPostCount())
	}
	if _, ok := result.ApplicationResult(); ok {
		t.Fatal("invalid payload fabricated application result")
	}
}
