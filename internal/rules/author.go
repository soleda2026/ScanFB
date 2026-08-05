package rules

import (
	"strings"
	"unicode"

	"github.com/soleda2026/ScanFB/internal/domain"
)

var anonymousDisplayNames = []string{
	"anonymous",
	"thanh vien an danh",
	"nguoi tham gia an danh",
}

// EvaluateAuthor checks deterministic anonymous and no-whitespace author exclusions.
func EvaluateAuthor(author domain.AuthorIdentity) Result {
	displayName := strings.TrimSpace(author.DisplayName)
	if displayName == "" {
		return excludeResult(ReasonAuthorDisplayNameMissing)
	}

	normalized := strings.ToLower(displayName)
	for _, anonymousName := range anonymousDisplayNames {
		if normalized == anonymousName {
			return excludeResult(ReasonAnonymousAuthor)
		}
	}

	if !containsWhitespace(displayName) {
		return excludeResult(ReasonAuthorNameHasNoWhitespace)
	}

	return includeResult()
}

func containsWhitespace(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
