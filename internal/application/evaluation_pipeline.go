package application

import (
	"errors"
	"time"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/dedup"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/rules"
)

var (
	ErrInvalidPipelineScanWindow     = errors.New("application: invalid pipeline scan window")
	ErrInvalidPipelineSearchProfile  = errors.New("application: invalid pipeline search profile")
	ErrInvalidPipelineGeographicMode = errors.New("application: invalid pipeline geographic mode")
)

// EvaluationPipelineInput contains one in-memory deterministic evaluation request.
type EvaluationPipelineInput struct {
	Posts          []domain.RawPost
	ScanWindow     domain.ScanWindow
	SearchProfile  domain.SearchProfile
	GeographicMode domain.GeographicMode
	Blocklist      blocklist.List
}

// EvaluatedPost preserves a RawPost and its exact rule result.
type EvaluatedPost struct {
	Post   domain.RawPost
	Result rules.Result
}

// ReviewPost preserves a RawPost that requires review.
type ReviewPost struct {
	Post   domain.RawPost
	Result rules.Result
}

// ExcludedPost preserves a RawPost excluded by deterministic rules.
type ExcludedPost struct {
	Post   domain.RawPost
	Result rules.Result
}

// EvaluationPipelineResult is the in-memory end-to-end deterministic pipeline output.
type EvaluationPipelineResult struct {
	evaluated     []EvaluatedPost
	eligible      []EvaluatedPost
	review        []ReviewPost
	excluded      []ExcludedPost
	aggregation   dedup.AggregationResult
	leadFiltering LeadFilterResult
}

// RunEvaluationPipeline runs rules, aggregation, and blocklist filtering in the approved order.
func RunEvaluationPipeline(input EvaluationPipelineInput) (EvaluationPipelineResult, error) {
	if err := validatePipelineInput(input); err != nil {
		return EvaluationPipelineResult{}, err
	}

	var result EvaluationPipelineResult
	var eligiblePosts []domain.RawPost
	for _, post := range input.Posts {
		ruleResult := rules.EvaluatePostForBuyerSearchAndGeography(
			post,
			input.ScanWindow,
			input.SearchProfile,
			input.GeographicMode,
		)
		evaluated := EvaluatedPost{Post: post, Result: copyRuleResult(ruleResult)}
		result.evaluated = append(result.evaluated, evaluated)

		switch ruleResult.Decision {
		case rules.DecisionInclude:
			result.eligible = append(result.eligible, evaluated)
			eligiblePosts = append(eligiblePosts, post)
		case rules.DecisionReview:
			result.review = append(result.review, ReviewPost{Post: post, Result: copyRuleResult(ruleResult)})
		default:
			result.excluded = append(result.excluded, ExcludedPost{Post: post, Result: copyRuleResult(ruleResult)})
		}
	}

	result.aggregation = dedup.AggregatePosts(eligiblePosts, input.SearchProfile)
	result.leadFiltering = FilterLeads(result.aggregation.Leads(), input.Blocklist)
	return result, nil
}

// Evaluated returns all rule-evaluated posts in input order.
func (r EvaluationPipelineResult) Evaluated() []EvaluatedPost {
	return copyEvaluatedPosts(r.evaluated)
}

// Eligible returns posts that were included by rules and passed to aggregation.
func (r EvaluationPipelineResult) Eligible() []EvaluatedPost {
	return copyEvaluatedPosts(r.eligible)
}

// Review returns posts that require review.
func (r EvaluationPipelineResult) Review() []ReviewPost {
	if len(r.review) == 0 {
		return nil
	}
	copied := make([]ReviewPost, len(r.review))
	for i, post := range r.review {
		copied[i] = post
		copied[i].Result = copyRuleResult(post.Result)
	}
	return copied
}

// Excluded returns posts excluded by deterministic rules.
func (r EvaluationPipelineResult) Excluded() []ExcludedPost {
	if len(r.excluded) == 0 {
		return nil
	}
	copied := make([]ExcludedPost, len(r.excluded))
	for i, post := range r.excluded {
		copied[i] = post
		copied[i].Result = copyRuleResult(post.Result)
	}
	return copied
}

// AggregatedLeads returns the aggregation leads before blocklist filtering.
func (r EvaluationPipelineResult) AggregatedLeads() []dedup.Lead {
	return r.aggregation.Leads()
}

// Unaggregated returns eligible posts that dedup could not aggregate.
func (r EvaluationPipelineResult) Unaggregated() []dedup.UnaggregatedPost {
	return r.aggregation.Unaggregated()
}

// Conflicts returns explicit aggregation source conflicts.
func (r EvaluationPipelineResult) Conflicts() []dedup.SourceConflict {
	return r.aggregation.Conflicts()
}

// Allowed returns post-aggregation leads allowed by blocklist filtering.
func (r EvaluationPipelineResult) Allowed() []AllowedLead {
	return r.leadFiltering.Allowed()
}

// Blocked returns post-aggregation leads blocked by blocklist filtering.
func (r EvaluationPipelineResult) Blocked() []BlockedLead {
	return r.leadFiltering.Blocked()
}

// Unresolved returns post-aggregation leads unresolved by blocklist filtering.
func (r EvaluationPipelineResult) Unresolved() []UnresolvedLead {
	return r.leadFiltering.Unresolved()
}

func validatePipelineInput(input EvaluationPipelineInput) error {
	if !validPipelineScanWindow(input.ScanWindow) {
		return ErrInvalidPipelineScanWindow
	}
	if !validPipelineSearchProfile(input.SearchProfile) {
		return ErrInvalidPipelineSearchProfile
	}
	if !input.GeographicMode.Valid() {
		return ErrInvalidPipelineGeographicMode
	}
	return nil
}

func validPipelineScanWindow(window domain.ScanWindow) bool {
	if window.Timezone() != domain.RequiredTimezone {
		return false
	}
	if window.ScanDate().IsZero() || window.StartOfDay().IsZero() || window.ScanStarted().IsZero() {
		return false
	}
	if window.StartOfDay().After(window.ScanStarted()) {
		return false
	}
	if window.StartOfDay().Hour() != 0 ||
		window.StartOfDay().Minute() != 0 ||
		window.StartOfDay().Second() != 0 ||
		window.StartOfDay().Nanosecond() != 0 {
		return false
	}
	return sameCalendarDay(window.ScanDate(), window.ScanStarted()) &&
		sameCalendarDay(window.StartOfDay(), window.ScanStarted())
}

func validPipelineSearchProfile(profile domain.SearchProfile) bool {
	return profile.IsEnabled() && profile.ID() != "" && profile.DisplayName() != "" && len(profile.ProductTerms()) > 0
}

func sameCalendarDay(left time.Time, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func copyEvaluatedPosts(posts []EvaluatedPost) []EvaluatedPost {
	if len(posts) == 0 {
		return nil
	}
	copied := make([]EvaluatedPost, len(posts))
	for i, post := range posts {
		copied[i] = post
		copied[i].Result = copyRuleResult(post.Result)
	}
	return copied
}

func copyRuleResult(result rules.Result) rules.Result {
	if len(result.Reasons) == 0 {
		result.Reasons = nil
		return result
	}
	reasons := make([]rules.ReasonCode, len(result.Reasons))
	copy(reasons, result.Reasons)
	result.Reasons = reasons
	return result
}
