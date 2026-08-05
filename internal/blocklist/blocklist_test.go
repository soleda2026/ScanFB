package blocklist

import (
	"errors"
	"reflect"
	"testing"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestIdentityKeyCreation(t *testing.T) {
	tests := []struct {
		name      string
		kind      IdentityKind
		value     string
		want      IdentityKey
		wantErr   error
		wantCause ReasonCode
	}{
		{
			name:  "Facebook user ID trims surrounding whitespace",
			kind:  IdentityKindFacebookUserID,
			value: " user-001 ",
			want:  IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-001"},
		},
		{
			name:  "profile URL trims surrounding whitespace",
			kind:  IdentityKindCanonicalProfileURL,
			value: " https://facebook.example/buyer.one ",
			want:  IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/buyer.one"},
		},
		{
			name:  "profile URL trailing slash normalization is deterministic",
			kind:  IdentityKindCanonicalProfileURL,
			value: " https://facebook.example/buyer.one/// ",
			want:  IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/buyer.one"},
		},
		{
			name:  "username trims and lowercases",
			kind:  IdentityKindUsername,
			value: " Buyer.One ",
			want:  IdentityKey{Kind: IdentityKindUsername, Value: "buyer.one"},
		},
		{
			name:      "empty identity is rejected",
			kind:      IdentityKindFacebookUserID,
			value:     "",
			wantErr:   ErrInvalidEntry,
			wantCause: ReasonInvalidEntry,
		},
		{
			name:      "whitespace-only identity is rejected",
			kind:      IdentityKindUsername,
			value:     " \t\n ",
			wantErr:   ErrInvalidEntry,
			wantCause: ReasonInvalidEntry,
		},
		{
			name:      "unsupported identity kind is rejected",
			kind:      IdentityKind("display_name"),
			value:     "Nguyen Van A",
			wantErr:   ErrUnsupportedIdentityKind,
			wantCause: ReasonUnsupportedIdentityKind,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.value
			got, err := NewIdentityKey(tt.kind, tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewIdentityKey error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantCause != "" {
				var entryErr EntryError
				if !errors.As(err, &entryErr) || entryErr.Reason != tt.wantCause {
					t.Fatalf("EntryError = %#v, want reason %q", err, tt.wantCause)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewIdentityKey returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NewIdentityKey = %#v, want %#v", got, tt.want)
			}
			if tt.value != original {
				t.Fatalf("NewIdentityKey mutated original string: got %q, want %q", tt.value, original)
			}
		})
	}
}

func TestDifferentIdentityKindsWithIdenticalTextProduceDifferentKeys(t *testing.T) {
	facebookID, err := NewIdentityKey(IdentityKindFacebookUserID, "buyer.one")
	if err != nil {
		t.Fatalf("Facebook user ID key setup failed: %v", err)
	}
	username, err := NewIdentityKey(IdentityKindUsername, "buyer.one")
	if err != nil {
		t.Fatalf("username key setup failed: %v", err)
	}

	if facebookID == username {
		t.Fatalf("different identity kinds collapsed into same key: %#v", facebookID)
	}
}

func TestAuthorKeySelection(t *testing.T) {
	tests := []struct {
		name   string
		author domain.AuthorIdentity
		want   IdentityKey
		wantOK bool
	}{
		{
			name: "Facebook user ID is preferred when present",
			author: domain.AuthorIdentity{
				FacebookUserID:      " user-001 ",
				CanonicalProfileURL: "https://facebook.example/buyer.one",
				Username:            "Buyer.One",
				DisplayName:         "Nguyen Van A",
			},
			want:   IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-001"},
			wantOK: true,
		},
		{
			name: "profile URL is used when Facebook user ID is absent",
			author: domain.AuthorIdentity{
				CanonicalProfileURL: " https://facebook.example/buyer.one/ ",
				Username:            "Buyer.One",
				DisplayName:         "Nguyen Van A",
			},
			want:   IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/buyer.one"},
			wantOK: true,
		},
		{
			name: "username is used when stronger identifiers are absent",
			author: domain.AuthorIdentity{
				Username:    " Buyer.One ",
				DisplayName: "Nguyen Van A",
			},
			want:   IdentityKey{Kind: IdentityKindUsername, Value: "buyer.one"},
			wantOK: true,
		},
		{
			name: "display name alone is insufficient",
			author: domain.AuthorIdentity{
				DisplayName: "Nguyen Van A",
			},
		},
		{
			name:   "empty author identity is insufficient",
			author: domain.AuthorIdentity{},
		},
		{
			name: "one-word display name is insufficient",
			author: domain.AuthorIdentity{
				DisplayName: "Nguyen",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.author
			got, ok := NewAuthorIdentityKey(tt.author)
			if ok != tt.wantOK {
				t.Fatalf("NewAuthorIdentityKey ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("NewAuthorIdentityKey = %#v, want %#v", got, tt.want)
			}
			if tt.author != original {
				t.Fatalf("NewAuthorIdentityKey mutated author: got %#v, want %#v", tt.author, original)
			}
		})
	}
}

func TestListConstruction(t *testing.T) {
	if list := NewList(nil); list.Len() != 0 || len(list.Entries()) != 0 {
		t.Fatalf("nil input list = len %d entries %#v, want empty", list.Len(), list.Entries())
	}
	if list := NewList([]Entry{}); list.Len() != 0 || len(list.Entries()) != 0 {
		t.Fatalf("empty input list = len %d entries %#v, want empty", list.Len(), list.Entries())
	}

	first := mustEntry(t, IdentityKindUsername, " Buyer.One ", "Nguyen Van A")
	second := mustEntry(t, IdentityKindFacebookUserID, "user-001", "Tran Thi B")
	duplicate := mustEntry(t, IdentityKindUsername, "buyer.one", "Changed Metadata")
	entries := []Entry{first, second, duplicate}
	original := append([]Entry(nil), entries...)

	list := NewList(entries)
	if list.Len() != 2 {
		t.Fatalf("Len = %d, want 2", list.Len())
	}
	gotEntries := list.Entries()
	if !reflect.DeepEqual(gotEntries, []Entry{first, second}) {
		t.Fatalf("Entries = %#v, want first occurrence order", gotEntries)
	}
	if !reflect.DeepEqual(entries, original) {
		t.Fatalf("NewList reordered or mutated input slice: got %#v, want %#v", entries, original)
	}

	gotEntries[0] = second
	again := list.Entries()
	if !reflect.DeepEqual(again, []Entry{first, second}) {
		t.Fatalf("Entries returned alias internal state: %#v", again)
	}

	reconstructed := NewList(entries)
	if !reflect.DeepEqual(reconstructed, list) {
		t.Fatalf("repeated list construction = %#v, want %#v", reconstructed, list)
	}
}

func TestListDuplicateNormalizationIsDeterministic(t *testing.T) {
	first := mustEntry(t, IdentityKindUsername, "Buyer.One", "first")
	second := mustEntry(t, IdentityKindUsername, " buyer.one ", "second")

	list := NewList([]Entry{first, second})
	if list.Len() != 1 {
		t.Fatalf("Len = %d, want 1", list.Len())
	}
	if got := list.Entries()[0].DisplayName(); got != "first" {
		t.Fatalf("duplicate metadata changed first entry: %q", got)
	}
}

func TestMatching(t *testing.T) {
	list := NewList([]Entry{
		mustEntry(t, IdentityKindFacebookUserID, "user-001", "Nguyen Van A"),
		mustEntry(t, IdentityKindCanonicalProfileURL, "https://facebook.example/buyer.two", "Tran Thi B"),
		mustEntry(t, IdentityKindUsername, "buyer.three", "Le Van C"),
		mustEntry(t, IdentityKindFacebookUserID, "same-text", "Same Text ID"),
	})

	tests := []struct {
		name   string
		author domain.AuthorIdentity
		want   MatchResult
	}{
		{
			name: "matching Facebook user ID blocks",
			author: domain.AuthorIdentity{
				FacebookUserID: " user-001 ",
				DisplayName:    "Nguyen Van A",
			},
			want: matchResult(
				MatchOutcomeBlocked,
				IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-001"},
				mustEntry(t, IdentityKindFacebookUserID, "user-001", "Nguyen Van A"),
				ReasonIdentityMatched,
			),
		},
		{
			name: "matching profile URL blocks",
			author: domain.AuthorIdentity{
				CanonicalProfileURL: "https://facebook.example/buyer.two/",
				DisplayName:         "Tran Thi B",
			},
			want: matchResult(
				MatchOutcomeBlocked,
				IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/buyer.two"},
				mustEntry(t, IdentityKindCanonicalProfileURL, "https://facebook.example/buyer.two", "Tran Thi B"),
				ReasonIdentityMatched,
			),
		},
		{
			name: "matching username with case difference blocks",
			author: domain.AuthorIdentity{
				Username:    "BUYER.THREE",
				DisplayName: "Le Van C",
			},
			want: matchResult(
				MatchOutcomeBlocked,
				IdentityKey{Kind: IdentityKindUsername, Value: "buyer.three"},
				mustEntry(t, IdentityKindUsername, "buyer.three", "Le Van C"),
				ReasonIdentityMatched,
			),
		},
		{
			name: "unmatched stable identity is not blocked",
			author: domain.AuthorIdentity{
				FacebookUserID: "user-002",
				DisplayName:    "Tran Thi B",
			},
			want: matchResult(
				MatchOutcomeNotBlocked,
				IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-002"},
				Entry{},
				ReasonIdentityNotFound,
			),
		},
		{
			name: "display name alone is insufficient",
			author: domain.AuthorIdentity{
				DisplayName: "Nguyen Van A",
			},
			want: matchResult(MatchOutcomeInsufficientIdentity, IdentityKey{}, Entry{}, ReasonStableAuthorIdentityMissing),
		},
		{
			name:   "empty identity is insufficient",
			author: domain.AuthorIdentity{},
			want:   matchResult(MatchOutcomeInsufficientIdentity, IdentityKey{}, Entry{}, ReasonStableAuthorIdentityMissing),
		},
		{
			name: "same string under different identity kinds does not match",
			author: domain.AuthorIdentity{
				Username:    "same-text",
				DisplayName: "Same Text Username",
			},
			want: matchResult(
				MatchOutcomeNotBlocked,
				IdentityKey{Kind: IdentityKindUsername, Value: "same-text"},
				Entry{},
				ReasonIdentityNotFound,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := list.MatchAuthor(tt.author)
			assertMatchResult(t, got, tt.want)

			again := list.MatchAuthor(tt.author)
			if !reflect.DeepEqual(again, got) {
				t.Fatalf("MatchAuthor repeated result = %#v, want %#v", again, got)
			}
			if hasDuplicateReasons(got.Reasons) {
				t.Fatalf("duplicate reasons returned: %#v", got.Reasons)
			}
		})
	}
}

func TestStrongestIdentityPolicy(t *testing.T) {
	list := NewList([]Entry{
		mustEntry(t, IdentityKindUsername, "buyer.one", "Old Username"),
		mustEntry(t, IdentityKindCanonicalProfileURL, "https://facebook.example/buyer.two", "Profile Entry"),
		mustEntry(t, IdentityKindFacebookUserID, "user-003", "ID Entry"),
	})

	tests := []struct {
		name   string
		author domain.AuthorIdentity
		want   MatchOutcome
		key    IdentityKey
	}{
		{
			name: "author with Facebook user ID and username uses Facebook user ID",
			author: domain.AuthorIdentity{
				FacebookUserID: "user-003",
				Username:       "buyer.one",
				DisplayName:    "Nguyen Van A",
			},
			want: MatchOutcomeBlocked,
			key:  IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-003"},
		},
		{
			name: "unmatched Facebook user ID does not fall back to blocked username",
			author: domain.AuthorIdentity{
				FacebookUserID: "user-999",
				Username:       "buyer.one",
				DisplayName:    "Nguyen Van A",
			},
			want: MatchOutcomeNotBlocked,
			key:  IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-999"},
		},
		{
			name: "author without Facebook user ID may match profile URL",
			author: domain.AuthorIdentity{
				CanonicalProfileURL: "https://facebook.example/buyer.two/",
				Username:            "buyer.one",
				DisplayName:         "Tran Thi B",
			},
			want: MatchOutcomeBlocked,
			key:  IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/buyer.two"},
		},
		{
			name: "author without Facebook user ID or profile URL may match username",
			author: domain.AuthorIdentity{
				Username:    "BUYER.ONE",
				DisplayName: "Nguyen Van A",
			},
			want: MatchOutcomeBlocked,
			key:  IdentityKey{Kind: IdentityKindUsername, Value: "buyer.one"},
		},
		{
			name: "unmatched profile URL does not fall back to username when profile URL exists",
			author: domain.AuthorIdentity{
				CanonicalProfileURL: "https://facebook.example/not-blocked",
				Username:            "buyer.one",
				DisplayName:         "Nguyen Van A",
			},
			want: MatchOutcomeNotBlocked,
			key:  IdentityKey{Kind: IdentityKindCanonicalProfileURL, Value: "https://facebook.example/not-blocked"},
		},
		{
			name: "stronger stable identity wins regardless of display name",
			author: domain.AuthorIdentity{
				FacebookUserID: "user-not-blocked",
				DisplayName:    "ID Entry",
			},
			want: MatchOutcomeNotBlocked,
			key:  IdentityKey{Kind: IdentityKindFacebookUserID, Value: "user-not-blocked"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := list.MatchAuthor(tt.author)
			if got.Outcome != tt.want {
				t.Fatalf("Outcome = %q, want %q: %#v", got.Outcome, tt.want, got)
			}
			if got.AuthorKey != tt.key {
				t.Fatalf("AuthorKey = %#v, want %#v", got.AuthorKey, tt.key)
			}
		})
	}
}

func TestFailClosedCases(t *testing.T) {
	invalid, err := NewEntry(IdentityKind("display_name"), "Nguyen Van A", "Nguyen Van A")
	if !errors.Is(err, ErrUnsupportedIdentityKind) {
		t.Fatalf("NewEntry unsupported error = %v, want %v", err, ErrUnsupportedIdentityKind)
	}
	listWithMalformedInput := NewList([]Entry{invalid})
	if got := listWithMalformedInput.MatchAuthor(authorWithUserID("user-001")); got.Outcome != MatchOutcomeNotBlocked {
		t.Fatalf("malformed entry blocked author: %#v", got)
	}

	emptyList := NewList(nil)
	if got := emptyList.MatchAuthor(authorWithUserID("user-001")); got.Outcome != MatchOutcomeNotBlocked {
		t.Fatalf("empty blocklist outcome = %#v, want not blocked", got)
	}

	list := NewList([]Entry{
		mustEntry(t, IdentityKindUsername, "buyer.one", "Nguyễn Văn A"),
		mustEntry(t, IdentityKindCanonicalProfileURL, "https://facebook.example/buyer.one", "Tran Thi B"),
	})

	if got := list.MatchAuthor(domain.AuthorIdentity{DisplayName: "nguyễn văn a"}); got.Outcome != MatchOutcomeInsufficientIdentity {
		t.Fatalf("display-name casing created match: %#v", got)
	}
	if got := list.MatchAuthor(domain.AuthorIdentity{Username: "buyer.on", DisplayName: "Nguyen Van A"}); got.Outcome != MatchOutcomeNotBlocked {
		t.Fatalf("fuzzy username matched: %#v", got)
	}
	if got := list.MatchAuthor(domain.AuthorIdentity{Username: "buyer.one.extra", DisplayName: "Nguyen Van A"}); got.Outcome != MatchOutcomeNotBlocked {
		t.Fatalf("substring username matched: %#v", got)
	}
	if got := list.MatchAuthor(domain.AuthorIdentity{CanonicalProfileURL: "https://facebook.example/buyer.one/photos", DisplayName: "Tran Thi B"}); got.Outcome != MatchOutcomeNotBlocked {
		t.Fatalf("partial URL matched: %#v", got)
	}
}

func TestPostFieldsDoNotAffectAuthorMatching(t *testing.T) {
	list := NewList([]Entry{mustEntry(t, IdentityKindFacebookUserID, "user-001", "Nguyen Van A")})
	post := domain.RawPost{
		GroupID:   "group-1",
		PostURL:   "https://facebook.example/posts/user-001",
		Body:      "user-001 buyer.one https://facebook.example/buyer.one",
		Author:    domain.AuthorIdentity{DisplayName: "Nguyen Van A"},
		GroupName: "Synthetic Group",
	}

	got := list.MatchAuthor(post.Author)
	assertMatchResult(t, got, matchResult(MatchOutcomeInsufficientIdentity, IdentityKey{}, Entry{}, ReasonStableAuthorIdentityMissing))
}

func mustEntry(t *testing.T, kind IdentityKind, value string, displayName string) Entry {
	t.Helper()

	entry, err := NewEntry(kind, value, displayName)
	if err != nil {
		t.Fatalf("NewEntry(%q, %q) setup failed: %v", kind, value, err)
	}
	return entry
}

func authorWithUserID(userID string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		FacebookUserID: userID,
		DisplayName:    "Nguyen Van A",
	}
}

func assertMatchResult(t *testing.T, got MatchResult, want MatchResult) {
	t.Helper()

	if got.Outcome != want.Outcome {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, want.Outcome)
	}
	if got.AuthorKey != want.AuthorKey {
		t.Fatalf("AuthorKey = %#v, want %#v", got.AuthorKey, want.AuthorKey)
	}
	if got.MatchedEntry != want.MatchedEntry {
		t.Fatalf("MatchedEntry = %#v, want %#v", got.MatchedEntry, want.MatchedEntry)
	}
	if !reflect.DeepEqual(got.Reasons, want.Reasons) {
		t.Fatalf("Reasons = %#v, want %#v", got.Reasons, want.Reasons)
	}
}

func hasDuplicateReasons(reasons []ReasonCode) bool {
	seen := make(map[ReasonCode]struct{}, len(reasons))
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			return true
		}
		seen[reason] = struct{}{}
	}
	return false
}
