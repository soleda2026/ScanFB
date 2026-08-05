package rules

// Decision is the machine-readable outcome of a limited deterministic rule set.
type Decision string

const (
	DecisionInclude Decision = "include"
	DecisionExclude Decision = "exclude"
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

func combineResults(results ...Result) Result {
	var reasons []ReasonCode
	seen := make(map[ReasonCode]struct{})
	for _, result := range results {
		if result.Decision == DecisionExclude {
			for _, reason := range result.Reasons {
				if _, ok := seen[reason]; ok {
					continue
				}
				seen[reason] = struct{}{}
				reasons = append(reasons, reason)
			}
		}
	}
	if len(reasons) == 0 {
		return includeResult()
	}
	return excludeResult(reasons...)
}

func copyReasons(reasons []ReasonCode) []ReasonCode {
	if len(reasons) == 0 {
		return nil
	}
	copied := make([]ReasonCode, len(reasons))
	copy(copied, reasons)
	return copied
}
