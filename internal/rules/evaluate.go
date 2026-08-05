package rules

import "github.com/soleda2026/ScanFB/internal/domain"

// EvaluatePost applies Phase 3A time and author rules in deterministic order.
func EvaluatePost(post domain.RawPost, window domain.ScanWindow) Result {
	return combineResults(
		EvaluatePostTime(post, window),
		EvaluateAuthor(post.Author),
	)
}
