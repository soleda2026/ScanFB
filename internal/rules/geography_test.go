package rules

import (
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestClassifyGeographyHCMVocabulary(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "HCM normal", body: "can mua MacBook tai HCM"},
		{name: "HCM case variation", body: "can mua MacBook tai hcm"},
		{name: "HCM surrounding punctuation", body: "can mua MacBook (HCM), gap"},
		{name: "TPHCM normal", body: "can mua MacBook tai TPHCM"},
		{name: "TPHCM case variation", body: "can mua MacBook tai tphcm"},
		{name: "TPHCM surrounding punctuation", body: "can mua MacBook [TPHCM]."},
		{name: "TP.HCM normal", body: "can mua MacBook tai TP.HCM"},
		{name: "TP.HCM case variation", body: "can mua MacBook tai tp.hcm"},
		{name: "TP.HCM surrounding punctuation", body: "can mua MacBook /TP.HCM/"},
		{name: "Ho Chi Minh normal", body: "can mua MacBook tai Ho Chi Minh"},
		{name: "Ho Chi Minh case and repeated whitespace", body: "can mua MacBook tai HO   CHI\tMINH"},
		{name: "Ho Chi Minh surrounding punctuation", body: "can mua MacBook, Ho Chi Minh."},
		{name: "Sai Gon normal", body: "can mua MacBook tai Sai Gon"},
		{name: "Sai Gon case and repeated whitespace", body: "can mua MacBook tai SAI   GON"},
		{name: "Sai Gon surrounding punctuation", body: "can mua MacBook - Sai Gon!"},
		{name: "Saigon normal", body: "can mua MacBook tai Saigon"},
		{name: "Saigon case variation", body: "can mua MacBook tai saigon"},
		{name: "Saigon surrounding punctuation", body: "can mua MacBook: Saigon?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := postWithBody(tt.body)
			originalBody := post.Body

			got := ClassifyGeography(post)
			assertGeographicClassification(t, got, GeographicClassHoChiMinhCity, nil)
			if post.Body != originalBody {
				t.Fatalf("ClassifyGeography mutated body: got %q, want %q", post.Body, originalBody)
			}
		})
	}
}

func TestClassifyGeographyHCMVocabularyDoesNotMatchInsideLongerTokens(t *testing.T) {
	tests := []string{
		"can mua MacBook tai abcHCMxyz",
		"can mua MacBook tai TPHCMcity",
		"can mua MacBook tai TP.HCMcity",
		"can mua MacBook tai Ho Chi Minhhhh",
		"can mua MacBook tai Sai Gonese",
		"can mua MacBook tai Saigoner",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			got := ClassifyGeography(postWithBody(body))
			assertGeographicClassification(t, got, GeographicClassUnknown, []ReasonCode{ReasonLocationUnknown})
		})
	}
}

func TestClassifyGeographyOutsideHCMVietnamVocabulary(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "Hà Nội normal", body: "can mua MacBook tai Hà Nội"},
		{name: "Hà Nội case and repeated whitespace", body: "can mua MacBook tai HÀ   NỘI"},
		{name: "Hà Nội surrounding punctuation", body: "can mua MacBook (Hà Nội)"},
		{name: "Ha Noi normal", body: "can mua MacBook tai Ha Noi"},
		{name: "Ha Noi case and repeated whitespace", body: "can mua MacBook tai HA   NOI"},
		{name: "Ha Noi surrounding punctuation", body: "can mua MacBook, Ha Noi."},
		{name: "Đà Nẵng normal", body: "can mua MacBook tai Đà Nẵng"},
		{name: "Đà Nẵng case and repeated whitespace", body: "can mua MacBook tai ĐÀ   NẴNG"},
		{name: "Đà Nẵng surrounding punctuation", body: "can mua MacBook /Đà Nẵng/"},
		{name: "Da Nang normal", body: "can mua MacBook tai Da Nang"},
		{name: "Da Nang case and repeated whitespace", body: "can mua MacBook tai DA   NANG"},
		{name: "Da Nang surrounding punctuation", body: "can mua MacBook: Da Nang?"},
		{name: "Cần Thơ normal", body: "can mua MacBook tai Cần Thơ"},
		{name: "Cần Thơ case and repeated whitespace", body: "can mua MacBook tai CẦN   THƠ"},
		{name: "Cần Thơ surrounding punctuation", body: "can mua MacBook - Cần Thơ!"},
		{name: "Can Tho normal", body: "can mua MacBook tai Can Tho"},
		{name: "Can Tho case and repeated whitespace", body: "can mua MacBook tai CAN   THO"},
		{name: "Can Tho surrounding punctuation", body: "can mua MacBook [Can Tho]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyGeography(postWithBody(tt.body))
			assertGeographicClassification(t, got, GeographicClassOutsideHoChiMinhCityVN, nil)
		})
	}
}

func TestClassifyGeographyOutsideHCMVietnamDoesNotMatchHCMOrLongerTokens(t *testing.T) {
	tests := []string{
		"can mua MacBook tai Hà Nộian",
		"can mua MacBook tai Ha Noisy",
		"can mua MacBook tai Đà Nẵngx",
		"can mua MacBook tai Da Nangville",
		"can mua MacBook tai Cần Thơm",
		"can mua MacBook tai Can Thong",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			got := ClassifyGeography(postWithBody(body))
			assertGeographicClassification(t, got, GeographicClassUnknown, []ReasonCode{ReasonLocationUnknown})
		})
	}
}

func TestClassifyGeographyUnknown(t *testing.T) {
	tests := []string{
		"",
		" \t\n ",
		"can mua MacBook",
		"can mua MacBook tai synthetic district reference",
		"can mua MacBook tai unsupported Vietnamese city",
		"can mua MacBook tai foreign-looking-place",
		"ship toàn quốc",
		"giao hàng toàn quốc",
		"ở đâu cũng được",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			got := ClassifyGeography(postWithBody(body))
			assertGeographicClassification(t, got, GeographicClassUnknown, []ReasonCode{ReasonLocationUnknown})

			again := ClassifyGeography(postWithBody(body))
			if !reflect.DeepEqual(again, got) {
				t.Fatalf("ClassifyGeography repeated result = %#v, want %#v", again, got)
			}
		})
	}
}

func TestClassifyGeographyConflict(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "one HCM and one outside-HCM term",
			body: "can mua MacBook tai HCM hoac Ha Noi",
		},
		{
			name: "repeated matching terms do not duplicate reasons",
			body: "can mua MacBook HCM HCM Ha Noi Ha Noi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyGeography(postWithBody(tt.body))
			assertGeographicClassification(t, got, GeographicClassConflict, []ReasonCode{ReasonLocationConflict})

			again := ClassifyGeography(postWithBody(tt.body))
			if !reflect.DeepEqual(again, got) {
				t.Fatalf("ClassifyGeography repeated result = %#v, want %#v", again, got)
			}
		})
	}
}

func TestEvaluateGeographicMode(t *testing.T) {
	tests := []struct {
		name string
		body string
		mode domain.GeographicMode
		want Result
	}{
		{
			name: "HCM mode includes HCM",
			body: "can mua MacBook tai HCM",
			mode: domain.GeographicModeHoChiMinhCity,
			want: includeResult(),
		},
		{
			name: "HCM mode excludes outside HCM",
			body: "can mua MacBook tai Ha Noi",
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonOutsideSelectedGeographicMode, ReasonHoChiMinhCityRequired),
		},
		{
			name: "HCM mode reviews unknown",
			body: "can mua MacBook",
			mode: domain.GeographicModeHoChiMinhCity,
			want: reviewResult(ReasonLocationUnknown),
		},
		{
			name: "HCM mode reviews conflict",
			body: "can mua MacBook tai HCM va Ha Noi",
			mode: domain.GeographicModeHoChiMinhCity,
			want: reviewResult(ReasonLocationConflict),
		},
		{
			name: "outside-HCM mode includes outside HCM",
			body: "can mua MacBook tai Da Nang",
			mode: domain.GeographicModeOutsideHoChiMinhCityVN,
			want: includeResult(),
		},
		{
			name: "outside-HCM mode excludes HCM",
			body: "can mua MacBook tai HCM",
			mode: domain.GeographicModeOutsideHoChiMinhCityVN,
			want: excludeResult(ReasonOutsideSelectedGeographicMode, ReasonOutsideHoChiMinhCityVNRequired),
		},
		{
			name: "outside-HCM mode reviews unknown",
			body: "can mua MacBook",
			mode: domain.GeographicModeOutsideHoChiMinhCityVN,
			want: reviewResult(ReasonLocationUnknown),
		},
		{
			name: "outside-HCM mode reviews conflict",
			body: "can mua MacBook tai HCM va Ha Noi",
			mode: domain.GeographicModeOutsideHoChiMinhCityVN,
			want: reviewResult(ReasonLocationConflict),
		},
		{
			name: "all Vietnam mode includes HCM",
			body: "can mua MacBook tai HCM",
			mode: domain.GeographicModeAllVietnam,
			want: includeResult(),
		},
		{
			name: "all Vietnam mode includes outside HCM",
			body: "can mua MacBook tai Can Tho",
			mode: domain.GeographicModeAllVietnam,
			want: includeResult(),
		},
		{
			name: "all Vietnam mode reviews unknown",
			body: "can mua MacBook",
			mode: domain.GeographicModeAllVietnam,
			want: reviewResult(ReasonLocationUnknown),
		},
		{
			name: "all Vietnam mode reviews conflict",
			body: "can mua MacBook tai HCM va Ha Noi",
			mode: domain.GeographicModeAllVietnam,
			want: reviewResult(ReasonLocationConflict),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateGeographicMode(postWithBody(tt.body), tt.mode)
			assertResult(t, got, tt.want)

			again := EvaluateGeographicMode(postWithBody(tt.body), tt.mode)
			assertResult(t, again, got)
		})
	}
}

func TestEvaluatePostForBuyerSearchAndGeographyComposition(t *testing.T) {
	window := testScanWindow(t)
	loc := testLocation()
	profile := domain.MacBookSearchProfile()

	tests := []struct {
		name string
		post domain.RawPost
		mode domain.GeographicMode
		want Result
	}{
		{
			name: "valid buyer post with HCM geography includes in HCM mode",
			post: postWithBody("Cần mua MacBook tai HCM"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: includeResult(),
		},
		{
			name: "valid buyer post with outside-HCM geography includes in outside-HCM mode",
			post: postWithBody("Cần mua MacBook tai Hà Nội"),
			mode: domain.GeographicModeOutsideHoChiMinhCityVN,
			want: includeResult(),
		},
		{
			name: "valid buyer post with HCM geography includes in all Vietnam mode",
			post: postWithBody("Cần mua MacBook tai Saigon"),
			mode: domain.GeographicModeAllVietnam,
			want: includeResult(),
		},
		{
			name: "valid buyer post with outside-HCM geography includes in all Vietnam mode",
			post: postWithBody("Cần mua MacBook tai Cần Thơ"),
			mode: domain.GeographicModeAllVietnam,
			want: includeResult(),
		},
		{
			name: "valid buyer post with wrong domestic geography excludes",
			post: postWithBody("Cần mua MacBook tai Đà Nẵng"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonOutsideSelectedGeographicMode, ReasonHoChiMinhCityRequired),
		},
		{
			name: "valid buyer post with unknown geography reviews",
			post: postWithBody("Cần mua MacBook"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: reviewResult(ReasonLocationUnknown),
		},
		{
			name: "valid buyer post with conflicting geography reviews",
			post: postWithBody("Cần mua MacBook tai HCM va Ha Noi"),
			mode: domain.GeographicModeAllVietnam,
			want: reviewResult(ReasonLocationConflict),
		},
		{
			name: "anonymous author remains excluded even if geography matches",
			post: func() domain.RawPost {
				post := postWithBody("Cần mua MacBook tai HCM")
				post.Author.DisplayName = "Anonymous"
				return post
			}(),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonAnonymousAuthor),
		},
		{
			name: "invalid time remains excluded even if geography matches",
			post: func() domain.RawPost {
				post := postWithBody("Cần mua MacBook tai HCM")
				post.CreatedAt = time.Date(2026, 8, 6, 0, 0, 0, 0, loc)
				return post
			}(),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonPostAfterScanStart),
		},
		{
			name: "seller noise remains excluded even if geography matches",
			post: postWithBody("Bán MacBook Pro tai HCM"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "buyer intent failure remains excluded even if geography matches",
			post: postWithBody("MacBook Pro tai HCM"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonBuyerIntentMissing),
		},
		{
			name: "geography review does not override seller exclusion",
			post: postWithBody("Bán MacBook Pro"),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(ReasonSellerIntent),
		},
		{
			name: "deterministic reason ordering across all excluding stages",
			post: func() domain.RawPost {
				post := postWithBody("Bán MacBook Pro tai Da Nang")
				post.CreatedAt = time.Date(2026, 8, 6, 0, 0, 0, 0, loc)
				post.Author.DisplayName = "Anonymous"
				return post
			}(),
			mode: domain.GeographicModeHoChiMinhCity,
			want: excludeResult(
				ReasonPostAfterScanStart,
				ReasonAnonymousAuthor,
				ReasonSellerIntent,
				ReasonOutsideSelectedGeographicMode,
				ReasonHoChiMinhCityRequired,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePostForBuyerSearchAndGeography(tt.post, window, profile, tt.mode)
			assertResult(t, got, tt.want)
			if hasDuplicateReasons(got.Reasons) {
				t.Fatalf("duplicate reasons returned: %#v", got.Reasons)
			}

			again := EvaluatePostForBuyerSearchAndGeography(tt.post, window, profile, tt.mode)
			assertResult(t, again, got)
		})
	}
}

func TestCombineResultsReviewSemantics(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    Result
	}{
		{
			name:    "review wins over include",
			results: []Result{includeResult(), reviewResult(ReasonLocationUnknown)},
			want:    reviewResult(ReasonLocationUnknown),
		},
		{
			name:    "exclude wins over review",
			results: []Result{excludeResult(ReasonSellerIntent), reviewResult(ReasonLocationUnknown)},
			want:    excludeResult(ReasonSellerIntent),
		},
		{
			name:    "review reasons deduplicate in order",
			results: []Result{reviewResult(ReasonLocationUnknown), reviewResult(ReasonLocationUnknown, ReasonLocationConflict)},
			want:    reviewResult(ReasonLocationUnknown, ReasonLocationConflict),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := combineResults(tt.results...)
			assertResult(t, got, tt.want)
		})
	}
}

func assertGeographicClassification(t *testing.T, got GeographicClassification, wantClass GeographicClass, wantReasons []ReasonCode) {
	t.Helper()

	if got.Class != wantClass {
		t.Fatalf("Class = %q, want %q", got.Class, wantClass)
	}
	if !reflect.DeepEqual(got.Reasons, wantReasons) {
		t.Fatalf("Reasons = %#v, want %#v", got.Reasons, wantReasons)
	}
}
