package persistence

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestNewBatchRecordPreservesCompletedBatchSnapshot(t *testing.T) {
	record := buildTestBatchRecord(t)

	if got, want := record.ID().String(), "batch-001"; got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}
	if got, want := record.SearchProfile().ID, "macbook"; got != want {
		t.Fatalf("SearchProfile().ID = %q, want %q", got, want)
	}
	if got, want := record.GeographicMode(), string(domain.GeographicModeAllVietnam); got != want {
		t.Fatalf("GeographicMode() = %q, want %q", got, want)
	}

	if got, want := postIDs(record.FlattenedPosts()), []string{"post-1", "post-2", "post-3", "post-4", "post-5", "post-6"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FlattenedPosts() IDs = %v, want %v", got, want)
	}
	if got, want := len(record.Groups()), 2; got != want {
		t.Fatalf("len(Groups()) = %d, want %d", got, want)
	}
	if got, want := record.Groups()[0].GroupID, "group-a"; got != want {
		t.Fatalf("first group = %q, want %q", got, want)
	}

	assertCount(t, "evaluated posts", len(record.EvaluatedPosts()), 6)
	assertCount(t, "included posts", len(record.IncludedPosts()), 4)
	assertCount(t, "review posts", len(record.ReviewPosts()), 1)
	assertCount(t, "excluded posts", len(record.ExcludedPosts()), 1)
	assertCount(t, "aggregated leads", len(record.Leads()), 2)
	assertCount(t, "allowed leads", len(record.AllowedLeads()), 1)
	assertCount(t, "blocked leads", len(record.BlockedLeads()), 1)
	assertCount(t, "unaggregated posts", len(record.Unaggregated()), 1)

	allowed := record.AllowedLeads()[0]
	if got, want := allowed.Match.Outcome, "not_blocked"; got != want {
		t.Fatalf("allowed match outcome = %q, want %q", got, want)
	}
	if got, want := allowed.Match.Reasons, []string{"blocklist.identity_not_found"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("allowed match reasons = %v, want %v", got, want)
	}
	if got, want := len(allowed.Lead.Sources), 2; got != want {
		t.Fatalf("allowed source count = %d, want %d", got, want)
	}
	if got, want := postIDsFromSources(allowed.Lead.Sources), []string{"post-1", "post-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("allowed source post IDs = %v, want %v", got, want)
	}

	blocked := record.BlockedLeads()[0]
	if got, want := blocked.Match.Outcome, "blocked"; got != want {
		t.Fatalf("blocked match outcome = %q, want %q", got, want)
	}
	if got, want := blocked.Match.Reasons, []string{"blocklist.identity_matched"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("blocked match reasons = %v, want %v", got, want)
	}
	if got, want := blocked.Match.MatchedEntry.DisplayName, "Blocked Buyer"; got != want {
		t.Fatalf("blocked matched entry display name = %q, want %q", got, want)
	}

	unaggregated := record.Unaggregated()[0]
	if got, want := unaggregated.Post.PostID, "post-6"; got != want {
		t.Fatalf("unaggregated post ID = %q, want %q", got, want)
	}
	if got, want := unaggregated.Reasons, []string{"dedup.stable_author_identity_missing"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unaggregated reasons = %v, want %v", got, want)
	}

	summary := record.Summary()
	if summary.InputPostCount != 6 ||
		summary.EvaluatedPostCount != 6 ||
		summary.IncludePostCount != 4 ||
		summary.ReviewPostCount != 1 ||
		summary.ExcludedPostCount != 1 ||
		summary.AggregatedLeadCount != 2 ||
		summary.AllowedLeadCount != 1 ||
		summary.BlockedLeadCount != 1 ||
		summary.UnaggregatedPostCount != 1 ||
		summary.AllowedLeadSourcePostCount != 2 ||
		summary.BlockedLeadSourcePostCount != 1 {
		t.Fatalf("summary was not preserved: %+v", summary)
	}
}

func TestBatchRecordAccessorsReturnCopies(t *testing.T) {
	record := buildTestBatchRecord(t)

	groups := record.Groups()
	groups[0].Posts[0].Body = "mutated"
	if got := record.Groups()[0].Posts[0].Body; got == "mutated" {
		t.Fatal("Groups returned mutable internal post slice")
	}

	profile := record.SearchProfile()
	profile.ProductTerms[0] = "mutated"
	if got := record.SearchProfile().ProductTerms[0]; got == "mutated" {
		t.Fatal("SearchProfile returned mutable internal terms")
	}

	allowed := record.AllowedLeads()
	allowed[0].Lead.Sources[0].Post.Body = "mutated"
	allowed[0].Match.Reasons[0] = "mutated"
	if got := record.AllowedLeads()[0].Lead.Sources[0].Post.Body; got == "mutated" {
		t.Fatal("AllowedLeads returned mutable internal source posts")
	}
	if got := record.AllowedLeads()[0].Match.Reasons[0]; got == "mutated" {
		t.Fatal("AllowedLeads returned mutable internal match reasons")
	}
}

func TestBatchRecordValidationRejectsMalformedRecords(t *testing.T) {
	tests := []struct {
		name string
		edit func(*BatchRecord)
		want error
	}{
		{
			name: "empty id",
			edit: func(record *BatchRecord) {
				record.id = ""
			},
			want: ErrEmptyBatchRecordID,
		},
		{
			name: "inactive search profile snapshot",
			edit: func(record *BatchRecord) {
				record.searchProfile.IsEnabled = false
			},
			want: ErrInvalidBatchRecord,
		},
		{
			name: "unsupported geographic mode",
			edit: func(record *BatchRecord) {
				record.geographicMode = "foreign"
			},
			want: ErrInvalidBatchRecord,
		},
		{
			name: "duplicate group id",
			edit: func(record *BatchRecord) {
				record.groups[1].GroupID = record.groups[0].GroupID
			},
			want: ErrInvalidBatchRecord,
		},
		{
			name: "wrong included decision",
			edit: func(record *BatchRecord) {
				record.includedPosts[0].Decision = "review"
			},
			want: ErrUnsupportedDecision,
		},
		{
			name: "unsupported decision",
			edit: func(record *BatchRecord) {
				record.evaluatedPosts[0].Decision = "hold"
			},
			want: ErrUnsupportedDecision,
		},
		{
			name: "wrong allowed outcome",
			edit: func(record *BatchRecord) {
				record.allowedLeads[0].Match.Outcome = "blocked"
			},
			want: ErrUnsupportedLeadOutcome,
		},
		{
			name: "summary mismatch",
			edit: func(record *BatchRecord) {
				record.summary.InputPostCount++
			},
			want: ErrInconsistentBatchSummary,
		},
		{
			name: "missing lead identity",
			edit: func(record *BatchRecord) {
				record.leads[0].Key.Value = ""
			},
			want: ErrInvalidBatchRecord,
		},
		{
			name: "source group mismatch",
			edit: func(record *BatchRecord) {
				record.groups[0].Posts[0].GroupID = "other-group"
			},
			want: ErrInvalidBatchRecord,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := buildTestBatchRecord(t)
			test.edit(&record)
			if err := record.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestBatchRecordValidationAcceptsUnresolvedOutcomeContract(t *testing.T) {
	record := buildTestBatchRecord(t)
	allowed := record.allowedLeads[0]
	record.allowedLeads = nil
	record.unresolvedLeads = []UnresolvedLeadRecord{{
		Lead: allowed.Lead,
		Match: BlocklistMatchRecord{
			Outcome: "insufficient_identity",
			Reasons: []string{"blocklist.stable_author_identity_missing"},
		},
		ApplicationReasons: []string{"application.blocklist_evaluation_unsupported"},
	}}
	record.summary.AllowedLeadCount = 0
	record.summary.AllowedLeadSourcePostCount = 0
	record.summary.UnresolvedLeadCount = 1

	if err := record.Validate(); err != nil {
		t.Fatalf("Validate() rejected unresolved outcome contract: %v", err)
	}
}

func TestBatchRecordIDRejectsBlankValues(t *testing.T) {
	if _, err := NewBatchRecordID(" \t\n "); !errors.Is(err, ErrEmptyBatchRecordID) {
		t.Fatalf("NewBatchRecordID(blank) error = %v, want %v", err, ErrEmptyBatchRecordID)
	}
}

func TestBatchRepositoryIsSaveOnlyContract(t *testing.T) {
	var _ BatchRepository = (*recordingBatchRepository)(nil)

	repo := &recordingBatchRepository{}
	record := buildTestBatchRecord(t)
	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if got, want := len(repo.saved), 1; got != want {
		t.Fatalf("saved records = %d, want %d", got, want)
	}
}

type recordingBatchRepository struct {
	saved []BatchRecord
}

func (repo *recordingBatchRepository) SaveBatch(record BatchRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	repo.saved = append(repo.saved, record)
	return nil
}

func buildTestBatchRecord(t *testing.T) BatchRecord {
	t.Helper()

	entry, err := blocklist.NewEntry(blocklist.IdentityKindFacebookUserID, "blocked-author", "Blocked Buyer")
	if err != nil {
		t.Fatalf("NewEntry() error = %v", err)
	}

	input := application.ScanBatchInput{
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
		Blocklist:      blocklist.NewList([]blocklist.Entry{entry}),
	}

	result, err := application.RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch() error = %v", err)
	}
	id, err := NewBatchRecordID("batch-001")
	if err != nil {
		t.Fatalf("NewBatchRecordID() error = %v", err)
	}
	record, err := NewBatchRecord(id, input, result)
	if err != nil {
		t.Fatalf("NewBatchRecord() error = %v", err)
	}
	return record
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

func postIDs(posts []domain.RawPost) []string {
	ids := make([]string, len(posts))
	for i, post := range posts {
		ids[i] = post.PostID
	}
	return ids
}

func postIDsFromSources(sources []SourcePostRecord) []string {
	ids := make([]string, len(sources))
	for i, source := range sources {
		ids[i] = source.Post.PostID
	}
	return ids
}

func assertCount(t *testing.T, label string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", label, got, want)
	}
}
