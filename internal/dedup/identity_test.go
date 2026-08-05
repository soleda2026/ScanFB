package dedup

import (
	"reflect"
	"testing"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestAuthorKeyPriorityAndNormalization(t *testing.T) {
	tests := []struct {
		name   string
		author domain.AuthorIdentity
		want   AuthorKey
	}{
		{
			name: "Facebook user ID preferred when present",
			author: domain.AuthorIdentity{
				FacebookUserID:      " user-1 ",
				CanonicalProfileURL: "https://example.test/user-1",
				Username:            "Buyer.One",
				DisplayName:         "Buyer One",
			},
			want: AuthorKey{Kind: AuthorIdentityKindFacebookUserID, Value: "user-1"},
		},
		{
			name: "canonical profile URL used when Facebook user ID absent",
			author: domain.AuthorIdentity{
				CanonicalProfileURL: " https://example.test/user-1/ ",
				Username:            "Buyer.One",
				DisplayName:         "Buyer One",
			},
			want: AuthorKey{Kind: AuthorIdentityKindCanonicalProfileURL, Value: "https://example.test/user-1"},
		},
		{
			name: "username used when stronger identifiers absent",
			author: domain.AuthorIdentity{
				Username:    " Buyer.One ",
				DisplayName: "Buyer One",
			},
			want: AuthorKey{Kind: AuthorIdentityKindUsername, Value: "buyer.one"},
		},
		{
			name: "display name alone is insufficient",
			author: domain.AuthorIdentity{
				DisplayName: "Buyer One",
			},
			want: AuthorKey{},
		},
		{
			name:   "empty identity is insufficient",
			author: domain.AuthorIdentity{},
			want:   AuthorKey{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.author
			got := NewAuthorKey(tt.author)
			if got != tt.want {
				t.Fatalf("NewAuthorKey = %#v, want %#v", got, tt.want)
			}
			if got.Sufficient() != (tt.want != (AuthorKey{})) {
				t.Fatalf("Sufficient = %v for key %#v", got.Sufficient(), got)
			}
			if tt.author != original {
				t.Fatalf("NewAuthorKey mutated author: got %#v, want %#v", tt.author, original)
			}
		})
	}
}

func TestNeedKeyNormalizationEvidenceAndFingerprint(t *testing.T) {
	profile := domain.MacBookSearchProfile()

	tests := []struct {
		name               string
		leftBody           string
		rightBody          string
		wantSame           bool
		wantSufficient     bool
		wantNormalizedBody string
	}{
		{
			name:               "equivalent repeated whitespace produces same key",
			leftBody:           "Cần mua MacBook Pro",
			rightBody:          "  cần   mua   macbook pro  ",
			wantSame:           true,
			wantSufficient:     true,
			wantNormalizedBody: "cần mua macbook pro",
		},
		{
			name:               "line breaks normalize to spaces",
			leftBody:           "Cần mua\nMacBook Pro",
			rightBody:          "cần mua macbook pro",
			wantSame:           true,
			wantSufficient:     true,
			wantNormalizedBody: "cần mua macbook pro",
		},
		{
			name:               "Vietnamese diacritics are preserved",
			leftBody:           "Cần mua MacBook Pro",
			rightBody:          "Can mua MacBook Pro",
			wantSame:           false,
			wantSufficient:     true,
			wantNormalizedBody: "cần mua macbook pro",
		},
		{
			name:               "materially different normalized text differs",
			leftBody:           "Cần mua MacBook Pro 16 inch",
			rightBody:          "Cần mua MacBook Air",
			wantSame:           false,
			wantSufficient:     true,
			wantNormalizedBody: "cần mua macbook pro 16 inch",
		},
		{
			name:               "product evidence is required",
			leftBody:           "Cần mua iPhone",
			rightBody:          "Cần mua iPhone",
			wantSame:           true,
			wantSufficient:     false,
			wantNormalizedBody: "cần mua iphone",
		},
		{
			name:               "buyer intent evidence is required",
			leftBody:           "MacBook Pro",
			rightBody:          "macbook pro",
			wantSame:           true,
			wantSufficient:     false,
			wantNormalizedBody: "macbook pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leftPost := postWithBody(tt.leftBody)
			rightPost := postWithBody(tt.rightBody)
			originalBody := leftPost.Body

			left := NewNeedKey(leftPost, profile)
			right := NewNeedKey(rightPost, profile)

			if left.NormalizedBody != tt.wantNormalizedBody {
				t.Fatalf("NormalizedBody = %q, want %q", left.NormalizedBody, tt.wantNormalizedBody)
			}
			if left.Sufficient() != tt.wantSufficient {
				t.Fatalf("Sufficient = %v, want %v for %#v", left.Sufficient(), tt.wantSufficient, left)
			}
			gotSame := left.SearchProfileID == right.SearchProfileID && left.BodyFingerprint == right.BodyFingerprint
			if gotSame != tt.wantSame {
				t.Fatalf("key equality = %v, want %v\nleft=%#v\nright=%#v", gotSame, tt.wantSame, left, right)
			}
			if leftPost.Body != originalBody {
				t.Fatalf("NewNeedKey mutated body: got %q, want %q", leftPost.Body, originalBody)
			}

			again := NewNeedKey(leftPost, profile)
			if !reflect.DeepEqual(again, left) {
				t.Fatalf("NewNeedKey repeated result = %#v, want %#v", again, left)
			}
		})
	}
}

func TestNeedKeyEvidenceAndSearchProfileID(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	key := NewNeedKey(postWithBody("Cần mua MacBook Pro"), profile)

	if key.SearchProfileID != profile.ID() {
		t.Fatalf("SearchProfileID = %q, want %q", key.SearchProfileID, profile.ID())
	}
	if !reflect.DeepEqual(key.ProductEvidence, []string{"macbook", "macbook pro"}) {
		t.Fatalf("ProductEvidence = %#v", key.ProductEvidence)
	}
	if !reflect.DeepEqual(key.BuyerIntentEvidence, []string{"cần mua"}) {
		t.Fatalf("BuyerIntentEvidence = %#v", key.BuyerIntentEvidence)
	}
	if key.BodyFingerprint != fingerprint("cần mua macbook pro") {
		t.Fatalf("BodyFingerprint = %q, want stable fingerprint", key.BodyFingerprint)
	}

	otherProfile := mustSearchProfile(t, "other-profile", []string{"MacBook"}, []string{"cần mua"}, nil)
	otherKey := NewNeedKey(postWithBody("Cần mua MacBook Pro"), otherProfile)
	if key.SearchProfileID == otherKey.SearchProfileID {
		t.Fatalf("different SearchProfiles were not represented in keys: %#v %#v", key, otherKey)
	}
}

func TestComparePosts(t *testing.T) {
	profile := domain.MacBookSearchProfile()

	tests := []struct {
		name  string
		left  domain.RawPost
		right domain.RawPost
		want  Comparison
	}{
		{
			name:  "same stable author plus same normalized need across different groups is duplicate",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua MacBook Pro"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "  cần   mua   macbook pro  "),
			want:  comparisonOnly(ComparisonOutcomeDuplicateNeed, ReasonDuplicateNeedMatched),
		},
		{
			name:  "same stable author plus same normalized need with different post IDs is duplicate",
			left:  withPostID(postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua MacBook Pro"), "post-1"),
			right: withPostID(postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Pro"), "post-2"),
			want:  comparisonOnly(ComparisonOutcomeDuplicateNeed, ReasonDuplicateNeedMatched),
		},
		{
			name:  "same post ID is same source",
			left:  withPostID(postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua MacBook Pro"), "post-1"),
			right: withPostID(postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Pro 16 inch"), " post-1 "),
			want:  comparisonOnly(ComparisonOutcomeSameSourcePost, ReasonSamePostID),
		},
		{
			name:  "different stable authors plus identical body is distinct",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua MacBook Pro"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-2"), "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonStableAuthorIdentityDiffers),
		},
		{
			name:  "same display name but no stable identifiers is insufficient",
			left:  postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "Buyer One"}, "group-1", "Cần mua MacBook Pro"),
			right: postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "Buyer One"}, "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeInsufficientIdentity, ReasonStableAuthorIdentityMissing),
		},
		{
			name:  "same stable author plus different need body is distinct",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua MacBook Pro 16 inch"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Air"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonNormalizedNeedDiffers),
		},
		{
			name:  "missing product evidence is not duplicate",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Cần mua iPhone"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonProductEvidenceMissing),
		},
		{
			name:  "missing buyer intent evidence is not duplicate",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "MacBook Pro"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonBuyerIntentEvidenceMissing),
		},
		{
			name:  "empty body is not duplicate",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", ""),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonProductEvidenceMissing, ReasonBuyerIntentEvidenceMissing),
		},
		{
			name:  "seller noise example is not duplicate buyer need",
			left:  postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "Bán MacBook Pro"),
			right: postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "Bán MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeDistinct, ReasonBuyerIntentEvidenceMissing),
		},
		{
			name:  "same author in same group can still be compared",
			left:  postWithAuthorGroupAndBody(authorWithUsername("buyer.one"), "group-1", "Mình đang tìm MacBook Air"),
			right: postWithAuthorGroupAndBody(authorWithUsername("BUYER.ONE"), "group-1", "mình đang tìm macbook air"),
			want:  comparisonOnly(ComparisonOutcomeDuplicateNeed, ReasonDuplicateNeedMatched),
		},
		{
			name:  "missing group ID does not establish author identity",
			left:  postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "Buyer One"}, "", "Cần mua MacBook Pro"),
			right: postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "Buyer One"}, "", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeInsufficientIdentity, ReasonStableAuthorIdentityMissing),
		},
		{
			name:  "post URL differences do not defeat duplicate detection",
			left:  withPostURL(postWithAuthorGroupAndBody(authorWithURL("https://example.test/u/1/"), "group-1", "Cần mua MacBook Pro"), "https://example.test/posts/1"),
			right: withPostURL(postWithAuthorGroupAndBody(authorWithURL("https://example.test/u/1"), "group-2", "Cần mua MacBook Pro"), "https://example.test/posts/2"),
			want:  comparisonOnly(ComparisonOutcomeDuplicateNeed, ReasonDuplicateNeedMatched),
		},
		{
			name:  "display-name casing does not create stable identity",
			left:  postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "buyer one"}, "group-1", "Cần mua MacBook Pro"),
			right: postWithAuthorGroupAndBody(domain.AuthorIdentity{DisplayName: "BUYER ONE"}, "group-2", "Cần mua MacBook Pro"),
			want:  comparisonOnly(ComparisonOutcomeInsufficientIdentity, ReasonStableAuthorIdentityMissing),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComparePosts(tt.left, tt.right, profile)
			assertComparison(t, got, tt.want)

			again := ComparePosts(tt.left, tt.right, profile)
			if !reflect.DeepEqual(again, got) {
				t.Fatalf("ComparePosts repeated result = %#v, want %#v", again, got)
			}
		})
	}
}

func TestComparePostsReasonOrdering(t *testing.T) {
	profile := domain.SearchProfile{}
	left := postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", "")
	right := postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-2", "")

	got := ComparePosts(left, right, profile)
	want := comparisonOnly(ComparisonOutcomeDistinct, ReasonProductEvidenceMissing, ReasonBuyerIntentEvidenceMissing)
	assertComparison(t, got, want)
}

func postWithBody(body string) domain.RawPost {
	return postWithAuthorGroupAndBody(authorWithUserID("user-1"), "group-1", body)
}

func postWithAuthorGroupAndBody(author domain.AuthorIdentity, groupID string, body string) domain.RawPost {
	return domain.RawPost{
		GroupID:   groupID,
		GroupName: "Synthetic Group",
		PostURL:   "https://example.test/posts/" + groupID,
		Author:    author,
		Body:      body,
	}
}

func authorWithUserID(userID string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		FacebookUserID: userID,
		DisplayName:    "Buyer One",
	}
}

func authorWithURL(url string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		CanonicalProfileURL: url,
		DisplayName:         "Buyer One",
	}
}

func authorWithUsername(username string) domain.AuthorIdentity {
	return domain.AuthorIdentity{
		Username:    username,
		DisplayName: "Buyer One",
	}
}

func withPostID(post domain.RawPost, postID string) domain.RawPost {
	post.PostID = postID
	return post
}

func withPostURL(post domain.RawPost, postURL string) domain.RawPost {
	post.PostURL = postURL
	return post
}

func mustSearchProfile(t *testing.T, id string, productTerms []string, buyerIntentTerms []string, noiseTerms []string) domain.SearchProfile {
	t.Helper()

	profile, err := domain.NewSearchProfile(id, "Test Profile", productTerms, buyerIntentTerms, noiseTerms, true)
	if err != nil {
		t.Fatalf("NewSearchProfile setup failed: %v", err)
	}
	return profile
}

func comparisonOnly(outcome ComparisonOutcome, reasons ...ReasonCode) Comparison {
	return Comparison{
		Outcome: outcome,
		Reasons: reasons,
	}
}

func assertComparison(t *testing.T, got Comparison, want Comparison) {
	t.Helper()

	if got.Outcome != want.Outcome {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, want.Outcome)
	}
	if !reflect.DeepEqual(got.Reasons, want.Reasons) {
		t.Fatalf("Reasons = %#v, want %#v", got.Reasons, want.Reasons)
	}
}
