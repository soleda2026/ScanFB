package application

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/dedup"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/rules"
)

func TestRunEvaluationPipelineBasicEndToEndFlow(t *testing.T) {
	input := validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
	})

	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}
	if len(result.Allowed()) != 1 || len(result.Blocked()) != 0 || len(result.Unresolved()) != 0 {
		t.Fatalf("lead filtering result = allowed %d blocked %d unresolved %d", len(result.Allowed()), len(result.Blocked()), len(result.Unresolved()))
	}
	assertPipelineLeadSourcePostIDs(t, result.Allowed()[0].Lead, []string{"post-1"})

	again, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline repeat returned error: %v", err)
	}
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("RunEvaluationPipeline repeated result = %#v, want %#v", again, result)
	}
}

func TestRunEvaluationPipelineAggregatesSameAuthorSameNeed(t *testing.T) {
	input := validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("post-2", "group-2", "https://facebook.example/post-002", authorWithUserID("user-001"), "  cần   mua   macbook pro tại hcm  "),
	})

	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}
	if len(result.Allowed()) != 1 {
		t.Fatalf("len(Allowed) = %d, want 1", len(result.Allowed()))
	}
	assertPipelineLeadSourcePostIDs(t, result.Allowed()[0].Lead, []string{"post-1", "post-2"})
	if len(result.Evaluated()) != 2 || len(result.Eligible()) != 2 {
		t.Fatalf("evaluated/eligible counts = %d/%d, want 2/2", len(result.Evaluated()), len(result.Eligible()))
	}
}

func TestRunEvaluationPipelineBlockedAuthor(t *testing.T) {
	input := validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
	})
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-001", "Nguyen Van A"),
	})

	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}
	if len(result.Blocked()) != 1 || len(result.Allowed()) != 0 {
		t.Fatalf("blocked author result = allowed %d blocked %d", len(result.Allowed()), len(result.Blocked()))
	}
	assertPipelineLeadSourcePostIDs(t, result.Blocked()[0].Lead, []string{"post-1"})
}

func TestRunEvaluationPipelineMixedRuleResults(t *testing.T) {
	posts := []domain.RawPost{
		pipelinePost("include-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("review-1", "group-1", "https://facebook.example/post-002", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		pipelinePost("exclude-1", "group-1", "https://facebook.example/post-003", authorWithUserID("user-003"), "Bán MacBook Pro giá tốt tại HCM"),
	}
	result, err := RunEvaluationPipeline(validPipelineInput(t, posts))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}

	assertEvaluatedPostIDs(t, result.Evaluated(), []string{"include-1", "review-1", "exclude-1"})
	assertEvaluatedPostIDs(t, result.Eligible(), []string{"include-1"})
	assertReviewPostIDs(t, result.Review(), []string{"review-1"})
	assertExcludedPostIDs(t, result.Excluded(), []string{"exclude-1"})
	if len(result.Allowed()) != 1 {
		t.Fatalf("len(Allowed) = %d, want 1", len(result.Allowed()))
	}
	if len(result.AggregatedLeads()) != 1 {
		t.Fatalf("aggregation received non-include posts: leads=%#v", result.AggregatedLeads())
	}
	assertRuleReasons(t, result.Review()[0].Result, rules.DecisionReview, []rules.ReasonCode{rules.ReasonLocationUnknown})
	assertRuleReasons(t, result.Excluded()[0].Result, rules.DecisionExclude, []rules.ReasonCode{rules.ReasonSellerIntent})
}

func TestRunEvaluationPipelineRuleStage(t *testing.T) {
	loc := pipelineLocation()
	tests := []struct {
		name         string
		post         domain.RawPost
		mode         domain.GeographicMode
		wantDecision rules.Decision
		wantReasons  []rules.ReasonCode
	}{
		{
			name:         "eligible include proceeds to aggregation",
			post:         pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionInclude,
		},
		{
			name:         "seller post is excluded",
			post:         pipelinePost("post-2", "group-1", "https://facebook.example/post-002", authorWithUserID("user-001"), "Bán MacBook Pro giá tốt tại HCM"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionExclude,
			wantReasons:  []rules.ReasonCode{rules.ReasonSellerIntent},
		},
		{
			name:         "anonymous author is excluded",
			post:         pipelinePost("post-3", "group-1", "https://facebook.example/post-003", domain.AuthorIdentity{FacebookUserID: "user-001", DisplayName: "Anonymous"}, "Cần mua MacBook Pro tại HCM"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionExclude,
			wantReasons:  []rules.ReasonCode{rules.ReasonAnonymousAuthor},
		},
		{
			name:         "one-word display name is excluded",
			post:         pipelinePost("post-4", "group-1", "https://facebook.example/post-004", domain.AuthorIdentity{FacebookUserID: "user-001", DisplayName: "Nguyen"}, "Cần mua MacBook Pro tại HCM"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionExclude,
			wantReasons:  []rules.ReasonCode{rules.ReasonAuthorNameHasNoWhitespace},
		},
		{
			name: "outside-today window is excluded",
			post: func() domain.RawPost {
				post := pipelinePost("post-5", "group-1", "https://facebook.example/post-005", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM")
				post.CreatedAt = time.Date(2026, 8, 4, 23, 59, 59, 0, loc)
				return post
			}(),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionExclude,
			wantReasons:  []rules.ReasonCode{rules.ReasonPostBeforeScanWindow},
		},
		{
			name:         "unknown geography becomes review",
			post:         pipelinePost("post-6", "group-1", "https://facebook.example/post-006", authorWithUserID("user-001"), "Cần mua MacBook Pro"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionReview,
			wantReasons:  []rules.ReasonCode{rules.ReasonLocationUnknown},
		},
		{
			name:         "conflicting geography becomes review",
			post:         pipelinePost("post-7", "group-1", "https://facebook.example/post-007", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM hoặc Hà Nội"),
			mode:         domain.GeographicModeAllVietnam,
			wantDecision: rules.DecisionReview,
			wantReasons:  []rules.ReasonCode{rules.ReasonLocationConflict},
		},
		{
			name:         "active geographic mode is respected",
			post:         pipelinePost("post-8", "group-1", "https://facebook.example/post-008", authorWithUserID("user-001"), "Cần mua MacBook Air tại Hà Nội"),
			mode:         domain.GeographicModeHoChiMinhCity,
			wantDecision: rules.DecisionExclude,
			wantReasons:  []rules.ReasonCode{rules.ReasonOutsideSelectedGeographicMode, rules.ReasonHoChiMinhCityRequired},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validPipelineInput(t, []domain.RawPost{tt.post})
			input.GeographicMode = tt.mode
			result, err := RunEvaluationPipeline(input)
			if err != nil {
				t.Fatalf("RunEvaluationPipeline returned error: %v", err)
			}
			if len(result.Evaluated()) != 1 {
				t.Fatalf("len(Evaluated) = %d, want 1", len(result.Evaluated()))
			}
			assertRuleReasons(t, result.Evaluated()[0].Result, tt.wantDecision, tt.wantReasons)
			if tt.wantDecision != rules.DecisionInclude && len(result.AggregatedLeads()) != 0 {
				t.Fatalf("non-include post proceeded to aggregation: %#v", result.AggregatedLeads())
			}
		})
	}
}

func TestRunEvaluationPipelineAggregationStage(t *testing.T) {
	tests := []struct {
		name             string
		posts            []domain.RawPost
		wantLeadCount    int
		wantUnaggregated int
		wantConflicts    int
	}{
		{
			name: "same author plus same need aggregates",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				pipelinePost("post-2", "group-2", "https://facebook.example/post-002", authorWithUserID("user-001"), "cần mua macbook pro tại hcm"),
			},
			wantLeadCount: 1,
		},
		{
			name: "different stable authors do not merge",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				pipelinePost("post-2", "group-2", "https://facebook.example/post-002", authorWithUserID("user-002"), "Cần mua MacBook Pro tại HCM"),
			},
			wantLeadCount: 2,
		},
		{
			name: "same author plus different need does not merge",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				pipelinePost("post-2", "group-2", "https://facebook.example/post-002", authorWithUserID("user-001"), "Mình đang tìm MacBook Air tại HCM"),
			},
			wantLeadCount: 2,
		},
		{
			name: "missing stable identity remains explicit through unaggregated",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", domain.AuthorIdentity{DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro tại HCM"),
			},
			wantUnaggregated: 1,
		},
		{
			name: "missing source identity remains unaggregated",
			posts: []domain.RawPost{
				pipelinePost("", "group-1", "", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
			},
			wantUnaggregated: 1,
		},
		{
			name: "duplicate source occurrence is not duplicated",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				pipelinePost(" post-1 ", "group-2", "https://facebook.example/post-duplicate", authorWithUserID("user-001"), "cần mua macbook pro tại hcm"),
			},
			wantLeadCount: 1,
		},
		{
			name: "source conflict remains explicit",
			posts: []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				pipelinePost("post-1", "group-2", "https://facebook.example/post-001", authorWithUserID("user-001"), "Mình đang tìm MacBook Air tại HCM"),
			},
			wantLeadCount: 1,
			wantConflicts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunEvaluationPipeline(validPipelineInput(t, tt.posts))
			if err != nil {
				t.Fatalf("RunEvaluationPipeline returned error: %v", err)
			}
			if len(result.AggregatedLeads()) != tt.wantLeadCount ||
				len(result.Unaggregated()) != tt.wantUnaggregated ||
				len(result.Conflicts()) != tt.wantConflicts {
				t.Fatalf("aggregation counts = leads %d unaggregated %d conflicts %d",
					len(result.AggregatedLeads()), len(result.Unaggregated()), len(result.Conflicts()))
			}
			if tt.wantUnaggregated > 0 && len(result.Allowed()) != 0 {
				t.Fatalf("unaggregated post became allowed lead: %#v", result.Allowed())
			}
		})
	}
}

func TestRunEvaluationPipelineBlocklistStage(t *testing.T) {
	tests := []struct {
		name           string
		author         domain.AuthorIdentity
		entryKind      blocklist.IdentityKind
		entryValue     string
		wantAllowed    int
		wantBlocked    int
		wantUnresolved int
	}{
		{name: "matching Facebook user ID blocks lead", author: authorWithUserID("user-001"), entryKind: blocklist.IdentityKindFacebookUserID, entryValue: "user-001", wantBlocked: 1},
		{name: "matching profile URL blocks lead", author: authorWithURL("https://facebook.example/buyer.one/"), entryKind: blocklist.IdentityKindCanonicalProfileURL, entryValue: "https://facebook.example/buyer.one", wantBlocked: 1},
		{name: "matching username blocks lead", author: authorWithUsername("buyer.one"), entryKind: blocklist.IdentityKindUsername, entryValue: "buyer.one", wantBlocked: 1},
		{name: "unmatched author remains allowed", author: authorWithUserID("user-002"), entryKind: blocklist.IdentityKindFacebookUserID, entryValue: "user-001", wantAllowed: 1},
		{
			name: "strongest identity no-fallback policy is preserved",
			author: domain.AuthorIdentity{
				FacebookUserID: "user-999",
				Username:       "buyer.one",
				DisplayName:    "Nguyen Van A",
			},
			entryKind:   blocklist.IdentityKindUsername,
			entryValue:  "buyer.one",
			wantAllowed: 1,
		},
		{name: "display name alone never blocks", author: domain.AuthorIdentity{FacebookUserID: "user-002", DisplayName: "Nguyen Van A"}, entryKind: blocklist.IdentityKindFacebookUserID, entryValue: "user-001", wantAllowed: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validPipelineInput(t, []domain.RawPost{
				pipelinePost("post-1", "group-1", "https://facebook.example/post-001", tt.author, "Cần mua MacBook Pro tại HCM"),
			})
			input.Blocklist = blocklist.NewList([]blocklist.Entry{
				mustBlocklistEntry(t, tt.entryKind, tt.entryValue, "Synthetic Block"),
			})

			result, err := RunEvaluationPipeline(input)
			if err != nil {
				t.Fatalf("RunEvaluationPipeline returned error: %v", err)
			}
			if len(result.Allowed()) != tt.wantAllowed ||
				len(result.Blocked()) != tt.wantBlocked ||
				len(result.Unresolved()) != tt.wantUnresolved {
				t.Fatalf("blocklist counts = allowed %d blocked %d unresolved %d",
					len(result.Allowed()), len(result.Blocked()), len(result.Unresolved()))
			}
		})
	}

	unresolvedInput := validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", domain.AuthorIdentity{DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro tại HCM"),
	})
	unresolved, err := RunEvaluationPipeline(unresolvedInput)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline unresolved setup returned error: %v", err)
	}
	if len(unresolved.Unaggregated()) != 1 || len(unresolved.Unresolved()) != 0 {
		t.Fatalf("insufficient stable identity should remain dedup-unaggregated before blocklist: unaggregated=%#v unresolved=%#v", unresolved.Unaggregated(), unresolved.Unresolved())
	}

	emptyBlocklist, err := RunEvaluationPipeline(validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-2", "group-1", "https://facebook.example/post-002", authorWithUsername("buyer.two"), "Cần mua MacBook Pro tại HCM"),
	}))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline empty blocklist returned error: %v", err)
	}
	if len(emptyBlocklist.Allowed()) != 1 {
		t.Fatalf("empty blocklist allowed count = %d, want 1", len(emptyBlocklist.Allowed()))
	}
}

func TestRunEvaluationPipelineSourcePreservation(t *testing.T) {
	createdAt := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC)
	post := pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM")
	post.CreatedAt = createdAt
	blockedPost := pipelinePost("post-2", "group-2", "https://facebook.example/post-002", authorWithUserID("user-002"), "Cần mua MacBook Pro tại HCM")

	input := validPipelineInput(t, []domain.RawPost{post, blockedPost})
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Tran Thi B"),
	})
	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}

	assertSourcePostPreserved(t, result.Allowed()[0].Lead.Sources()[0].Post, post)
	assertSourcePostPreserved(t, result.Blocked()[0].Lead.Sources()[0].Post, blockedPost)
}

func TestRunEvaluationPipelineOrdering(t *testing.T) {
	posts := []domain.RawPost{
		pipelinePost("allowed-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("review-1", "group-1", "https://facebook.example/post-002", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		pipelinePost("excluded-1", "group-1", "https://facebook.example/post-003", authorWithUserID("user-003"), "Bán MacBook Pro giá tốt tại HCM"),
		pipelinePost("blocked-1", "group-1", "https://facebook.example/post-004", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("allowed-2", "group-2", "https://facebook.example/post-005", authorWithUserID("user-005"), "Mình đang tìm MacBook Air tại HCM"),
	}
	input := validPipelineInput(t, posts)
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-004", "Blocked Buyer"),
	})
	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}

	assertEvaluatedPostIDs(t, result.Evaluated(), []string{"allowed-1", "review-1", "excluded-1", "blocked-1", "allowed-2"})
	assertReviewPostIDs(t, result.Review(), []string{"review-1"})
	assertExcludedPostIDs(t, result.Excluded(), []string{"excluded-1"})
	assertPipelineLeadPostIDs(t, result.Allowed(), []string{"allowed-1", "allowed-2"})
	assertPipelineBlockedLeadPostIDs(t, result.Blocked(), []string{"blocked-1"})

	again, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline repeat returned error: %v", err)
	}
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("ordering changed on repeat: %#v", again)
	}
}

func TestRunEvaluationPipelineConfiguration(t *testing.T) {
	valid := validPipelineInput(t, nil)
	result, err := RunEvaluationPipeline(valid)
	if err != nil {
		t.Fatalf("nil posts with valid configuration returned error: %v", err)
	}
	if len(result.Evaluated()) != 0 || len(result.Allowed()) != 0 {
		t.Fatalf("nil posts result = %#v, want empty success", result)
	}

	valid.Posts = []domain.RawPost{}
	result, err = RunEvaluationPipeline(valid)
	if err != nil {
		t.Fatalf("empty posts with valid configuration returned error: %v", err)
	}
	if len(result.Evaluated()) != 0 || len(result.Allowed()) != 0 {
		t.Fatalf("empty posts result = %#v, want empty success", result)
	}

	tests := []struct {
		name    string
		mutate  func(EvaluationPipelineInput) EvaluationPipelineInput
		wantErr error
	}{
		{
			name: "invalid ScanWindow fails explicitly",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				input.ScanWindow = domain.ScanWindow{}
				return input
			},
			wantErr: ErrInvalidPipelineScanWindow,
		},
		{
			name: "zero SearchProfile fails explicitly",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				input.SearchProfile = domain.SearchProfile{}
				return input
			},
			wantErr: ErrInvalidPipelineSearchProfile,
		},
		{
			name: "disabled SearchProfile fails explicitly",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				profile, err := domain.NewSearchProfile(
					"macbook-disabled",
					"MacBook Disabled",
					[]string{"MacBook"},
					[]string{"cần mua"},
					nil,
					false,
				)
				if err != nil {
					t.Fatalf("NewSearchProfile setup failed: %v", err)
				}
				input.SearchProfile = profile
				return input
			},
			wantErr: ErrInvalidPipelineSearchProfile,
		},
		{
			name: "unsupported GeographicMode fails explicitly",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				input.GeographicMode = domain.GeographicMode("outside_vietnam")
				return input
			},
			wantErr: ErrInvalidPipelineGeographicMode,
		},
		{
			name: "zero-value blocklist is safe",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				input.Posts = []domain.RawPost{
					pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				}
				input.Blocklist = blocklist.List{}
				return input
			},
		},
		{
			name: "empty blocklist is safe",
			mutate: func(input EvaluationPipelineInput) EvaluationPipelineInput {
				input.Posts = []domain.RawPost{
					pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
				}
				input.Blocklist = blocklist.NewList(nil)
				return input
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunEvaluationPipeline(tt.mutate(validPipelineInput(t, nil)))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RunEvaluationPipeline error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if len(result.Allowed()) != 0 || len(result.Blocked()) != 0 || len(result.Evaluated()) != 0 {
					t.Fatalf("invalid config produced misleading data: %#v", result)
				}
				return
			}
			if len(result.Allowed()) != 1 {
				t.Fatalf("valid zero/empty blocklist allowed count = %d, want 1", len(result.Allowed()))
			}
		})
	}
}

func TestRunEvaluationPipelineImmutabilityAndDefensiveCopies(t *testing.T) {
	posts := []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("post-2", "group-1", "https://facebook.example/post-002", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		pipelinePost("post-3", "group-1", "https://facebook.example/post-003", authorWithUserID("user-003"), "Bán MacBook Pro giá tốt tại HCM"),
		pipelinePost("post-4", "group-1", "https://facebook.example/post-004", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM"),
	}
	originalPosts := append([]domain.RawPost(nil), posts...)
	input := validPipelineInput(t, posts)
	input.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-004", "Tran Thi B"),
	})
	originalProfileTerms := input.SearchProfile.ProductTerms()
	originalBlocklistEntries := input.Blocklist.Entries()

	result, err := RunEvaluationPipeline(input)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline returned error: %v", err)
	}
	if !reflect.DeepEqual(posts, originalPosts) {
		t.Fatalf("pipeline mutated input posts: got %#v want %#v", posts, originalPosts)
	}
	if !reflect.DeepEqual(input.SearchProfile.ProductTerms(), originalProfileTerms) {
		t.Fatalf("pipeline mutated profile terms")
	}
	if !reflect.DeepEqual(input.Blocklist.Entries(), originalBlocklistEntries) {
		t.Fatalf("pipeline mutated blocklist")
	}

	evaluated := result.Evaluated()
	evaluated[0] = EvaluatedPost{}
	if result.Evaluated()[0].Post.PostID != "post-1" {
		t.Fatalf("Evaluated returned alias internal state: %#v", result.Evaluated())
	}
	review := result.Review()
	review[0] = ReviewPost{}
	if result.Review()[0].Post.PostID != "post-2" {
		t.Fatalf("Review returned alias internal state: %#v", result.Review())
	}
	excluded := result.Excluded()
	excluded[0] = ExcludedPost{}
	if result.Excluded()[0].Post.PostID != "post-3" {
		t.Fatalf("Excluded returned alias internal state: %#v", result.Excluded())
	}
	allowed := result.Allowed()
	allowed[0] = AllowedLead{}
	if result.Allowed()[0].Lead.Sources()[0].Post.PostID != "post-1" {
		t.Fatalf("Allowed returned alias internal state: %#v", result.Allowed())
	}
	blocked := result.Blocked()
	blocked[0] = BlockedLead{}
	if result.Blocked()[0].Lead.Sources()[0].Post.PostID != "post-4" {
		t.Fatalf("Blocked returned alias internal state: %#v", result.Blocked())
	}

	evaluated = result.Evaluated()
	evaluated[1].Result.Reasons[0] = rules.ReasonSellerIntent
	if result.Evaluated()[1].Result.Reasons[0] != rules.ReasonLocationUnknown {
		t.Fatalf("evaluated reason slice aliases internal state: %#v", result.Evaluated()[1].Result.Reasons)
	}
}

func TestEvaluationPipelineResultPreservesUnresolvedFilteringOutput(t *testing.T) {
	lead := dedup.Lead{
		Author: dedup.AuthorKey{
			Kind:  dedup.AuthorIdentityKind("legacy_external_id"),
			Value: "external-001",
		},
	}
	result := EvaluationPipelineResult{
		leadFiltering: FilterLeads([]dedup.Lead{lead}, blocklist.NewList(nil)),
	}

	unresolved := result.Unresolved()
	if len(unresolved) != 1 {
		t.Fatalf("len(Unresolved) = %d, want 1", len(unresolved))
	}
	if !reflect.DeepEqual(unresolved[0].Reasons, []ReasonCode{ReasonBlocklistEvaluationUnsupported}) {
		t.Fatalf("Unresolved reasons = %#v, want unsupported reason", unresolved[0].Reasons)
	}
	unresolved[0] = UnresolvedLead{}
	if len(result.Unresolved()) != 1 {
		t.Fatalf("Unresolved returned alias internal state: %#v", result.Unresolved())
	}
}

func TestRunEvaluationPipelineFailClosedBehavior(t *testing.T) {
	unknown, err := RunEvaluationPipeline(validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-1", "group-1", "https://facebook.example/post-001", authorWithUserID("user-001"), "Cần mua MacBook Pro"),
	}))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline unknown geography returned error: %v", err)
	}
	if len(unknown.Review()) != 1 || len(unknown.Allowed()) != 0 {
		t.Fatalf("unknown geography became included: review=%#v allowed=%#v", unknown.Review(), unknown.Allowed())
	}

	insufficient, err := RunEvaluationPipeline(validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-2", "group-1", "https://facebook.example/post-002", domain.AuthorIdentity{DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro tại HCM"),
	}))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline insufficient identity returned error: %v", err)
	}
	if len(insufficient.Unaggregated()) != 1 || len(insufficient.Allowed()) != 0 {
		t.Fatalf("insufficient stable identity became allowed: unaggregated=%#v allowed=%#v", insufficient.Unaggregated(), insufficient.Allowed())
	}

	conflict, err := RunEvaluationPipeline(validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-3", "group-1", "https://facebook.example/post-003", authorWithUserID("user-001"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("post-3", "group-2", "https://facebook.example/post-003", authorWithUserID("user-001"), "Mình đang tìm MacBook Air tại HCM"),
	}))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline conflict returned error: %v", err)
	}
	if len(conflict.Conflicts()) != 1 {
		t.Fatalf("source conflict was silently resolved: %#v", conflict.Conflicts())
	}

	blockedInput := validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-4", "group-1", "https://facebook.example/post-004", authorWithUserID("user-004"), "Cần mua MacBook Pro tại HCM"),
	})
	blockedInput.Blocklist = blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-004", "Blocked"),
	})
	blocked, err := RunEvaluationPipeline(blockedInput)
	if err != nil {
		t.Fatalf("RunEvaluationPipeline blocked returned error: %v", err)
	}
	if len(blocked.Blocked()) != 1 || len(blocked.Allowed()) != 0 {
		t.Fatalf("blocked lead appeared in allowed: blocked=%#v allowed=%#v", blocked.Blocked(), blocked.Allowed())
	}

	differentAuthors, err := RunEvaluationPipeline(validPipelineInput(t, []domain.RawPost{
		pipelinePost("post-5", "group-1", "https://facebook.example/post-005", authorWithUserID("user-005"), "Cần mua MacBook Pro tại HCM"),
		pipelinePost("post-6", "group-2", "https://facebook.example/post-006", authorWithUserID("user-006"), "Cần mua MacBook Pro tại HCM"),
	}))
	if err != nil {
		t.Fatalf("RunEvaluationPipeline different authors returned error: %v", err)
	}
	if len(differentAuthors.Allowed()) != 2 {
		t.Fatalf("body similarity across different authors merged: allowed=%#v", differentAuthors.Allowed())
	}
}

func validPipelineInput(t *testing.T, posts []domain.RawPost) EvaluationPipelineInput {
	t.Helper()

	return EvaluationPipelineInput{
		Posts:          posts,
		ScanWindow:     validPipelineWindow(t),
		SearchProfile:  domain.MacBookSearchProfile(),
		GeographicMode: domain.GeographicModeHoChiMinhCity,
		Blocklist:      blocklist.NewList(nil),
	}
}

func validPipelineWindow(t *testing.T) domain.ScanWindow {
	t.Helper()

	loc := pipelineLocation()
	window, err := domain.NewScanWindow(
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 23, 0, 0, 0, loc),
	)
	if err != nil {
		t.Fatalf("NewScanWindow setup failed: %v", err)
	}
	return window
}

func pipelineLocation() *time.Location {
	return time.FixedZone(domain.RequiredTimezone, 7*60*60)
}

func pipelinePost(postID string, groupID string, postURL string, author domain.AuthorIdentity, body string) domain.RawPost {
	post := sourcePost(postID, groupID, postURL, author, body)
	post.CreatedAt = time.Date(2026, 8, 5, 9, 0, 0, 0, pipelineLocation())
	post.CapturedAt = time.Date(2026, 8, 5, 9, 1, 0, 0, pipelineLocation())
	return post
}

func assertEvaluatedPostIDs(t *testing.T, posts []EvaluatedPost, want []string) {
	t.Helper()

	if len(posts) != len(want) {
		t.Fatalf("len(EvaluatedPost) = %d, want %d", len(posts), len(want))
	}
	for i, post := range posts {
		if post.Post.PostID != want[i] {
			t.Fatalf("post[%d].PostID = %q, want %q", i, post.Post.PostID, want[i])
		}
	}
}

func assertReviewPostIDs(t *testing.T, posts []ReviewPost, want []string) {
	t.Helper()

	if len(posts) != len(want) {
		t.Fatalf("len(ReviewPost) = %d, want %d", len(posts), len(want))
	}
	for i, post := range posts {
		if post.Post.PostID != want[i] {
			t.Fatalf("review[%d].PostID = %q, want %q", i, post.Post.PostID, want[i])
		}
	}
}

func assertExcludedPostIDs(t *testing.T, posts []ExcludedPost, want []string) {
	t.Helper()

	if len(posts) != len(want) {
		t.Fatalf("len(ExcludedPost) = %d, want %d", len(posts), len(want))
	}
	for i, post := range posts {
		if post.Post.PostID != want[i] {
			t.Fatalf("excluded[%d].PostID = %q, want %q", i, post.Post.PostID, want[i])
		}
	}
}

func assertPipelineLeadPostIDs(t *testing.T, leads []AllowedLead, want []string) {
	t.Helper()

	if len(leads) != len(want) {
		t.Fatalf("len(AllowedLead) = %d, want %d", len(leads), len(want))
	}
	for i, lead := range leads {
		sources := lead.Lead.Sources()
		if len(sources) == 0 {
			t.Fatalf("allowed lead %d has no sources", i)
		}
		if sources[0].Post.PostID != want[i] {
			t.Fatalf("allowed lead %d first PostID = %q, want %q", i, sources[0].Post.PostID, want[i])
		}
	}
}

func assertPipelineBlockedLeadPostIDs(t *testing.T, leads []BlockedLead, want []string) {
	t.Helper()

	if len(leads) != len(want) {
		t.Fatalf("len(BlockedLead) = %d, want %d", len(leads), len(want))
	}
	for i, lead := range leads {
		sources := lead.Lead.Sources()
		if len(sources) == 0 {
			t.Fatalf("blocked lead %d has no sources", i)
		}
		if sources[0].Post.PostID != want[i] {
			t.Fatalf("blocked lead %d first PostID = %q, want %q", i, sources[0].Post.PostID, want[i])
		}
	}
}

func assertPipelineLeadSourcePostIDs(t *testing.T, lead dedup.Lead, want []string) {
	t.Helper()

	sources := lead.Sources()
	if len(sources) != len(want) {
		t.Fatalf("len(Sources) = %d, want %d", len(sources), len(want))
	}
	for i, source := range sources {
		if source.Post.PostID != want[i] {
			t.Fatalf("source[%d].PostID = %q, want %q", i, source.Post.PostID, want[i])
		}
	}
}

func assertRuleReasons(t *testing.T, got rules.Result, wantDecision rules.Decision, wantReasons []rules.ReasonCode) {
	t.Helper()

	if got.Decision != wantDecision {
		t.Fatalf("Decision = %q, want %q", got.Decision, wantDecision)
	}
	if !reflect.DeepEqual(got.Reasons, wantReasons) {
		t.Fatalf("Reasons = %#v, want %#v", got.Reasons, wantReasons)
	}
}
