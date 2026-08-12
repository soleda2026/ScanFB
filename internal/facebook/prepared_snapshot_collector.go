package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

const (
	PreparedSnapshotSchemaVersion       = 1
	PreparedSnapshotMaxPayloadBytes     = 1_048_576
	PreparedSnapshotMinPosts            = 1
	PreparedSnapshotMaxPosts            = 100
	PreparedSnapshotMaxBodyBytes        = 65_536
	PreparedSnapshotMaxDisplayNameBytes = 512
	PreparedSnapshotMaxUsernameBytes    = 256
	PreparedSnapshotMaxFacebookIDBytes  = 128
	PreparedSnapshotMaxURLBytes         = 4_096
	PreparedSnapshotMaxPostIDBytes      = 256
	preparedSnapshotMaxCreatedAtBytes   = 64
)

var (
	ErrPreparedSnapshotEmptyPayload      = errors.New("facebook: prepared snapshot payload is empty")
	ErrPreparedSnapshotOversizedPayload  = errors.New("facebook: prepared snapshot payload exceeds byte limit")
	ErrPreparedSnapshotMalformedJSON     = errors.New("facebook: prepared snapshot JSON is malformed")
	ErrPreparedSnapshotUnsupportedSchema = errors.New("facebook: prepared snapshot schema is unsupported")
	ErrPreparedSnapshotUnknownField      = errors.New("facebook: prepared snapshot contains an unknown field")
	ErrPreparedSnapshotTrailingContent   = errors.New("facebook: prepared snapshot contains trailing JSON content")
	ErrPreparedSnapshotDuplicateKey      = errors.New("facebook: prepared snapshot contains a duplicate JSON object key")
	ErrPreparedSnapshotInvalidPostCount  = errors.New("facebook: prepared snapshot post count is invalid")
	ErrPreparedSnapshotFieldTooLarge     = errors.New("facebook: prepared snapshot field exceeds byte limit")
	ErrPreparedSnapshotInvalidCreatedAt  = errors.New("facebook: prepared snapshot created_at must be RFC3339 with +07:00 offset")
	ErrPreparedSnapshotMissingAuthor     = errors.New("facebook: prepared snapshot post has no author identity")
	ErrPreparedSnapshotInvalidRequest    = errors.New("facebook: prepared snapshot collection request is invalid")
	ErrPreparedSnapshotExtractionFailed  = errors.New("facebook: prepared snapshot extraction failed")
)

type preparedSnapshotDTO struct {
	SchemaVersion int                    `json:"schema_version"`
	Posts         []preparedSnapshotPost `json:"posts"`
}

type preparedSnapshotPost struct {
	PostID    string                 `json:"post_id"`
	PostURL   string                 `json:"post_url"`
	Author    preparedSnapshotAuthor `json:"author"`
	Body      string                 `json:"body"`
	CreatedAt string                 `json:"created_at"`
}

type preparedSnapshotAuthor struct {
	FacebookUserID      string `json:"facebook_user_id"`
	CanonicalProfileURL string `json:"canonical_profile_url"`
	Username            string `json:"username"`
	DisplayName         string `json:"display_name"`
}

// PreparedSnapshotCollector maps one immutable caller-supplied JSON payload through Phase 10A.
type PreparedSnapshotCollector struct {
	payload         []byte
	payloadTooLarge bool
}

var _ application.GroupPostCollector = (*PreparedSnapshotCollector)(nil)

// NewPreparedSnapshotCollector defensively owns the payload for deterministic collection calls.
func NewPreparedSnapshotCollector(payload []byte) *PreparedSnapshotCollector {
	if len(payload) > PreparedSnapshotMaxPayloadBytes {
		return &PreparedSnapshotCollector{payloadTooLarge: true}
	}
	owned := make([]byte, len(payload))
	copy(owned, payload)
	return &PreparedSnapshotCollector{payload: owned}
}

// CollectGroupPosts validates one bounded payload and returns posts bound to the requested group.
func (c *PreparedSnapshotCollector) CollectGroupPosts(
	ctx context.Context,
	request application.GroupCollectionRequest,
) (application.GroupCollectionResult, error) {
	if ctx == nil || c == nil {
		return application.GroupCollectionResult{}, ErrPreparedSnapshotInvalidRequest
	}
	if err := ctx.Err(); err != nil {
		return application.GroupCollectionResult{}, err
	}
	if c.payloadTooLarge {
		return application.GroupCollectionResult{}, ErrPreparedSnapshotOversizedPayload
	}
	if err := validatePreparedSnapshotRequest(request); err != nil {
		return application.GroupCollectionResult{}, err
	}

	dto, err := decodePreparedSnapshot(c.payload)
	if err != nil {
		return application.GroupCollectionResult{}, err
	}

	snapshot, err := dto.preparedPageSnapshot(request)
	if err != nil {
		return application.GroupCollectionResult{}, err
	}
	posts, err := ExtractPreparedPage(snapshot)
	if err != nil {
		return application.GroupCollectionResult{}, fmt.Errorf("%w: %w", ErrPreparedSnapshotExtractionFailed, err)
	}

	return application.GroupCollectionResult{
		WatchedGroupID: request.WatchedGroup.ID(),
		Posts:          posts,
	}, nil
}

func validatePreparedSnapshotRequest(request application.GroupCollectionRequest) error {
	if err := request.WatchedGroup.Validate(); err != nil || !request.WatchedGroup.IsActive() {
		return ErrPreparedSnapshotInvalidRequest
	}
	if request.ScanWindow.Timezone() != domain.RequiredTimezone || request.ScanWindow.ScanStarted().IsZero() {
		return ErrPreparedSnapshotInvalidRequest
	}
	return nil
}

func decodePreparedSnapshot(payload []byte) (preparedSnapshotDTO, error) {
	if len(payload) > PreparedSnapshotMaxPayloadBytes {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotOversizedPayload
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotEmptyPayload
	}
	if !utf8.Valid(payload) {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotMalformedJSON
	}
	if err := validatePreparedSnapshotJSONStructure(payload); err != nil {
		return preparedSnapshotDTO{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var decoded *preparedSnapshotDTO
	if err := decoder.Decode(&decoded); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			return preparedSnapshotDTO{}, fmt.Errorf("%w: %v", ErrPreparedSnapshotUnknownField, err)
		}
		return preparedSnapshotDTO{}, fmt.Errorf("%w: %v", ErrPreparedSnapshotMalformedJSON, err)
	}
	if decoded == nil {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotMalformedJSON
	}
	if decoded.SchemaVersion != PreparedSnapshotSchemaVersion {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotUnsupportedSchema
	}
	if len(decoded.Posts) < PreparedSnapshotMinPosts || len(decoded.Posts) > PreparedSnapshotMaxPosts {
		return preparedSnapshotDTO{}, ErrPreparedSnapshotInvalidPostCount
	}
	return *decoded, nil
}

func validatePreparedSnapshotJSONStructure(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := walkPreparedSnapshotJSONValue(decoder); err != nil {
		if errors.Is(err, ErrPreparedSnapshotDuplicateKey) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrPreparedSnapshotMalformedJSON, err)
	}
	if _, err := decoder.Token(); err == io.EOF {
		return nil
	}
	return ErrPreparedSnapshotTrailingContent
}

func walkPreparedSnapshotJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return ErrPreparedSnapshotMalformedJSON
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("%w: %q", ErrPreparedSnapshotDuplicateKey, key)
			}
			seen[key] = struct{}{}
			if err := walkPreparedSnapshotJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return ErrPreparedSnapshotMalformedJSON
		}
	case '[':
		for decoder.More() {
			if err := walkPreparedSnapshotJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return ErrPreparedSnapshotMalformedJSON
		}
	default:
		return ErrPreparedSnapshotMalformedJSON
	}
	return nil
}

func (dto preparedSnapshotDTO) preparedPageSnapshot(
	request application.GroupCollectionRequest,
) (PreparedPageSnapshot, error) {
	preparedPosts := make([]PreparedPost, len(dto.Posts))
	for index, post := range dto.Posts {
		if err := post.validateBounds(); err != nil {
			return PreparedPageSnapshot{}, err
		}
		if !post.Author.hasIdentity() {
			return PreparedPageSnapshot{}, ErrPreparedSnapshotMissingAuthor
		}
		if !validPreparedSnapshotCreatedAt(post.CreatedAt) {
			return PreparedPageSnapshot{}, ErrPreparedSnapshotInvalidCreatedAt
		}
		preparedPosts[index] = PreparedPost{
			PostID:  post.PostID,
			GroupID: request.WatchedGroup.ID(),
			PostURL: post.PostURL,
			Author: domain.AuthorIdentity{
				FacebookUserID:      post.Author.FacebookUserID,
				CanonicalProfileURL: post.Author.CanonicalProfileURL,
				Username:            post.Author.Username,
				DisplayName:         post.Author.DisplayName,
			},
			Body:      post.Body,
			CreatedAt: post.CreatedAt,
		}
	}

	return PreparedPageSnapshot{
		SchemaVersion:    PreparedPageSnapshotSchemaVersion,
		WatchedGroupID:   request.WatchedGroup.ID(),
		WatchedGroupName: request.WatchedGroup.Name(),
		CapturedAt:       request.ScanWindow.ScanStarted(),
		Posts:            preparedPosts,
	}, nil
}

func (post preparedSnapshotPost) validateBounds() error {
	fields := []struct {
		name  string
		value string
		max   int
	}{
		{name: "post_id", value: post.PostID, max: PreparedSnapshotMaxPostIDBytes},
		{name: "post_url", value: post.PostURL, max: PreparedSnapshotMaxURLBytes},
		{name: "body", value: post.Body, max: PreparedSnapshotMaxBodyBytes},
		{name: "created_at", value: post.CreatedAt, max: preparedSnapshotMaxCreatedAtBytes},
		{name: "author.facebook_user_id", value: post.Author.FacebookUserID, max: PreparedSnapshotMaxFacebookIDBytes},
		{name: "author.canonical_profile_url", value: post.Author.CanonicalProfileURL, max: PreparedSnapshotMaxURLBytes},
		{name: "author.username", value: post.Author.Username, max: PreparedSnapshotMaxUsernameBytes},
		{name: "author.display_name", value: post.Author.DisplayName, max: PreparedSnapshotMaxDisplayNameBytes},
	}
	for _, field := range fields {
		if len(field.value) > field.max {
			return fmt.Errorf("%w: %s", ErrPreparedSnapshotFieldTooLarge, field.name)
		}
	}
	return nil
}

func (author preparedSnapshotAuthor) hasIdentity() bool {
	return strings.TrimSpace(author.FacebookUserID) != "" ||
		strings.TrimSpace(author.CanonicalProfileURL) != "" ||
		strings.TrimSpace(author.Username) != "" ||
		strings.TrimSpace(author.DisplayName) != ""
}

func validPreparedSnapshotCreatedAt(value string) bool {
	if !strings.HasSuffix(value, "+07:00") {
		return false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return false
	}
	_, offset := parsed.Zone()
	return offset == 7*60*60
}
