package rules

import "github.com/soleda2026/ScanFB/internal/domain"

// EvaluatePost applies Phase 3A time and author rules in deterministic order.
func EvaluatePost(post domain.RawPost, window domain.ScanWindow) Result {
	return combineResults(
		EvaluatePostTime(post, window),
		EvaluateAuthor(post.Author),
	)
}

// EvaluatePostForBuyerSearch applies time, author, and buyer-intent rules in deterministic order.
func EvaluatePostForBuyerSearch(post domain.RawPost, window domain.ScanWindow, profile domain.SearchProfile) Result {
	return combineResults(
		EvaluatePostTime(post, window),
		EvaluateAuthor(post.Author),
		EvaluateBuyerIntent(post, profile),
	)
}

// EvaluatePostForBuyerSearchAndGeography applies time, author, buyer-intent, and geography rules in deterministic order.
func EvaluatePostForBuyerSearchAndGeography(post domain.RawPost, window domain.ScanWindow, profile domain.SearchProfile, mode domain.GeographicMode) Result {
	return combineResults(
		EvaluatePostTime(post, window),
		EvaluateAuthor(post.Author),
		EvaluateBuyerIntent(post, profile),
		EvaluateGeographicMode(post, mode),
	)
}
