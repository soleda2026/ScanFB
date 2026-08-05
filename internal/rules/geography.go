package rules

import "github.com/soleda2026/ScanFB/internal/domain"

// GeographicClass is the deterministic MVP classification of RawPost.Body.
type GeographicClass string

const (
	GeographicClassHoChiMinhCity          GeographicClass = "hcm"
	GeographicClassOutsideHoChiMinhCityVN GeographicClass = "non_hcm_vietnam"
	GeographicClassUnknown                GeographicClass = "unknown"
	GeographicClassConflict               GeographicClass = "conflict"
)

// GeographicClassification contains deterministic geography output.
type GeographicClassification struct {
	Class   GeographicClass
	Reasons []ReasonCode
}

var hcmGeographicTerms = []string{
	"HCM",
	"TPHCM",
	"TP.HCM",
	"Ho Chi Minh",
	"Sai Gon",
	"Saigon",
}

var outsideHCMVietnamGeographicTerms = []string{
	"Hà Nội",
	"Ha Noi",
	"Đà Nẵng",
	"Da Nang",
	"Cần Thơ",
	"Can Tho",
}

// ClassifyGeography classifies RawPost.Body using only approved MVP vocabulary.
func ClassifyGeography(post domain.RawPost) GeographicClassification {
	body := normalizeText(post.Body)
	hcmMatches := matchedTerms(body, hcmGeographicTerms)
	outsideHCMMatches := matchedTerms(body, outsideHCMVietnamGeographicTerms)

	switch {
	case len(hcmMatches) > 0 && len(outsideHCMMatches) == 0:
		return GeographicClassification{
			Class: GeographicClassHoChiMinhCity,
		}
	case len(outsideHCMMatches) > 0 && len(hcmMatches) == 0:
		return GeographicClassification{
			Class: GeographicClassOutsideHoChiMinhCityVN,
		}
	case len(hcmMatches) > 0 && len(outsideHCMMatches) > 0:
		return GeographicClassification{
			Class:   GeographicClassConflict,
			Reasons: []ReasonCode{ReasonLocationConflict},
		}
	default:
		return GeographicClassification{
			Class:   GeographicClassUnknown,
			Reasons: []ReasonCode{ReasonLocationUnknown},
		}
	}
}

// EvaluateGeographicMode applies a selected GeographicMode to RawPost.Body geography.
func EvaluateGeographicMode(post domain.RawPost, mode domain.GeographicMode) Result {
	classification := ClassifyGeography(post)

	switch classification.Class {
	case GeographicClassUnknown:
		return reviewResult(ReasonLocationUnknown)
	case GeographicClassConflict:
		return reviewResult(ReasonLocationConflict)
	}

	switch mode {
	case domain.GeographicModeHoChiMinhCity:
		if classification.Class == GeographicClassHoChiMinhCity {
			return includeResult()
		}
		return excludeResult(ReasonOutsideSelectedGeographicMode, ReasonHoChiMinhCityRequired)
	case domain.GeographicModeOutsideHoChiMinhCityVN:
		if classification.Class == GeographicClassOutsideHoChiMinhCityVN {
			return includeResult()
		}
		return excludeResult(ReasonOutsideSelectedGeographicMode, ReasonOutsideHoChiMinhCityVNRequired)
	case domain.GeographicModeAllVietnam:
		return includeResult()
	default:
		return reviewResult(ReasonLocationUnknown)
	}
}

func matchedTerms(normalizedBody string, terms []string) []string {
	var matches []string
	for _, term := range terms {
		normalizedTerm := normalizeText(term)
		if normalizedTerm == "" {
			continue
		}
		if containsTerm(normalizedBody, normalizedTerm) {
			matches = append(matches, term)
		}
	}
	return matches
}
