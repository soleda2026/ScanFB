package rules

import (
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestEvaluateBuyerIntentProductMatching(t *testing.T) {
	profile := mustSearchProfile(t,
		[]string{"MacBook Pro"},
		[]string{"cần mua"},
		nil,
	)

	tests := []struct {
		name string
		body string
		want Result
	}{
		{
			name: "product term alone is not sufficient",
			body: "MacBook Pro",
			want: excludeResult(ReasonBuyerIntentMissing),
		},
		{
			name: "exact product phrase matches",
			body: "cần mua MacBook Pro",
			want: includeResult(),
		},
		{
			name: "case differences match",
			body: "CẦN MUA MACBOOK PRO",
			want: includeResult(),
		},
		{
			name: "surrounding and repeated whitespace are handled",
			body: " \tcần   mua   MacBook     Pro\n",
			want: includeResult(),
		},
		{
			name: "no product term produces product exclusion reason",
			body: "cần mua iPhone",
			want: excludeResult(ReasonTargetKeywordMissing),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateBuyerIntent(postWithBody(tt.body), profile)
			assertResult(t, got, tt.want)
		})
	}
}

func TestEvaluateBuyerIntentSingleWordBoundary(t *testing.T) {
	profile := mustSearchProfile(t,
		[]string{"Pro"},
		[]string{"cần mua"},
		nil,
	)

	got := EvaluateBuyerIntent(postWithBody("cần mua Projector"), profile)
	assertResult(t, got, excludeResult(ReasonTargetKeywordMissing))
}

func TestEvaluateBuyerIntentTerms(t *testing.T) {
	profile := mustSearchProfile(t,
		[]string{"MacBook"},
		[]string{"cần mua", "tìm mua"},
		nil,
	)

	tests := []struct {
		name string
		body string
		want Result
	}{
		{
			name: "product plus buyer-intent term is included",
			body: "cần mua MacBook",
			want: includeResult(),
		},
		{
			name: "buyer-intent term without product is excluded",
			body: "cần mua máy",
			want: excludeResult(ReasonTargetKeywordMissing),
		},
		{
			name: "product without buyer-intent term is excluded",
			body: "MacBook",
			want: excludeResult(ReasonBuyerIntentMissing),
		},
		{
			name: "multiple buyer-intent terms do not duplicate reason codes",
			body: "cần mua tìm mua MacBook",
			want: includeResult(),
		},
		{
			name: "case variants match",
			body: "TÌM MUA MACBOOK",
			want: includeResult(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateBuyerIntent(postWithBody(tt.body), profile)
			assertResult(t, got, tt.want)
		})
	}
}

func TestEvaluateBuyerIntentSellerNoise(t *testing.T) {
	profile := mustSearchProfile(t,
		[]string{"MacBook"},
		[]string{"cần mua"},
		[]string{"bán", "có sẵn", "quảng cáo"},
	)

	tests := []struct {
		name string
		body string
		want Result
	}{
		{
			name: "product plus seller term is excluded",
			body: "bán MacBook",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "product plus buyer term plus seller term is excluded",
			body: "cần mua MacBook nhưng bài này bán MacBook",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "seller exclusion takes precedence over missing buyer term",
			body: "MacBook có sẵn",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "advertising example is excluded",
			body: "quảng cáo MacBook",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "seller term matching is boundary-aware",
			body: "cần mua MacBook để bàn",
			want: includeResult(),
		},
		{
			name: "no seller term does not cause exclusion",
			body: "cần mua MacBook",
			want: includeResult(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateBuyerIntent(postWithBody(tt.body), profile)
			assertResult(t, got, tt.want)
		})
	}
}

func TestEvaluateBuyerIntentEmptyAndEdgeInput(t *testing.T) {
	profile := mustSearchProfile(t,
		[]string{"MacBook"},
		[]string{"cần mua"},
		nil,
	)

	tests := []struct {
		name    string
		post    domain.RawPost
		profile domain.SearchProfile
		want    Result
	}{
		{
			name:    "empty body excluded",
			post:    postWithBody(""),
			profile: profile,
			want:    excludeResult(ReasonPostBodyMissing),
		},
		{
			name:    "whitespace-only body excluded",
			post:    postWithBody(" \t\n "),
			profile: profile,
			want:    excludeResult(ReasonPostBodyMissing),
		},
		{
			name:    "empty profile term slices fail closed",
			post:    postWithBody("cần mua MacBook"),
			profile: domain.SearchProfile{},
			want:    excludeResult(ReasonTargetKeywordMissing, ReasonBuyerIntentMissing),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalBody := tt.post.Body
			got := EvaluateBuyerIntent(tt.post, tt.profile)
			assertResult(t, got, tt.want)
			if tt.post.Body != originalBody {
				t.Fatalf("EvaluateBuyerIntent mutated body: got %q, want %q", tt.post.Body, originalBody)
			}

			again := EvaluateBuyerIntent(tt.post, tt.profile)
			assertResult(t, again, got)
		})
	}
}

func TestEvaluateBuyerIntentMacBookProfileExamples(t *testing.T) {
	profile := domain.MacBookSearchProfile()

	tests := []struct {
		name string
		body string
		want Result
	}{
		{
			name: "buyer can mua MacBook Pro",
			body: "Cần mua MacBook Pro",
			want: includeResult(),
		},
		{
			name: "buyer dang tim MacBook Air",
			body: "Mình đang tìm MacBook Air",
			want: includeResult(),
		},
		{
			name: "buyer documented phrase",
			body: "Có ai bán MacBook không?",
			want: includeResult(),
		},
		{
			name: "seller ban MacBook Pro",
			body: "Bán MacBook Pro",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "seller stock offer",
			body: "MacBook Air có sẵn",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "advertising seller noise",
			body: "Quảng cáo MacBook giá tốt",
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "ambiguous no buyer intent",
			body: "MacBook Pro",
			want: excludeResult(ReasonBuyerIntentMissing),
		},
		{
			name: "review no buyer intent",
			body: "Ai dùng MacBook rồi cho xin review",
			want: excludeResult(ReasonBuyerIntentMissing),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateBuyerIntent(postWithBody(tt.body), profile)
			assertResult(t, got, tt.want)
		})
	}
}

func TestEvaluatePostForBuyerSearchComposition(t *testing.T) {
	window := testScanWindow(t)
	loc := testLocation()
	profile := domain.MacBookSearchProfile()

	tests := []struct {
		name string
		post domain.RawPost
		want Result
	}{
		{
			name: "valid time acceptable author and buyer intent included",
			post: postWithBody("Cần mua MacBook Pro"),
			want: includeResult(),
		},
		{
			name: "valid time and author but no buyer intent excluded",
			post: postWithBody("MacBook Pro"),
			want: excludeResult(ReasonBuyerIntentMissing),
		},
		{
			name: "buyer intent cannot override anonymous author",
			post: func() domain.RawPost {
				post := postWithBody("Cần mua MacBook Pro")
				post.Author.DisplayName = "Anonymous"
				return post
			}(),
			want: excludeResult(ReasonAnonymousAuthor),
		},
		{
			name: "buyer intent cannot override invalid time",
			post: func() domain.RawPost {
				post := postWithBody("Cần mua MacBook Pro")
				post.CreatedAt = time.Date(2026, 8, 6, 0, 0, 0, 0, loc)
				return post
			}(),
			want: excludeResult(ReasonPostAfterScanStart),
		},
		{
			name: "seller reason appears after existing reasons",
			post: func() domain.RawPost {
				post := postWithBody("Bán MacBook Pro")
				post.CreatedAt = time.Date(2026, 8, 6, 0, 0, 0, 0, loc)
				post.Author.DisplayName = "Anonymous"
				return post
			}(),
			want: excludeResult(ReasonPostAfterScanStart, ReasonAnonymousAuthor, ReasonSellerIntent),
		},
		{
			name: "no duplicate reasons across composed evaluation",
			post: func() domain.RawPost {
				post := postWithBody("MacBook")
				post.CreatedAt = time.Time{}
				return post
			}(),
			want: excludeResult(ReasonPostCreatedAtMissing, ReasonBuyerIntentMissing),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePostForBuyerSearch(tt.post, window, profile)
			assertResult(t, got, tt.want)
			if hasDuplicateReasons(got.Reasons) {
				t.Fatalf("duplicate reasons returned: %#v", got.Reasons)
			}

			again := EvaluatePostForBuyerSearch(tt.post, window, profile)
			assertResult(t, again, got)
		})
	}
}

func mustSearchProfile(t *testing.T, productTerms []string, buyerIntentTerms []string, noiseTerms []string) domain.SearchProfile {
	t.Helper()

	profile, err := domain.NewSearchProfile("test-profile", "Test Profile", productTerms, buyerIntentTerms, noiseTerms, true)
	if err != nil {
		t.Fatalf("NewSearchProfile setup failed: %v", err)
	}
	return profile
}

func postWithBody(body string) domain.RawPost {
	post := testPost()
	post.Body = body
	return post
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

func TestCombineResultsDeduplicatesReasons(t *testing.T) {
	got := combineResults(
		excludeResult(ReasonBuyerIntentMissing),
		excludeResult(ReasonBuyerIntentMissing, ReasonSellerIntent),
	)
	want := excludeResult(ReasonBuyerIntentMissing, ReasonSellerIntent)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("combineResults = %#v, want %#v", got, want)
	}
}
