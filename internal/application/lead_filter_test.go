package application

import (
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/dedup"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestFilterLeadsBasicFiltering(t *testing.T) {
	allowedLead := testLead(t, "post-1", authorWithUserID("user-001"), "Cần mua MacBook Pro")
	blockedLead := testLead(t, "post-2", authorWithUserID("user-002"), "Mình đang tìm MacBook Air")
	list := blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Tran Thi B"),
	})

	result := FilterLeads([]dedup.Lead{allowedLead, blockedLead}, list)

	assertLeadPostIDs(t, allowedLeads(result), []string{"post-1"})
	assertLeadPostIDs(t, blockedLeads(result), []string{"post-2"})
	if len(result.Unresolved()) != 0 {
		t.Fatalf("len(Unresolved) = %d, want 0", len(result.Unresolved()))
	}
	if total := len(result.Allowed()) + len(result.Blocked()) + len(result.Unresolved()); total != 2 {
		t.Fatalf("total result count = %d, want 2", total)
	}

	again := FilterLeads([]dedup.Lead{allowedLead, blockedLead}, list)
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("FilterLeads repeated result = %#v, want %#v", again, result)
	}
}

func TestFilterLeadsIdentityKinds(t *testing.T) {
	tests := []struct {
		name        string
		lead        dedup.Lead
		entryKind   blocklist.IdentityKind
		entryValue  string
		wantOutcome blocklist.MatchOutcome
	}{
		{
			name:        "Facebook user ID block filters lead",
			lead:        testLead(t, "post-1", authorWithUserID("user-001"), "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindFacebookUserID,
			entryValue:  "user-001",
			wantOutcome: blocklist.MatchOutcomeBlocked,
		},
		{
			name:        "profile URL block filters lead",
			lead:        testLead(t, "post-2", authorWithURL("https://facebook.example/buyer.one/"), "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindCanonicalProfileURL,
			entryValue:  "https://facebook.example/buyer.one",
			wantOutcome: blocklist.MatchOutcomeBlocked,
		},
		{
			name:        "username block filters lead",
			lead:        testLead(t, "post-3", authorWithUsername("buyer.one"), "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindUsername,
			entryValue:  "buyer.one",
			wantOutcome: blocklist.MatchOutcomeBlocked,
		},
		{
			name:        "username case normalization follows blocklist",
			lead:        testLead(t, "post-4", authorWithUsername("BUYER.ONE"), "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindUsername,
			entryValue:  "buyer.one",
			wantOutcome: blocklist.MatchOutcomeBlocked,
		},
		{
			name:        "same string under different identity kinds does not block",
			lead:        testLead(t, "post-5", authorWithUsername("same-text"), "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindFacebookUserID,
			entryValue:  "same-text",
			wantOutcome: blocklist.MatchOutcomeNotBlocked,
		},
		{
			name: "unmatched strongest Facebook user ID does not fall back to blocked username",
			lead: testLead(t, "post-6", domain.AuthorIdentity{
				FacebookUserID: "user-999",
				Username:       "buyer.one",
				DisplayName:    "Nguyen Van A",
			}, "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindUsername,
			entryValue:  "buyer.one",
			wantOutcome: blocklist.MatchOutcomeNotBlocked,
		},
		{
			name: "unmatched profile URL does not fall back to blocked username",
			lead: testLead(t, "post-7", domain.AuthorIdentity{
				CanonicalProfileURL: "https://facebook.example/not-blocked",
				Username:            "buyer.one",
				DisplayName:         "Nguyen Van A",
			}, "Cần mua MacBook Pro"),
			entryKind:   blocklist.IdentityKindUsername,
			entryValue:  "buyer.one",
			wantOutcome: blocklist.MatchOutcomeNotBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := blocklist.NewList([]blocklist.Entry{
				mustBlocklistEntry(t, tt.entryKind, tt.entryValue, "Synthetic Block"),
			})
			result := FilterLeads([]dedup.Lead{tt.lead}, list)

			switch tt.wantOutcome {
			case blocklist.MatchOutcomeBlocked:
				if len(result.Blocked()) != 1 || result.Blocked()[0].Match.Outcome != tt.wantOutcome {
					t.Fatalf("blocked result = %#v, want one blocked", result.Blocked())
				}
				if len(result.Allowed()) != 0 || len(result.Unresolved()) != 0 {
					t.Fatalf("unexpected other results: allowed=%#v unresolved=%#v", result.Allowed(), result.Unresolved())
				}
			case blocklist.MatchOutcomeNotBlocked:
				if len(result.Allowed()) != 1 || result.Allowed()[0].Match.Outcome != tt.wantOutcome {
					t.Fatalf("allowed result = %#v, want one allowed", result.Allowed())
				}
				if len(result.Blocked()) != 0 || len(result.Unresolved()) != 0 {
					t.Fatalf("unexpected other results: blocked=%#v unresolved=%#v", result.Blocked(), result.Unresolved())
				}
			}
		})
	}
}

func TestFilterLeadsDisplayNameSimilarityNeverBlocks(t *testing.T) {
	lead := testLead(t, "post-1", domain.AuthorIdentity{FacebookUserID: "user-001", DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro")
	list := blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Nguyen Van A"),
	})

	result := FilterLeads([]dedup.Lead{lead}, list)
	if len(result.Allowed()) != 1 {
		t.Fatalf("display-name similarity blocked lead: %#v", result)
	}
}

func TestFilterLeadsUnresolvedCases(t *testing.T) {
	displayNameOnly := testLeadFromPosts(t, []domain.RawPost{
		sourcePost("post-1", "", "", domain.AuthorIdentity{DisplayName: "Nguyen Van A"}, "Cần mua MacBook Pro"),
	})
	unsupported := dedup.Lead{
		Author: dedup.AuthorKey{Kind: dedup.AuthorIdentityKind("display_name"), Value: "Nguyen Van A"},
	}

	tests := []struct {
		name       string
		lead       dedup.Lead
		wantReason []ReasonCode
	}{
		{name: "insufficient stable identity becomes unresolved", lead: displayNameOnly},
		{name: "display-name-only lead becomes unresolved", lead: displayNameOnly},
		{name: "zero-value lead becomes unresolved", lead: dedup.Lead{}},
		{name: "empty sources do not create inferred identity", lead: dedup.Lead{Author: dedup.AuthorKey{}}},
		{name: "malformed unsupported identity does not falsely block", lead: unsupported, wantReason: []ReasonCode{ReasonBlocklistEvaluationUnsupported}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLeads([]dedup.Lead{tt.lead}, blocklist.NewList(nil))
			if len(result.Unresolved()) != 1 {
				t.Fatalf("len(Unresolved) = %d, want 1: %#v", len(result.Unresolved()), result)
			}
			if len(result.Allowed()) != 0 || len(result.Blocked()) != 0 {
				t.Fatalf("unresolved lead also appeared elsewhere: allowed=%#v blocked=%#v", result.Allowed(), result.Blocked())
			}
			unresolved := result.Unresolved()[0]
			if unresolved.Match.Outcome != blocklist.MatchOutcomeInsufficientIdentity {
				t.Fatalf("Outcome = %q, want insufficient_identity", unresolved.Match.Outcome)
			}
			assertBlocklistReasons(t, unresolved.Match.Reasons, []blocklist.ReasonCode{blocklist.ReasonStableAuthorIdentityMissing})
			if !reflect.DeepEqual(unresolved.Reasons, tt.wantReason) {
				t.Fatalf("Reasons = %#v, want %#v", unresolved.Reasons, tt.wantReason)
			}
		})
	}
}

func TestFilterLeadsEmptyInputAndEmptyList(t *testing.T) {
	if got := FilterLeads(nil, blocklist.NewList(nil)); len(got.Allowed()) != 0 || len(got.Blocked()) != 0 || len(got.Unresolved()) != 0 {
		t.Fatalf("nil leads result = %#v, want empty", got)
	}
	if got := FilterLeads([]dedup.Lead{}, blocklist.NewList(nil)); len(got.Allowed()) != 0 || len(got.Blocked()) != 0 || len(got.Unresolved()) != 0 {
		t.Fatalf("empty leads result = %#v, want empty", got)
	}

	sufficient := testLead(t, "post-1", authorWithUserID("user-001"), "Cần mua MacBook Pro")
	insufficient := dedup.Lead{}
	result := FilterLeads([]dedup.Lead{sufficient, insufficient}, blocklist.List{})
	if len(result.Allowed()) != 1 || len(result.Blocked()) != 0 || len(result.Unresolved()) != 1 {
		t.Fatalf("zero-value list result = allowed %d blocked %d unresolved %d", len(result.Allowed()), len(result.Blocked()), len(result.Unresolved()))
	}
	if result.Allowed()[0].Match.Outcome != blocklist.MatchOutcomeNotBlocked {
		t.Fatalf("empty blocklist sufficient match = %#v", result.Allowed()[0].Match)
	}
}

func TestFilterLeadsOrdering(t *testing.T) {
	leads := []dedup.Lead{
		testLead(t, "allowed-1", authorWithUserID("user-001"), "Cần mua MacBook Pro"),
		testLead(t, "blocked-1", authorWithUserID("user-002"), "Cần mua MacBook Pro"),
		dedup.Lead{},
		testLead(t, "allowed-2", authorWithUserID("user-003"), "Mình đang tìm MacBook Air"),
		testLead(t, "blocked-2", authorWithUsername("buyer.two"), "Cần mua MacBook Pro"),
	}
	list := blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Tran Thi B"),
		mustBlocklistEntry(t, blocklist.IdentityKindUsername, "buyer.two", "Le Van C"),
	})

	result := FilterLeads(leads, list)
	assertLeadPostIDs(t, allowedLeads(result), []string{"allowed-1", "allowed-2"})
	assertLeadPostIDs(t, blockedLeads(result), []string{"blocked-1", "blocked-2"})
	if len(result.Unresolved()) != 1 {
		t.Fatalf("len(Unresolved) = %d, want 1", len(result.Unresolved()))
	}

	again := FilterLeads(leads, list)
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("ordering changed on repeat: %#v", again)
	}
}

func TestFilterLeadsSourcePreservation(t *testing.T) {
	createdAt := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC)
	capturedAt := time.Date(2026, 8, 5, 9, 31, 0, 0, time.UTC)
	allowedPost := sourcePost("post-1", "group-1", "https://facebook.example/posts/1", authorWithUserID("user-001"), "Cần mua MacBook Pro")
	allowedPost.CreatedAt = createdAt
	allowedPost.CapturedAt = capturedAt
	blockedPost := sourcePost("post-2", "group-2", "https://facebook.example/posts/2", authorWithUserID("user-002"), "Mình đang tìm MacBook Air")

	allowedLead := testLeadFromPosts(t, []domain.RawPost{allowedPost})
	blockedLead := testLeadFromPosts(t, []domain.RawPost{blockedPost})
	result := FilterLeads([]dedup.Lead{allowedLead, blockedLead}, blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Tran Thi B"),
	}))

	assertSourcePostPreserved(t, result.Allowed()[0].Lead.Sources()[0].Post, allowedPost)
	assertSourcePostPreserved(t, result.Blocked()[0].Lead.Sources()[0].Post, blockedPost)
}

func TestFilterLeadsImmutabilityAndDefensiveCopies(t *testing.T) {
	leadOne := testLead(t, "post-1", authorWithUserID("user-001"), "Cần mua MacBook Pro")
	leadTwo := testLead(t, "post-2", authorWithUserID("user-002"), "Cần mua MacBook Pro")
	leadThree := dedup.Lead{}
	leads := []dedup.Lead{leadOne, leadTwo, leadThree}
	originalLeads := append([]dedup.Lead(nil), leads...)
	entry := mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-002", "Tran Thi B")
	entries := []blocklist.Entry{entry}
	list := blocklist.NewList(entries)
	originalEntries := list.Entries()

	result := FilterLeads(leads, list)

	if !reflect.DeepEqual(leads, originalLeads) {
		t.Fatalf("FilterLeads mutated or reordered input leads: got %#v want %#v", leads, originalLeads)
	}
	if !reflect.DeepEqual(list.Entries(), originalEntries) {
		t.Fatalf("FilterLeads mutated blocklist entries: got %#v want %#v", list.Entries(), originalEntries)
	}

	allowed := result.Allowed()
	allowed[0] = AllowedLead{}
	if len(result.Allowed()) != 1 || result.Allowed()[0].Lead.Key.Value != leadOne.Key.Value {
		t.Fatalf("Allowed returned alias internal state: %#v", result.Allowed())
	}

	blocked := result.Blocked()
	blocked[0] = BlockedLead{}
	if len(result.Blocked()) != 1 || result.Blocked()[0].Lead.Key.Value != leadTwo.Key.Value {
		t.Fatalf("Blocked returned alias internal state: %#v", result.Blocked())
	}

	unresolved := result.Unresolved()
	unresolved[0] = UnresolvedLead{}
	if len(result.Unresolved()) != 1 || result.Unresolved()[0].Match.Outcome != blocklist.MatchOutcomeInsufficientIdentity {
		t.Fatalf("Unresolved returned alias internal state: %#v", result.Unresolved())
	}

	allowed = result.Allowed()
	allowed[0].Match.Reasons[0] = blocklist.ReasonIdentityMatched
	if result.Allowed()[0].Match.Reasons[0] != blocklist.ReasonIdentityNotFound {
		t.Fatalf("Allowed match reasons alias internal state: %#v", result.Allowed()[0].Match.Reasons)
	}
}

func TestFilterLeadsFailClosedPolicy(t *testing.T) {
	list := blocklist.NewList([]blocklist.Entry{
		mustBlocklistEntry(t, blocklist.IdentityKindFacebookUserID, "user-001", "Nguyen Van A"),
		mustBlocklistEntry(t, blocklist.IdentityKindUsername, "buyer.one", "Buyer One"),
	})

	tests := []struct {
		name string
		lead dedup.Lead
	}{
		{name: "post-body similarity never blocks", lead: testLead(t, "post-1", authorWithUserID("user-999"), "user-001 buyer.one")},
		{name: "group ID never blocks", lead: testLeadFromPosts(t, []domain.RawPost{sourcePost("post-2", "user-001", "", authorWithUserID("user-999"), "Cần mua MacBook Pro")})},
		{name: "post URL never establishes blocked-author identity", lead: testLeadFromPosts(t, []domain.RawPost{sourcePost("post-3", "group-1", "https://facebook.example/user-001", authorWithUserID("user-999"), "Cần mua MacBook Pro")})},
		{name: "no substring matching", lead: testLead(t, "post-4", authorWithUsername("buyer.one.extra"), "Cần mua MacBook Pro")},
		{name: "no fuzzy username matching", lead: testLead(t, "post-5", authorWithUsername("buyer.on"), "Cần mua MacBook Pro")},
		{name: "different stable author remains independently evaluated", lead: testLead(t, "post-6", authorWithUserID("user-002"), "Cần mua MacBook Pro")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLeads([]dedup.Lead{tt.lead}, list)
			if len(result.Allowed()) != 1 || len(result.Blocked()) != 0 || len(result.Unresolved()) != 0 {
				t.Fatalf("fail-closed case was not allowed only: allowed=%#v blocked=%#v unresolved=%#v", result.Allowed(), result.Blocked(), result.Unresolved())
			}
		})
	}
}

func testLead(t *testing.T, postID string, author domain.AuthorIdentity, body string) dedup.Lead {
	t.Helper()
	return testLeadFromPosts(t, []domain.RawPost{
		sourcePost(postID, "group-1", "https://facebook.example/posts/"+postID, author, body),
	})
}

func testLeadFromPosts(t *testing.T, posts []domain.RawPost) dedup.Lead {
	t.Helper()

	result := dedup.AggregatePosts(posts, domain.MacBookSearchProfile())
	leads := result.Leads()
	if len(leads) == 0 {
		if len(result.Unaggregated()) == 0 {
			return dedup.Lead{}
		}
		return dedup.Lead{
			Author: result.Unaggregated()[0].Candidate.Author,
			Need:   result.Unaggregated()[0].Candidate.Need,
		}
	}
	if len(leads) != 1 {
		t.Fatalf("testLeadFromPosts got %d leads, want 1", len(leads))
	}
	return leads[0]
}

func sourcePost(postID string, groupID string, postURL string, author domain.AuthorIdentity, body string) domain.RawPost {
	return domain.RawPost{
		PostID:     postID,
		GroupID:    groupID,
		GroupName:  "Synthetic Group",
		PostURL:    postURL,
		Author:     author,
		Body:       body,
		CreatedAt:  time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC),
		CapturedAt: time.Date(2026, 8, 5, 9, 1, 0, 0, time.UTC),
	}
}

func authorWithUserID(userID string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		FacebookUserID: userID,
		DisplayName:    "Nguyen Van A",
	}
}

func authorWithURL(url string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		CanonicalProfileURL: url,
		DisplayName:         "Nguyen Van A",
	}
}

func authorWithUsername(username string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		Username:    username,
		DisplayName: "Nguyen Van A",
	}
}

func mustBlocklistEntry(t *testing.T, kind blocklist.IdentityKind, value string, displayName string) blocklist.Entry {
	t.Helper()

	entry, err := blocklist.NewEntry(kind, value, displayName)
	if err != nil {
		t.Fatalf("NewEntry(%q, %q) setup failed: %v", kind, value, err)
	}
	return entry
}

func allowedLeads(result LeadFilterResult) []dedup.Lead {
	allowed := result.Allowed()
	leads := make([]dedup.Lead, len(allowed))
	for i, lead := range allowed {
		leads[i] = lead.Lead
	}
	return leads
}

func blockedLeads(result LeadFilterResult) []dedup.Lead {
	blocked := result.Blocked()
	leads := make([]dedup.Lead, len(blocked))
	for i, lead := range blocked {
		leads[i] = lead.Lead
	}
	return leads
}

func assertLeadPostIDs(t *testing.T, leads []dedup.Lead, wantPostIDs []string) {
	t.Helper()

	if len(leads) != len(wantPostIDs) {
		t.Fatalf("lead count = %d, want %d", len(leads), len(wantPostIDs))
	}
	for i, lead := range leads {
		sources := lead.Sources()
		if len(sources) == 0 {
			t.Fatalf("lead %d has no sources", i)
		}
		if sources[0].Post.PostID != wantPostIDs[i] {
			t.Fatalf("lead %d first source PostID = %q, want %q", i, sources[0].Post.PostID, wantPostIDs[i])
		}
	}
}

func assertBlocklistReasons(t *testing.T, got []blocklist.ReasonCode, want []blocklist.ReasonCode) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("blocklist reasons = %#v, want %#v", got, want)
	}
}

func assertSourcePostPreserved(t *testing.T, got domain.RawPost, want domain.RawPost) {
	t.Helper()

	if got.PostID != want.PostID ||
		got.GroupID != want.GroupID ||
		got.PostURL != want.PostURL ||
		got.Body != want.Body ||
		got.Author != want.Author ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.CapturedAt.Equal(want.CapturedAt) {
		t.Fatalf("source post not preserved:\ngot  %#v\nwant %#v", got, want)
	}
}
