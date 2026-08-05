package application

import (
	"errors"
	"strings"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
	"github.com/soleda2026/ScanFB/internal/rules"
)

var (
	ErrScanBatchNoGroups            = errors.New("application: scan batch requires at least one group")
	ErrScanBatchTooManyGroups       = errors.New("application: scan batch allows at most five groups")
	ErrScanBatchEmptyGroupID        = errors.New("application: scan batch group id is empty")
	ErrScanBatchDuplicateGroupID    = errors.New("application: scan batch group id is duplicate")
	ErrScanBatchPostGroupIDMismatch = errors.New("application: scan batch post group id mismatch")
)

// GroupBatch contains one explicit source group and its already-collected posts.
type GroupBatch struct {
	GroupID   string
	GroupName string
	Posts     []domain.RawPost
}

// ScanBatchInput contains one deterministic in-memory manual batch request.
type ScanBatchInput struct {
	Groups         []GroupBatch
	ScanWindow     domain.ScanWindow
	SearchProfile  domain.SearchProfile
	GeographicMode domain.GeographicMode
	Blocklist      blocklist.List
}

// ScanBatchSummary contains count-only batch output derived from actual results.
type ScanBatchSummary struct {
	GroupCount                 int
	InputPostCount             int
	EvaluatedPostCount         int
	IncludePostCount           int
	ReviewPostCount            int
	ExcludedPostCount          int
	AggregatedLeadCount        int
	AllowedLeadCount           int
	BlockedLeadCount           int
	UnresolvedLeadCount        int
	UnaggregatedPostCount      int
	SourceConflictCount        int
	AllowedLeadSourcePostCount int
	BlockedLeadSourcePostCount int
}

// GroupSummary contains count-only rule-stage accounting for one input group.
type GroupSummary struct {
	GroupID            string
	InputPostCount     int
	EvaluatedPostCount int
	IncludePostCount   int
	ReviewPostCount    int
	ExcludedPostCount  int
}

// ScanBatchResult preserves validated group order, flattened input, pipeline output, and summaries.
type ScanBatchResult struct {
	groups         []GroupBatch
	flattenedPosts []domain.RawPost
	pipeline       EvaluationPipelineResult
	summary        ScanBatchSummary
	groupSummaries []GroupSummary
}

// RunScanBatch validates and evaluates one in-memory manual batch.
func RunScanBatch(input ScanBatchInput) (ScanBatchResult, error) {
	if err := validatePipelineInput(EvaluationPipelineInput{
		ScanWindow:     input.ScanWindow,
		SearchProfile:  input.SearchProfile,
		GeographicMode: input.GeographicMode,
		Blocklist:      input.Blocklist,
	}); err != nil {
		return ScanBatchResult{}, err
	}

	groups, err := validateBatchGroups(input.Groups)
	if err != nil {
		return ScanBatchResult{}, err
	}

	flattened, groupIndexes := flattenBatchPosts(groups)
	pipeline, err := RunEvaluationPipeline(EvaluationPipelineInput{
		Posts:          flattened,
		ScanWindow:     input.ScanWindow,
		SearchProfile:  input.SearchProfile,
		GeographicMode: input.GeographicMode,
		Blocklist:      input.Blocklist,
	})
	if err != nil {
		return ScanBatchResult{}, err
	}

	groupSummaries := buildGroupSummaries(groups, groupIndexes, pipeline)
	return ScanBatchResult{
		groups:         copyGroupBatches(groups),
		flattenedPosts: copyRawPosts(flattened),
		pipeline:       pipeline,
		summary:        buildBatchSummary(len(groups), len(flattened), pipeline),
		groupSummaries: copyGroupSummaries(groupSummaries),
	}, nil
}

// Groups returns validated groups in input order.
func (r ScanBatchResult) Groups() []GroupBatch {
	return copyGroupBatches(r.groups)
}

// FlattenedPosts returns posts in group order, then original order within each group.
func (r ScanBatchResult) FlattenedPosts() []domain.RawPost {
	return copyRawPosts(r.flattenedPosts)
}

// Pipeline returns the complete Phase 5A pipeline result.
func (r ScanBatchResult) Pipeline() EvaluationPipelineResult {
	return r.pipeline
}

// Summary returns count-only batch accounting.
func (r ScanBatchResult) Summary() ScanBatchSummary {
	return r.summary
}

// GroupSummaries returns per-group rule-stage summaries in input order.
func (r ScanBatchResult) GroupSummaries() []GroupSummary {
	return copyGroupSummaries(r.groupSummaries)
}

func validateBatchGroups(groups []GroupBatch) ([]GroupBatch, error) {
	if len(groups) == 0 {
		return nil, ErrScanBatchNoGroups
	}
	if len(groups) > domain.MaxScanRequestGroups {
		return nil, ErrScanBatchTooManyGroups
	}

	validated := make([]GroupBatch, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for i, group := range groups {
		groupID := strings.TrimSpace(group.GroupID)
		if groupID == "" {
			return nil, ErrScanBatchEmptyGroupID
		}
		if _, exists := seen[groupID]; exists {
			return nil, ErrScanBatchDuplicateGroupID
		}
		seen[groupID] = struct{}{}

		for _, post := range group.Posts {
			postGroupID := strings.TrimSpace(post.GroupID)
			if postGroupID != "" && postGroupID != groupID {
				return nil, ErrScanBatchPostGroupIDMismatch
			}
		}

		validated[i] = GroupBatch{
			GroupID:   groupID,
			GroupName: group.GroupName,
			Posts:     copyRawPosts(group.Posts),
		}
	}

	return validated, nil
}

func flattenBatchPosts(groups []GroupBatch) ([]domain.RawPost, []int) {
	var posts []domain.RawPost
	var groupIndexes []int
	for groupIndex, group := range groups {
		for _, post := range group.Posts {
			posts = append(posts, post)
			groupIndexes = append(groupIndexes, groupIndex)
		}
	}
	return posts, groupIndexes
}

func buildBatchSummary(groupCount int, inputPostCount int, pipeline EvaluationPipelineResult) ScanBatchSummary {
	allowed := pipeline.Allowed()
	blocked := pipeline.Blocked()
	return ScanBatchSummary{
		GroupCount:                 groupCount,
		InputPostCount:             inputPostCount,
		EvaluatedPostCount:         len(pipeline.Evaluated()),
		IncludePostCount:           len(pipeline.Eligible()),
		ReviewPostCount:            len(pipeline.Review()),
		ExcludedPostCount:          len(pipeline.Excluded()),
		AggregatedLeadCount:        len(pipeline.AggregatedLeads()),
		AllowedLeadCount:           len(allowed),
		BlockedLeadCount:           len(blocked),
		UnresolvedLeadCount:        len(pipeline.Unresolved()),
		UnaggregatedPostCount:      len(pipeline.Unaggregated()),
		SourceConflictCount:        len(pipeline.Conflicts()),
		AllowedLeadSourcePostCount: countAllowedLeadSources(allowed),
		BlockedLeadSourcePostCount: countBlockedLeadSources(blocked),
	}
}

func buildGroupSummaries(groups []GroupBatch, groupIndexes []int, pipeline EvaluationPipelineResult) []GroupSummary {
	summaries := make([]GroupSummary, len(groups))
	for i, group := range groups {
		summaries[i] = GroupSummary{
			GroupID:        group.GroupID,
			InputPostCount: len(group.Posts),
		}
	}

	evaluated := pipeline.Evaluated()
	for i, post := range evaluated {
		if i >= len(groupIndexes) {
			break
		}
		summary := &summaries[groupIndexes[i]]
		summary.EvaluatedPostCount++
		switch post.Result.Decision {
		case rules.DecisionInclude:
			summary.IncludePostCount++
		case rules.DecisionReview:
			summary.ReviewPostCount++
		default:
			summary.ExcludedPostCount++
		}
	}
	return summaries
}

func countAllowedLeadSources(leads []AllowedLead) int {
	count := 0
	for _, lead := range leads {
		count += len(lead.Lead.Sources())
	}
	return count
}

func countBlockedLeadSources(leads []BlockedLead) int {
	count := 0
	for _, lead := range leads {
		count += len(lead.Lead.Sources())
	}
	return count
}

func copyGroupBatches(groups []GroupBatch) []GroupBatch {
	if len(groups) == 0 {
		return nil
	}
	copied := make([]GroupBatch, len(groups))
	for i, group := range groups {
		copied[i] = group
		copied[i].Posts = copyRawPosts(group.Posts)
	}
	return copied
}

func copyRawPosts(posts []domain.RawPost) []domain.RawPost {
	if len(posts) == 0 {
		return nil
	}
	copied := make([]domain.RawPost, len(posts))
	copy(copied, posts)
	return copied
}

func copyGroupSummaries(summaries []GroupSummary) []GroupSummary {
	if len(summaries) == 0 {
		return nil
	}
	copied := make([]GroupSummary, len(summaries))
	copy(copied, summaries)
	return copied
}
