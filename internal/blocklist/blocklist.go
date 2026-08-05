// Package blocklist owns deterministic local blocklist identity primitives.
//
// It intentionally does not implement persistence, scan orchestration, UI, CLI,
// Facebook integration, network lookup, or fuzzy matching.
package blocklist

import (
	"errors"
	"strings"

	"github.com/soleda2026/ScanFB/internal/domain"
)

// IdentityKind names a stable author identity kind supported by the blocklist.
type IdentityKind string

const (
	IdentityKindFacebookUserID      IdentityKind = "facebook_user_id"
	IdentityKindCanonicalProfileURL IdentityKind = "canonical_profile_url"
	IdentityKindUsername            IdentityKind = "username"
)

// ReasonCode is a stable machine-readable blocklist reason.
type ReasonCode string

const (
	ReasonIdentityMatched             ReasonCode = "blocklist.identity_matched"
	ReasonIdentityNotFound            ReasonCode = "blocklist.identity_not_found"
	ReasonStableAuthorIdentityMissing ReasonCode = "blocklist.stable_author_identity_missing"
	ReasonInvalidEntry                ReasonCode = "blocklist.invalid_entry"
	ReasonUnsupportedIdentityKind     ReasonCode = "blocklist.unsupported_identity_kind"
)

// MatchOutcome is the deterministic blocklist match outcome.
type MatchOutcome string

const (
	MatchOutcomeBlocked              MatchOutcome = "blocked"
	MatchOutcomeNotBlocked           MatchOutcome = "not_blocked"
	MatchOutcomeInsufficientIdentity MatchOutcome = "insufficient_identity"
)

var (
	ErrInvalidEntry            = errors.New("blocklist: invalid entry")
	ErrUnsupportedIdentityKind = errors.New("blocklist: unsupported identity kind")
)

// EntryError exposes a stable reason for invalid entry/key construction.
type EntryError struct {
	Reason ReasonCode
}

func (e EntryError) Error() string {
	return string(e.Reason)
}

func (e EntryError) Unwrap() error {
	switch e.Reason {
	case ReasonUnsupportedIdentityKind:
		return ErrUnsupportedIdentityKind
	default:
		return ErrInvalidEntry
	}
}

// IdentityKey is a normalized stable author identity.
type IdentityKey struct {
	Kind  IdentityKind
	Value string
}

// NewIdentityKey validates and normalizes one stable blocklist identity.
func NewIdentityKey(kind IdentityKind, value string) (IdentityKey, error) {
	normalized, supported := normalizeIdentityValue(kind, value)
	if !supported {
		return IdentityKey{}, EntryError{Reason: ReasonUnsupportedIdentityKind}
	}
	if normalized == "" {
		return IdentityKey{}, EntryError{Reason: ReasonInvalidEntry}
	}
	return IdentityKey{Kind: kind, Value: normalized}, nil
}

// Entry is one immutable-style in-memory blocklist entry.
type Entry struct {
	key         IdentityKey
	displayName string
}

// NewEntry creates a blocklist entry from deterministic identity data.
func NewEntry(kind IdentityKind, value string, displayName string) (Entry, error) {
	key, err := NewIdentityKey(kind, value)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		key:         key,
		displayName: strings.TrimSpace(displayName),
	}, nil
}

// Key returns the normalized stable identity key for this entry.
func (e Entry) Key() IdentityKey {
	return e.key
}

// DisplayName returns optional user-entered display metadata.
func (e Entry) DisplayName() string {
	return e.displayName
}

// List is a deterministic in-memory blocklist value.
type List struct {
	entries []Entry
	index   map[IdentityKey]int
}

// NewList constructs a deterministic in-memory list and ignores later exact duplicates.
func NewList(entries []Entry) List {
	list := List{
		index: make(map[IdentityKey]int, len(entries)),
	}
	for _, entry := range entries {
		key := entry.Key()
		if !key.valid() {
			continue
		}
		if _, exists := list.index[key]; exists {
			continue
		}
		list.index[key] = len(list.entries)
		list.entries = append(list.entries, entry)
	}
	return list
}

// Len returns the number of unique blocklist entries.
func (l List) Len() int {
	return len(l.entries)
}

// Entries returns entries in first occurrence order.
func (l List) Entries() []Entry {
	return copyEntries(l.entries)
}

// Lookup returns the exact entry for a normalized identity key.
func (l List) Lookup(key IdentityKey) (Entry, bool) {
	position, ok := l.index[key]
	if !ok {
		return Entry{}, false
	}
	return l.entries[position], true
}

// MatchResult contains one deterministic author blocklist evaluation.
type MatchResult struct {
	Outcome      MatchOutcome
	Reasons      []ReasonCode
	AuthorKey    IdentityKey
	MatchedEntry Entry
}

// MatchAuthor matches an author using only the strongest available stable identity.
func (l List) MatchAuthor(author domain.AuthorIdentity) MatchResult {
	key, ok := NewAuthorIdentityKey(author)
	if !ok {
		return matchResult(MatchOutcomeInsufficientIdentity, IdentityKey{}, Entry{}, ReasonStableAuthorIdentityMissing)
	}

	entry, found := l.Lookup(key)
	if !found {
		return matchResult(MatchOutcomeNotBlocked, key, Entry{}, ReasonIdentityNotFound)
	}

	return matchResult(MatchOutcomeBlocked, key, entry, ReasonIdentityMatched)
}

// NewAuthorIdentityKey derives the strongest stable author identity key.
func NewAuthorIdentityKey(author domain.AuthorIdentity) (IdentityKey, bool) {
	if key, ok := authorIdentityKey(IdentityKindFacebookUserID, author.FacebookUserID); ok {
		return key, true
	}
	if key, ok := authorIdentityKey(IdentityKindCanonicalProfileURL, author.CanonicalProfileURL); ok {
		return key, true
	}
	if key, ok := authorIdentityKey(IdentityKindUsername, author.Username); ok {
		return key, true
	}
	return IdentityKey{}, false
}

func (k IdentityKey) valid() bool {
	if k.Value == "" {
		return false
	}
	switch k.Kind {
	case IdentityKindFacebookUserID, IdentityKindCanonicalProfileURL, IdentityKindUsername:
		return true
	default:
		return false
	}
}

func authorIdentityKey(kind IdentityKind, value string) (IdentityKey, bool) {
	key, err := NewIdentityKey(kind, value)
	if err != nil {
		return IdentityKey{}, false
	}
	return key, true
}

func matchResult(outcome MatchOutcome, authorKey IdentityKey, entry Entry, reasons ...ReasonCode) MatchResult {
	return MatchResult{
		Outcome:      outcome,
		Reasons:      copyReasons(reasons),
		AuthorKey:    authorKey,
		MatchedEntry: entry,
	}
}

func normalizeIdentityValue(kind IdentityKind, value string) (string, bool) {
	switch kind {
	case IdentityKindFacebookUserID:
		return strings.TrimSpace(value), true
	case IdentityKindCanonicalProfileURL:
		return strings.TrimRight(strings.TrimSpace(value), "/"), true
	case IdentityKindUsername:
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}

func copyEntries(entries []Entry) []Entry {
	if len(entries) == 0 {
		return nil
	}
	copied := make([]Entry, len(entries))
	copy(copied, entries)
	return copied
}

func copyReasons(reasons []ReasonCode) []ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}
