package facebook

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestExtractPreparedPageMapsOneRawPostExactly(t *testing.T) {
	snapshot := phase10AValidSnapshot()

	posts, err := ExtractPreparedPage(snapshot)
	if err != nil {
		t.Fatalf("ExtractPreparedPage() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("post count = %d, want 1", len(posts))
	}

	post := posts[0]
	if post.PostID != "post-001" || post.GroupID != "group-001" || post.GroupName != "MacBook Buyers Vietnam" {
		t.Fatalf("post identity mapping = %#v", post)
	}
	if post.PostURL != "https://www.facebook.com/groups/group-001/posts/post-001" {
		t.Fatalf("PostURL = %q", post.PostURL)
	}
	if post.Body != "Cần mua MacBook Pro M2 tại HCM, ngân sách 25 triệu." {
		t.Fatalf("Body = %q", post.Body)
	}
	if post.Author != (domain.AuthorIdentity{
		FacebookUserID:      "user-001",
		CanonicalProfileURL: "https://www.facebook.com/profile.php?id=user-001",
		Username:            "nguyen.van.a",
		DisplayName:         "Nguyễn Văn A",
	}) {
		t.Fatalf("Author = %#v", post.Author)
	}
	wantCreatedAt, err := time.Parse(time.RFC3339Nano, "2026-08-11T09:15:00+07:00")
	if err != nil {
		t.Fatalf("parse expected CreatedAt: %v", err)
	}
	if !post.CreatedAt.Equal(wantCreatedAt) {
		t.Fatalf("CreatedAt = %v, want %v", post.CreatedAt, wantCreatedAt)
	}
	if !post.CapturedAt.Equal(snapshot.CapturedAt) {
		t.Fatalf("CapturedAt = %v, want caller value %v", post.CapturedAt, snapshot.CapturedAt)
	}
}

func TestExtractPreparedPagePreservesOrderAndDisplayOnlyAuthorWithoutMutation(t *testing.T) {
	snapshot := phase10AValidSnapshot()
	snapshot.Posts = append(snapshot.Posts, PreparedPost{
		PostID:    "post-002",
		GroupID:   "group-001",
		PostURL:   "https://www.facebook.com/groups/group-001/posts/post-002",
		Author:    domain.AuthorIdentity{DisplayName: "Trần Thị B"},
		Body:      "Mình cần mua MacBook Air, ưu tiên máy còn bảo hành.",
		CreatedAt: "2026-08-11T09:20:00+07:00",
	})
	beforePosts := append([]PreparedPost(nil), snapshot.Posts...)

	first, err := ExtractPreparedPage(snapshot)
	if err != nil {
		t.Fatalf("first ExtractPreparedPage() error = %v", err)
	}
	second, err := ExtractPreparedPage(snapshot)
	if err != nil {
		t.Fatalf("second ExtractPreparedPage() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated extraction changed: first=%#v second=%#v", first, second)
	}
	if len(first) != 2 || first[0].PostID != "post-001" || first[1].PostID != "post-002" {
		t.Fatalf("post order = %#v", first)
	}
	if first[1].Body != "Mình cần mua MacBook Air, ưu tiên máy còn bảo hành." {
		t.Fatalf("second body = %q", first[1].Body)
	}
	if first[1].Author != (domain.AuthorIdentity{DisplayName: "Trần Thị B"}) {
		t.Fatalf("display-only author was altered or upgraded: %#v", first[1].Author)
	}
	for i, post := range first {
		if post.GroupID != snapshot.WatchedGroupID || post.GroupName != snapshot.WatchedGroupName {
			t.Fatalf("post[%d] group = (%q, %q)", i, post.GroupID, post.GroupName)
		}
		if !post.CapturedAt.Equal(snapshot.CapturedAt) {
			t.Fatalf("post[%d].CapturedAt = %v, want %v", i, post.CapturedAt, snapshot.CapturedAt)
		}
	}
	if !reflect.DeepEqual(snapshot.Posts, beforePosts) {
		t.Fatalf("input posts mutated: got %#v want %#v", snapshot.Posts, beforePosts)
	}
}

func TestExtractPreparedPageAcceptsAnySuppliedAuthorIdentityFieldExactly(t *testing.T) {
	tests := []struct {
		name   string
		author domain.AuthorIdentity
	}{
		{
			name:   "Facebook user ID only",
			author: domain.AuthorIdentity{FacebookUserID: "user-only-001"},
		},
		{
			name:   "canonical profile URL only",
			author: domain.AuthorIdentity{CanonicalProfileURL: "https://www.facebook.com/profile.php?id=user-only-002"},
		},
		{
			name:   "username only",
			author: domain.AuthorIdentity{Username: "buyer.only"},
		},
		{
			name:   "display name only",
			author: domain.AuthorIdentity{DisplayName: "Người dùng chỉ có tên"},
		},
		{
			name: "stable fields preserved without normalization",
			author: domain.AuthorIdentity{
				FacebookUserID:      " user-raw ",
				CanonicalProfileURL: " profile-key-raw ",
				Username:            " username-raw ",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := phase10AValidSnapshot()
			snapshot.Posts[0].Author = tc.author

			posts, err := ExtractPreparedPage(snapshot)
			if err != nil {
				t.Fatalf("ExtractPreparedPage() error = %v", err)
			}
			if len(posts) != 1 {
				t.Fatalf("post count = %d, want 1", len(posts))
			}
			if posts[0].Author != tc.author {
				t.Fatalf("Author = %#v, want exact %#v", posts[0].Author, tc.author)
			}
		})
	}
}

func TestExtractPreparedPageFailsClosedForMalformedPreparedSnapshots(t *testing.T) {
	tests := []struct {
		name     string
		snapshot func() PreparedPageSnapshot
		wantErr  error
	}{
		{
			name:     "empty snapshot",
			snapshot: func() PreparedPageSnapshot { return PreparedPageSnapshot{} },
			wantErr:  ErrEmptyPreparedPageSnapshot,
		},
		{
			name: "unsupported unrecognized structure",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.SchemaVersion = PreparedPageSnapshotSchemaVersion + 1
				return snapshot
			},
			wantErr: ErrUnsupportedPreparedPageSnapshotVersion,
		},
		{
			name: "missing watched group identity",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.WatchedGroupID = " \t "
				return snapshot
			},
			wantErr: ErrInvalidPreparedPageGroupIdentity,
		},
		{
			name: "non canonical watched group identity",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.WatchedGroupID = " group-001 "
				return snapshot
			},
			wantErr: ErrInvalidPreparedPageGroupIdentity,
		},
		{
			name: "missing captured at",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.CapturedAt = time.Time{}
				return snapshot
			},
			wantErr: ErrInvalidPreparedPageCapturedAt,
		},
		{
			name: "missing critical post body",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].Body = " \n\t "
				return snapshot
			},
			wantErr: ErrMissingPreparedPostBody,
		},
		{
			name: "entirely empty author",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].Author = domain.AuthorIdentity{}
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostAuthor,
		},
		{
			name: "whitespace-only author",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].Author = domain.AuthorIdentity{
					FacebookUserID:      " \t ",
					CanonicalProfileURL: "\n",
					Username:            " ",
					DisplayName:         "\t",
				}
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostAuthor,
		},
		{
			name: "relative created at",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].CreatedAt = "2h ago"
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostCreatedAt,
		},
		{
			name: "created at without timezone",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].CreatedAt = "2026-08-11T09:15:00"
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostCreatedAt,
		},
		{
			name: "relative post URL",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].PostURL = "/groups/group-001/posts/post-001"
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostURL,
		},
		{
			name: "non HTTPS post URL",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].PostURL = "http://www.facebook.com/groups/group-001/posts/post-001"
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostURL,
		},
		{
			name: "post URL without hostname",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].PostURL = "https://:443/post-001"
				return snapshot
			},
			wantErr: ErrInvalidPreparedPostURL,
		},
		{
			name: "embedded group identity conflict",
			snapshot: func() PreparedPageSnapshot {
				snapshot := phase10AValidSnapshot()
				snapshot.Posts[0].GroupID = "different-group"
				return snapshot
			},
			wantErr: ErrPreparedPostGroupConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := tc.snapshot()
			before := snapshot
			before.Posts = append([]PreparedPost(nil), snapshot.Posts...)

			posts, err := ExtractPreparedPage(snapshot)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ExtractPreparedPage() error = %v, want %v", err, tc.wantErr)
			}
			if posts != nil {
				t.Fatalf("malformed snapshot returned posts %#v", posts)
			}
			if !reflect.DeepEqual(snapshot, before) {
				t.Fatalf("malformed snapshot mutated: got %#v want %#v", snapshot, before)
			}
		})
	}
}

func TestExtractPreparedPageDoesNotFabricateUnavailableOptionalFields(t *testing.T) {
	snapshot := phase10AValidSnapshot()
	snapshot.Posts[0].PostID = ""
	snapshot.Posts[0].GroupID = ""
	snapshot.Posts[0].PostURL = ""
	snapshot.Posts[0].Author = domain.AuthorIdentity{DisplayName: "Người dùng mẫu"}

	posts, err := ExtractPreparedPage(snapshot)
	if err != nil {
		t.Fatalf("ExtractPreparedPage() error = %v", err)
	}
	post := posts[0]
	if post.PostID != "" || post.PostURL != "" {
		t.Fatalf("optional fields were fabricated: PostID=%q PostURL=%q", post.PostID, post.PostURL)
	}
	if post.GroupID != snapshot.WatchedGroupID {
		t.Fatalf("GroupID = %q, want caller group %q", post.GroupID, snapshot.WatchedGroupID)
	}
	if post.Author != (domain.AuthorIdentity{DisplayName: "Người dùng mẫu"}) {
		t.Fatalf("display-only author was upgraded: %#v", post.Author)
	}
}

func TestPhase10APreparedPageSourceExcludesDeferredInfrastructure(t *testing.T) {
	body, err := os.ReadFile("prepared_page.go")
	if err != nil {
		t.Fatalf("read prepared_page.go: %v", err)
	}
	forbidden := []string{
		"time.Now(",
		"math/rand",
		"crypto/rand",
		"net/http",
		"net.Dial",
		"localhost",
		"WebSocket",
		"websocket",
		"os/exec",
		"syscall",
		"chromedp",
		"playwright",
		"selenium",
		"WebKit",
		"AppleScript",
		"Accessibility",
		"cookie",
		"credential",
		"session",
		"keychain",
		"RunScanBatch(",
		"StartNextPending(",
		"StartAttempt(",
		"SucceedAttempt(",
		"FailAttempt(",
		"SkipAttempt(",
		"ExpireAtDayBoundary(",
		"database/sql",
		"sqlite",
		"persistence",
		"bridge",
		"Swift",
		"Xcode",
		"go func",
		"sync.",
		"scheduler",
		"retry",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 10A source contains deferred behavior %q", fragment)
		}
	}
}

func phase10AValidSnapshot() PreparedPageSnapshot {
	return PreparedPageSnapshot{
		SchemaVersion:    PreparedPageSnapshotSchemaVersion,
		WatchedGroupID:   "group-001",
		WatchedGroupName: "MacBook Buyers Vietnam",
		CapturedAt:       time.Date(2026, time.August, 11, 9, 30, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)),
		Posts: []PreparedPost{
			{
				PostID:  "post-001",
				GroupID: "group-001",
				PostURL: "https://www.facebook.com/groups/group-001/posts/post-001",
				Author: domain.AuthorIdentity{
					FacebookUserID:      "user-001",
					CanonicalProfileURL: "https://www.facebook.com/profile.php?id=user-001",
					Username:            "nguyen.van.a",
					DisplayName:         "Nguyễn Văn A",
				},
				Body:      "Cần mua MacBook Pro M2 tại HCM, ngân sách 25 triệu.",
				CreatedAt: "2026-08-11T09:15:00+07:00",
			},
		},
	}
}
