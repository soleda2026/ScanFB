package rules

import (
	"strings"
	"unicode"

	"github.com/soleda2026/ScanFB/internal/domain"
)

// EvaluateBuyerIntent checks RawPost.Body against the active buyer SearchProfile.
func EvaluateBuyerIntent(post domain.RawPost, profile domain.SearchProfile) Result {
	body := normalizeText(post.Body)
	if body == "" {
		return excludeResult(ReasonPostBodyMissing)
	}

	if hasAnyTerm(body, profile.NoiseTerms()) {
		return excludeResult(ReasonSellerIntent)
	}

	var reasons []ReasonCode
	if !hasAnyTerm(body, profile.ProductTerms()) {
		reasons = append(reasons, ReasonTargetKeywordMissing)
	}
	if !hasAnyTerm(body, profile.BuyerIntentTerms()) {
		reasons = append(reasons, ReasonBuyerIntentMissing)
	}
	if len(reasons) > 0 {
		return excludeResult(reasons...)
	}

	return includeResult()
}

func hasAnyTerm(normalizedBody string, terms []string) bool {
	for _, term := range terms {
		normalizedTerm := normalizeText(term)
		if normalizedTerm == "" {
			continue
		}
		if containsTerm(normalizedBody, normalizedTerm) {
			return true
		}
	}
	return false
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
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
