package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestPreparedSnapshotCollectorMapsValidPayloadExactly(t *testing.T) {
	request := preparedSnapshotRequest(t)
	payload := preparedSnapshotPayload(t,
		preparedSnapshotPost{
			PostID:  "post-001",
			PostURL: "https://www.facebook.com/groups/source/posts/post-001",
			Author: preparedSnapshotAuthor{
				FacebookUserID:      " user-001 ",
				CanonicalProfileURL: " profile-path ",
				Username:            " buyer.one ",
				DisplayName:         " Nguyễn Văn A ",
			},
			Body:      "  Cần mua MacBook Pro M2 tại HCM.  ",
			CreatedAt: "2026-08-12T09:15:00.123456789+07:00",
		},
		preparedSnapshotPost{
			PostID:    "post-002",
			Author:    preparedSnapshotAuthor{DisplayName: "Buyer Two"},
			Body:      "Cần mua MacBook Air tại Hà Nội.",
			CreatedAt: "2026-08-12T09:30:00+07:00",
		},
	)

	result, err := NewPreparedSnapshotCollector(payload).CollectGroupPosts(context.Background(), request)
	if err != nil {
		t.Fatalf("CollectGroupPosts() error = %v", err)
	}
	if result.WatchedGroupID != request.WatchedGroup.ID() {
		t.Fatalf("WatchedGroupID = %q, want %q", result.WatchedGroupID, request.WatchedGroup.ID())
	}
	posts := result.OrderedPosts()
	if got := []string{posts[0].PostID, posts[1].PostID}; !reflect.DeepEqual(got, []string{"post-001", "post-002"}) {
		t.Fatalf("post order = %#v", got)
	}
	if posts[0].GroupID != request.WatchedGroup.ID() || posts[0].GroupName != request.WatchedGroup.Name() {
		t.Fatalf("post group = %q/%q", posts[0].GroupID, posts[0].GroupName)
	}
	if !posts[0].CapturedAt.Equal(request.ScanWindow.ScanStarted()) {
		t.Fatalf("CapturedAt = %v, want %v", posts[0].CapturedAt, request.ScanWindow.ScanStarted())
	}
	if posts[0].Body != "  Cần mua MacBook Pro M2 tại HCM.  " {
		t.Fatalf("Body = %q", posts[0].Body)
	}
	wantAuthor := domain.AuthorIdentity{
		FacebookUserID:      " user-001 ",
		CanonicalProfileURL: " profile-path ",
		Username:            " buyer.one ",
		DisplayName:         " Nguyễn Văn A ",
	}
	if posts[0].Author != wantAuthor {
		t.Fatalf("Author = %#v, want %#v", posts[0].Author, wantAuthor)
	}
	if posts[1].PostURL != "" || posts[1].Author.DisplayName != "Buyer Two" || posts[1].Author.FacebookUserID != "" {
		t.Fatalf("optional/display-only values changed: %#v", posts[1])
	}
}

func TestPreparedSnapshotCollectorStrictJSONFailures(t *testing.T) {
	request := preparedSnapshotRequest(t)
	tests := []struct {
		name    string
		payload string
		want    error
	}{
		{name: "empty", payload: " \n\t", want: ErrPreparedSnapshotEmptyPayload},
		{name: "invalid UTF-8", payload: string([]byte{'{', '"', 0xff, '"', ':', '1', '}'}), want: ErrPreparedSnapshotMalformedJSON},
		{name: "malformed", payload: `{"schema_version":1,"posts":[`, want: ErrPreparedSnapshotMalformedJSON},
		{name: "null", payload: `null`, want: ErrPreparedSnapshotMalformedJSON},
		{name: "unsupported schema", payload: `{"schema_version":2,"posts":[]}`, want: ErrPreparedSnapshotUnsupportedSchema},
		{name: "wrong schema type", payload: `{"schema_version":"1","posts":[]}`, want: ErrPreparedSnapshotMalformedJSON},
		{name: "unknown top level", payload: `{"schema_version":1,"posts":[],"group_id":"other"}`, want: ErrPreparedSnapshotUnknownField},
		{name: "unknown post", payload: `{"schema_version":1,"posts":[{"post_id":"1","group_id":"other","author":{"display_name":"A"},"body":"x","created_at":"2026-08-12T09:00:00+07:00"}]}`, want: ErrPreparedSnapshotUnknownField},
		{name: "unknown author", payload: `{"schema_version":1,"posts":[{"author":{"display_name":"A","email":"private"},"body":"x","created_at":"2026-08-12T09:00:00+07:00"}]}`, want: ErrPreparedSnapshotUnknownField},
		{name: "trailing object", payload: `{"schema_version":1,"posts":[]} {}`, want: ErrPreparedSnapshotTrailingContent},
		{name: "trailing scalar", payload: `{"schema_version":1,"posts":[]} true`, want: ErrPreparedSnapshotTrailingContent},
		{name: "duplicate top key", payload: `{"schema_version":1,"schema_version":1,"posts":[]}`, want: ErrPreparedSnapshotDuplicateKey},
		{name: "duplicate nested key", payload: `{"schema_version":1,"posts":[{"author":{"display_name":"A","display_name":"B"},"body":"x","created_at":"2026-08-12T09:00:00+07:00"}]}`, want: ErrPreparedSnapshotDuplicateKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewPreparedSnapshotCollector([]byte(tt.payload)).CollectGroupPosts(context.Background(), request)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CollectGroupPosts() error = %v, want %v", err, tt.want)
			}
			if !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
				t.Fatalf("result = %#v, want zero", result)
			}
		})
	}
}

func TestPreparedSnapshotCollectorPayloadByteBoundary(t *testing.T) {
	request := preparedSnapshotRequest(t)
	exact := preparedSnapshotExactMaxPayload(t)
	if len(exact) != PreparedSnapshotMaxPayloadBytes {
		t.Fatalf("exact payload bytes = %d", len(exact))
	}
	if _, err := NewPreparedSnapshotCollector(exact).CollectGroupPosts(context.Background(), request); err != nil {
		t.Fatalf("exact max CollectGroupPosts() error = %v", err)
	}

	oversized := append(append([]byte(nil), exact...), ' ')
	result, err := NewPreparedSnapshotCollector(oversized).CollectGroupPosts(context.Background(), request)
	if !errors.Is(err, ErrPreparedSnapshotOversizedPayload) {
		t.Fatalf("oversized error = %v", err)
	}
	if !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
		t.Fatalf("oversized result = %#v", result)
	}
}

func TestPreparedSnapshotCollectorPostCountBounds(t *testing.T) {
	request := preparedSnapshotRequest(t)
	zero := preparedSnapshotPayload(t)
	if _, err := NewPreparedSnapshotCollector(zero).CollectGroupPosts(context.Background(), request); !errors.Is(err, ErrPreparedSnapshotInvalidPostCount) {
		t.Fatalf("zero posts error = %v", err)
	}

	posts := make([]preparedSnapshotPost, PreparedSnapshotMaxPosts)
	for index := range posts {
		posts[index] = validPreparedSnapshotPost()
		posts[index].PostID = "post-" + strings.Repeat("x", index%10)
	}
	result, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, posts...)).CollectGroupPosts(context.Background(), request)
	if err != nil || len(result.Posts) != PreparedSnapshotMaxPosts {
		t.Fatalf("100 posts result/error = %d/%v", len(result.Posts), err)
	}

	posts = append(posts, validPreparedSnapshotPost())
	if _, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, posts...)).CollectGroupPosts(context.Background(), request); !errors.Is(err, ErrPreparedSnapshotInvalidPostCount) {
		t.Fatalf("101 posts error = %v", err)
	}
}

func TestPreparedSnapshotCollectorPerFieldByteBounds(t *testing.T) {
	request := preparedSnapshotRequest(t)
	tests := []struct {
		name   string
		limit  int
		set    func(*preparedSnapshotPost, string)
		prefix string
	}{
		{name: "body", limit: PreparedSnapshotMaxBodyBytes, set: func(p *preparedSnapshotPost, value string) { p.Body = value }},
		{name: "display name", limit: PreparedSnapshotMaxDisplayNameBytes, set: func(p *preparedSnapshotPost, value string) { p.Author = preparedSnapshotAuthor{DisplayName: value} }},
		{name: "username", limit: PreparedSnapshotMaxUsernameBytes, set: func(p *preparedSnapshotPost, value string) { p.Author = preparedSnapshotAuthor{Username: value} }},
		{name: "facebook ID", limit: PreparedSnapshotMaxFacebookIDBytes, set: func(p *preparedSnapshotPost, value string) { p.Author = preparedSnapshotAuthor{FacebookUserID: value} }},
		{name: "profile URL", limit: PreparedSnapshotMaxURLBytes, prefix: "profile:", set: func(p *preparedSnapshotPost, value string) {
			p.Author = preparedSnapshotAuthor{CanonicalProfileURL: value}
		}},
		{name: "post URL", limit: PreparedSnapshotMaxURLBytes, prefix: "https://example.com/", set: func(p *preparedSnapshotPost, value string) { p.PostURL = value }},
		{name: "post ID", limit: PreparedSnapshotMaxPostIDBytes, set: func(p *preparedSnapshotPost, value string) { p.PostID = value }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := validPreparedSnapshotPost()
			exact := tt.prefix + strings.Repeat("x", tt.limit-len(tt.prefix))
			tt.set(&post, exact)
			if _, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, post)).CollectGroupPosts(context.Background(), request); err != nil {
				t.Fatalf("exact boundary error = %v", err)
			}

			tt.set(&post, exact+"x")
			result, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, post)).CollectGroupPosts(context.Background(), request)
			if !errors.Is(err, ErrPreparedSnapshotFieldTooLarge) {
				t.Fatalf("over boundary error = %v", err)
			}
			if !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
				t.Fatalf("over boundary result = %#v", result)
			}
		})
	}
}

func TestPreparedSnapshotCollectorAuthorSemantics(t *testing.T) {
	request := preparedSnapshotRequest(t)
	tests := []struct {
		name   string
		author preparedSnapshotAuthor
		valid  bool
	}{
		{name: "display name only", author: preparedSnapshotAuthor{DisplayName: "Buyer"}, valid: true},
		{name: "stable ID only", author: preparedSnapshotAuthor{FacebookUserID: "12345"}, valid: true},
		{name: "profile only", author: preparedSnapshotAuthor{CanonicalProfileURL: "profile-value"}, valid: true},
		{name: "username only", author: preparedSnapshotAuthor{Username: "buyer"}, valid: true},
		{name: "empty", author: preparedSnapshotAuthor{}, valid: false},
		{name: "whitespace", author: preparedSnapshotAuthor{DisplayName: " \t"}, valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := validPreparedSnapshotPost()
			post.Author = tt.author
			result, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, post)).CollectGroupPosts(context.Background(), request)
			if tt.valid {
				if err != nil || len(result.Posts) != 1 || result.Posts[0].Author != (domain.AuthorIdentity{
					FacebookUserID:      tt.author.FacebookUserID,
					CanonicalProfileURL: tt.author.CanonicalProfileURL,
					Username:            tt.author.Username,
					DisplayName:         tt.author.DisplayName,
				}) {
					t.Fatalf("valid author result/error = %#v/%v", result, err)
				}
				return
			}
			if !errors.Is(err, ErrPreparedSnapshotMissingAuthor) || !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
				t.Fatalf("invalid author result/error = %#v/%v", result, err)
			}
		})
	}
}

func TestPreparedSnapshotCollectorCreatedAtRequiresExactHCMOffset(t *testing.T) {
	request := preparedSnapshotRequest(t)
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "RFC3339", value: "2026-08-12T09:00:00+07:00", valid: true},
		{name: "RFC3339Nano", value: "2026-08-12T09:00:00.123456789+07:00", valid: true},
		{name: "UTC", value: "2026-08-12T02:00:00Z"},
		{name: "other offset", value: "2026-08-12T08:00:00+06:00"},
		{name: "timezone less", value: "2026-08-12T09:00:00"},
		{name: "relative", value: "2 hours ago"},
		{name: "malformed", value: "not-a-time"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := validPreparedSnapshotPost()
			post.CreatedAt = tt.value
			result, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, post)).CollectGroupPosts(context.Background(), request)
			if tt.valid {
				if err != nil || len(result.Posts) != 1 {
					t.Fatalf("valid time result/error = %#v/%v", result, err)
				}
				return
			}
			if !errors.Is(err, ErrPreparedSnapshotInvalidCreatedAt) || !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
				t.Fatalf("invalid time result/error = %#v/%v", result, err)
			}
		})
	}
}

func TestPreparedSnapshotCollectorReusesPhase10AValidationAllOrNothing(t *testing.T) {
	request := preparedSnapshotRequest(t)
	tests := []struct {
		name   string
		mutate func(*preparedSnapshotPost)
		want   error
	}{
		{name: "empty body", mutate: func(post *preparedSnapshotPost) { post.Body = " \t" }, want: ErrMissingPreparedPostBody},
		{name: "invalid post URL", mutate: func(post *preparedSnapshotPost) { post.PostURL = "http://example.com/post" }, want: ErrInvalidPreparedPostURL},
		{name: "URL with user info", mutate: func(post *preparedSnapshotPost) { post.PostURL = "https://user@example.com/post" }, want: ErrInvalidPreparedPostURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := validPreparedSnapshotPost()
			second := validPreparedSnapshotPost()
			second.PostID = "post-invalid"
			tt.mutate(&second)
			result, err := NewPreparedSnapshotCollector(preparedSnapshotPayload(t, first, second)).CollectGroupPosts(context.Background(), request)
			if !errors.Is(err, ErrPreparedSnapshotExtractionFailed) || !errors.Is(err, tt.want) {
				t.Fatalf("CollectGroupPosts() error = %v, want extraction and %v", err, tt.want)
			}
			if !reflect.DeepEqual(result, application.GroupCollectionResult{}) {
				t.Fatalf("partial result = %#v", result)
			}
		})
	}
}

func TestPreparedSnapshotCollectorDefensiveCopyAndDeterminism(t *testing.T) {
	request := preparedSnapshotRequest(t)
	original := preparedSnapshotPayload(t, validPreparedSnapshotPost())
	wantPayload := append([]byte(nil), original...)
	collector := NewPreparedSnapshotCollector(original)
	for index := range original {
		original[index] = 'x'
	}

	first, err := collector.CollectGroupPosts(context.Background(), request)
	if err != nil {
		t.Fatalf("first CollectGroupPosts() error = %v; original payload was %q", err, wantPayload)
	}
	second, err := collector.CollectGroupPosts(context.Background(), request)
	if err != nil {
		t.Fatalf("second CollectGroupPosts() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated results differ:\nfirst=%#v\nsecond=%#v", first, second)
	}
	first.Posts[0].Body = "mutated result"
	third, err := collector.CollectGroupPosts(context.Background(), request)
	if err != nil || third.Posts[0].Body == "mutated result" {
		t.Fatalf("collector result was not defensive: %#v/%v", third, err)
	}
}

func TestPreparedSnapshotCollectorRejectsInvalidRequestAndCancellation(t *testing.T) {
	request := preparedSnapshotRequest(t)
	payload := preparedSnapshotPayload(t, validPreparedSnapshotPost())
	request.WatchedGroup = request.WatchedGroup.WithActive(false)
	if _, err := NewPreparedSnapshotCollector(payload).CollectGroupPosts(context.Background(), request); !errors.Is(err, ErrPreparedSnapshotInvalidRequest) {
		t.Fatalf("inactive request error = %v", err)
	}

	request = preparedSnapshotRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewPreparedSnapshotCollector(payload).CollectGroupPosts(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context error = %v", err)
	}
}

func TestPreparedSnapshotCollectorSourceBoundaries(t *testing.T) {
	source, err := os.ReadFile("prepared_snapshot_collector.go")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := bytes.Count(source, []byte("ExtractPreparedPage(snapshot)")); got != 1 {
		t.Fatalf("ExtractPreparedPage(snapshot) call count = %d, want 1", got)
	}
	forbidden := []string{
		"time.Now(",
		"crypto/rand",
		"math/rand",
		"net/http",
		"net.Dial",
		"os.Open",
		"os.ReadFile",
		"os.WriteFile",
		"os.Stdin",
		"bufio.NewReader",
		"io.ReadAll",
		"exec.Command",
		"uuid",
		"Safari",
		"AppleScript",
		"WebKit",
		"Accessibility",
		"NSPasteboard",
		"clipboard",
		"database/sql",
		"sqlite",
		"persistence",
		"bridge",
		"cursor",
		"SelectNextFive",
		"Advance",
		"RunScanBatch",
		"go func",
		"sync.",
		"retry",
		"scheduler",
	}
	for _, fragment := range forbidden {
		if bytes.Contains(source, []byte(fragment)) {
			t.Errorf("prepared_snapshot_collector.go contains forbidden behavior %q", fragment)
		}
	}
}

func preparedSnapshotRequest(t *testing.T) application.GroupCollectionRequest {
	t.Helper()
	location, err := time.LoadLocation(domain.RequiredTimezone)
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	createdAt := time.Date(2026, time.August, 12, 8, 0, 0, 0, location)
	group, err := domain.NewWatchedGroup(
		"group-authoritative",
		"facebook-group-authoritative",
		"https://www.facebook.com/groups/authoritative",
		"Authoritative Group Name",
		createdAt,
	)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	startOfDay := time.Date(2026, time.August, 12, 0, 0, 0, 0, location)
	scanStarted := time.Date(2026, time.August, 12, 10, 0, 0, 123, location)
	window, err := domain.NewScanWindow(startOfDay, startOfDay, scanStarted)
	if err != nil {
		t.Fatalf("NewScanWindow() error = %v", err)
	}
	return application.GroupCollectionRequest{WatchedGroup: group, ScanWindow: window}
}

func validPreparedSnapshotPost() preparedSnapshotPost {
	return preparedSnapshotPost{
		PostID:    "post-001",
		PostURL:   "https://www.facebook.com/groups/source/posts/post-001",
		Author:    preparedSnapshotAuthor{DisplayName: "Buyer One"},
		Body:      "Cần mua MacBook Pro tại HCM.",
		CreatedAt: "2026-08-12T09:00:00+07:00",
	}
}

func preparedSnapshotPayload(t *testing.T, posts ...preparedSnapshotPost) []byte {
	t.Helper()
	payload, err := json.Marshal(preparedSnapshotDTO{SchemaVersion: PreparedSnapshotSchemaVersion, Posts: posts})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return payload
}

func preparedSnapshotExactMaxPayload(t *testing.T) []byte {
	t.Helper()
	posts := make([]preparedSnapshotPost, 16)
	for index := range posts {
		posts[index] = validPreparedSnapshotPost()
		posts[index].PostID = ""
		posts[index].PostURL = ""
		posts[index].Body = strings.Repeat("x", PreparedSnapshotMaxBodyBytes)
	}
	payload := preparedSnapshotPayload(t, posts...)
	over := len(payload) - PreparedSnapshotMaxPayloadBytes
	if over < 0 || over >= len(posts[len(posts)-1].Body) {
		t.Fatalf("cannot construct exact max payload: initial=%d over=%d", len(payload), over)
	}
	posts[len(posts)-1].Body = posts[len(posts)-1].Body[:len(posts[len(posts)-1].Body)-over]
	payload = preparedSnapshotPayload(t, posts...)
	if len(payload) != PreparedSnapshotMaxPayloadBytes {
		t.Fatalf("adjusted payload bytes = %d", len(payload))
	}
	return payload
}
