package dedup

import (
	"reflect"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestAggregatePostsBasicAggregation(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	posts := []domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
		sourcePost("post-2", "group-2", "https://example.test/posts/2", authorWithUserID("user-1"), "  cần   mua   macbook pro  "),
		sourcePost("post-3", "group-1", "https://example.test/posts/3", authorWithUserID("user-1"), "cần mua macbook pro"),
	}

	result := AggregatePosts(posts, profile)
	leads := result.Leads()
	if len(leads) != 1 {
		t.Fatalf("len(Leads) = %d, want 1: %#v", len(leads), leads)
	}
	if leads[0].SourceCount() != 3 {
		t.Fatalf("SourceCount = %d, want 3", leads[0].SourceCount())
	}
	assertSourceOrder(t, leads[0], []string{"post-1", "post-2", "post-3"})
	if len(result.Unaggregated()) != 0 || len(result.Conflicts()) != 0 {
		t.Fatalf("unexpected non-lead output: unaggregated=%#v conflicts=%#v", result.Unaggregated(), result.Conflicts())
	}

	again := AggregatePosts(posts, profile)
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("AggregatePosts repeated result = %#v, want %#v", again, result)
	}
}

func TestAggregatePostsLeadOrderAndSeparation(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	posts := []domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
		sourcePost("post-2", "group-2", "https://example.test/posts/2", authorWithUserID("user-2"), "Cần mua MacBook Pro"),
		sourcePost("post-3", "group-3", "https://example.test/posts/3", authorWithUserID("user-1"), "Cần mua MacBook Air"),
		sourcePost("post-4", "group-4", "https://example.test/posts/4", authorWithUserID("user-1"), "cần mua macbook pro"),
	}

	result := AggregatePosts(posts, profile)
	leads := result.Leads()
	if len(leads) != 3 {
		t.Fatalf("len(Leads) = %d, want 3", len(leads))
	}
	assertSourceOrder(t, leads[0], []string{"post-1", "post-4"})
	assertSourceOrder(t, leads[1], []string{"post-2"})
	assertSourceOrder(t, leads[2], []string{"post-3"})
	if leads[0].Key.Value == leads[1].Key.Value || leads[0].Key.Value == leads[2].Key.Value {
		t.Fatalf("separate leads share identity: %#v", leads)
	}
}

func TestAggregatePostsSourcePreservationAndImmutability(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	createdAt := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC)
	capturedAt := time.Date(2026, 8, 5, 9, 31, 0, 0, time.UTC)
	post := sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro")
	post.CreatedAt = createdAt
	post.CapturedAt = capturedAt
	posts := []domain.RawPost{post}
	originalPosts := append([]domain.RawPost(nil), posts...)

	result := AggregatePosts(posts, profile)
	sources := result.Leads()[0].Sources()
	if len(sources) != 1 {
		t.Fatalf("len(Sources) = %d, want 1", len(sources))
	}
	gotPost := sources[0].Post
	if gotPost.PostID != post.PostID || gotPost.GroupID != post.GroupID || gotPost.PostURL != post.PostURL || gotPost.Body != post.Body || gotPost.Author != post.Author {
		t.Fatalf("source post not preserved:\ngot  %#v\nwant %#v", gotPost, post)
	}
	if !gotPost.CreatedAt.Equal(createdAt) || !gotPost.CapturedAt.Equal(capturedAt) {
		t.Fatalf("timestamps not preserved: %#v", gotPost)
	}
	if !reflect.DeepEqual(posts, originalPosts) {
		t.Fatalf("AggregatePosts mutated input slice/posts:\ngot  %#v\nwant %#v", posts, originalPosts)
	}

	sources[0].Post.Body = "changed"
	sourcesAgain := result.Leads()[0].Sources()
	if sourcesAgain[0].Post.Body != post.Body {
		t.Fatalf("returned sources alias internal state: got %q, want %q", sourcesAgain[0].Post.Body, post.Body)
	}
}

func TestAggregatePostsUnaggregated(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	posts := []domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", domain.AuthorIdentity{DisplayName: "Buyer One"}, "Cần mua MacBook Pro"),
		sourcePost("post-2", "group-2", "https://example.test/posts/2", authorWithUserID("user-1"), "Cần mua iPhone"),
		sourcePost("post-3", "group-3", "https://example.test/posts/3", authorWithUserID("user-1"), "MacBook Pro"),
		sourcePost("post-4", "group-4", "https://example.test/posts/4", authorWithUserID("user-1"), ""),
		sourcePost("", "group-5", "", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
	}

	result := AggregatePosts(posts, profile)
	if len(result.Leads()) != 0 {
		t.Fatalf("len(Leads) = %d, want 0", len(result.Leads()))
	}
	unaggregated := result.Unaggregated()
	if len(unaggregated) != 5 {
		t.Fatalf("len(Unaggregated) = %d, want 5", len(unaggregated))
	}
	assertReasons(t, unaggregated[0].Reasons, []ReasonCode{ReasonStableAuthorIdentityMissing})
	assertReasons(t, unaggregated[1].Reasons, []ReasonCode{ReasonProductEvidenceMissing})
	assertReasons(t, unaggregated[2].Reasons, []ReasonCode{ReasonBuyerIntentEvidenceMissing})
	assertReasons(t, unaggregated[3].Reasons, []ReasonCode{ReasonProductEvidenceMissing, ReasonBuyerIntentEvidenceMissing})
	assertReasons(t, unaggregated[4].Reasons, []ReasonCode{ReasonSourceIdentityMissing})
}

func TestAggregatePostsDuplicateSourceOccurrences(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	byPostID := []domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
		sourcePost(" post-1 ", "group-1", "https://example.test/posts/1-copy", authorWithUserID("user-1"), "cần mua macbook pro"),
	}
	byURL := []domain.RawPost{
		sourcePost("", "group-1", "https://example.test/posts/url-only", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
		sourcePost("", "group-2", " https://example.test/posts/url-only ", authorWithUserID("user-1"), "cần mua macbook pro"),
	}

	for _, posts := range [][]domain.RawPost{byPostID, byURL} {
		result := AggregatePosts(posts, profile)
		leads := result.Leads()
		if len(leads) != 1 || leads[0].SourceCount() != 1 {
			t.Fatalf("duplicate source was not ignored once: leads=%#v", leads)
		}
		if len(result.Conflicts()) != 0 {
			t.Fatalf("consistent duplicate source produced conflicts: %#v", result.Conflicts())
		}
		again := AggregatePosts(posts, profile)
		if !reflect.DeepEqual(again, result) {
			t.Fatalf("duplicate source repeated result = %#v, want %#v", again, result)
		}
	}
}

func TestAggregatePostsSourceConflicts(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	tests := []struct {
		name  string
		posts []domain.RawPost
		want  []ReasonCode
	}{
		{
			name: "same PostID with different stable authors",
			posts: []domain.RawPost{
				sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
				sourcePost("post-1", "group-2", "https://example.test/posts/1", authorWithUserID("user-2"), "Cần mua MacBook Pro"),
			},
			want: []ReasonCode{ReasonSourceIdentityConflict, ReasonStableAuthorIdentityDiffers},
		},
		{
			name: "same PostID with materially different body",
			posts: []domain.RawPost{
				sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
				sourcePost("post-1", "group-2", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Air"),
			},
			want: []ReasonCode{ReasonSourceIdentityConflict, ReasonNormalizedNeedDiffers},
		},
		{
			name: "same source identity with different deterministic need identity",
			posts: []domain.RawPost{
				sourcePost("", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro 16 inch"),
				sourcePost("", "group-2", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Air"),
			},
			want: []ReasonCode{ReasonSourceIdentityConflict, ReasonNormalizedNeedDiffers},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregatePosts(tt.posts, profile)
			if len(result.Leads()) != 1 || result.Leads()[0].SourceCount() != 1 {
				t.Fatalf("conflict should preserve prior source only: %#v", result.Leads())
			}
			conflicts := result.Conflicts()
			if len(conflicts) != 1 {
				t.Fatalf("len(Conflicts) = %d, want 1: %#v", len(conflicts), conflicts)
			}
			assertReasons(t, conflicts[0].Reasons, tt.want)
			if conflicts[0].ExistingSource.Post.Body != tt.posts[0].Body || conflicts[0].Post.Body != tt.posts[1].Body {
				t.Fatalf("conflict did not preserve posts: %#v", conflicts[0])
			}
		})
	}
}

func TestAggregatePostsProfileBehaviorAndEdgeCases(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	otherProfile := mustSearchProfile(t, "other-profile", []string{"MacBook"}, []string{"cần mua"}, nil)

	if got := AggregatePosts(nil, profile); len(got.Leads()) != 0 || len(got.Unaggregated()) != 0 || len(got.Conflicts()) != 0 {
		t.Fatalf("nil input result = %#v, want empty", got)
	}
	if got := AggregatePosts([]domain.RawPost{}, profile); len(got.Leads()) != 0 || len(got.Unaggregated()) != 0 || len(got.Conflicts()) != 0 {
		t.Fatalf("empty input result = %#v, want empty", got)
	}

	inactive := AggregatePosts([]domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
	}, domain.SearchProfile{})
	if len(inactive.Leads()) != 0 || len(inactive.Unaggregated()) != 1 {
		t.Fatalf("empty profile result = %#v, want one unaggregated post", inactive)
	}
	assertReasons(t, inactive.Unaggregated()[0].Reasons, []ReasonCode{ReasonProductEvidenceMissing, ReasonBuyerIntentEvidenceMissing})

	validWithoutGroupOrURL := sourcePost("post-1", "", "", authorWithUserID("user-1"), "Cần mua MacBook Pro")
	result := AggregatePosts([]domain.RawPost{validWithoutGroupOrURL}, profile)
	if len(result.Leads()) != 1 || result.Leads()[0].SourceCount() != 1 {
		t.Fatalf("PostID-backed source with missing group/url should aggregate: %#v", result)
	}

	left := AggregatePosts([]domain.RawPost{sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro")}, profile)
	right := AggregatePosts([]domain.RawPost{sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro")}, otherProfile)
	if left.Leads()[0].Key.Value == right.Leads()[0].Key.Value {
		t.Fatalf("different profile identities mixed lead keys: left=%#v right=%#v", left.Leads()[0].Key, right.Leads()[0].Key)
	}
}

func TestAggregatePostsVietnameseDiacriticsRemainPreserved(t *testing.T) {
	profile := domain.MacBookSearchProfile()
	posts := []domain.RawPost{
		sourcePost("post-1", "group-1", "https://example.test/posts/1", authorWithUserID("user-1"), "Cần mua MacBook Pro"),
		sourcePost("post-2", "group-2", "https://example.test/posts/2", authorWithUserID("user-1"), "Can mua MacBook Pro"),
	}

	result := AggregatePosts(posts, profile)
	if len(result.Leads()) != 2 {
		t.Fatalf("len(Leads) = %d, want 2", len(result.Leads()))
	}
	if result.Leads()[0].Key.Value == result.Leads()[1].Key.Value {
		t.Fatalf("diacritic-preserving need identities collapsed: %#v", result.Leads())
	}
	if len(result.Unaggregated()) != 0 {
		t.Fatalf("len(Unaggregated) = %d, want 0", len(result.Unaggregated()))
	}
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

func assertSourceOrder(t *testing.T, lead Lead, wantPostIDs []string) {
	t.Helper()

	sources := lead.Sources()
	if len(sources) != len(wantPostIDs) {
		t.Fatalf("len(Sources) = %d, want %d", len(sources), len(wantPostIDs))
	}
	for i, want := range wantPostIDs {
		if sources[i].Post.PostID != want {
			t.Fatalf("source[%d].PostID = %q, want %q", i, sources[i].Post.PostID, want)
		}
	}
}

func assertReasons(t *testing.T, got []ReasonCode, want []ReasonCode) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reasons = %#v, want %#v", got, want)
	}
}
