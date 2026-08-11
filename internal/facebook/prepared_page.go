package facebook

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

const PreparedPageSnapshotSchemaVersion = 1

var (
	ErrEmptyPreparedPageSnapshot              = errors.New("facebook: prepared page snapshot is empty")
	ErrUnsupportedPreparedPageSnapshotVersion = errors.New("facebook: prepared page snapshot version is unsupported")
	ErrInvalidPreparedPageGroupIdentity       = errors.New("facebook: prepared page watched group identity is invalid")
	ErrInvalidPreparedPageCapturedAt          = errors.New("facebook: prepared page captured at is invalid")
	ErrMissingPreparedPostBody                = errors.New("facebook: prepared post body is missing")
	ErrInvalidPreparedPostAuthor              = errors.New("facebook: prepared post author is invalid")
	ErrInvalidPreparedPostCreatedAt           = errors.New("facebook: prepared post created at is invalid")
	ErrInvalidPreparedPostURL                 = errors.New("facebook: prepared post url is invalid")
	ErrPreparedPostGroupConflict              = errors.New("facebook: prepared post group identity conflicts with watched group")
)

// PreparedPageSnapshot is caller-supplied, pre-normalized fixture content for one watched group.
type PreparedPageSnapshot struct {
	SchemaVersion    int
	WatchedGroupID   string
	WatchedGroupName string
	CapturedAt       time.Time
	Posts            []PreparedPost
}

// PreparedPost contains explicit source values from one prepared fixture record.
type PreparedPost struct {
	PostID    string
	GroupID   string
	PostURL   string
	Author    domain.AuthorIdentity
	Body      string
	CreatedAt string
}

// ExtractPreparedPage maps one typed local snapshot to RawPost values without browser access.
func ExtractPreparedPage(snapshot PreparedPageSnapshot) ([]domain.RawPost, error) {
	if len(snapshot.Posts) == 0 {
		return nil, ErrEmptyPreparedPageSnapshot
	}
	if snapshot.SchemaVersion != PreparedPageSnapshotSchemaVersion {
		return nil, ErrUnsupportedPreparedPageSnapshotVersion
	}
	groupID := strings.TrimSpace(snapshot.WatchedGroupID)
	if groupID == "" || groupID != snapshot.WatchedGroupID {
		return nil, ErrInvalidPreparedPageGroupIdentity
	}
	if snapshot.CapturedAt.IsZero() {
		return nil, ErrInvalidPreparedPageCapturedAt
	}

	posts := make([]domain.RawPost, len(snapshot.Posts))
	for i, prepared := range snapshot.Posts {
		if strings.TrimSpace(prepared.Body) == "" {
			return nil, ErrMissingPreparedPostBody
		}
		if !validPreparedPostAuthor(prepared.Author) {
			return nil, ErrInvalidPreparedPostAuthor
		}
		if prepared.GroupID != "" && prepared.GroupID != groupID {
			return nil, ErrPreparedPostGroupConflict
		}
		createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(prepared.CreatedAt))
		if err != nil {
			return nil, ErrInvalidPreparedPostCreatedAt
		}
		if prepared.PostURL != "" && !validPreparedPostURL(prepared.PostURL) {
			return nil, ErrInvalidPreparedPostURL
		}
		posts[i] = domain.RawPost{
			PostID:     prepared.PostID,
			GroupID:    groupID,
			GroupName:  snapshot.WatchedGroupName,
			PostURL:    prepared.PostURL,
			Author:     prepared.Author,
			Body:       prepared.Body,
			CreatedAt:  createdAt,
			CapturedAt: snapshot.CapturedAt,
		}
	}
	return posts, nil
}

func validPreparedPostAuthor(author domain.AuthorIdentity) bool {
	return strings.TrimSpace(author.FacebookUserID) != "" ||
		strings.TrimSpace(author.CanonicalProfileURL) != "" ||
		strings.TrimSpace(author.Username) != "" ||
		strings.TrimSpace(author.DisplayName) != ""
}

func validPreparedPostURL(value string) bool {
	if value != strings.TrimSpace(value) {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil
}
