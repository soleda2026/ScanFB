package rules

import (
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestEvaluatePostTime(t *testing.T) {
	window := testScanWindow(t)
	loc := testLocation()

	tests := []struct {
		name       string
		createdAt  time.Time
		capturedAt time.Time
		want       Result
	}{
		{
			name:      "at start of day included",
			createdAt: time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
			want:      includeResult(),
		},
		{
			name:      "at scan start included",
			createdAt: time.Date(2026, 8, 5, 10, 30, 0, 0, loc),
			want:      includeResult(),
		},
		{
			name:      "one nanosecond before start of day excluded",
			createdAt: time.Date(2026, 8, 4, 23, 59, 59, int(time.Second)-1, loc),
			want:      excludeResult(ReasonPostBeforeScanWindow),
		},
		{
			name:      "one nanosecond after scan start excluded",
			createdAt: time.Date(2026, 8, 5, 10, 30, 0, 1, loc),
			want:      excludeResult(ReasonPostAfterScanStart),
		},
		{
			name:      "previous day excluded",
			createdAt: time.Date(2026, 8, 4, 12, 0, 0, 0, loc),
			want:      excludeResult(ReasonPostBeforeScanWindow),
		},
		{
			name:      "next day excluded",
			createdAt: time.Date(2026, 8, 6, 0, 0, 0, 0, loc),
			want:      excludeResult(ReasonPostAfterScanStart),
		},
		{
			name:       "zero created at excluded even when captured at is inside window",
			createdAt:  time.Time{},
			capturedAt: time.Date(2026, 8, 5, 9, 0, 0, 0, loc),
			want:       excludeResult(ReasonPostCreatedAtMissing),
		},
		{
			name:      "another location representing instant inside window included",
			createdAt: time.Date(2026, 8, 5, 2, 0, 0, 0, time.UTC),
			want:      includeResult(),
		},
		{
			name:       "captured at inside window does not rescue created at before window",
			createdAt:  time.Date(2026, 8, 4, 12, 0, 0, 0, loc),
			capturedAt: time.Date(2026, 8, 5, 9, 0, 0, 0, loc),
			want:       excludeResult(ReasonPostBeforeScanWindow),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := testPost()
			post.CreatedAt = tt.createdAt
			post.CapturedAt = tt.capturedAt

			got := EvaluatePostTime(post, window)
			assertResult(t, got, tt.want)
		})
	}
}

func TestEvaluateAuthor(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		identity    domain.AuthorIdentity
		want        Result
	}{
		{
			name:        "Anonymous excluded",
			displayName: "Anonymous",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "lowercase anonymous excluded",
			displayName: "anonymous",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "mixed case anonymous excluded",
			displayName: "aNoNyMoUs",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "surrounding whitespace anonymous excluded",
			displayName: " \tAnonymous\n ",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "documented Vietnamese member anonymous label excluded",
			displayName: "Thanh vien an danh",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "documented Vietnamese participant anonymous label excluded",
			displayName: "Nguoi tham gia an danh",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "case variant Vietnamese anonymous label excluded",
			displayName: "tHaNh ViEn An DaNh",
			want:        excludeResult(ReasonAnonymousAuthor),
		},
		{
			name:        "motivated alias excluded",
			displayName: "motivatedsalamander3113",
			want:        excludeResult(ReasonAuthorNameHasNoWhitespace),
		},
		{
			name:        "single word name excluded by policy",
			displayName: "Nguyen",
			want:        excludeResult(ReasonAuthorNameHasNoWhitespace),
		},
		{
			name:        "multi word name accepted",
			displayName: "Nguyen Van A",
			want:        includeResult(),
		},
		{
			name:        "Unicode whitespace accepted",
			displayName: "Trần\u00a0Minh",
			want:        includeResult(),
		},
		{
			name:        "empty display name excluded",
			displayName: " \t\n ",
			want:        excludeResult(ReasonAuthorDisplayNameMissing),
		},
		{
			name:        "stable IDs do not override anonymous exclusion",
			displayName: "Anonymous",
			identity: domain.AuthorIdentity{
				FacebookUserID:      "user-1",
				CanonicalProfileURL: "https://example.test/user-1",
				Username:            "buyer.one",
			},
			want: excludeResult(ReasonAnonymousAuthor),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			author := tt.identity
			author.DisplayName = tt.displayName
			originalDisplayName := author.DisplayName

			got := EvaluateAuthor(author)
			assertResult(t, got, tt.want)

			if author.DisplayName != originalDisplayName {
				t.Fatalf("EvaluateAuthor mutated display name: got %q, want %q", author.DisplayName, originalDisplayName)
			}
		})
	}
}

func TestEvaluatePostComposition(t *testing.T) {
	window := testScanWindow(t)
	loc := testLocation()

	tests := []struct {
		name string
		post domain.RawPost
		want Result
	}{
		{
			name: "valid time and acceptable author included",
			post: testPost(),
			want: includeResult(),
		},
		{
			name: "invalid time and acceptable author excluded with time reason",
			post: func() domain.RawPost {
				post := testPost()
				post.CreatedAt = time.Date(2026, 8, 4, 23, 0, 0, 0, loc)
				return post
			}(),
			want: excludeResult(ReasonPostBeforeScanWindow),
		},
		{
			name: "valid time and anonymous author excluded with author reason",
			post: func() domain.RawPost {
				post := testPost()
				post.Author.DisplayName = "Anonymous"
				return post
			}(),
			want: excludeResult(ReasonAnonymousAuthor),
		},
		{
			name: "multiple failures produce deterministic order",
			post: func() domain.RawPost {
				post := testPost()
				post.CreatedAt = time.Date(2026, 8, 6, 0, 0, 0, 0, loc)
				post.Author.DisplayName = "motivatedsalamander3113"
				return post
			}(),
			want: excludeResult(ReasonPostAfterScanStart, ReasonAuthorNameHasNoWhitespace),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluatePost(tt.post, window)
			assertResult(t, got, tt.want)

			again := EvaluatePost(tt.post, window)
			assertResult(t, again, got)
		})
	}
}

func testPost() domain.RawPost {
	loc := testLocation()
	return domain.RawPost{
		PostID:    "post-1",
		GroupID:   "group-1",
		GroupName: "Group One",
		PostURL:   "https://example.test/posts/post-1",
		Author: domain.AuthorIdentity{
			FacebookUserID:      "user-1",
			CanonicalProfileURL: "https://example.test/user-1",
			Username:            "buyer.one",
			DisplayName:         "Nguyen Van A",
		},
		Body:       "can mua MacBook",
		CreatedAt:  time.Date(2026, 8, 5, 9, 0, 0, 0, loc),
		CapturedAt: time.Date(2026, 8, 5, 9, 1, 0, 0, loc),
	}
}

func testScanWindow(t *testing.T) domain.ScanWindow {
	t.Helper()

	loc := testLocation()
	window, err := domain.NewScanWindow(
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 0, 0, 0, 0, loc),
		time.Date(2026, 8, 5, 10, 30, 0, 0, loc),
	)
	if err != nil {
		t.Fatalf("NewScanWindow setup failed: %v", err)
	}
	return window
}

func testLocation() *time.Location {
	return time.FixedZone(domain.RequiredTimezone, 7*60*60)
}

func assertResult(t *testing.T, got Result, want Result) {
	t.Helper()

	if got.Decision != want.Decision {
		t.Fatalf("Decision = %q, want %q", got.Decision, want.Decision)
	}
	if !reflect.DeepEqual(got.Reasons, want.Reasons) {
		t.Fatalf("Reasons = %#v, want %#v", got.Reasons, want.Reasons)
	}
}
