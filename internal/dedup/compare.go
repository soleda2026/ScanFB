package dedup

import (
	"strings"

	"github.com/soleda2026/ScanFB/internal/domain"
)

// ComparisonOutcome is the deterministic outcome of comparing two posts.
type ComparisonOutcome string

const (
	ComparisonOutcomeSameSourcePost       ComparisonOutcome = "same_source_post"
	ComparisonOutcomeDuplicateNeed        ComparisonOutcome = "duplicate_need"
	ComparisonOutcomeDistinct             ComparisonOutcome = "distinct"
	ComparisonOutcomeInsufficientIdentity ComparisonOutcome = "insufficient_identity"
)

// ReasonCode is a stable machine-readable dedup reason.
type ReasonCode string

const (
	ReasonSamePostID                  ReasonCode = "dedup.same_post_id"
	ReasonStableAuthorIdentityMissing ReasonCode = "dedup.stable_author_identity_missing"
	ReasonStableAuthorIdentityDiffers ReasonCode = "dedup.stable_author_identity_differs"
	ReasonProductEvidenceMissing      ReasonCode = "dedup.product_evidence_missing"
	ReasonBuyerIntentEvidenceMissing  ReasonCode = "dedup.buyer_intent_evidence_missing"
	ReasonNormalizedNeedDiffers       ReasonCode = "dedup.normalized_need_differs"
	ReasonDuplicateNeedMatched        ReasonCode = "dedup.duplicate_need_matched"
)

// Comparison contains the deterministic duplicate comparison result.
type Comparison struct {
	Outcome ComparisonOutcome
	Reasons []ReasonCode
	Left    CandidateKey
	Right   CandidateKey
}

// ComparePosts compares two normalized posts under one active SearchProfile.
func ComparePosts(left domain.RawPost, right domain.RawPost, profile domain.SearchProfile) Comparison {
	leftPostID := strings.TrimSpace(left.PostID)
	rightPostID := strings.TrimSpace(right.PostID)
	if leftPostID != "" && leftPostID == rightPostID {
		return comparison(ComparisonOutcomeSameSourcePost, nil, nil, ReasonSamePostID)
	}

	leftKey := NewCandidateKey(left, profile)
	rightKey := NewCandidateKey(right, profile)

	if !leftKey.Author.Sufficient() || !rightKey.Author.Sufficient() {
		return comparison(ComparisonOutcomeInsufficientIdentity, &leftKey, &rightKey, ReasonStableAuthorIdentityMissing)
	}
	if leftKey.Author != rightKey.Author {
		return comparison(ComparisonOutcomeDistinct, &leftKey, &rightKey, ReasonStableAuthorIdentityDiffers)
	}

	var reasons []ReasonCode
	if len(leftKey.Need.ProductEvidence) == 0 || len(rightKey.Need.ProductEvidence) == 0 {
		reasons = append(reasons, ReasonProductEvidenceMissing)
	}
	if len(leftKey.Need.BuyerIntentEvidence) == 0 || len(rightKey.Need.BuyerIntentEvidence) == 0 {
		reasons = append(reasons, ReasonBuyerIntentEvidenceMissing)
	}
	if len(reasons) > 0 {
		return comparison(ComparisonOutcomeDistinct, &leftKey, &rightKey, reasons...)
	}

	if leftKey.Need.SearchProfileID != rightKey.Need.SearchProfileID || leftKey.Need.BodyFingerprint != rightKey.Need.BodyFingerprint {
		return comparison(ComparisonOutcomeDistinct, &leftKey, &rightKey, ReasonNormalizedNeedDiffers)
	}

	return comparison(ComparisonOutcomeDuplicateNeed, &leftKey, &rightKey, ReasonDuplicateNeedMatched)
}

func comparison(outcome ComparisonOutcome, left *CandidateKey, right *CandidateKey, reasons ...ReasonCode) Comparison {
	result := Comparison{
		Outcome: outcome,
		Reasons: copyReasonCodes(reasons),
	}
	if left != nil {
		result.Left = *left
	}
	if right != nil {
		result.Right = *right
	}
	return result
}

func copyReasonCodes(reasons []ReasonCode) []ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}
