package rules

import "github.com/soleda2026/ScanFB/internal/domain"

// EvaluatePostTime checks RawPost.CreatedAt against the supplied ScanWindow.
func EvaluatePostTime(post domain.RawPost, window domain.ScanWindow) Result {
	if post.CreatedAt.IsZero() {
		return excludeResult(ReasonPostCreatedAtMissing)
	}

	createdAt := post.CreatedAt
	if createdAt.Before(window.StartOfDay()) {
		return excludeResult(ReasonPostBeforeScanWindow)
	}
	if createdAt.After(window.ScanStarted()) {
		return excludeResult(ReasonPostAfterScanStart)
	}

	return includeResult()
}
