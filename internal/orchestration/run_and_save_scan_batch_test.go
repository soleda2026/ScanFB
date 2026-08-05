package orchestration

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/persistence"
)

func TestRunAndSaveScanBatchSavesCompletedResult(t *testing.T) {
	recordID := validBatchRecordID(t, "batch-001")
	input := validScanBatchInput(t)
	repository := &recordingBatchRepository{}

	result, err := RunAndSaveScanBatch(recordID, input, repository)
	if err != nil {
		t.Fatalf("RunAndSaveScanBatch() error = %v", err)
	}

	if got, want := repository.saveCalls, 1; got != want {
		t.Fatalf("SaveBatch calls = %d, want %d", got, want)
	}
	if got, want := len(repository.saved), 1; got != want {
		t.Fatalf("saved records = %d, want %d", got, want)
	}
	if !reflect.DeepEqual(repository.saved[0], result.BatchRecord) {
		t.Fatalf("saved record differs from returned record:\nsaved %#v\nreturned %#v", repository.saved[0], result.BatchRecord)
	}
	if got, want := result.BatchRecord.ID(), recordID; got != want {
		t.Fatalf("BatchRecord.ID() = %q, want %q", got, want)
	}
	if got, want := result.ScanBatchResult.Summary(), result.BatchRecord.Summary(); !sameSummary(got, want) {
		t.Fatalf("summary not preserved:\nscan %#v\nrecord %#v", got, want)
	}
	if got, want := postIDs(result.ScanBatchResult.FlattenedPosts()), []string{"post-1", "post-2", "post-3", "post-4", "post-5", "post-6"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("flattened post order = %#v, want %#v", got, want)
	}
	if got, want := postIDsFromRecord(result.BatchRecord.FlattenedPosts()), []string{"post-1", "post-2", "post-3", "post-4", "post-5", "post-6"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("record post order = %#v, want %#v", got, want)
	}
	if got, want := result.BatchRecord.Summary().AllowedLeadCount, 1; got != want {
		t.Fatalf("AllowedLeadCount = %d, want %d", got, want)
	}
	if got, want := result.BatchRecord.Summary().BlockedLeadCount, 1; got != want {
		t.Fatalf("BlockedLeadCount = %d, want %d", got, want)
	}
	if got, want := result.BatchRecord.Summary().ReviewPostCount, 1; got != want {
		t.Fatalf("ReviewPostCount = %d, want %d", got, want)
	}
	if got, want := result.BatchRecord.Summary().ExcludedPostCount, 1; got != want {
		t.Fatalf("ExcludedPostCount = %d, want %d", got, want)
	}
	if got, want := result.BatchRecord.Summary().UnaggregatedPostCount, 1; got != want {
		t.Fatalf("UnaggregatedPostCount = %d, want %d", got, want)
	}
}

func TestRunAndSaveScanBatchUsesSaveOnlyRepositoryContract(t *testing.T) {
	var _ persistence.BatchRepository = (*recordingBatchRepository)(nil)

	result, err := RunAndSaveScanBatch(validBatchRecordID(t, "batch-001"), validScanBatchInput(t), &recordingBatchRepository{})
	if err != nil {
		t.Fatalf("RunAndSaveScanBatch() error = %v", err)
	}
	if err := result.BatchRecord.Validate(); err != nil {
		t.Fatalf("returned BatchRecord invalid: %v", err)
	}
}

func TestRunAndSaveScanBatchPropagatesSaveErrorAfterSingleSaveAttempt(t *testing.T) {
	wantErr := errors.New("save failed")
	repository := &recordingBatchRepository{err: wantErr}

	result, err := RunAndSaveScanBatch(validBatchRecordID(t, "batch-001"), validScanBatchInput(t), repository)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunAndSaveScanBatch() error = %v, want %v", err, wantErr)
	}
	if !reflect.DeepEqual(result, RunAndSaveScanBatchResult{}) {
		t.Fatalf("result on save failure = %#v, want zero", result)
	}
	if got, want := repository.saveCalls, 1; got != want {
		t.Fatalf("SaveBatch calls = %d, want %d", got, want)
	}
	if got, want := len(repository.saved), 1; got != want {
		t.Fatalf("saved attempts = %d, want %d", got, want)
	}
}

func TestRunAndSaveScanBatchRejectsNilRepositoryBeforeWork(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		result, err := RunAndSaveScanBatch(validBatchRecordID(t, "batch-001"), validScanBatchInput(t), nil)
		if !errors.Is(err, ErrNilBatchRepository) {
			t.Fatalf("RunAndSaveScanBatch() error = %v, want %v", err, ErrNilBatchRepository)
		}
		if !reflect.DeepEqual(result, RunAndSaveScanBatchResult{}) {
			t.Fatalf("result = %#v, want zero", result)
		}
	})

	t.Run("typed nil", func(t *testing.T) {
		var repository *recordingBatchRepository
		result, err := RunAndSaveScanBatch(validBatchRecordID(t, "batch-001"), validScanBatchInput(t), repository)
		if !errors.Is(err, ErrNilBatchRepository) {
			t.Fatalf("RunAndSaveScanBatch() error = %v, want %v", err, ErrNilBatchRepository)
		}
		if !reflect.DeepEqual(result, RunAndSaveScanBatchResult{}) {
			t.Fatalf("result = %#v, want zero", result)
		}
	})
}

func TestRunAndSaveScanBatchRejectsEmptyRecordIDBeforeSave(t *testing.T) {
	repository := &recordingBatchRepository{}

	result, err := RunAndSaveScanBatch("", validScanBatchInput(t), repository)
	if !errors.Is(err, persistence.ErrEmptyBatchRecordID) {
		t.Fatalf("RunAndSaveScanBatch() error = %v, want %v", err, persistence.ErrEmptyBatchRecordID)
	}
	if !reflect.DeepEqual(result, RunAndSaveScanBatchResult{}) {
		t.Fatalf("result = %#v, want zero", result)
	}
	if got := repository.saveCalls; got != 0 {
		t.Fatalf("SaveBatch calls = %d, want 0", got)
	}
}

func TestRunAndSaveScanBatchDoesNotSaveWhenScanFails(t *testing.T) {
	repository := &recordingBatchRepository{}
	input := validScanBatchInput(t)
	input.Groups = nil

	result, err := RunAndSaveScanBatch(validBatchRecordID(t, "batch-001"), input, repository)
	if !errors.Is(err, application.ErrScanBatchNoGroups) {
		t.Fatalf("RunAndSaveScanBatch() error = %v, want %v", err, application.ErrScanBatchNoGroups)
	}
	if !reflect.DeepEqual(result, RunAndSaveScanBatchResult{}) {
		t.Fatalf("result = %#v, want zero", result)
	}
	if got := repository.saveCalls; got != 0 {
		t.Fatalf("SaveBatch calls = %d, want 0", got)
	}
}

func TestRunAndSaveScanBatchPropagatesDuplicateIDFromRepository(t *testing.T) {
	repository := persistence.NewInMemoryBatchRepository()
	recordID := validBatchRecordID(t, "batch-001")
	input := validScanBatchInput(t)

	first, err := RunAndSaveScanBatch(recordID, input, repository)
	if err != nil {
		t.Fatalf("first RunAndSaveScanBatch() error = %v", err)
	}
	if got, want := repository.Count(), 1; got != want {
		t.Fatalf("Count() after first save = %d, want %d", got, want)
	}

	second, err := RunAndSaveScanBatch(recordID, input, repository)
	if !errors.Is(err, persistence.ErrBatchRecordAlreadyExists) {
		t.Fatalf("second RunAndSaveScanBatch() error = %v, want %v", err, persistence.ErrBatchRecordAlreadyExists)
	}
	if !reflect.DeepEqual(second, RunAndSaveScanBatchResult{}) {
		t.Fatalf("second result = %#v, want zero", second)
	}
	if got, want := repository.Count(), 1; got != want {
		t.Fatalf("Count() after duplicate = %d, want %d", got, want)
	}
	stored, ok := repository.RecordByID(recordID)
	if !ok {
		t.Fatalf("RecordByID(%q) not found", recordID)
	}
	if !reflect.DeepEqual(stored, first.BatchRecord) {
		t.Fatalf("duplicate save changed stored record:\ngot  %#v\nwant %#v", stored, first.BatchRecord)
	}
}

type recordingBatchRepository struct {
	saveCalls int
	saved     []persistence.BatchRecord
	err       error
}

func (repository *recordingBatchRepository) SaveBatch(record persistence.BatchRecord) error {
	repository.saveCalls++
	repository.saved = append(repository.saved, record)
	return repository.err
}

func validBatchRecordID(t *testing.T, value string) persistence.BatchRecordID {
	t.Helper()

	id, err := persistence.NewBatchRecordID(value)
	if err != nil {
		t.Fatalf("NewBatchRecordID(%q) error = %v", value, err)
	}
	return id
}

func validScanBatchInput(t *testing.T) application.ScanBatchInput {
	t.Helper()

	blockedEntry, err := blocklist.NewEntry(blocklist.IdentityKindFacebookUserID, "blocked-author", "Blocked Buyer")
	if err != nil {
		t.Fatalf("NewEntry() error = %v", err)
	}

	return application.ScanBatchInput{
		Groups: []application.GroupBatch{
			{
				GroupID:   "group-a",
				GroupName: "Group A",
				Posts: []domain.RawPost{
					testPost("post-1", "group-a", authorWithFacebookID("allowed-author"), "can mua MacBook HCM"),
					testPost("post-2", "group-a", authorWithFacebookID("allowed-author"), "can mua MacBook HCM"),
					testPost("post-3", "group-a", authorWithFacebookID("blocked-author"), "can mua MacBook HCM"),
				},
			},
			{
				GroupID:   "group-b",
				GroupName: "Group B",
				Posts: []domain.RawPost{
					testPost("post-4", "group-b", authorWithFacebookID("seller-author"), "Bán MacBook Pro HCM"),
					testPost("post-5", "group-b", authorWithFacebookID("review-author"), "can mua MacBook"),
					testPost("post-6", "group-b", domain.AuthorIdentity{DisplayName: "Needs Human"}, "can mua MacBook HCM"),
				},
			},
		},
		ScanWindow:     validScanWindow(t),
		SearchProfile:  domain.MacBookSearchProfile(),
		GeographicMode: domain.GeographicModeAllVietnam,
		Blocklist:      blocklist.NewList([]blocklist.Entry{blockedEntry}),
	}
}

func validScanWindow(t *testing.T) domain.ScanWindow {
	t.Helper()

	location, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	startOfDay := time.Date(2026, 8, 5, 0, 0, 0, 0, location)
	scanStarted := time.Date(2026, 8, 5, 10, 30, 0, 0, location)
	window, err := domain.NewScanWindow(startOfDay, startOfDay, scanStarted)
	if err != nil {
		t.Fatalf("NewScanWindow() error = %v", err)
	}
	return window
}

func testPost(postID string, groupID string, author domain.AuthorIdentity, body string) domain.RawPost {
	location, _ := time.LoadLocation(domain.RequiredTimezone)
	return domain.RawPost{
		PostID:     postID,
		GroupID:    groupID,
		GroupName:  groupID + " name",
		PostURL:    "https://facebook.example/posts/" + postID,
		Author:     author,
		Body:       body,
		CreatedAt:  time.Date(2026, 8, 5, 9, 0, 0, 0, location),
		CapturedAt: time.Date(2026, 8, 5, 10, 30, 0, 0, location),
	}
}

func authorWithFacebookID(id string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		FacebookUserID: id,
		DisplayName:    "Buyer " + id,
	}
}

func sameSummary(scan application.ScanBatchSummary, record persistence.BatchSummaryRecord) bool {
	return scan.GroupCount == record.GroupCount &&
		scan.InputPostCount == record.InputPostCount &&
		scan.EvaluatedPostCount == record.EvaluatedPostCount &&
		scan.IncludePostCount == record.IncludePostCount &&
		scan.ReviewPostCount == record.ReviewPostCount &&
		scan.ExcludedPostCount == record.ExcludedPostCount &&
		scan.AggregatedLeadCount == record.AggregatedLeadCount &&
		scan.AllowedLeadCount == record.AllowedLeadCount &&
		scan.BlockedLeadCount == record.BlockedLeadCount &&
		scan.UnresolvedLeadCount == record.UnresolvedLeadCount &&
		scan.UnaggregatedPostCount == record.UnaggregatedPostCount &&
		scan.SourceConflictCount == record.SourceConflictCount &&
		scan.AllowedLeadSourcePostCount == record.AllowedLeadSourcePostCount &&
		scan.BlockedLeadSourcePostCount == record.BlockedLeadSourcePostCount
}

func postIDs(posts []domain.RawPost) []string {
	ids := make([]string, len(posts))
	for i, post := range posts {
		ids[i] = post.PostID
	}
	return ids
}

func postIDsFromRecord(posts []domain.RawPost) []string {
	return postIDs(posts)
}
