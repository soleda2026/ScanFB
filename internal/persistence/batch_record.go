package persistence

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

var (
	ErrEmptyBatchRecordID       = errors.New("empty batch record id")
	ErrInvalidBatchRecord       = errors.New("invalid batch record")
	ErrInconsistentBatchSummary = errors.New("inconsistent batch summary")
	ErrUnsupportedDecision      = errors.New("unsupported decision")
	ErrUnsupportedLeadOutcome   = errors.New("unsupported lead outcome")
	ErrUnsupportedBlockOutcome  = errors.New("unsupported blocklist outcome")
)

// BatchRepository is the minimal persistence-facing contract for completed
// batch snapshots. Implementations must save the whole record atomically.
type BatchRepository interface {
	SaveBatch(record BatchRecord) error
}

// BatchRecordID is an opaque identifier supplied by a future application boundary.
type BatchRecordID string

// NewBatchRecordID validates an externally supplied batch record identifier.
func NewBatchRecordID(value string) (BatchRecordID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrEmptyBatchRecordID
	}
	return BatchRecordID(value), nil
}

func (id BatchRecordID) String() string {
	return string(id)
}

type ScanWindowRecord struct {
	ScanDate    time.Time
	StartOfDay  time.Time
	ScanStarted time.Time
	Timezone    string
}

type SearchProfileRecord struct {
	ID               string
	DisplayName      string
	ProductTerms     []string
	BuyerIntentTerms []string
	NoiseTerms       []string
	IsEnabled        bool
}

type GroupRecord struct {
	GroupID   string
	GroupName string
	Posts     []domain.RawPost
}

type IdentityRecord struct {
	Kind  string
	Value string
}

type SourceIdentityRecord struct {
	Kind  string
	Value string
}

type NeedRecord struct {
	SearchProfileID     string
	NormalizedBody      string
	BodyFingerprint     string
	ProductEvidence     []string
	BuyerIntentEvidence []string
}

type LeadKeyRecord struct {
	Value  string
	Author IdentityRecord
	Need   NeedRecord
}

type SourcePostRecord struct {
	Key  SourceIdentityRecord
	Post domain.RawPost
}

type LeadRecord struct {
	Key     LeadKeyRecord
	Author  IdentityRecord
	Need    NeedRecord
	Sources []SourcePostRecord
}

type CandidateRecord struct {
	Author IdentityRecord
	Need   NeedRecord
}

type EvaluatedPostRecord struct {
	Post              domain.RawPost
	Decision          string
	Reasons           []string
	GeographicClass   string
	GeographicReasons []string
}

type BlocklistEntryRecord struct {
	Key         IdentityRecord
	DisplayName string
}

type BlocklistMatchRecord struct {
	Outcome      string
	Reasons      []string
	AuthorKey    IdentityRecord
	MatchedEntry BlocklistEntryRecord
}

type AllowedLeadRecord struct {
	Lead  LeadRecord
	Match BlocklistMatchRecord
}

type BlockedLeadRecord struct {
	Lead  LeadRecord
	Match BlocklistMatchRecord
}

type UnresolvedLeadRecord struct {
	Lead               LeadRecord
	Match              BlocklistMatchRecord
	ApplicationReasons []string
}

type UnaggregatedPostRecord struct {
	Post      domain.RawPost
	Candidate CandidateRecord
	Source    SourceIdentityRecord
	Reasons   []string
}

type SourceConflictRecord struct {
	Post           domain.RawPost
	ExistingSource SourcePostRecord
	Candidate      CandidateRecord
	Source         SourceIdentityRecord
	Reasons        []string
}

type BatchSummaryRecord struct {
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

type GroupSummaryRecord struct {
	GroupID            string
	InputPostCount     int
	EvaluatedPostCount int
	IncludePostCount   int
	ReviewPostCount    int
	ExcludedPostCount  int
}

// BatchRecord is an immutable completed scan-batch snapshot prepared for a
// future local storage adapter. It intentionally exposes no update, delete,
// search, pagination, migration, transaction, or ID-generation behavior.
type BatchRecord struct {
	id              BatchRecordID
	scanWindow      ScanWindowRecord
	searchProfile   SearchProfileRecord
	geographicMode  string
	groups          []GroupRecord
	flattenedPosts  []domain.RawPost
	evaluatedPosts  []EvaluatedPostRecord
	includedPosts   []EvaluatedPostRecord
	reviewPosts     []EvaluatedPostRecord
	excludedPosts   []EvaluatedPostRecord
	leads           []LeadRecord
	allowedLeads    []AllowedLeadRecord
	blockedLeads    []BlockedLeadRecord
	unresolvedLeads []UnresolvedLeadRecord
	unaggregated    []UnaggregatedPostRecord
	conflicts       []SourceConflictRecord
	summary         BatchSummaryRecord
	groupSummaries  []GroupSummaryRecord
}

// NewBatchRecord creates a deterministic completed-batch snapshot. It copies
// source data in input/result order and does not mutate or recompute decisions.
func NewBatchRecord(id BatchRecordID, input application.ScanBatchInput, result application.ScanBatchResult) (BatchRecord, error) {
	record := BatchRecord{
		id:             id,
		scanWindow:     scanWindowRecord(input.ScanWindow),
		searchProfile:  searchProfileRecord(input.SearchProfile),
		geographicMode: string(input.GeographicMode),
		groups:         groupRecords(result.Groups()),
		flattenedPosts: copyRawPosts(result.FlattenedPosts()),
		summary:        summaryRecord(result.Summary()),
		groupSummaries: groupSummaryRecords(result.GroupSummaries()),
	}

	pipeline := result.Pipeline()
	for _, evaluated := range pipeline.Evaluated() {
		record.evaluatedPosts = append(record.evaluatedPosts, evaluatedPostRecord(evaluated.Post, string(evaluated.Result.Decision), reasonStrings(evaluated.Result.Reasons)))
	}
	for _, included := range pipeline.Eligible() {
		record.includedPosts = append(record.includedPosts, evaluatedPostRecord(included.Post, string(included.Result.Decision), reasonStrings(included.Result.Reasons)))
	}
	for _, review := range pipeline.Review() {
		record.reviewPosts = append(record.reviewPosts, evaluatedPostRecord(review.Post, string(review.Result.Decision), reasonStrings(review.Result.Reasons)))
	}
	for _, excluded := range pipeline.Excluded() {
		record.excludedPosts = append(record.excludedPosts, evaluatedPostRecord(excluded.Post, string(excluded.Result.Decision), reasonStrings(excluded.Result.Reasons)))
	}

	for _, lead := range pipeline.AggregatedLeads() {
		sources := lead.Sources()
		sourceRecords := make([]SourcePostRecord, len(sources))
		for i, source := range sources {
			sourceRecords[i] = SourcePostRecord{
				Key:  SourceIdentityRecord{Kind: string(source.Key.Kind), Value: source.Key.Value},
				Post: source.Post,
			}
		}
		record.leads = append(record.leads, leadRecordFromLeadParts(
			lead.Key.Value,
			string(lead.Author.Kind), lead.Author.Value,
			lead.Need.SearchProfileID, lead.Need.NormalizedBody, lead.Need.BodyFingerprint,
			copyStrings(lead.Need.ProductEvidence), copyStrings(lead.Need.BuyerIntentEvidence),
			sourceRecords,
		))
	}
	for _, allowed := range pipeline.Allowed() {
		lead := allowed.Lead
		sources := lead.Sources()
		sourceRecords := make([]SourcePostRecord, len(sources))
		for i, source := range sources {
			sourceRecords[i] = SourcePostRecord{
				Key:  SourceIdentityRecord{Kind: string(source.Key.Kind), Value: source.Key.Value},
				Post: source.Post,
			}
		}
		match := allowed.Match
		record.allowedLeads = append(record.allowedLeads, AllowedLeadRecord{
			Lead: leadRecordFromLeadParts(
				lead.Key.Value,
				string(lead.Author.Kind), lead.Author.Value,
				lead.Need.SearchProfileID, lead.Need.NormalizedBody, lead.Need.BodyFingerprint,
				copyStrings(lead.Need.ProductEvidence), copyStrings(lead.Need.BuyerIntentEvidence),
				sourceRecords,
			),
			Match: BlocklistMatchRecord{
				Outcome: string(match.Outcome),
				Reasons: blocklistReasonStrings(match.Reasons),
				AuthorKey: IdentityRecord{
					Kind:  string(match.AuthorKey.Kind),
					Value: match.AuthorKey.Value,
				},
				MatchedEntry: BlocklistEntryRecord{
					Key: IdentityRecord{
						Kind:  string(match.MatchedEntry.Key().Kind),
						Value: match.MatchedEntry.Key().Value,
					},
					DisplayName: match.MatchedEntry.DisplayName(),
				},
			},
		})
	}
	for _, blocked := range pipeline.Blocked() {
		lead := blocked.Lead
		sources := lead.Sources()
		sourceRecords := make([]SourcePostRecord, len(sources))
		for i, source := range sources {
			sourceRecords[i] = SourcePostRecord{
				Key:  SourceIdentityRecord{Kind: string(source.Key.Kind), Value: source.Key.Value},
				Post: source.Post,
			}
		}
		match := blocked.Match
		record.blockedLeads = append(record.blockedLeads, BlockedLeadRecord{
			Lead: leadRecordFromLeadParts(
				lead.Key.Value,
				string(lead.Author.Kind), lead.Author.Value,
				lead.Need.SearchProfileID, lead.Need.NormalizedBody, lead.Need.BodyFingerprint,
				copyStrings(lead.Need.ProductEvidence), copyStrings(lead.Need.BuyerIntentEvidence),
				sourceRecords,
			),
			Match: BlocklistMatchRecord{
				Outcome: string(match.Outcome),
				Reasons: blocklistReasonStrings(match.Reasons),
				AuthorKey: IdentityRecord{
					Kind:  string(match.AuthorKey.Kind),
					Value: match.AuthorKey.Value,
				},
				MatchedEntry: BlocklistEntryRecord{
					Key: IdentityRecord{
						Kind:  string(match.MatchedEntry.Key().Kind),
						Value: match.MatchedEntry.Key().Value,
					},
					DisplayName: match.MatchedEntry.DisplayName(),
				},
			},
		})
	}
	for _, unresolved := range pipeline.Unresolved() {
		lead := unresolved.Lead
		sources := lead.Sources()
		sourceRecords := make([]SourcePostRecord, len(sources))
		for i, source := range sources {
			sourceRecords[i] = SourcePostRecord{
				Key:  SourceIdentityRecord{Kind: string(source.Key.Kind), Value: source.Key.Value},
				Post: source.Post,
			}
		}
		match := unresolved.Match
		record.unresolvedLeads = append(record.unresolvedLeads, UnresolvedLeadRecord{
			Lead: leadRecordFromLeadParts(
				lead.Key.Value,
				string(lead.Author.Kind), lead.Author.Value,
				lead.Need.SearchProfileID, lead.Need.NormalizedBody, lead.Need.BodyFingerprint,
				copyStrings(lead.Need.ProductEvidence), copyStrings(lead.Need.BuyerIntentEvidence),
				sourceRecords,
			),
			Match: BlocklistMatchRecord{
				Outcome: string(match.Outcome),
				Reasons: blocklistReasonStrings(match.Reasons),
				AuthorKey: IdentityRecord{
					Kind:  string(match.AuthorKey.Kind),
					Value: match.AuthorKey.Value,
				},
				MatchedEntry: BlocklistEntryRecord{
					Key: IdentityRecord{
						Kind:  string(match.MatchedEntry.Key().Kind),
						Value: match.MatchedEntry.Key().Value,
					},
					DisplayName: match.MatchedEntry.DisplayName(),
				},
			},
			ApplicationReasons: applicationReasonStrings(unresolved.Reasons),
		})
	}
	for _, unaggregated := range pipeline.Unaggregated() {
		record.unaggregated = append(record.unaggregated, UnaggregatedPostRecord{
			Post: unaggregated.Post,
			Candidate: candidateRecord(
				string(unaggregated.Candidate.Author.Kind), unaggregated.Candidate.Author.Value,
				unaggregated.Candidate.Need.SearchProfileID, unaggregated.Candidate.Need.NormalizedBody, unaggregated.Candidate.Need.BodyFingerprint,
				copyStrings(unaggregated.Candidate.Need.ProductEvidence), copyStrings(unaggregated.Candidate.Need.BuyerIntentEvidence),
			),
			Source:  SourceIdentityRecord{Kind: string(unaggregated.Source.Kind), Value: unaggregated.Source.Value},
			Reasons: dedupReasonStrings(unaggregated.Reasons),
		})
	}
	for _, conflict := range pipeline.Conflicts() {
		record.conflicts = append(record.conflicts, SourceConflictRecord{
			Post:           conflict.Post,
			ExistingSource: SourcePostRecord{Key: SourceIdentityRecord{Kind: string(conflict.ExistingSource.Key.Kind), Value: conflict.ExistingSource.Key.Value}, Post: conflict.ExistingSource.Post},
			Candidate: candidateRecord(
				string(conflict.Candidate.Author.Kind), conflict.Candidate.Author.Value,
				conflict.Candidate.Need.SearchProfileID, conflict.Candidate.Need.NormalizedBody, conflict.Candidate.Need.BodyFingerprint,
				copyStrings(conflict.Candidate.Need.ProductEvidence), copyStrings(conflict.Candidate.Need.BuyerIntentEvidence),
			),
			Source:  SourceIdentityRecord{Kind: string(conflict.Source.Kind), Value: conflict.Source.Value},
			Reasons: dedupReasonStrings(conflict.Reasons),
		})
	}

	if err := record.Validate(); err != nil {
		return BatchRecord{}, err
	}
	return record, nil
}

func (record BatchRecord) ID() BatchRecordID {
	return record.id
}

func (record BatchRecord) ScanWindow() ScanWindowRecord {
	return record.scanWindow
}

func (record BatchRecord) SearchProfile() SearchProfileRecord {
	profile := record.searchProfile
	profile.ProductTerms = copyStrings(profile.ProductTerms)
	profile.BuyerIntentTerms = copyStrings(profile.BuyerIntentTerms)
	profile.NoiseTerms = copyStrings(profile.NoiseTerms)
	return profile
}

func (record BatchRecord) GeographicMode() string {
	return record.geographicMode
}

func (record BatchRecord) Groups() []GroupRecord {
	return copyGroupRecords(record.groups)
}

func (record BatchRecord) FlattenedPosts() []domain.RawPost {
	return copyRawPosts(record.flattenedPosts)
}

func (record BatchRecord) EvaluatedPosts() []EvaluatedPostRecord {
	return copyEvaluatedPostRecords(record.evaluatedPosts)
}

func (record BatchRecord) IncludedPosts() []EvaluatedPostRecord {
	return copyEvaluatedPostRecords(record.includedPosts)
}

func (record BatchRecord) ReviewPosts() []EvaluatedPostRecord {
	return copyEvaluatedPostRecords(record.reviewPosts)
}

func (record BatchRecord) ExcludedPosts() []EvaluatedPostRecord {
	return copyEvaluatedPostRecords(record.excludedPosts)
}

func (record BatchRecord) Leads() []LeadRecord {
	return copyLeadRecords(record.leads)
}

func (record BatchRecord) AllowedLeads() []AllowedLeadRecord {
	return copyAllowedLeadRecords(record.allowedLeads)
}

func (record BatchRecord) BlockedLeads() []BlockedLeadRecord {
	return copyBlockedLeadRecords(record.blockedLeads)
}

func (record BatchRecord) UnresolvedLeads() []UnresolvedLeadRecord {
	return copyUnresolvedLeadRecords(record.unresolvedLeads)
}

func (record BatchRecord) Unaggregated() []UnaggregatedPostRecord {
	return copyUnaggregatedPostRecords(record.unaggregated)
}

func (record BatchRecord) Conflicts() []SourceConflictRecord {
	return copySourceConflictRecords(record.conflicts)
}

func (record BatchRecord) Summary() BatchSummaryRecord {
	return record.summary
}

func (record BatchRecord) GroupSummaries() []GroupSummaryRecord {
	return append([]GroupSummaryRecord(nil), record.groupSummaries...)
}

func (record BatchRecord) Validate() error {
	if strings.TrimSpace(record.id.String()) == "" {
		return ErrEmptyBatchRecordID
	}
	if err := validateScanWindowRecord(record.scanWindow); err != nil {
		return err
	}
	if err := validateSearchProfileRecord(record.searchProfile); err != nil {
		return err
	}
	if !domain.GeographicMode(record.geographicMode).Valid() {
		return fmt.Errorf("%w: geographic mode", ErrInvalidBatchRecord)
	}
	if err := validateGroups(record.groups, record.flattenedPosts); err != nil {
		return err
	}
	if err := validateDecisions(record.evaluatedPosts, "", false); err != nil {
		return err
	}
	if err := validateDecisions(record.includedPosts, "include", true); err != nil {
		return err
	}
	if err := validateDecisions(record.reviewPosts, "review", true); err != nil {
		return err
	}
	if err := validateDecisions(record.excludedPosts, "exclude", true); err != nil {
		return err
	}
	if err := validateLeadRecords(record.leads); err != nil {
		return err
	}
	for _, allowed := range record.allowedLeads {
		if err := validateLeadRecord(allowed.Lead); err != nil {
			return err
		}
		if err := validateBlocklistMatch(allowed.Match, "not_blocked"); err != nil {
			return err
		}
	}
	for _, blocked := range record.blockedLeads {
		if err := validateLeadRecord(blocked.Lead); err != nil {
			return err
		}
		if err := validateBlocklistMatch(blocked.Match, "blocked"); err != nil {
			return err
		}
	}
	for _, unresolved := range record.unresolvedLeads {
		if err := validateLeadRecord(unresolved.Lead); err != nil {
			return err
		}
		if err := validateBlocklistMatch(unresolved.Match, "insufficient_identity"); err != nil {
			return err
		}
	}
	for _, unaggregated := range record.unaggregated {
		if len(unaggregated.Reasons) == 0 {
			return fmt.Errorf("%w: unaggregated reasons", ErrInvalidBatchRecord)
		}
	}
	for _, conflict := range record.conflicts {
		if !sourceIdentityPresent(conflict.ExistingSource.Key) || !sourceIdentityPresent(conflict.Source) {
			return fmt.Errorf("%w: conflict source identity", ErrInvalidBatchRecord)
		}
	}
	return validateSummary(record)
}

func validateScanWindowRecord(window ScanWindowRecord) error {
	if window.ScanDate.IsZero() || window.StartOfDay.IsZero() || window.ScanStarted.IsZero() || strings.TrimSpace(window.Timezone) == "" {
		return fmt.Errorf("%w: scan window", ErrInvalidBatchRecord)
	}
	if window.StartOfDay.After(window.ScanStarted) {
		return fmt.Errorf("%w: scan window", ErrInvalidBatchRecord)
	}
	if window.StartOfDay.Hour() != 0 || window.StartOfDay.Minute() != 0 || window.StartOfDay.Second() != 0 || window.StartOfDay.Nanosecond() != 0 {
		return fmt.Errorf("%w: scan window", ErrInvalidBatchRecord)
	}
	if !sameDate(window.ScanDate, window.StartOfDay) || !sameDate(window.ScanDate, window.ScanStarted) {
		return fmt.Errorf("%w: scan window", ErrInvalidBatchRecord)
	}
	return nil
}

func validateSearchProfileRecord(profile SearchProfileRecord) error {
	if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.DisplayName) == "" || !profile.IsEnabled || len(profile.ProductTerms) == 0 {
		return fmt.Errorf("%w: search profile", ErrInvalidBatchRecord)
	}
	for _, term := range profile.ProductTerms {
		if strings.TrimSpace(term) == "" {
			return fmt.Errorf("%w: search profile term", ErrInvalidBatchRecord)
		}
	}
	return nil
}

func validateGroups(groups []GroupRecord, flattened []domain.RawPost) error {
	if len(groups) == 0 || len(groups) > domain.MaxScanRequestGroups {
		return fmt.Errorf("%w: groups", ErrInvalidBatchRecord)
	}
	seen := map[string]struct{}{}
	total := 0
	for _, group := range groups {
		groupID := strings.TrimSpace(group.GroupID)
		if groupID == "" {
			return fmt.Errorf("%w: group id", ErrInvalidBatchRecord)
		}
		if _, ok := seen[groupID]; ok {
			return fmt.Errorf("%w: duplicate group id", ErrInvalidBatchRecord)
		}
		seen[groupID] = struct{}{}
		total += len(group.Posts)
		for _, post := range group.Posts {
			if post.GroupID != group.GroupID {
				return fmt.Errorf("%w: group post mismatch", ErrInvalidBatchRecord)
			}
		}
	}
	if total != len(flattened) {
		return ErrInconsistentBatchSummary
	}
	return nil
}

func validateDecisions(posts []EvaluatedPostRecord, expected string, requireExpected bool) error {
	for _, post := range posts {
		if !supportedDecision(post.Decision) {
			return ErrUnsupportedDecision
		}
		if requireExpected && post.Decision != expected {
			return ErrUnsupportedDecision
		}
		if post.Decision != "include" && len(post.Reasons) == 0 {
			return fmt.Errorf("%w: decision reasons", ErrInvalidBatchRecord)
		}
	}
	return nil
}

func validateLeadRecords(leads []LeadRecord) error {
	for _, lead := range leads {
		if err := validateLeadRecord(lead); err != nil {
			return err
		}
	}
	return nil
}

func validateLeadRecord(lead LeadRecord) error {
	if strings.TrimSpace(lead.Key.Value) == "" {
		return fmt.Errorf("%w: lead key", ErrInvalidBatchRecord)
	}
	if !identityPresent(lead.Author) || !identityPresent(lead.Key.Author) {
		return fmt.Errorf("%w: lead author identity", ErrInvalidBatchRecord)
	}
	if strings.TrimSpace(lead.Need.SearchProfileID) == "" || strings.TrimSpace(lead.Need.NormalizedBody) == "" || strings.TrimSpace(lead.Need.BodyFingerprint) == "" {
		return fmt.Errorf("%w: lead need", ErrInvalidBatchRecord)
	}
	if len(lead.Sources) == 0 {
		return fmt.Errorf("%w: lead sources", ErrInvalidBatchRecord)
	}
	for _, source := range lead.Sources {
		if !sourceIdentityPresent(source.Key) {
			return fmt.Errorf("%w: source identity", ErrInvalidBatchRecord)
		}
		if strings.TrimSpace(source.Post.GroupID) == "" || strings.TrimSpace(source.Post.Body) == "" {
			return fmt.Errorf("%w: source post", ErrInvalidBatchRecord)
		}
	}
	return nil
}

func validateBlocklistMatch(match BlocklistMatchRecord, expected string) error {
	if !supportedBlocklistOutcome(match.Outcome) {
		return ErrUnsupportedBlockOutcome
	}
	if match.Outcome != expected {
		return ErrUnsupportedLeadOutcome
	}
	if len(match.Reasons) == 0 {
		return fmt.Errorf("%w: blocklist reasons", ErrInvalidBatchRecord)
	}
	return nil
}

func validateSummary(record BatchRecord) error {
	summary := record.summary
	if summary.GroupCount != len(record.groups) ||
		summary.InputPostCount != len(record.flattenedPosts) ||
		summary.EvaluatedPostCount != len(record.evaluatedPosts) ||
		summary.IncludePostCount != len(record.includedPosts) ||
		summary.ReviewPostCount != len(record.reviewPosts) ||
		summary.ExcludedPostCount != len(record.excludedPosts) ||
		summary.AggregatedLeadCount != len(record.leads) ||
		summary.AllowedLeadCount != len(record.allowedLeads) ||
		summary.BlockedLeadCount != len(record.blockedLeads) ||
		summary.UnresolvedLeadCount != len(record.unresolvedLeads) ||
		summary.UnaggregatedPostCount != len(record.unaggregated) ||
		summary.SourceConflictCount != len(record.conflicts) {
		return ErrInconsistentBatchSummary
	}
	if summary.AllowedLeadSourcePostCount != sourceCountForAllowed(record.allowedLeads) ||
		summary.BlockedLeadSourcePostCount != sourceCountForBlocked(record.blockedLeads) {
		return ErrInconsistentBatchSummary
	}
	if len(record.groupSummaries) != len(record.groups) {
		return ErrInconsistentBatchSummary
	}
	var inputTotal, evaluatedTotal, includeTotal, reviewTotal, excludedTotal int
	for i, group := range record.groups {
		groupSummary := record.groupSummaries[i]
		if groupSummary.GroupID != group.GroupID || groupSummary.InputPostCount != len(group.Posts) {
			return ErrInconsistentBatchSummary
		}
		inputTotal += groupSummary.InputPostCount
		evaluatedTotal += groupSummary.EvaluatedPostCount
		includeTotal += groupSummary.IncludePostCount
		reviewTotal += groupSummary.ReviewPostCount
		excludedTotal += groupSummary.ExcludedPostCount
	}
	if inputTotal != summary.InputPostCount ||
		evaluatedTotal != summary.EvaluatedPostCount ||
		includeTotal != summary.IncludePostCount ||
		reviewTotal != summary.ReviewPostCount ||
		excludedTotal != summary.ExcludedPostCount {
		return ErrInconsistentBatchSummary
	}
	return nil
}

func supportedDecision(decision string) bool {
	switch decision {
	case "include", "exclude", "review":
		return true
	default:
		return false
	}
}

func supportedBlocklistOutcome(outcome string) bool {
	switch outcome {
	case "blocked", "not_blocked", "insufficient_identity":
		return true
	default:
		return false
	}
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func identityPresent(identity IdentityRecord) bool {
	return strings.TrimSpace(identity.Kind) != "" && strings.TrimSpace(identity.Value) != ""
}

func sourceIdentityPresent(identity SourceIdentityRecord) bool {
	return strings.TrimSpace(identity.Kind) != "" && strings.TrimSpace(identity.Value) != ""
}

func sourceCountForAllowed(leads []AllowedLeadRecord) int {
	var count int
	for _, lead := range leads {
		count += len(lead.Lead.Sources)
	}
	return count
}

func sourceCountForBlocked(leads []BlockedLeadRecord) int {
	var count int
	for _, lead := range leads {
		count += len(lead.Lead.Sources)
	}
	return count
}

func scanWindowRecord(window domain.ScanWindow) ScanWindowRecord {
	return ScanWindowRecord{
		ScanDate:    window.ScanDate(),
		StartOfDay:  window.StartOfDay(),
		ScanStarted: window.ScanStarted(),
		Timezone:    window.Timezone(),
	}
}

func searchProfileRecord(profile domain.SearchProfile) SearchProfileRecord {
	return SearchProfileRecord{
		ID:               profile.ID(),
		DisplayName:      profile.DisplayName(),
		ProductTerms:     copyStrings(profile.ProductTerms()),
		BuyerIntentTerms: copyStrings(profile.BuyerIntentTerms()),
		NoiseTerms:       copyStrings(profile.NoiseTerms()),
		IsEnabled:        profile.IsEnabled(),
	}
}

func groupRecords(groups []application.GroupBatch) []GroupRecord {
	records := make([]GroupRecord, len(groups))
	for i, group := range groups {
		records[i] = GroupRecord{
			GroupID:   group.GroupID,
			GroupName: group.GroupName,
			Posts:     copyRawPosts(group.Posts),
		}
	}
	return records
}

func evaluatedPostRecord(post domain.RawPost, decision string, reasons []string) EvaluatedPostRecord {
	return EvaluatedPostRecord{
		Post:     post,
		Decision: decision,
		Reasons:  copyStrings(reasons),
	}
}

func summaryRecord(summary application.ScanBatchSummary) BatchSummaryRecord {
	return BatchSummaryRecord{
		GroupCount:                 summary.GroupCount,
		InputPostCount:             summary.InputPostCount,
		EvaluatedPostCount:         summary.EvaluatedPostCount,
		IncludePostCount:           summary.IncludePostCount,
		ReviewPostCount:            summary.ReviewPostCount,
		ExcludedPostCount:          summary.ExcludedPostCount,
		AggregatedLeadCount:        summary.AggregatedLeadCount,
		AllowedLeadCount:           summary.AllowedLeadCount,
		BlockedLeadCount:           summary.BlockedLeadCount,
		UnresolvedLeadCount:        summary.UnresolvedLeadCount,
		UnaggregatedPostCount:      summary.UnaggregatedPostCount,
		SourceConflictCount:        summary.SourceConflictCount,
		AllowedLeadSourcePostCount: summary.AllowedLeadSourcePostCount,
		BlockedLeadSourcePostCount: summary.BlockedLeadSourcePostCount,
	}
}

func groupSummaryRecords(summaries []application.GroupSummary) []GroupSummaryRecord {
	records := make([]GroupSummaryRecord, len(summaries))
	for i, summary := range summaries {
		records[i] = GroupSummaryRecord{
			GroupID:            summary.GroupID,
			InputPostCount:     summary.InputPostCount,
			EvaluatedPostCount: summary.EvaluatedPostCount,
			IncludePostCount:   summary.IncludePostCount,
			ReviewPostCount:    summary.ReviewPostCount,
			ExcludedPostCount:  summary.ExcludedPostCount,
		}
	}
	return records
}

func leadRecordFromLeadParts(keyValue, authorKind, authorValue, profileID, normalizedBody, fingerprint string, productEvidence, buyerIntentEvidence []string, sources []SourcePostRecord) LeadRecord {
	author := IdentityRecord{Kind: authorKind, Value: authorValue}
	need := NeedRecord{
		SearchProfileID:     profileID,
		NormalizedBody:      normalizedBody,
		BodyFingerprint:     fingerprint,
		ProductEvidence:     copyStrings(productEvidence),
		BuyerIntentEvidence: copyStrings(buyerIntentEvidence),
	}
	return LeadRecord{
		Key:     LeadKeyRecord{Value: keyValue, Author: author, Need: need},
		Author:  author,
		Need:    need,
		Sources: copySourcePostRecords(sources),
	}
}

func candidateRecord(authorKind, authorValue, profileID, normalizedBody, fingerprint string, productEvidence, buyerIntentEvidence []string) CandidateRecord {
	return CandidateRecord{
		Author: IdentityRecord{Kind: authorKind, Value: authorValue},
		Need: NeedRecord{
			SearchProfileID:     profileID,
			NormalizedBody:      normalizedBody,
			BodyFingerprint:     fingerprint,
			ProductEvidence:     copyStrings(productEvidence),
			BuyerIntentEvidence: copyStrings(buyerIntentEvidence),
		},
	}
}

func reasonStrings[T ~string](reasons []T) []string {
	values := make([]string, len(reasons))
	for i, reason := range reasons {
		values[i] = string(reason)
	}
	return values
}

func applicationReasonStrings[T ~string](reasons []T) []string {
	return reasonStrings(reasons)
}

func blocklistReasonStrings[T ~string](reasons []T) []string {
	return reasonStrings(reasons)
}

func dedupReasonStrings[T ~string](reasons []T) []string {
	return reasonStrings(reasons)
}

func copyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func copyRawPosts(posts []domain.RawPost) []domain.RawPost {
	return append([]domain.RawPost(nil), posts...)
}

func copyGroupRecords(groups []GroupRecord) []GroupRecord {
	if len(groups) == 0 {
		return nil
	}
	copied := make([]GroupRecord, len(groups))
	for i, group := range groups {
		copied[i] = group
		copied[i].Posts = copyRawPosts(group.Posts)
	}
	return copied
}

func copyEvaluatedPostRecords(posts []EvaluatedPostRecord) []EvaluatedPostRecord {
	if len(posts) == 0 {
		return nil
	}
	copied := make([]EvaluatedPostRecord, len(posts))
	for i, post := range posts {
		copied[i] = post
		copied[i].Reasons = copyStrings(post.Reasons)
		copied[i].GeographicReasons = copyStrings(post.GeographicReasons)
	}
	return copied
}

func copyLeadRecords(leads []LeadRecord) []LeadRecord {
	if len(leads) == 0 {
		return nil
	}
	copied := make([]LeadRecord, len(leads))
	for i, lead := range leads {
		copied[i] = lead
		copied[i].Need.ProductEvidence = copyStrings(lead.Need.ProductEvidence)
		copied[i].Need.BuyerIntentEvidence = copyStrings(lead.Need.BuyerIntentEvidence)
		copied[i].Key.Need.ProductEvidence = copyStrings(lead.Key.Need.ProductEvidence)
		copied[i].Key.Need.BuyerIntentEvidence = copyStrings(lead.Key.Need.BuyerIntentEvidence)
		copied[i].Sources = copySourcePostRecords(lead.Sources)
	}
	return copied
}

func copySourcePostRecords(sources []SourcePostRecord) []SourcePostRecord {
	if len(sources) == 0 {
		return nil
	}
	return append([]SourcePostRecord(nil), sources...)
}

func copyAllowedLeadRecords(leads []AllowedLeadRecord) []AllowedLeadRecord {
	if len(leads) == 0 {
		return nil
	}
	copied := make([]AllowedLeadRecord, len(leads))
	for i, lead := range leads {
		copied[i] = lead
		copied[i].Lead = copyLeadRecords([]LeadRecord{lead.Lead})[0]
		copied[i].Match = copyBlocklistMatchRecord(lead.Match)
	}
	return copied
}

func copyBlockedLeadRecords(leads []BlockedLeadRecord) []BlockedLeadRecord {
	if len(leads) == 0 {
		return nil
	}
	copied := make([]BlockedLeadRecord, len(leads))
	for i, lead := range leads {
		copied[i] = lead
		copied[i].Lead = copyLeadRecords([]LeadRecord{lead.Lead})[0]
		copied[i].Match = copyBlocklistMatchRecord(lead.Match)
	}
	return copied
}

func copyUnresolvedLeadRecords(leads []UnresolvedLeadRecord) []UnresolvedLeadRecord {
	if len(leads) == 0 {
		return nil
	}
	copied := make([]UnresolvedLeadRecord, len(leads))
	for i, lead := range leads {
		copied[i] = lead
		copied[i].Lead = copyLeadRecords([]LeadRecord{lead.Lead})[0]
		copied[i].Match = copyBlocklistMatchRecord(lead.Match)
		copied[i].ApplicationReasons = copyStrings(lead.ApplicationReasons)
	}
	return copied
}

func copyBlocklistMatchRecord(match BlocklistMatchRecord) BlocklistMatchRecord {
	match.Reasons = copyStrings(match.Reasons)
	return match
}

func copyUnaggregatedPostRecords(posts []UnaggregatedPostRecord) []UnaggregatedPostRecord {
	if len(posts) == 0 {
		return nil
	}
	copied := make([]UnaggregatedPostRecord, len(posts))
	for i, post := range posts {
		copied[i] = post
		copied[i].Candidate = copyCandidateRecord(post.Candidate)
		copied[i].Reasons = copyStrings(post.Reasons)
	}
	return copied
}

func copySourceConflictRecords(conflicts []SourceConflictRecord) []SourceConflictRecord {
	if len(conflicts) == 0 {
		return nil
	}
	copied := make([]SourceConflictRecord, len(conflicts))
	for i, conflict := range conflicts {
		copied[i] = conflict
		copied[i].Candidate = copyCandidateRecord(conflict.Candidate)
		copied[i].Reasons = copyStrings(conflict.Reasons)
	}
	return copied
}

func copyCandidateRecord(candidate CandidateRecord) CandidateRecord {
	candidate.Need.ProductEvidence = copyStrings(candidate.Need.ProductEvidence)
	candidate.Need.BuyerIntentEvidence = copyStrings(candidate.Need.BuyerIntentEvidence)
	return candidate
}
