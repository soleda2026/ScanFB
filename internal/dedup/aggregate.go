package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/soleda2026/ScanFB/internal/domain"
)

// SourceIdentityKind names the source field used to preserve unique posts.
type SourceIdentityKind string

const (
	SourceIdentityKindPostID  SourceIdentityKind = "post_id"
	SourceIdentityKindPostURL SourceIdentityKind = "post_url"
)

// SourceKey identifies one source post without using body similarity.
type SourceKey struct {
	Kind  SourceIdentityKind
	Value string
}

// NewSourceKey derives deterministic source identity from RawPost.
func NewSourceKey(post domain.RawPost) SourceKey {
	if value := strings.TrimSpace(post.PostID); value != "" {
		return SourceKey{Kind: SourceIdentityKindPostID, Value: value}
	}
	if value := strings.TrimSpace(post.PostURL); value != "" {
		return SourceKey{Kind: SourceIdentityKindPostURL, Value: value}
	}
	return SourceKey{}
}

// Sufficient reports whether a source can be safely deduplicated inside a lead.
func (k SourceKey) Sufficient() bool {
	return k.Kind != "" && k.Value != ""
}

// LeadKey is deterministic lead identity derived only from stable author and need identity.
type LeadKey struct {
	Value  string
	Author AuthorKey
	Need   NeedKey
}

// NewLeadKey derives a deterministic lead key from a candidate.
func NewLeadKey(candidate CandidateKey) LeadKey {
	return LeadKey{
		Value:  leadFingerprint(candidate.Author, candidate.Need),
		Author: candidate.Author,
		Need:   candidate.Need,
	}
}

// SourcePost preserves the complete original RawPost plus its source identity.
type SourcePost struct {
	Key  SourceKey
	Post domain.RawPost
}

// Lead is an in-memory logical buyer lead with all preserved sources.
type Lead struct {
	Key     LeadKey
	Author  AuthorKey
	Need    NeedKey
	sources []SourcePost
}

// Sources returns preserved source posts in deterministic first-occurrence order.
func (l Lead) Sources() []SourcePost {
	return copySourcePosts(l.sources)
}

// SourceCount returns the number of preserved source posts.
func (l Lead) SourceCount() int {
	return len(l.sources)
}

// UnaggregatedPost preserves a post that could not be automatically aggregated.
type UnaggregatedPost struct {
	Post      domain.RawPost
	Candidate CandidateKey
	Source    SourceKey
	Reasons   []ReasonCode
}

// SourceConflict preserves a post whose source identity conflicts with a prior source.
type SourceConflict struct {
	Post           domain.RawPost
	ExistingSource SourcePost
	Candidate      CandidateKey
	Source         SourceKey
	Reasons        []ReasonCode
}

// AggregationResult is the deterministic in-memory aggregation output.
type AggregationResult struct {
	leads        []Lead
	unaggregated []UnaggregatedPost
	conflicts    []SourceConflict
}

// Leads returns leads in deterministic first accepted source order.
func (r AggregationResult) Leads() []Lead {
	return copyLeads(r.leads)
}

// Unaggregated returns posts that could not be automatically aggregated.
func (r AggregationResult) Unaggregated() []UnaggregatedPost {
	return copyUnaggregated(r.unaggregated)
}

// Conflicts returns posts with conflicting source identity.
func (r AggregationResult) Conflicts() []SourceConflict {
	return copyConflicts(r.conflicts)
}

// AggregatePosts groups duplicate buyer posts in memory while preserving every source post.
func AggregatePosts(posts []domain.RawPost, profile domain.SearchProfile) AggregationResult {
	var result AggregationResult
	leadIndex := make(map[string]int)
	sourceIndex := make(map[string]sourceRecord)

	for _, post := range posts {
		candidate := NewCandidateKey(post, profile)
		source := NewSourceKey(post)
		if reasons := aggregationRejectionReasons(candidate, source); len(reasons) > 0 {
			result.unaggregated = append(result.unaggregated, UnaggregatedPost{
				Post:      post,
				Candidate: candidate,
				Source:    source,
				Reasons:   copyReasonCodes(reasons),
			})
			continue
		}

		leadKey := NewLeadKey(candidate)
		sourceIdentity := source.identity()
		if existing, ok := sourceIndex[sourceIdentity]; ok {
			if sourceConsistent(existing, post, candidate, leadKey) {
				continue
			}

			result.conflicts = append(result.conflicts, SourceConflict{
				Post:           post,
				ExistingSource: existing.source,
				Candidate:      candidate,
				Source:         source,
				Reasons:        sourceConflictReasons(existing, post, candidate, leadKey),
			})
			continue
		}

		leadPosition, exists := leadIndex[leadKey.Value]
		sourcePost := SourcePost{Key: source, Post: post}
		if !exists {
			result.leads = append(result.leads, Lead{
				Key:     leadKey,
				Author:  candidate.Author,
				Need:    candidate.Need,
				sources: []SourcePost{sourcePost},
			})
			leadPosition = len(result.leads) - 1
			leadIndex[leadKey.Value] = leadPosition
		} else {
			lead := &result.leads[leadPosition]
			comparison := ComparePosts(lead.sources[0].Post, post, profile)
			if comparison.Outcome != ComparisonOutcomeDuplicateNeed {
				result.conflicts = append(result.conflicts, SourceConflict{
					Post:           post,
					ExistingSource: lead.sources[0],
					Candidate:      candidate,
					Source:         source,
					Reasons:        copyReasonCodes(comparison.Reasons),
				})
				continue
			}
			lead.sources = append(lead.sources, sourcePost)
		}

		sourceIndex[sourceIdentity] = sourceRecord{
			source:    sourcePost,
			candidate: candidate,
			leadKey:   leadKey,
		}
	}

	return result
}

type sourceRecord struct {
	source    SourcePost
	candidate CandidateKey
	leadKey   LeadKey
}

func aggregationRejectionReasons(candidate CandidateKey, source SourceKey) []ReasonCode {
	var reasons []ReasonCode
	if !candidate.Author.Sufficient() {
		reasons = append(reasons, ReasonStableAuthorIdentityMissing)
	}
	if len(candidate.Need.ProductEvidence) == 0 {
		reasons = append(reasons, ReasonProductEvidenceMissing)
	}
	if len(candidate.Need.BuyerIntentEvidence) == 0 {
		reasons = append(reasons, ReasonBuyerIntentEvidenceMissing)
	}
	if !source.Sufficient() {
		reasons = append(reasons, ReasonSourceIdentityMissing)
	}
	return reasons
}

func sourceConsistent(existing sourceRecord, post domain.RawPost, candidate CandidateKey, leadKey LeadKey) bool {
	return existing.leadKey.Value == leadKey.Value &&
		existing.candidate.Author == candidate.Author &&
		existing.candidate.Need.SearchProfileID == candidate.Need.SearchProfileID &&
		existing.candidate.Need.BodyFingerprint == candidate.Need.BodyFingerprint &&
		normalizeText(existing.source.Post.Body) == normalizeText(post.Body)
}

func sourceConflictReasons(existing sourceRecord, post domain.RawPost, candidate CandidateKey, leadKey LeadKey) []ReasonCode {
	reasons := []ReasonCode{ReasonSourceIdentityConflict}
	if existing.candidate.Author != candidate.Author {
		reasons = append(reasons, ReasonStableAuthorIdentityDiffers)
	}
	if existing.candidate.Need.SearchProfileID != candidate.Need.SearchProfileID ||
		existing.candidate.Need.BodyFingerprint != candidate.Need.BodyFingerprint ||
		normalizeText(existing.source.Post.Body) != normalizeText(post.Body) {
		reasons = append(reasons, ReasonNormalizedNeedDiffers)
	}
	return reasons
}

func leadFingerprint(author AuthorKey, need NeedKey) string {
	canonical := strings.Join([]string{
		string(author.Kind),
		author.Value,
		need.SearchProfileID,
		need.BodyFingerprint,
	}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func (k SourceKey) identity() string {
	return string(k.Kind) + "\x00" + k.Value
}

func copySourcePosts(sources []SourcePost) []SourcePost {
	if len(sources) == 0 {
		return nil
	}
	copied := make([]SourcePost, len(sources))
	copy(copied, sources)
	return copied
}

func copyLeads(leads []Lead) []Lead {
	if len(leads) == 0 {
		return nil
	}
	copied := make([]Lead, len(leads))
	for i, lead := range leads {
		copied[i] = lead
		copied[i].sources = copySourcePosts(lead.sources)
	}
	return copied
}

func copyUnaggregated(posts []UnaggregatedPost) []UnaggregatedPost {
	if len(posts) == 0 {
		return nil
	}
	copied := make([]UnaggregatedPost, len(posts))
	for i, post := range posts {
		copied[i] = post
		copied[i].Reasons = copyReasonCodes(post.Reasons)
	}
	return copied
}

func copyConflicts(conflicts []SourceConflict) []SourceConflict {
	if len(conflicts) == 0 {
		return nil
	}
	copied := make([]SourceConflict, len(conflicts))
	for i, conflict := range conflicts {
		copied[i] = conflict
		copied[i].Reasons = copyReasonCodes(conflict.Reasons)
	}
	return copied
}
