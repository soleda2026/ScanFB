package application

import (
	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/dedup"
	"github.com/soleda2026/ScanFB/internal/domain"
)

// ReasonCode is a stable machine-readable application orchestration reason.
type ReasonCode string

const (
	ReasonBlocklistEvaluationUnsupported ReasonCode = "application.blocklist_evaluation_unsupported"
)

// AllowedLead preserves a lead that did not match the local blocklist.
type AllowedLead struct {
	Lead  dedup.Lead
	Match blocklist.MatchResult
}

// BlockedLead preserves a lead that matched the local blocklist.
type BlockedLead struct {
	Lead  dedup.Lead
	Match blocklist.MatchResult
}

// UnresolvedLead preserves a lead that could not be safely evaluated.
type UnresolvedLead struct {
	Lead    dedup.Lead
	Match   blocklist.MatchResult
	Reasons []ReasonCode
}

// LeadFilterResult is the deterministic in-memory output of blocklist filtering.
type LeadFilterResult struct {
	allowed    []AllowedLead
	blocked    []BlockedLead
	unresolved []UnresolvedLead
}

// FilterLeads separates already-aggregated leads through the local blocklist.
func FilterLeads(leads []dedup.Lead, list blocklist.List) LeadFilterResult {
	var result LeadFilterResult
	for _, lead := range leads {
		author, ok := authorIdentityFromLead(lead)
		if !ok {
			result.unresolved = append(result.unresolved, UnresolvedLead{
				Lead:    lead,
				Match:   insufficientMatch(),
				Reasons: unsupportedEvaluationReasons(lead.Author),
			})
			continue
		}

		match := list.MatchAuthor(author)
		switch match.Outcome {
		case blocklist.MatchOutcomeBlocked:
			result.blocked = append(result.blocked, BlockedLead{Lead: lead, Match: copyMatchResult(match)})
		case blocklist.MatchOutcomeNotBlocked:
			result.allowed = append(result.allowed, AllowedLead{Lead: lead, Match: copyMatchResult(match)})
		default:
			result.unresolved = append(result.unresolved, UnresolvedLead{Lead: lead, Match: copyMatchResult(match)})
		}
	}
	return result
}

// Allowed returns allowed leads in deterministic input-relative order.
func (r LeadFilterResult) Allowed() []AllowedLead {
	if len(r.allowed) == 0 {
		return nil
	}
	copied := make([]AllowedLead, len(r.allowed))
	for i, lead := range r.allowed {
		copied[i] = lead
		copied[i].Match = copyMatchResult(lead.Match)
	}
	return copied
}

// Blocked returns blocked leads in deterministic input-relative order.
func (r LeadFilterResult) Blocked() []BlockedLead {
	if len(r.blocked) == 0 {
		return nil
	}
	copied := make([]BlockedLead, len(r.blocked))
	for i, lead := range r.blocked {
		copied[i] = lead
		copied[i].Match = copyMatchResult(lead.Match)
	}
	return copied
}

// Unresolved returns safely unresolved leads in deterministic input-relative order.
func (r LeadFilterResult) Unresolved() []UnresolvedLead {
	if len(r.unresolved) == 0 {
		return nil
	}
	copied := make([]UnresolvedLead, len(r.unresolved))
	for i, lead := range r.unresolved {
		copied[i] = lead
		copied[i].Match = copyMatchResult(lead.Match)
		copied[i].Reasons = copyApplicationReasons(lead.Reasons)
	}
	return copied
}

func authorIdentityFromLead(lead dedup.Lead) (domain.AuthorIdentity, bool) {
	if !lead.Author.Sufficient() {
		return domain.AuthorIdentity{}, false
	}
	switch lead.Author.Kind {
	case dedup.AuthorIdentityKindFacebookUserID:
		return domain.AuthorIdentity{FacebookUserID: lead.Author.Value}, true
	case dedup.AuthorIdentityKindCanonicalProfileURL:
		return domain.AuthorIdentity{CanonicalProfileURL: lead.Author.Value}, true
	case dedup.AuthorIdentityKindUsername:
		return domain.AuthorIdentity{Username: lead.Author.Value}, true
	default:
		return domain.AuthorIdentity{}, false
	}
}

func insufficientMatch() blocklist.MatchResult {
	return blocklist.MatchResult{
		Outcome: blocklist.MatchOutcomeInsufficientIdentity,
		Reasons: []blocklist.ReasonCode{blocklist.ReasonStableAuthorIdentityMissing},
	}
}

func unsupportedEvaluationReasons(author dedup.AuthorKey) []ReasonCode {
	if author.Kind == "" {
		return nil
	}
	switch author.Kind {
	case dedup.AuthorIdentityKindFacebookUserID, dedup.AuthorIdentityKindCanonicalProfileURL, dedup.AuthorIdentityKindUsername:
		return nil
	default:
		return []ReasonCode{ReasonBlocklistEvaluationUnsupported}
	}
}

func copyMatchResult(match blocklist.MatchResult) blocklist.MatchResult {
	match.Reasons = copyBlocklistReasons(match.Reasons)
	return match
}

func copyBlocklistReasons(reasons []blocklist.ReasonCode) []blocklist.ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]blocklist.ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}

func copyApplicationReasons(reasons []ReasonCode) []ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}
