package application

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestRunScanBatchValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(ScanBatchInput) ScanBatchInput
		wantErr error
	}{
		{
			name: "nil groups fail explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = nil
				return input
			},
			wantErr: ErrScanBatchNoGroups,
		},
		{
			name: "zero groups fail explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{}
				return input
			},
			wantErr: ErrScanBatchNoGroups,
		},
		{
			name: "one group with nil posts succeeds",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: "group-001"}}
				return input
			},
		},
		{
			name: "one group with empty posts succeeds",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: "group-001", Posts: []domain.RawPost{}}}
				return input
			},
		},
		{
			name: "five groups succeed",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{
					{GroupID: "group-001"},
					{GroupID: "group-002"},
					{GroupID: "group-003"},
					{GroupID: "group-004"},
					{GroupID: "group-005"},
				}
				return input
			},
		},
		{
			name: "six groups fail explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{
					{GroupID: "group-001"},
					{GroupID: "group-002"},
					{GroupID: "group-003"},
					{GroupID: "group-004"},
					{GroupID: "group-005"},
					{GroupID: "group-006"},
				}
				return input
			},
			wantErr: ErrScanBatchTooManyGroups,
		},
		{
			name: "empty GroupID fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: ""}}
				return input
			},
			wantErr: ErrScanBatchEmptyGroupID,
		},
		{
			name: "whitespace-only GroupID fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: " \t\n "}}
				return input
			},
			wantErr: ErrScanBatchEmptyGroupID,
		},
		{
			name: "duplicate GroupID fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: "group-001"}, {GroupID: "group-001"}}
				return input
			},
			wantErr: ErrScanBatchDuplicateGroupID,
		},
		{
			name: "duplicate normalized GroupID fails deterministically",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{GroupID: " group-001 "}, {GroupID: "group-001"}}
				return input
			},
			wantErr: ErrScanBatchDuplicateGroupID,
		},
		{
			name: "invalid ScanWindow fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.ScanWindow = domain.ScanWindow{}
				return input
			},
			wantErr: ErrInvalidPipelineScanWindow,
		},
		{
			name: "invalid SearchProfile fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.SearchProfile = domain.SearchProfile{}
				return input
			},
			wantErr: ErrInvalidPipelineSearchProfile,
		},
		{
			name: "unsupported GeographicMode fails explicitly",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.GeographicMode = domain.GeographicMode("outside_vietnam")
				return input
			},
			wantErr: ErrInvalidPipelineGeographicMode,
		},
		{
			name: "zero-value blocklist is safe",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{
					GroupID: "group-001",
					Posts: []domain.RawPost{
						batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
					},
				}}
				input.Blocklist = blocklist.List{}
				return input
			},
		},
		{
			name: "empty blocklist is safe",
			mutate: func(input ScanBatchInput) ScanBatchInput {
				input.Groups = []GroupBatch{{
					GroupID: "group-001",
					Posts: []domain.RawPost{
						batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
					},
				}}
				input.Blocklist = blocklist.NewList(nil)
				return input
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunScanBatch(tt.mutate(validScanBatchInput(t, nil)))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RunScanBatch error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if result.Summary() != (ScanBatchSummary{}) ||
					len(result.Groups()) != 0 ||
					len(result.FlattenedPosts()) != 0 ||
					len(result.Pipeline().Evaluated()) != 0 {
					t.Fatalf("invalid batch produced misleading output: %#v", result)
				}
				return
			}
			if result.Summary().GroupCount != len(result.Groups()) {
				t.Fatalf("GroupCount = %d, groups = %d", result.Summary().GroupCount, len(result.Groups()))
			}
		})
	}
}

func TestRunScanBatchGroupPostConsistency(t *testing.T) {
	matching := validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		},
	}})
	if _, err := RunScanBatch(matching); err != nil {
		t.Fatalf("matching post GroupID returned error: %v", err)
	}

	mismatchedPost := batchPost("post-002", "group-002", authorWithUserID("user-002"), "Cần mua MacBook Pro tại HCM")
	mismatched := validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts:   []domain.RawPost{mismatchedPost},
	}})
	result, err := RunScanBatch(mismatched)
	if !errors.Is(err, ErrScanBatchPostGroupIDMismatch) {
		t.Fatalf("mismatched post GroupID error = %v, want %v", err, ErrScanBatchPostGroupIDMismatch)
	}
	if result.Summary() != (ScanBatchSummary{}) || mismatched.Groups[0].Posts[0].GroupID != "group-002" {
		t.Fatalf("mismatched input was corrected or produced output: result=%#v input=%#v", result, mismatched.Groups[0].Posts[0])
	}

	missingGroupPost := batchPost("post-003", "", authorWithUserID("user-003"), "Cần mua MacBook Pro tại HCM")
	missing := validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts:   []domain.RawPost{missingGroupPost},
	}})
	missingResult, err := RunScanBatch(missing)
	if err != nil {
		t.Fatalf("missing post GroupID returned error: %v", err)
	}
	if got := missingResult.Pipeline().Allowed()[0].Lead.Sources()[0].Post.GroupID; got != "" {
		t.Fatalf("missing post GroupID was mutated to %q", got)
	}

	groups := []GroupBatch{
		{GroupID: "group-001", Posts: []domain.RawPost{batchPost("post-004", "group-001", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM")}},
		{GroupID: "group-002", Posts: []domain.RawPost{batchPost("post-005", "group-002", authorWithUserID("user-005"), "Cần mua MacBook Pro tại HCM")}},
	}
	original := copyGroupBatches(groups)
	ordered, err := RunScanBatch(validScanBatchInput(t, groups))
	if err != nil {
		t.Fatalf("ordered setup returned error: %v", err)
	}
	if !reflect.DeepEqual(groups, original) {
		t.Fatalf("RunScanBatch mutated input groups: got %#v want %#v", groups, original)
	}
	assertGroupIDs(t, ordered.Groups(), []string{"group-001", "group-002"})
	assertRawPostIDs(t, ordered.FlattenedPosts(), []string{"post-004", "post-005"})
}

func TestRunScanBatchFlatteningOrder(t *testing.T) {
	input := validScanBatchInput(t, []GroupBatch{
		{
			GroupID: "group-001",
			Posts: []domain.RawPost{
				batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				batchPost("post-002", "group-001", authorWithUserID("user-002"), "Cần mua MacBook Pro tại HCM"),
			},
		},
		{
			GroupID: "group-002",
			Posts: []domain.RawPost{
				batchPost("post-003", "group-002", authorWithUserID("user-003"), "Cần mua MacBook Pro tại HCM"),
				batchPost("post-004", "group-002", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM"),
			},
		},
	})

	result, err := RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch returned error: %v", err)
	}
	assertRawPostIDs(t, result.FlattenedPosts(), []string{"post-001", "post-002", "post-003", "post-004"})
	assertEvaluatedPostIDs(t, result.Pipeline().Evaluated(), []string{"post-001", "post-002", "post-003", "post-004"})

	again, err := RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch repeat returned error: %v", err)
	}
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("RunScanBatch repeated result changed:\ngot  %#v\nwant %#v", again, result)
	}
}

func TestRunScanBatchEndToEndFlowAndSummary(t *testing.T) {
	input := validScanBatchInput(t, []GroupBatch{
		{
			GroupID: "group-001",
			Posts: []domain.RawPost{
				batchPost("include-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				batchPost("review-001", "group-001", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
				batchPost("excluded-001", "group-001", authorWithUserID("user-003"), "Bán MacBook Pro giá tốt tại HCM"),
				batchPost("unaggregated-001", "group-001", domain.AuthorIdentity{DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro tại HCM"),
				batchPost("conflict-source", "group-001", authorWithUserID("user-008"), "Cần mua MacBook Pro 16 inch tại HCM"),
			},
		},
		{
			GroupID: "group-002",
			Posts: []domain.RawPost{
				batchPost("include-002", "group-002", authorWithUserID("user-001"), "  cần   mua   macbook pro tại HCM  "),
				batchPost("blocked-001", "group-002", authorWithUserID("user-005"), "Cần mua MacBook Pro tại HCM"),
				batchPost("conflict-source", "group-002", authorWithUserID("user-008"), "Mình đang tìm MacBook Air tại HCM"),
			},
		},
	})
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-005", "Blocked Buyer"),
	})

	result, err := RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch returned error: %v", err)
	}

	pipeline := result.Pipeline()
	assertEvaluatedPostIDs(t, pipeline.Evaluated(), []string{
		"include-001",
		"review-001",
		"excluded-001",
		"unaggregated-001",
		"conflict-source",
		"include-002",
		"blocked-001",
		"conflict-source",
	})
	if len(pipeline.Allowed()) != 2 || len(pipeline.Blocked()) != 1 || len(pipeline.Unaggregated()) != 1 || len(pipeline.Conflicts()) != 1 {
		t.Fatalf("pipeline counts = allowed %d blocked %d unaggregated %d conflicts %d",
			len(pipeline.Allowed()), len(pipeline.Blocked()), len(pipeline.Unaggregated()), len(pipeline.Conflicts()))
	}
	assertPipelineLeadSourcePostIDs(t, pipeline.Allowed()[0].Lead, []string{"include-001", "include-002"})
	assertPipelineLeadSourcePostIDs(t, pipeline.Allowed()[1].Lead, []string{"conflict-source"})
	assertPipelineLeadSourcePostIDs(t, pipeline.Blocked()[0].Lead, []string{"blocked-001"})

	wantSummary := ScanBatchSummary{
		GroupCount:                 2,
		InputPostCount:             8,
		EvaluatedPostCount:         8,
		IncludePostCount:           6,
		ReviewPostCount:            1,
		ExcludedPostCount:          1,
		AggregatedLeadCount:        3,
		AllowedLeadCount:           2,
		BlockedLeadCount:           1,
		UnresolvedLeadCount:        0,
		UnaggregatedPostCount:      1,
		SourceConflictCount:        1,
		AllowedLeadSourcePostCount: 3,
		BlockedLeadSourcePostCount: 1,
	}
	if got := result.Summary(); got != wantSummary {
		t.Fatalf("Summary = %#v, want %#v", got, wantSummary)
	}

	wantGroupSummaries := []GroupSummary{
		{GroupID: "group-001", InputPostCount: 5, EvaluatedPostCount: 5, IncludePostCount: 3, ReviewPostCount: 1, ExcludedPostCount: 1},
		{GroupID: "group-002", InputPostCount: 3, EvaluatedPostCount: 3, IncludePostCount: 3},
	}
	if got := result.GroupSummaries(); !reflect.DeepEqual(got, wantGroupSummaries) {
		t.Fatalf("GroupSummaries = %#v, want %#v", got, wantGroupSummaries)
	}

	groupRuleTotal := 0
	for _, summary := range result.GroupSummaries() {
		if summary.IncludePostCount+summary.ReviewPostCount+summary.ExcludedPostCount != summary.EvaluatedPostCount {
			t.Fatalf("group rule counts do not balance: %#v", summary)
		}
		groupRuleTotal += summary.EvaluatedPostCount
	}
	if groupRuleTotal != result.Summary().EvaluatedPostCount {
		t.Fatalf("group evaluated total = %d, batch evaluated = %d", groupRuleTotal, result.Summary().EvaluatedPostCount)
	}

	again, err := RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch repeat returned error: %v", err)
	}
	if !reflect.DeepEqual(again.Summary(), result.Summary()) {
		t.Fatalf("summary changed on repeat: got %#v want %#v", again.Summary(), result.Summary())
	}
}

func TestRunScanBatchSourcePreservation(t *testing.T) {
	createdAt := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC)
	capturedAt := time.Date(2026, 8, 5, 9, 31, 0, 0, time.UTC)
	first := batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM")
	first.CreatedAt = createdAt
	first.CapturedAt = capturedAt
	second := batchPost("post-002", "group-002", authorWithUserID("user-001"), "cần mua macbook pro tại HCM")

	result, err := RunScanBatch(validScanBatchInput(t, []GroupBatch{
		{GroupID: "group-001", Posts: []domain.RawPost{first}},
		{GroupID: "group-002", Posts: []domain.RawPost{second}},
	}))
	if err != nil {
		t.Fatalf("RunScanBatch returned error: %v", err)
	}

	sources := result.Pipeline().Allowed()[0].Lead.Sources()
	if len(sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(sources))
	}
	assertSourcePostPreserved(t, sources[0].Post, first)
	assertSourcePostPreserved(t, sources[1].Post, second)
}

func TestRunScanBatchImmutabilityAndDefensiveCopies(t *testing.T) {
	groups := []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-001", "group-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
			batchPost("post-002", "group-001", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		},
	}}
	originalGroups := copyGroupBatches(groups)
	input := validScanBatchInput(t, groups)
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-999", "Unused Block"),
	})
	originalProfileTerms := input.SearchProfile.ProductTerms()
	originalBlocklistEntries := input.Blocklist.Entries()

	result, err := RunScanBatch(input)
	if err != nil {
		t.Fatalf("RunScanBatch returned error: %v", err)
	}
	if !reflect.DeepEqual(groups, originalGroups) {
		t.Fatalf("RunScanBatch mutated group input: got %#v want %#v", groups, originalGroups)
	}
	if !reflect.DeepEqual(input.SearchProfile.ProductTerms(), originalProfileTerms) {
		t.Fatalf("RunScanBatch mutated SearchProfile terms")
	}
	if !reflect.DeepEqual(input.Blocklist.Entries(), originalBlocklistEntries) {
		t.Fatalf("RunScanBatch mutated blocklist")
	}

	returnedGroups := result.Groups()
	returnedGroups[0].GroupID = "changed"
	returnedGroups[0].Posts[0].Body = "changed"
	if result.Groups()[0].GroupID != "group-001" || result.Groups()[0].Posts[0].Body != groups[0].Posts[0].Body {
		t.Fatalf("Groups returned alias internal state: %#v", result.Groups())
	}

	flattened := result.FlattenedPosts()
	flattened[0].PostID = "changed"
	if result.FlattenedPosts()[0].PostID != "post-001" {
		t.Fatalf("FlattenedPosts returned alias internal state: %#v", result.FlattenedPosts())
	}

	summary := result.Summary()
	summary.InputPostCount = 0
	if result.Summary().InputPostCount != 2 {
		t.Fatalf("Summary returned alias internal state: %#v", result.Summary())
	}

	groupSummaries := result.GroupSummaries()
	groupSummaries[0].InputPostCount = 0
	if result.GroupSummaries()[0].InputPostCount != 2 {
		t.Fatalf("GroupSummaries returned alias internal state: %#v", result.GroupSummaries())
	}

	if !reflect.DeepEqual(result.GroupSummaries(), result.GroupSummaries()) ||
		!reflect.DeepEqual(result.FlattenedPosts(), result.FlattenedPosts()) {
		t.Fatalf("repeated reads were not stable")
	}
}

func TestRunScanBatchFailClosedBehavior(t *testing.T) {
	tooMany := validScanBatchInput(t, []GroupBatch{
		{GroupID: "group-001"},
		{GroupID: "group-002"},
		{GroupID: "group-003"},
		{GroupID: "group-004"},
		{GroupID: "group-005"},
		{GroupID: "group-006"},
	})
	if result, err := RunScanBatch(tooMany); !errors.Is(err, ErrScanBatchTooManyGroups) || result.Summary() != (ScanBatchSummary{}) {
		t.Fatalf("too many groups result = %#v err = %v", result, err)
	}

	duplicates := validScanBatchInput(t, []GroupBatch{{GroupID: "group-001"}, {GroupID: "group-001"}})
	if result, err := RunScanBatch(duplicates); !errors.Is(err, ErrScanBatchDuplicateGroupID) || len(result.Groups()) != 0 {
		t.Fatalf("duplicate groups were merged or processed: result=%#v err=%v", result, err)
	}

	mismatched := validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-001", "group-002", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		},
	}})
	if result, err := RunScanBatch(mismatched); !errors.Is(err, ErrScanBatchPostGroupIDMismatch) || len(result.FlattenedPosts()) != 0 {
		t.Fatalf("mismatched GroupID was corrected or processed: result=%#v err=%v", result, err)
	}

	unknown, err := RunScanBatch(validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-002", "group-001", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		},
	}}))
	if err != nil {
		t.Fatalf("unknown geography setup returned error: %v", err)
	}
	if len(unknown.Pipeline().Review()) != 1 || unknown.Summary().AllowedLeadCount != 0 {
		t.Fatalf("unknown geography did not remain review: summary=%#v review=%#v", unknown.Summary(), unknown.Pipeline().Review())
	}

	blockedInput := validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-003", "group-001", authorWithUserID("user-003"), "Cần mua MacBook Pro tại HCM"),
		},
	}})
	blockedInput.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-003", "Blocked Buyer"),
	})
	blocked, err := RunScanBatch(blockedInput)
	if err != nil {
		t.Fatalf("blocked setup returned error: %v", err)
	}
	if len(blocked.Pipeline().Blocked()) != 1 || len(blocked.Pipeline().Allowed()) != 0 {
		t.Fatalf("blocked lead appeared allowed: allowed=%#v blocked=%#v", blocked.Pipeline().Allowed(), blocked.Pipeline().Blocked())
	}

	conflict, err := RunScanBatch(validScanBatchInput(t, []GroupBatch{
		{
			GroupID: "group-001",
			Posts: []domain.RawPost{
				batchPost("post-004", "group-001", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM"),
			},
		},
		{
			GroupID: "group-002",
			Posts: []domain.RawPost{
				batchPost("post-004", "group-002", authorWithUserID("user-004"), "Mình đang tìm MacBook Air tại HCM"),
			},
		},
	}))
	if err != nil {
		t.Fatalf("conflict setup returned error: %v", err)
	}
	if len(conflict.Pipeline().Conflicts()) != 1 || conflict.Summary().SourceConflictCount != 1 {
		t.Fatalf("conflict was silently resolved: summary=%#v conflicts=%#v", conflict.Summary(), conflict.Pipeline().Conflicts())
	}

	differentAuthors, err := RunScanBatch(validScanBatchInput(t, []GroupBatch{{
		GroupID: "group-001",
		Posts: []domain.RawPost{
			batchPost("post-005", "group-001", authorWithUserID("user-005"), "Cần mua MacBook Pro tại HCM"),
			batchPost("post-006", "group-001", authorWithUserID("user-006"), "Cần mua MacBook Pro tại HCM"),
		},
	}}))
	if err != nil {
		t.Fatalf("different authors setup returned error: %v", err)
	}
	if differentAuthors.Summary().AllowedLeadCount != 2 {
		t.Fatalf("different authors merged from body similarity: summary=%#v", differentAuthors.Summary())
	}
}

func validScanBatchInput(t *testing.T, groups []GroupBatch) ScanBatchInput {
	t.Helper()

	return ScanBatchInput{
		Groups:         groups,
		ScanWindow:     validPipelineWindow(t),
		SearchProfile:  domain.MacBookSearchProfile(),
		GeographicMode: domain.GeographicModeHoChiMinhCity,
		Blocklist:      blocklist.NewList(nil),
	}
}

func batchPost(postID string, groupID string, author domain.AuthorIdentity, body string) domain.RawPost {
	return pipelinePost(postID, groupID, "https://facebook.example/posts/"+postID, author, body)
}

func assertRawPostIDs(t *testing.T, posts []domain.RawPost, want []string) {
	t.Helper()

	if len(posts) != len(want) {
		t.Fatalf("len(RawPost) = %d, want %d", len(posts), len(want))
	}
	for i, post := range posts {
		if post.PostID != want[i] {
			t.Fatalf("post[%d].PostID = %q, want %q", i, post.PostID, want[i])
		}
	}
}

func assertGroupIDs(t *testing.T, groups []GroupBatch, want []string) {
	t.Helper()

	if len(groups) != len(want) {
		t.Fatalf("len(GroupBatch) = %d, want %d", len(groups), len(want))
	}
	for i, group := range groups {
		if group.GroupID != want[i] {
			t.Fatalf("group[%d].GroupID = %q, want %q", i, group.GroupID, want[i])
		}
	}
}
