package domain

import (
	"net/url"
	"strings"
	"time"
)

// WatchedGroupIdentityKind identifies the source key used for duplicate checks.
type WatchedGroupIdentityKind string

const (
	WatchedGroupIdentityFacebookGroupID WatchedGroupIdentityKind = "facebook_group_id"
	WatchedGroupIdentityCanonicalURL    WatchedGroupIdentityKind = "canonical_url"
)

// WatchedGroupIdentityKey is the comparable authoritative identity of a group.
type WatchedGroupIdentityKey struct {
	kind  WatchedGroupIdentityKind
	value string
}

// WatchedGroupMetadata contains the user-editable Phase 9B metadata.
type WatchedGroupMetadata struct {
	Name                 string
	Notes                string
	LastSuccessfulScanAt time.Time
	LastError            string
	DisplayOrder         int
}

// WatchedGroup represents one Facebook group explicitly added by the user.
type WatchedGroup struct {
	id                   string
	facebookGroupID      string
	canonicalURL         string
	createdAt            time.Time
	active               bool
	name                 string
	notes                string
	lastSuccessfulScanAt time.Time
	lastError            string
	displayOrder         int
}

// NewWatchedGroup creates an active watched group from caller-supplied identity and time.
func NewWatchedGroup(id string, facebookGroupID string, canonicalURL string, name string, createdAt time.Time) (WatchedGroup, error) {
	group := WatchedGroup{
		id:              strings.TrimSpace(id),
		facebookGroupID: strings.TrimSpace(facebookGroupID),
		canonicalURL:    strings.TrimSpace(canonicalURL),
		createdAt:       createdAt,
		active:          true,
		name:            strings.TrimSpace(name),
	}
	if err := group.Validate(); err != nil {
		return WatchedGroup{}, err
	}
	return group, nil
}

// Validate checks the complete WatchedGroup value without mutating it.
func (g WatchedGroup) Validate() error {
	if strings.TrimSpace(g.id) == "" {
		return ErrInvalidWatchedGroupID
	}
	if strings.TrimSpace(g.name) == "" {
		return ErrInvalidWatchedGroupName
	}
	if g.createdAt.IsZero() {
		return ErrInvalidWatchedGroupCreatedAt
	}
	if strings.TrimSpace(g.facebookGroupID) == "" && strings.TrimSpace(g.canonicalURL) == "" {
		return ErrMissingWatchedGroupIdentity
	}
	if g.canonicalURL != "" && !validWatchedGroupCanonicalURL(g.canonicalURL) {
		return ErrInvalidWatchedGroupCanonicalURL
	}
	if !g.lastSuccessfulScanAt.IsZero() && g.lastSuccessfulScanAt.Before(g.createdAt) {
		return ErrWatchedGroupScanBeforeCreated
	}
	return nil
}

// WithMetadata returns an updated copy while preserving identity and lifecycle state.
func (g WatchedGroup) WithMetadata(metadata WatchedGroupMetadata) (WatchedGroup, error) {
	updated := g
	updated.name = strings.TrimSpace(metadata.Name)
	updated.notes = metadata.Notes
	updated.lastSuccessfulScanAt = metadata.LastSuccessfulScanAt
	updated.lastError = metadata.LastError
	updated.displayOrder = metadata.DisplayOrder
	if err := updated.Validate(); err != nil {
		return WatchedGroup{}, err
	}
	return updated, nil
}

// WithActive returns a copy with the requested active state.
func (g WatchedGroup) WithActive(active bool) WatchedGroup {
	g.active = active
	return g
}

func (g WatchedGroup) ID() string {
	return g.id
}

func (g WatchedGroup) FacebookGroupID() string {
	return g.facebookGroupID
}

func (g WatchedGroup) CanonicalURL() string {
	return g.canonicalURL
}

func (g WatchedGroup) Name() string {
	return g.name
}

func (g WatchedGroup) CreatedAt() time.Time {
	return g.createdAt
}

func (g WatchedGroup) IsActive() bool {
	return g.active
}

func (g WatchedGroup) Notes() string {
	return g.notes
}

func (g WatchedGroup) LastSuccessfulScanAt() (time.Time, bool) {
	if g.lastSuccessfulScanAt.IsZero() {
		return time.Time{}, false
	}
	return g.lastSuccessfulScanAt, true
}

func (g WatchedGroup) LastError() string {
	return g.lastError
}

func (g WatchedGroup) DisplayOrder() int {
	return g.displayOrder
}

func (g WatchedGroup) Metadata() WatchedGroupMetadata {
	return WatchedGroupMetadata{
		Name:                 g.name,
		Notes:                g.notes,
		LastSuccessfulScanAt: g.lastSuccessfulScanAt,
		LastError:            g.lastError,
		DisplayOrder:         g.displayOrder,
	}
}

// IdentityKey returns facebookGroupId when present, otherwise canonicalUrl.
func (g WatchedGroup) IdentityKey() WatchedGroupIdentityKey {
	if g.facebookGroupID != "" {
		return WatchedGroupIdentityKey{kind: WatchedGroupIdentityFacebookGroupID, value: g.facebookGroupID}
	}
	return WatchedGroupIdentityKey{kind: WatchedGroupIdentityCanonicalURL, value: g.canonicalURL}
}

func (k WatchedGroupIdentityKey) Kind() WatchedGroupIdentityKind {
	return k.kind
}

func (k WatchedGroupIdentityKey) Value() string {
	return k.value
}

func validWatchedGroupCanonicalURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && strings.EqualFold(parsed.Scheme, "https") && parsed.Host != ""
}
