package rules

// Decision is the machine-readable outcome of a limited deterministic rule set.
type Decision string

const (
	DecisionInclude Decision = "include"
	DecisionExclude Decision = "exclude"
	DecisionReview  Decision = "review"
)

// ReasonCode is a stable machine-readable reason identifier.
type ReasonCode string

const (
	// ReasonPostBeforeScanWindow means RawPost.CreatedAt is before the ScanWindow start of day.
	ReasonPostBeforeScanWindow ReasonCode = "excluded.previous_day"
	// ReasonPostAfterScanStart means RawPost.CreatedAt is after the user-triggered scan start.
	ReasonPostAfterScanStart ReasonCode = "excluded.post_after_scan_start"
	// ReasonPostCreatedAtMissing means RawPost.CreatedAt is zero or unusable.
	ReasonPostCreatedAtMissing ReasonCode = "excluded.post_created_at_missing"
	// ReasonAnonymousAuthor means the author display name is a documented anonymous label.
	ReasonAnonymousAuthor ReasonCode = "excluded.anonymous_author"
	// ReasonAuthorDisplayNameMissing means the author display name is empty after trimming.
	ReasonAuthorDisplayNameMissing ReasonCode = "excluded.author_display_name_missing"
	// ReasonAuthorNameHasNoWhitespace means the trimmed display name contains no whitespace.
	ReasonAuthorNameHasNoWhitespace ReasonCode = "excluded.author_name_has_no_space"
	// ReasonPostBodyMissing means RawPost.Body is empty after trimming.
	ReasonPostBodyMissing ReasonCode = "excluded.post_body_missing"
	// ReasonTargetKeywordMissing means no SearchProfile product term matched RawPost.Body.
	ReasonTargetKeywordMissing ReasonCode = "excluded.target_keyword_missing"
	// ReasonBuyerIntentMissing means no SearchProfile buyer-intent term matched RawPost.Body.
	ReasonBuyerIntentMissing ReasonCode = "excluded.buyer_intent_missing"
	// ReasonSellerIntent means a SearchProfile seller/noise term matched RawPost.Body.
	ReasonSellerIntent ReasonCode = "excluded.seller_intent"
	// ReasonLocationUnknown means RawPost.Body does not match approved MVP geographic vocabulary.
	ReasonLocationUnknown ReasonCode = "review.unknown_location"
	// ReasonLocationConflict means RawPost.Body matches incompatible domestic geographic terms.
	ReasonLocationConflict ReasonCode = "review.location_conflict"
	// ReasonOutsideSelectedGeographicMode means location is outside the selected geographic mode.
	ReasonOutsideSelectedGeographicMode ReasonCode = "excluded.outside_scope"
	// ReasonHoChiMinhCityRequired means HCM mode was selected but no HCM geography matched.
	ReasonHoChiMinhCityRequired ReasonCode = "excluded.hcm_required_not_matched"
	// ReasonOutsideHoChiMinhCityVNRequired means outside-HCM Vietnam mode was selected but no outside-HCM geography matched.
	ReasonOutsideHoChiMinhCityVNRequired ReasonCode = "excluded.outside_hcm_vietnam_required_not_matched"
)

// Result contains the decision and stable reason codes for a limited rule evaluation.
type Result struct {
	Decision Decision
	Reasons  []ReasonCode
}

func includeResult() Result {
	return Result{Decision: DecisionInclude}
}

func excludeResult(reasons ...ReasonCode) Result {
	return Result{Decision: DecisionExclude, Reasons: copyReasons(reasons)}
}

func reviewResult(reasons ...ReasonCode) Result {
	return Result{Decision: DecisionReview, Reasons: copyReasons(reasons)}
}

func combineResults(results ...Result) Result {
	var excludeReasons []ReasonCode
	var reviewReasons []ReasonCode
	seenExclude := make(map[ReasonCode]struct{})
	seenReview := make(map[ReasonCode]struct{})
	for _, result := range results {
		switch result.Decision {
		case DecisionExclude:
			for _, reason := range result.Reasons {
				if _, ok := seenExclude[reason]; ok {
					continue
				}
				seenExclude[reason] = struct{}{}
				excludeReasons = append(excludeReasons, reason)
			}
		case DecisionReview:
			for _, reason := range result.Reasons {
				if _, ok := seenReview[reason]; ok {
					continue
				}
				seenReview[reason] = struct{}{}
				reviewReasons = append(reviewReasons, reason)
			}
		}
	}
	if len(excludeReasons) > 0 {
		return excludeResult(excludeReasons...)
	}
	if len(reviewReasons) > 0 {
		return reviewResult(reviewReasons...)
	}
	return includeResult()
}

func copyReasons(reasons []ReasonCode) []ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}
