package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/soleda2026/ScanFB/internal/domain"
)

// AuthorIdentityKind names the stable author field used for deduplication.
type AuthorIdentityKind string

const (
	AuthorIdentityKindFacebookUserID      AuthorIdentityKind = "facebook_user_id"
	AuthorIdentityKindCanonicalProfileURL AuthorIdentityKind = "canonical_profile_url"
	AuthorIdentityKindUsername            AuthorIdentityKind = "username"
)

// AuthorKey is the stable author identity selected by documented priority.
type AuthorKey struct {
	Kind  AuthorIdentityKind
	Value string
}

// NewAuthorKey returns the strongest stable author key available.
func NewAuthorKey(author domain.AuthorIdentity) AuthorKey {
	if value := strings.TrimSpace(author.FacebookUserID); value != "" {
		return AuthorKey{Kind: AuthorIdentityKindFacebookUserID, Value: value}
	}
	if value := normalizeCanonicalProfileURL(author.CanonicalProfileURL); value != "" {
		return AuthorKey{Kind: AuthorIdentityKindCanonicalProfileURL, Value: value}
	}
	if value := normalizeUsername(author.Username); value != "" {
		return AuthorKey{Kind: AuthorIdentityKindUsername, Value: value}
	}
	return AuthorKey{}
}

// Sufficient reports whether the key can authorize automatic deduplication.
func (k AuthorKey) Sufficient() bool {
	return k.Kind != "" && k.Value != ""
}

// NeedKey is the deterministic identity for a buyer need under one SearchProfile.
type NeedKey struct {
	SearchProfileID     string
	NormalizedBody      string
	BodyFingerprint     string
	ProductEvidence     []string
	BuyerIntentEvidence []string
}

// NewNeedKey derives deterministic buyer-need identity from RawPost.Body and SearchProfile.
func NewNeedKey(post domain.RawPost, profile domain.SearchProfile) NeedKey {
	normalizedBody := normalizeText(post.Body)
	return NeedKey{
		SearchProfileID:     strings.TrimSpace(profile.ID()),
		NormalizedBody:      normalizedBody,
		BodyFingerprint:     fingerprint(normalizedBody),
		ProductEvidence:     matchedEvidence(normalizedBody, profile.ProductTerms()),
		BuyerIntentEvidence: matchedEvidence(normalizedBody, profile.BuyerIntentTerms()),
	}
}

// Sufficient reports whether the key contains product and buyer-intent evidence.
func (k NeedKey) Sufficient() bool {
	return k.SearchProfileID != "" && k.NormalizedBody != "" && len(k.ProductEvidence) > 0 && len(k.BuyerIntentEvidence) > 0
}

// CandidateKey combines stable author and buyer-need identity for a RawPost.
type CandidateKey struct {
	Author AuthorKey
	Need   NeedKey
}

// NewCandidateKey derives the dedup identity candidate for one post.
func NewCandidateKey(post domain.RawPost, profile domain.SearchProfile) CandidateKey {
	return CandidateKey{
		Author: NewAuthorKey(post.Author),
		Need:   NewNeedKey(post, profile),
	}
}

func normalizeCanonicalProfileURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func matchedEvidence(normalizedBody string, terms []string) []string {
	var evidence []string
	for _, term := range terms {
		normalizedTerm := normalizeText(term)
		if normalizedTerm == "" {
			continue
		}
		if containsTerm(normalizedBody, normalizedTerm) && !containsString(evidence, normalizedTerm) {
			evidence = append(evidence, normalizedTerm)
		}
	}
	return evidence
}

func containsTerm(normalizedBody string, normalizedTerm string) bool {
	start := 0
	for {
		index := strings.Index(normalizedBody[start:], normalizedTerm)
		if index < 0 {
			return false
		}

		absoluteIndex := start + index
		beforeOK := absoluteIndex == 0 || !isWordRune(runeBefore(normalizedBody, absoluteIndex))
		afterIndex := absoluteIndex + len(normalizedTerm)
		afterOK := afterIndex == len(normalizedBody) || !isWordRune(runeAfter(normalizedBody, afterIndex))
		if beforeOK && afterOK {
			return true
		}

		start = absoluteIndex + len(normalizedTerm)
		if start >= len(normalizedBody) {
			return false
		}
	}
}

func runeBefore(value string, byteIndex int) rune {
	var previous rune
	for _, r := range value[:byteIndex] {
		previous = r
	}
	return previous
}

func runeAfter(value string, byteIndex int) rune {
	for _, r := range value[byteIndex:] {
		return r
	}
	return 0
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
