package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
	"modernc.org/sqlite"
)

const sqliteConstraintUniqueCode = 2067

func (repo *SQLiteBatchRepository) SaveBatch(record BatchRecord) (err error) {
	if repo == nil || repo.db == nil {
		return ErrSQLiteRepositoryClosed
	}
	if err := record.Validate(); err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite batch save transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err != nil {
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
	}()

	batchPK, err := insertSQLiteScanBatch(ctx, tx, record)
	if err != nil {
		return translateSQLiteBatchRootInsertError(err)
	}
	if err := insertSQLiteSearchProfileTerms(ctx, tx, batchPK, record.SearchProfile()); err != nil {
		return err
	}
	groupPKs, postLookup, err := insertSQLiteGroupsAndRawPosts(ctx, tx, batchPK, record.Groups())
	if err != nil {
		return err
	}
	if err := insertSQLiteEvaluatedPosts(ctx, tx, batchPK, record.EvaluatedPosts(), postLookup.resolver()); err != nil {
		return err
	}
	if err := insertSQLiteBucketedPosts(ctx, tx, batchPK, "include", record.IncludedPosts(), postLookup.resolver()); err != nil {
		return err
	}
	if err := insertSQLiteBucketedPosts(ctx, tx, batchPK, "review", record.ReviewPosts(), postLookup.resolver()); err != nil {
		return err
	}
	if err := insertSQLiteBucketedPosts(ctx, tx, batchPK, "exclude", record.ExcludedPosts(), postLookup.resolver()); err != nil {
		return err
	}
	leadLookup, err := insertSQLiteLeads(ctx, tx, batchPK, record.Leads(), postLookup.resolver())
	if err != nil {
		return err
	}
	if err := insertSQLiteLeadOutcomes(ctx, tx, batchPK, leadLookup, record.AllowedLeads(), record.BlockedLeads(), record.UnresolvedLeads()); err != nil {
		return err
	}
	if err := insertSQLiteUnaggregatedPosts(ctx, tx, batchPK, record.Unaggregated(), postLookup.resolver()); err != nil {
		return err
	}
	if err := insertSQLiteSourceConflicts(ctx, tx, batchPK, record.Conflicts(), postLookup.resolver(), postLookup.resolver()); err != nil {
		return err
	}
	if err := insertSQLiteGroupSummaries(ctx, tx, batchPK, groupPKs, record.GroupSummaries()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite batch save transaction: %w", err)
	}
	committed = true
	return nil
}

func insertSQLiteScanBatch(ctx context.Context, tx *sql.Tx, record BatchRecord) (int64, error) {
	window := record.ScanWindow()
	profile := record.SearchProfile()
	summary := record.Summary()
	return sqliteInsert(ctx, tx, `
		INSERT INTO scan_batches (
			batch_record_id,
			schema_version,
			scan_date,
			scan_start_of_day,
			scan_started_at,
			scan_timezone,
			search_profile_id,
			search_profile_display_name,
			search_profile_is_enabled,
			geographic_mode,
			summary_group_count,
			summary_input_post_count,
			summary_evaluated_post_count,
			summary_include_post_count,
			summary_review_post_count,
			summary_excluded_post_count,
			summary_aggregated_lead_count,
			summary_allowed_lead_count,
			summary_blocked_lead_count,
			summary_unresolved_lead_count,
			summary_unaggregated_post_count,
			summary_source_conflict_count,
			summary_allowed_lead_source_post_count,
			summary_blocked_lead_source_post_count
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID().String(),
		CurrentSQLiteSchemaVersion,
		sqliteTime(window.ScanDate),
		sqliteTime(window.StartOfDay),
		sqliteTime(window.ScanStarted),
		window.Timezone,
		profile.ID,
		profile.DisplayName,
		sqliteBool(profile.IsEnabled),
		record.GeographicMode(),
		summary.GroupCount,
		summary.InputPostCount,
		summary.EvaluatedPostCount,
		summary.IncludePostCount,
		summary.ReviewPostCount,
		summary.ExcludedPostCount,
		summary.AggregatedLeadCount,
		summary.AllowedLeadCount,
		summary.BlockedLeadCount,
		summary.UnresolvedLeadCount,
		summary.UnaggregatedPostCount,
		summary.SourceConflictCount,
		summary.AllowedLeadSourcePostCount,
		summary.BlockedLeadSourcePostCount,
	)
}

func insertSQLiteSearchProfileTerms(ctx context.Context, tx *sql.Tx, batchPK int64, profile SearchProfileRecord) error {
	if err := insertSQLiteTerms(ctx, tx, batchPK, "product", profile.ProductTerms); err != nil {
		return err
	}
	if err := insertSQLiteTerms(ctx, tx, batchPK, "buyer_intent", profile.BuyerIntentTerms); err != nil {
		return err
	}
	return insertSQLiteTerms(ctx, tx, batchPK, "noise", profile.NoiseTerms)
}

func insertSQLiteTerms(ctx context.Context, tx *sql.Tx, batchPK int64, kind string, terms []string) error {
	for position, term := range terms {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO batch_search_profile_terms (batch_pk, term_kind, term_position, term_value)
			VALUES (?, ?, ?, ?)
		`, batchPK, kind, position, term); err != nil {
			return fmt.Errorf("insert sqlite search profile term: %w", err)
		}
	}
	return nil
}

func insertSQLiteGroupsAndRawPosts(ctx context.Context, tx *sql.Tx, batchPK int64, groups []GroupRecord) ([]int64, *sqlitePostOccurrenceLookup, error) {
	groupPKs := make([]int64, len(groups))
	postLookup := newSQLitePostOccurrenceLookup()
	flattenedPosition := 0
	for groupPosition, group := range groups {
		groupPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (?, ?, ?, ?)
		`, batchPK, groupPosition, group.GroupID, group.GroupName)
		if err != nil {
			return nil, nil, fmt.Errorf("insert sqlite batch group: %w", err)
		}
		groupPKs[groupPosition] = groupPK

		for groupPostPosition, post := range group.Posts {
			postPK, err := insertSQLiteRawPostOccurrence(ctx, tx, batchPK, groupPK, groupPosition, groupPostPosition, flattenedPosition, post)
			if err != nil {
				return nil, nil, err
			}
			postLookup.add(post, postPK)
			flattenedPosition++
		}
	}
	return groupPKs, postLookup, nil
}

func insertSQLiteRawPostOccurrence(ctx context.Context, tx *sql.Tx, batchPK int64, groupPK int64, groupPosition int, groupPostPosition int, flattenedPosition int, post domain.RawPost) (int64, error) {
	return sqliteInsert(ctx, tx, `
		INSERT INTO raw_post_occurrences (
			batch_pk,
			group_pk,
			group_position,
			group_post_position,
			flattened_position,
			post_id,
			group_id,
			group_name,
			post_url,
			author_facebook_user_id,
			author_canonical_profile_url,
			author_username,
			author_display_name,
			body,
			created_at,
			captured_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		batchPK,
		groupPK,
		groupPosition,
		groupPostPosition,
		flattenedPosition,
		post.PostID,
		post.GroupID,
		post.GroupName,
		post.PostURL,
		post.Author.FacebookUserID,
		post.Author.CanonicalProfileURL,
		post.Author.Username,
		post.Author.DisplayName,
		post.Body,
		sqliteTime(post.CreatedAt),
		sqliteTime(post.CapturedAt),
	)
}

func insertSQLiteEvaluatedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, posts []EvaluatedPostRecord, postResolver *sqlitePostOccurrenceResolver) error {
	for position, post := range posts {
		postPK, err := postResolver.resolve(post.Post)
		if err != nil {
			return err
		}
		evaluatedPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO evaluated_posts (
				batch_pk,
				post_occurrence_pk,
				evaluated_position,
				decision,
				geographic_class,
				geographic_reason_set_present
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`, batchPK, postPK, position, post.Decision, post.GeographicClass, sqliteBool(len(post.GeographicReasons) > 0))
		if err != nil {
			return fmt.Errorf("insert sqlite evaluated post: %w", err)
		}
		if err := insertSQLiteCategorizedReasons(ctx, tx, "evaluated_post_reasons", "evaluated_post_pk", evaluatedPK, post.Reasons, post.GeographicReasons); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteBucketedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, bucket string, posts []EvaluatedPostRecord, postResolver *sqlitePostOccurrenceResolver) error {
	for position, post := range posts {
		postPK, err := postResolver.resolve(post.Post)
		if err != nil {
			return err
		}
		bucketedPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO bucketed_posts (
				batch_pk,
				bucket,
				bucket_position,
				post_occurrence_pk,
				decision,
				geographic_class
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`, batchPK, bucket, position, postPK, post.Decision, post.GeographicClass)
		if err != nil {
			return fmt.Errorf("insert sqlite bucketed post: %w", err)
		}
		if err := insertSQLiteCategorizedReasons(ctx, tx, "bucketed_post_reasons", "bucketed_post_pk", bucketedPK, post.Reasons, post.GeographicReasons); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteLeads(ctx context.Context, tx *sql.Tx, batchPK int64, leads []LeadRecord, postResolver *sqlitePostOccurrenceResolver) (*sqliteLeadLookup, error) {
	leadLookup := newSQLiteLeadLookup()
	for position, lead := range leads {
		leadPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO leads (
				batch_pk,
				lead_position,
				lead_key_value,
				lead_key_author_kind,
				lead_key_author_value,
				lead_key_need_search_profile_id,
				lead_key_need_normalized_body,
				lead_key_need_body_fingerprint,
				author_kind,
				author_value,
				need_search_profile_id,
				need_normalized_body,
				need_body_fingerprint
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			batchPK,
			position,
			lead.Key.Value,
			lead.Key.Author.Kind,
			lead.Key.Author.Value,
			lead.Key.Need.SearchProfileID,
			lead.Key.Need.NormalizedBody,
			lead.Key.Need.BodyFingerprint,
			lead.Author.Kind,
			lead.Author.Value,
			lead.Need.SearchProfileID,
			lead.Need.NormalizedBody,
			lead.Need.BodyFingerprint,
		)
		if err != nil {
			return nil, fmt.Errorf("insert sqlite lead: %w", err)
		}
		leadLookup.add(lead, leadPK)
		if err := insertSQLiteEvidence(ctx, tx, "lead_key_need_product_evidence", "lead_pk", leadPK, lead.Key.Need.ProductEvidence); err != nil {
			return nil, err
		}
		if err := insertSQLiteEvidence(ctx, tx, "lead_key_need_buyer_intent_evidence", "lead_pk", leadPK, lead.Key.Need.BuyerIntentEvidence); err != nil {
			return nil, err
		}
		if err := insertSQLiteEvidence(ctx, tx, "lead_need_product_evidence", "lead_pk", leadPK, lead.Need.ProductEvidence); err != nil {
			return nil, err
		}
		if err := insertSQLiteEvidence(ctx, tx, "lead_need_buyer_intent_evidence", "lead_pk", leadPK, lead.Need.BuyerIntentEvidence); err != nil {
			return nil, err
		}
		leadPostResolver := postResolver.lookup.resolver()
		for sourcePosition, source := range lead.Sources {
			postPK, err := leadPostResolver.resolve(source.Post)
			if err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO lead_sources (
					lead_pk,
					source_position,
					post_occurrence_pk,
					source_key_kind,
					source_key_value
				)
				VALUES (?, ?, ?, ?, ?)
			`, leadPK, sourcePosition, postPK, source.Key.Kind, source.Key.Value); err != nil {
				return nil, fmt.Errorf("insert sqlite lead source: %w", err)
			}
		}
	}
	return leadLookup, nil
}

func insertSQLiteLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, lookup *sqliteLeadLookup, allowed []AllowedLeadRecord, blocked []BlockedLeadRecord, unresolved []UnresolvedLeadRecord) error {
	resolver := lookup.resolver()
	for position, record := range allowed {
		if err := insertSQLiteLeadOutcome(ctx, tx, batchPK, resolver, "allowed", position, record.Lead, record.Match, nil); err != nil {
			return err
		}
	}
	for position, record := range blocked {
		if err := insertSQLiteLeadOutcome(ctx, tx, batchPK, resolver, "blocked", position, record.Lead, record.Match, nil); err != nil {
			return err
		}
	}
	for position, record := range unresolved {
		if err := insertSQLiteLeadOutcome(ctx, tx, batchPK, resolver, "unresolved", position, record.Lead, record.Match, record.ApplicationReasons); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteLeadOutcome(ctx context.Context, tx *sql.Tx, batchPK int64, resolver *sqliteLeadResolver, bucket string, position int, lead LeadRecord, match BlocklistMatchRecord, applicationReasons []string) error {
	leadPK, err := resolver.resolve(lead)
	if err != nil {
		return err
	}
	outcomePK, err := sqliteInsert(ctx, tx, `
		INSERT INTO lead_outcomes (
			batch_pk,
			lead_pk,
			outcome_bucket,
			bucket_position,
			match_outcome,
			match_author_key_kind,
			match_author_key_value,
			matched_entry_key_kind,
			matched_entry_key_value,
			matched_entry_display_name
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		batchPK,
		leadPK,
		bucket,
		position,
		match.Outcome,
		match.AuthorKey.Kind,
		match.AuthorKey.Value,
		match.MatchedEntry.Key.Kind,
		match.MatchedEntry.Key.Value,
		match.MatchedEntry.DisplayName,
	)
	if err != nil {
		return fmt.Errorf("insert sqlite lead outcome: %w", err)
	}
	if err := insertSQLiteReasons(ctx, tx, "lead_outcome_blocklist_reasons", "lead_outcome_pk", outcomePK, match.Reasons); err != nil {
		return err
	}
	return insertSQLiteReasons(ctx, tx, "lead_outcome_application_reasons", "lead_outcome_pk", outcomePK, applicationReasons)
}

func insertSQLiteUnaggregatedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, posts []UnaggregatedPostRecord, postResolver *sqlitePostOccurrenceResolver) error {
	for position, record := range posts {
		postPK, err := postResolver.resolve(record.Post)
		if err != nil {
			return err
		}
		unaggregatedPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO unaggregated_posts (
				batch_pk,
				unaggregated_position,
				post_occurrence_pk,
				candidate_author_kind,
				candidate_author_value,
				candidate_need_search_profile_id,
				candidate_need_normalized_body,
				candidate_need_body_fingerprint,
				source_key_kind,
				source_key_value
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			batchPK,
			position,
			postPK,
			record.Candidate.Author.Kind,
			record.Candidate.Author.Value,
			record.Candidate.Need.SearchProfileID,
			record.Candidate.Need.NormalizedBody,
			record.Candidate.Need.BodyFingerprint,
			record.Source.Kind,
			record.Source.Value,
		)
		if err != nil {
			return fmt.Errorf("insert sqlite unaggregated post: %w", err)
		}
		if err := insertSQLiteEvidence(ctx, tx, "unaggregated_candidate_product_evidence", "unaggregated_pk", unaggregatedPK, record.Candidate.Need.ProductEvidence); err != nil {
			return err
		}
		if err := insertSQLiteEvidence(ctx, tx, "unaggregated_candidate_buyer_intent_evidence", "unaggregated_pk", unaggregatedPK, record.Candidate.Need.BuyerIntentEvidence); err != nil {
			return err
		}
		if err := insertSQLiteReasons(ctx, tx, "unaggregated_post_reasons", "unaggregated_pk", unaggregatedPK, record.Reasons); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteSourceConflicts(ctx context.Context, tx *sql.Tx, batchPK int64, conflicts []SourceConflictRecord, postResolver *sqlitePostOccurrenceResolver, existingPostResolver *sqlitePostOccurrenceResolver) error {
	for position, record := range conflicts {
		postPK, err := postResolver.resolve(record.Post)
		if err != nil {
			return err
		}
		existingPostPK, err := existingPostResolver.resolve(record.ExistingSource.Post)
		if err != nil {
			return err
		}
		conflictPK, err := sqliteInsert(ctx, tx, `
			INSERT INTO source_conflicts (
				batch_pk,
				conflict_position,
				post_occurrence_pk,
				existing_post_occurrence_pk,
				existing_source_key_kind,
				existing_source_key_value,
				candidate_author_kind,
				candidate_author_value,
				candidate_need_search_profile_id,
				candidate_need_normalized_body,
				candidate_need_body_fingerprint,
				source_key_kind,
				source_key_value
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			batchPK,
			position,
			postPK,
			existingPostPK,
			record.ExistingSource.Key.Kind,
			record.ExistingSource.Key.Value,
			record.Candidate.Author.Kind,
			record.Candidate.Author.Value,
			record.Candidate.Need.SearchProfileID,
			record.Candidate.Need.NormalizedBody,
			record.Candidate.Need.BodyFingerprint,
			record.Source.Kind,
			record.Source.Value,
		)
		if err != nil {
			return fmt.Errorf("insert sqlite source conflict: %w", err)
		}
		if err := insertSQLiteEvidence(ctx, tx, "source_conflict_candidate_product_evidence", "source_conflict_pk", conflictPK, record.Candidate.Need.ProductEvidence); err != nil {
			return err
		}
		if err := insertSQLiteEvidence(ctx, tx, "source_conflict_candidate_buyer_intent_evidence", "source_conflict_pk", conflictPK, record.Candidate.Need.BuyerIntentEvidence); err != nil {
			return err
		}
		if err := insertSQLiteReasons(ctx, tx, "source_conflict_reasons", "source_conflict_pk", conflictPK, record.Reasons); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteGroupSummaries(ctx context.Context, tx *sql.Tx, batchPK int64, groupPKs []int64, summaries []GroupSummaryRecord) error {
	if len(groupPKs) != len(summaries) {
		return ErrInconsistentBatchSummary
	}
	for position, summary := range summaries {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO group_summaries (
				batch_pk,
				group_pk,
				group_position,
				group_id,
				input_post_count,
				evaluated_post_count,
				include_post_count,
				review_post_count,
				excluded_post_count
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			batchPK,
			groupPKs[position],
			position,
			summary.GroupID,
			summary.InputPostCount,
			summary.EvaluatedPostCount,
			summary.IncludePostCount,
			summary.ReviewPostCount,
			summary.ExcludedPostCount,
		); err != nil {
			return fmt.Errorf("insert sqlite group summary: %w", err)
		}
	}
	return nil
}

func insertSQLiteCategorizedReasons(ctx context.Context, tx *sql.Tx, table string, parentColumn string, parentPK int64, ruleReasons []string, geographicReasons []string) error {
	for position, reason := range ruleReasons {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, reason_category, reason_position, reason_code)
			VALUES (?, 'rule', ?, ?)
		`, table, parentColumn), parentPK, position, reason); err != nil {
			return fmt.Errorf("insert sqlite rule reason: %w", err)
		}
	}
	for position, reason := range geographicReasons {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, reason_category, reason_position, reason_code)
			VALUES (?, 'geographic', ?, ?)
		`, table, parentColumn), parentPK, position, reason); err != nil {
			return fmt.Errorf("insert sqlite geographic reason: %w", err)
		}
	}
	return nil
}

func insertSQLiteEvidence(ctx context.Context, tx *sql.Tx, table string, parentColumn string, parentPK int64, values []string) error {
	for position, value := range values {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, evidence_position, evidence_value)
			VALUES (?, ?, ?)
		`, table, parentColumn), parentPK, position, value); err != nil {
			return fmt.Errorf("insert sqlite evidence: %w", err)
		}
	}
	return nil
}

func insertSQLiteReasons(ctx context.Context, tx *sql.Tx, table string, parentColumn string, parentPK int64, reasons []string) error {
	for position, reason := range reasons {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (%s, reason_position, reason_code)
			VALUES (?, ?, ?)
		`, table, parentColumn), parentPK, position, reason); err != nil {
			return fmt.Errorf("insert sqlite reason: %w", err)
		}
	}
	return nil
}

func sqliteInsert(ctx context.Context, tx *sql.Tx, statement string, args ...any) (int64, error) {
	result, err := tx.ExecContext(ctx, statement, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read sqlite inserted row id: %w", err)
	}
	return id, nil
}

func translateSQLiteBatchRootInsertError(err error) error {
	if isDuplicateSQLiteBatchRecordIDError(err) {
		return fmt.Errorf("%w: %v", ErrBatchRecordAlreadyExists, err)
	}
	return fmt.Errorf("insert sqlite scan batch: %w", err)
}

func isDuplicateSQLiteBatchRecordIDError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqliteConstraintUniqueCode {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "scan_batches.batch_record_id") ||
		strings.Contains(message, "idx_scan_batches_batch_record_id_unique")
}

func sqliteTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func sqliteBool(value bool) int {
	if value {
		return 1
	}
	return 0
}

type sqlitePostOccurrenceLookup struct {
	idsBySignature map[string][]int64
}

func newSQLitePostOccurrenceLookup() *sqlitePostOccurrenceLookup {
	return &sqlitePostOccurrenceLookup{idsBySignature: map[string][]int64{}}
}

func (lookup *sqlitePostOccurrenceLookup) add(post domain.RawPost, postPK int64) {
	signature := rawPostSignature(post)
	lookup.idsBySignature[signature] = append(lookup.idsBySignature[signature], postPK)
}

func (lookup *sqlitePostOccurrenceLookup) resolver() *sqlitePostOccurrenceResolver {
	return &sqlitePostOccurrenceResolver{
		lookup:  lookup,
		offsets: map[string]int{},
	}
}

type sqlitePostOccurrenceResolver struct {
	lookup  *sqlitePostOccurrenceLookup
	offsets map[string]int
}

func (resolver *sqlitePostOccurrenceResolver) resolve(post domain.RawPost) (int64, error) {
	signature := rawPostSignature(post)
	ids := resolver.lookup.idsBySignature[signature]
	offset := resolver.offsets[signature]
	if offset >= len(ids) {
		return 0, fmt.Errorf("%w: raw post occurrence reference", ErrInvalidBatchRecord)
	}
	resolver.offsets[signature] = offset + 1
	return ids[offset], nil
}

func rawPostSignature(post domain.RawPost) string {
	return strings.Join([]string{
		post.PostID,
		post.GroupID,
		post.GroupName,
		post.PostURL,
		post.Author.FacebookUserID,
		post.Author.CanonicalProfileURL,
		post.Author.Username,
		post.Author.DisplayName,
		post.Body,
		sqliteTime(post.CreatedAt),
		sqliteTime(post.CapturedAt),
	}, "\x00")
}

type sqliteLeadLookup struct {
	idsBySignature map[string][]int64
}

func newSQLiteLeadLookup() *sqliteLeadLookup {
	return &sqliteLeadLookup{idsBySignature: map[string][]int64{}}
}

func (lookup *sqliteLeadLookup) add(lead LeadRecord, leadPK int64) {
	signature := leadSignature(lead)
	lookup.idsBySignature[signature] = append(lookup.idsBySignature[signature], leadPK)
}

func (lookup *sqliteLeadLookup) resolver() *sqliteLeadResolver {
	return &sqliteLeadResolver{
		lookup:  lookup,
		offsets: map[string]int{},
	}
}

type sqliteLeadResolver struct {
	lookup  *sqliteLeadLookup
	offsets map[string]int
}

func (resolver *sqliteLeadResolver) resolve(lead LeadRecord) (int64, error) {
	signature := leadSignature(lead)
	ids := resolver.lookup.idsBySignature[signature]
	offset := resolver.offsets[signature]
	if offset >= len(ids) {
		return 0, fmt.Errorf("%w: lead reference", ErrInvalidBatchRecord)
	}
	resolver.offsets[signature] = offset + 1
	return ids[offset], nil
}

func leadSignature(lead LeadRecord) string {
	parts := []string{
		lead.Key.Value,
		lead.Key.Author.Kind,
		lead.Key.Author.Value,
		lead.Key.Need.SearchProfileID,
		lead.Key.Need.NormalizedBody,
		lead.Key.Need.BodyFingerprint,
		lead.Author.Kind,
		lead.Author.Value,
		lead.Need.SearchProfileID,
		lead.Need.NormalizedBody,
		lead.Need.BodyFingerprint,
	}
	parts = append(parts, lead.Key.Need.ProductEvidence...)
	parts = append(parts, lead.Key.Need.BuyerIntentEvidence...)
	parts = append(parts, lead.Need.ProductEvidence...)
	parts = append(parts, lead.Need.BuyerIntentEvidence...)
	for _, source := range lead.Sources {
		parts = append(parts, source.Key.Kind, source.Key.Value, rawPostSignature(source.Post))
	}
	return strings.Join(parts, "\x00")
}
