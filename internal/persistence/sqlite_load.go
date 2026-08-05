package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/domain"
)

var (
	ErrBatchRecordNotFound      = errors.New("batch record not found")
	ErrInvalidStoredBatchRecord = errors.New("invalid stored batch record")
)

type sqliteLoadedRoot struct {
	record  BatchRecord
	batchPK int64
	loc     *time.Location
}

// LoadBatch reconstructs one complete BatchRecord from SQLite schema version 1.
func (repo *SQLiteBatchRepository) LoadBatch(id BatchRecordID) (BatchRecord, error) {
	if repo == nil || repo.db == nil {
		return BatchRecord{}, ErrSQLiteRepositoryClosed
	}
	recordID, err := NewBatchRecordID(id.String())
	if err != nil {
		return BatchRecord{}, err
	}

	ctx := context.Background()
	if err := validateSQLiteSchema(ctx, repo.db); err != nil {
		return BatchRecord{}, err
	}

	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return BatchRecord{}, fmt.Errorf("begin sqlite batch load transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	root, err := loadSQLiteRoot(ctx, tx, recordID)
	if err != nil {
		return BatchRecord{}, err
	}
	record := root.record

	record.searchProfile.ProductTerms, record.searchProfile.BuyerIntentTerms, record.searchProfile.NoiseTerms, err = loadSQLiteSearchProfileTerms(ctx, tx, root.batchPK)
	if err != nil {
		return BatchRecord{}, err
	}
	groups, groupPKs, groupPositionByPK, err := loadSQLiteGroups(ctx, tx, root.batchPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.groups, record.flattenedPosts, err = loadSQLiteRawPostOccurrences(ctx, tx, root.batchPK, groups, groupPKs, groupPositionByPK, root.loc)
	if err != nil {
		return BatchRecord{}, err
	}
	postsByPK, err := loadSQLitePostsByPK(ctx, tx, root.batchPK, root.loc)
	if err != nil {
		return BatchRecord{}, err
	}
	record.evaluatedPosts, err = loadSQLiteEvaluatedPosts(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.includedPosts, record.reviewPosts, record.excludedPosts, err = loadSQLiteBucketedPosts(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.leads, err = loadSQLiteLeads(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	leadsByPK, err := loadSQLiteLeadsByPK(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.allowedLeads, record.blockedLeads, record.unresolvedLeads, err = loadSQLiteLeadOutcomes(ctx, tx, root.batchPK, leadsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.unaggregated, err = loadSQLiteUnaggregatedPosts(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.conflicts, err = loadSQLiteSourceConflicts(ctx, tx, root.batchPK, postsByPK)
	if err != nil {
		return BatchRecord{}, err
	}
	record.groupSummaries, err = loadSQLiteGroupSummaries(ctx, tx, root.batchPK, groupPKs, record.groups)
	if err != nil {
		return BatchRecord{}, err
	}

	if err := record.Validate(); err != nil {
		return BatchRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return BatchRecord{}, fmt.Errorf("commit sqlite batch load transaction: %w", err)
	}
	committed = true
	return record, nil
}

func loadSQLiteRoot(ctx context.Context, tx *sql.Tx, id BatchRecordID) (sqliteLoadedRoot, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			batch_pk,
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
		FROM scan_batches
		WHERE batch_record_id = ?
	`, id.String())
	if err != nil {
		return sqliteLoadedRoot{}, fmt.Errorf("query sqlite scan batch: %w", err)
	}
	defer rows.Close()

	var loaded *sqliteLoadedRoot
	for rows.Next() {
		if loaded != nil {
			return sqliteLoadedRoot{}, fmt.Errorf("%w: duplicate root batch", ErrInvalidStoredBatchRecord)
		}
		var (
			root              sqliteLoadedRoot
			idValue           string
			schemaVersion     int
			scanDate          string
			startOfDay        string
			scanStarted       string
			enabled           int
			geographicMode    string
			profileID         string
			profileName       string
			timezoneName      string
			summaryGroup      int
			summaryInput      int
			summaryEvaluated  int
			summaryInclude    int
			summaryReview     int
			summaryExcluded   int
			summaryLeads      int
			summaryAllowed    int
			summaryBlocked    int
			summaryUnresolved int
			summaryUnagg      int
			summaryConflicts  int
			summaryAllowSrc   int
			summaryBlockSrc   int
		)
		if err := rows.Scan(
			&root.batchPK,
			&idValue,
			&schemaVersion,
			&scanDate,
			&startOfDay,
			&scanStarted,
			&timezoneName,
			&profileID,
			&profileName,
			&enabled,
			&geographicMode,
			&summaryGroup,
			&summaryInput,
			&summaryEvaluated,
			&summaryInclude,
			&summaryReview,
			&summaryExcluded,
			&summaryLeads,
			&summaryAllowed,
			&summaryBlocked,
			&summaryUnresolved,
			&summaryUnagg,
			&summaryConflicts,
			&summaryAllowSrc,
			&summaryBlockSrc,
		); err != nil {
			return sqliteLoadedRoot{}, fmt.Errorf("scan sqlite scan batch: %w", err)
		}
		if schemaVersion != CurrentSQLiteSchemaVersion {
			return sqliteLoadedRoot{}, fmt.Errorf("%w: %d", ErrUnsupportedSQLiteSchemaVersion, schemaVersion)
		}
		recordID, err := NewBatchRecordID(idValue)
		if err != nil {
			return sqliteLoadedRoot{}, err
		}
		if recordID != id {
			return sqliteLoadedRoot{}, fmt.Errorf("%w: batch record id mismatch", ErrInvalidStoredBatchRecord)
		}
		root.loc, err = time.LoadLocation(timezoneName)
		if err != nil || root.loc.String() != domain.RequiredTimezone {
			return sqliteLoadedRoot{}, fmt.Errorf("%w: scan timezone", ErrInvalidStoredBatchRecord)
		}
		isEnabled, err := decodeSQLiteBool(enabled, "search profile enabled")
		if err != nil {
			return sqliteLoadedRoot{}, err
		}
		if !domain.GeographicMode(geographicMode).Valid() {
			return sqliteLoadedRoot{}, fmt.Errorf("%w: geographic mode", ErrInvalidStoredBatchRecord)
		}
		decodedScanDate, err := decodeSQLiteTimeInLocation(scanDate, "scan date", root.loc)
		if err != nil {
			return sqliteLoadedRoot{}, err
		}
		decodedStartOfDay, err := decodeSQLiteTimeInLocation(startOfDay, "scan start of day", root.loc)
		if err != nil {
			return sqliteLoadedRoot{}, err
		}
		decodedScanStarted, err := decodeSQLiteTimeInLocation(scanStarted, "scan started at", root.loc)
		if err != nil {
			return sqliteLoadedRoot{}, err
		}
		root.record = BatchRecord{
			id: recordID,
			scanWindow: ScanWindowRecord{
				ScanDate:    decodedScanDate,
				StartOfDay:  decodedStartOfDay,
				ScanStarted: decodedScanStarted,
				Timezone:    timezoneName,
			},
			searchProfile: SearchProfileRecord{
				ID:          profileID,
				DisplayName: profileName,
				IsEnabled:   isEnabled,
			},
			geographicMode: geographicMode,
			summary: BatchSummaryRecord{
				GroupCount:                 summaryGroup,
				InputPostCount:             summaryInput,
				EvaluatedPostCount:         summaryEvaluated,
				IncludePostCount:           summaryInclude,
				ReviewPostCount:            summaryReview,
				ExcludedPostCount:          summaryExcluded,
				AggregatedLeadCount:        summaryLeads,
				AllowedLeadCount:           summaryAllowed,
				BlockedLeadCount:           summaryBlocked,
				UnresolvedLeadCount:        summaryUnresolved,
				UnaggregatedPostCount:      summaryUnagg,
				SourceConflictCount:        summaryConflicts,
				AllowedLeadSourcePostCount: summaryAllowSrc,
				BlockedLeadSourcePostCount: summaryBlockSrc,
			},
		}
		loaded = &root
	}
	if err := rows.Err(); err != nil {
		return sqliteLoadedRoot{}, fmt.Errorf("iterate sqlite scan batch: %w", err)
	}
	if loaded == nil {
		return sqliteLoadedRoot{}, ErrBatchRecordNotFound
	}
	return *loaded, nil
}

func loadSQLiteSearchProfileTerms(ctx context.Context, tx *sql.Tx, batchPK int64) ([]string, []string, []string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT term_kind, term_position, term_value
		FROM batch_search_profile_terms
		WHERE batch_pk = ?
		ORDER BY term_kind, term_position
	`, batchPK)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query sqlite search profile terms: %w", err)
	}
	defer rows.Close()

	next := map[string]int{"product": 0, "buyer_intent": 0, "noise": 0}
	var product, buyerIntent, noise []string
	for rows.Next() {
		var kind, value string
		var position int
		if err := rows.Scan(&kind, &position, &value); err != nil {
			return nil, nil, nil, fmt.Errorf("scan sqlite search profile term: %w", err)
		}
		if _, ok := next[kind]; !ok {
			return nil, nil, nil, fmt.Errorf("%w: search profile term kind %q", ErrInvalidStoredBatchRecord, kind)
		}
		if err := validateSQLitePosition(kind+" term", position, next[kind]); err != nil {
			return nil, nil, nil, err
		}
		next[kind]++
		switch kind {
		case "product":
			product = append(product, value)
		case "buyer_intent":
			buyerIntent = append(buyerIntent, value)
		case "noise":
			noise = append(noise, value)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("iterate sqlite search profile terms: %w", err)
	}
	return product, buyerIntent, noise, nil
}

func loadSQLiteGroups(ctx context.Context, tx *sql.Tx, batchPK int64) ([]GroupRecord, []int64, map[int64]int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT group_pk, group_position, group_id, group_name
		FROM batch_groups
		WHERE batch_pk = ?
		ORDER BY group_position
	`, batchPK)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query sqlite groups: %w", err)
	}
	defer rows.Close()

	var groups []GroupRecord
	var groupPKs []int64
	groupPositionByPK := map[int64]int{}
	seenGroupIDs := map[string]struct{}{}
	for position := 0; rows.Next(); position++ {
		var groupPK int64
		var storedPosition int
		var group GroupRecord
		if err := rows.Scan(&groupPK, &storedPosition, &group.GroupID, &group.GroupName); err != nil {
			return nil, nil, nil, fmt.Errorf("scan sqlite group: %w", err)
		}
		if err := validateSQLitePosition("group", storedPosition, position); err != nil {
			return nil, nil, nil, err
		}
		if strings.TrimSpace(group.GroupID) == "" {
			return nil, nil, nil, fmt.Errorf("%w: blank group id", ErrInvalidStoredBatchRecord)
		}
		if _, ok := seenGroupIDs[group.GroupID]; ok {
			return nil, nil, nil, fmt.Errorf("%w: duplicate group id", ErrInvalidStoredBatchRecord)
		}
		seenGroupIDs[group.GroupID] = struct{}{}
		groups = append(groups, group)
		groupPKs = append(groupPKs, groupPK)
		groupPositionByPK[groupPK] = position
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("iterate sqlite groups: %w", err)
	}
	return groups, groupPKs, groupPositionByPK, nil
}

func loadSQLiteRawPostOccurrences(ctx context.Context, tx *sql.Tx, batchPK int64, groups []GroupRecord, groupPKs []int64, groupPositionByPK map[int64]int, loc *time.Location) ([]GroupRecord, []domain.RawPost, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			post_occurrence_pk,
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
		FROM raw_post_occurrences
		WHERE batch_pk = ?
		ORDER BY flattened_position
	`, batchPK)
	if err != nil {
		return nil, nil, fmt.Errorf("query sqlite raw post occurrences: %w", err)
	}
	defer rows.Close()

	nextGroupPost := make([]int, len(groups))
	lastGroupPosition := -1
	var flattened []domain.RawPost
	for flattenedPosition := 0; rows.Next(); flattenedPosition++ {
		var (
			postPK              int64
			groupPK             int64
			groupPosition       int
			groupPostPosition   int
			storedFlatPosition  int
			createdAt, captured string
			post                domain.RawPost
		)
		if err := rows.Scan(
			&postPK,
			&groupPK,
			&groupPosition,
			&groupPostPosition,
			&storedFlatPosition,
			&post.PostID,
			&post.GroupID,
			&post.GroupName,
			&post.PostURL,
			&post.Author.FacebookUserID,
			&post.Author.CanonicalProfileURL,
			&post.Author.Username,
			&post.Author.DisplayName,
			&post.Body,
			&createdAt,
			&captured,
		); err != nil {
			return nil, nil, fmt.Errorf("scan sqlite raw post occurrence: %w", err)
		}
		_ = postPK
		if err := validateSQLitePosition("flattened post", storedFlatPosition, flattenedPosition); err != nil {
			return nil, nil, err
		}
		mappedPosition, ok := groupPositionByPK[groupPK]
		if !ok || groupPosition < 0 || groupPosition >= len(groups) || mappedPosition != groupPosition {
			return nil, nil, fmt.Errorf("%w: raw post group reference", ErrInvalidStoredBatchRecord)
		}
		if groupPKs[groupPosition] != groupPK || post.GroupID != groups[groupPosition].GroupID {
			return nil, nil, fmt.Errorf("%w: raw post group mismatch", ErrInvalidStoredBatchRecord)
		}
		if groupPosition < lastGroupPosition {
			return nil, nil, fmt.Errorf("%w: raw post group order", ErrInvalidStoredBatchRecord)
		}
		lastGroupPosition = groupPosition
		if err := validateSQLitePosition("group post", groupPostPosition, nextGroupPost[groupPosition]); err != nil {
			return nil, nil, err
		}
		nextGroupPost[groupPosition]++
		var err error
		post.CreatedAt, err = decodeSQLiteTimeInLocation(createdAt, "raw post created at", loc)
		if err != nil {
			return nil, nil, err
		}
		post.CapturedAt, err = decodeSQLiteTimeInLocation(captured, "raw post captured at", loc)
		if err != nil {
			return nil, nil, err
		}
		groups[groupPosition].Posts = append(groups[groupPosition].Posts, post)
		flattened = append(flattened, post)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sqlite raw post occurrences: %w", err)
	}
	return groups, flattened, nil
}

func loadSQLitePostsByPK(ctx context.Context, tx *sql.Tx, batchPK int64, loc *time.Location) (map[int64]domain.RawPost, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			post_occurrence_pk,
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
		FROM raw_post_occurrences
		WHERE batch_pk = ?
		ORDER BY flattened_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite posts by pk: %w", err)
	}
	defer rows.Close()

	postsByPK := map[int64]domain.RawPost{}
	for rows.Next() {
		var postPK int64
		var createdAt, capturedAt string
		var post domain.RawPost
		if err := rows.Scan(
			&postPK,
			&post.PostID,
			&post.GroupID,
			&post.GroupName,
			&post.PostURL,
			&post.Author.FacebookUserID,
			&post.Author.CanonicalProfileURL,
			&post.Author.Username,
			&post.Author.DisplayName,
			&post.Body,
			&createdAt,
			&capturedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sqlite post by pk: %w", err)
		}
		var err error
		post.CreatedAt, err = decodeSQLiteTimeInLocation(createdAt, "raw post created at", loc)
		if err != nil {
			return nil, err
		}
		post.CapturedAt, err = decodeSQLiteTimeInLocation(capturedAt, "raw post captured at", loc)
		if err != nil {
			return nil, err
		}
		postsByPK[postPK] = post
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite posts by pk: %w", err)
	}
	return postsByPK, nil
}

func loadSQLiteEvaluatedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) ([]EvaluatedPostRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT evaluated_post_pk, post_occurrence_pk, evaluated_position, decision, geographic_class, geographic_reason_set_present
		FROM evaluated_posts
		WHERE batch_pk = ?
		ORDER BY evaluated_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite evaluated posts: %w", err)
	}
	defer rows.Close()

	var records []EvaluatedPostRecord
	for position := 0; rows.Next(); position++ {
		var evaluatedPK, postPK int64
		var storedPosition, geoPresent int
		var record EvaluatedPostRecord
		if err := rows.Scan(&evaluatedPK, &postPK, &storedPosition, &record.Decision, &record.GeographicClass, &geoPresent); err != nil {
			return nil, fmt.Errorf("scan sqlite evaluated post: %w", err)
		}
		if err := validateSQLitePosition("evaluated post", storedPosition, position); err != nil {
			return nil, err
		}
		post, ok := postsByPK[postPK]
		if !ok {
			return nil, fmt.Errorf("%w: evaluated post occurrence reference", ErrInvalidStoredBatchRecord)
		}
		if !supportedDecision(record.Decision) {
			return nil, ErrUnsupportedDecision
		}
		present, err := decodeSQLiteBool(geoPresent, "evaluated geographic reason presence")
		if err != nil {
			return nil, err
		}
		record.Post = post
		record.Reasons, record.GeographicReasons, err = loadSQLiteCategorizedReasons(ctx, tx, "evaluated_post_reasons", "evaluated_post_pk", evaluatedPK)
		if err != nil {
			return nil, err
		}
		if present != (len(record.GeographicReasons) > 0) {
			return nil, fmt.Errorf("%w: evaluated geographic reason presence", ErrInvalidStoredBatchRecord)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite evaluated posts: %w", err)
	}
	return records, nil
}

func loadSQLiteBucketedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) ([]EvaluatedPostRecord, []EvaluatedPostRecord, []EvaluatedPostRecord, error) {
	if err := validateSQLiteBucketValues(ctx, tx, batchPK); err != nil {
		return nil, nil, nil, err
	}
	included, err := loadSQLiteBucket(ctx, tx, batchPK, postsByPK, "include")
	if err != nil {
		return nil, nil, nil, err
	}
	review, err := loadSQLiteBucket(ctx, tx, batchPK, postsByPK, "review")
	if err != nil {
		return nil, nil, nil, err
	}
	excluded, err := loadSQLiteBucket(ctx, tx, batchPK, postsByPK, "exclude")
	if err != nil {
		return nil, nil, nil, err
	}
	return included, review, excluded, nil
}

func validateSQLiteBucketValues(ctx context.Context, tx *sql.Tx, batchPK int64) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT bucket
		FROM bucketed_posts
		WHERE batch_pk = ?
		ORDER BY bucket
	`, batchPK)
	if err != nil {
		return fmt.Errorf("query sqlite bucket values: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bucket string
		if err := rows.Scan(&bucket); err != nil {
			return fmt.Errorf("scan sqlite bucket value: %w", err)
		}
		if bucket != "include" && bucket != "review" && bucket != "exclude" {
			return fmt.Errorf("%w: bucket %q", ErrInvalidStoredBatchRecord, bucket)
		}
	}
	return rows.Err()
}

func loadSQLiteBucket(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost, bucket string) ([]EvaluatedPostRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT bucketed_post_pk, post_occurrence_pk, bucket_position, decision, geographic_class
		FROM bucketed_posts
		WHERE batch_pk = ? AND bucket = ?
		ORDER BY bucket_position
	`, batchPK, bucket)
	if err != nil {
		return nil, fmt.Errorf("query sqlite bucketed posts: %w", err)
	}
	defer rows.Close()

	var records []EvaluatedPostRecord
	for position := 0; rows.Next(); position++ {
		var bucketedPK, postPK int64
		var storedPosition int
		var record EvaluatedPostRecord
		if err := rows.Scan(&bucketedPK, &postPK, &storedPosition, &record.Decision, &record.GeographicClass); err != nil {
			return nil, fmt.Errorf("scan sqlite bucketed post: %w", err)
		}
		if err := validateSQLitePosition(bucket+" bucket", storedPosition, position); err != nil {
			return nil, err
		}
		post, ok := postsByPK[postPK]
		if !ok {
			return nil, fmt.Errorf("%w: bucketed post occurrence reference", ErrInvalidStoredBatchRecord)
		}
		if !supportedDecision(record.Decision) {
			return nil, ErrUnsupportedDecision
		}
		record.Post = post
		var err error
		record.Reasons, record.GeographicReasons, err = loadSQLiteCategorizedReasons(ctx, tx, "bucketed_post_reasons", "bucketed_post_pk", bucketedPK)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite bucketed posts: %w", err)
	}
	return records, nil
}

func loadSQLiteLeads(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) ([]LeadRecord, error) {
	leadsByPK, err := loadSQLiteLeadsByPK(ctx, tx, batchPK, postsByPK)
	if err != nil {
		return nil, err
	}
	return orderedLeadsFromPKMap(ctx, tx, batchPK, leadsByPK)
}

func loadSQLiteLeadsByPK(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) (map[int64]LeadRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			lead_pk,
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
		FROM leads
		WHERE batch_pk = ?
		ORDER BY lead_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite leads: %w", err)
	}
	defer rows.Close()

	leadsByPK := map[int64]LeadRecord{}
	for position := 0; rows.Next(); position++ {
		var leadPK int64
		var storedPosition int
		var lead LeadRecord
		if err := rows.Scan(
			&leadPK,
			&storedPosition,
			&lead.Key.Value,
			&lead.Key.Author.Kind,
			&lead.Key.Author.Value,
			&lead.Key.Need.SearchProfileID,
			&lead.Key.Need.NormalizedBody,
			&lead.Key.Need.BodyFingerprint,
			&lead.Author.Kind,
			&lead.Author.Value,
			&lead.Need.SearchProfileID,
			&lead.Need.NormalizedBody,
			&lead.Need.BodyFingerprint,
		); err != nil {
			return nil, fmt.Errorf("scan sqlite lead: %w", err)
		}
		if err := validateSQLitePosition("lead", storedPosition, position); err != nil {
			return nil, err
		}
		var err error
		lead.Key.Need.ProductEvidence, err = loadSQLiteOrderedValues(ctx, tx, "lead_key_need_product_evidence", "lead_pk", leadPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		lead.Key.Need.BuyerIntentEvidence, err = loadSQLiteOrderedValues(ctx, tx, "lead_key_need_buyer_intent_evidence", "lead_pk", leadPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		lead.Need.ProductEvidence, err = loadSQLiteOrderedValues(ctx, tx, "lead_need_product_evidence", "lead_pk", leadPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		lead.Need.BuyerIntentEvidence, err = loadSQLiteOrderedValues(ctx, tx, "lead_need_buyer_intent_evidence", "lead_pk", leadPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		lead.Sources, err = loadSQLiteLeadSources(ctx, tx, leadPK, postsByPK)
		if err != nil {
			return nil, err
		}
		leadsByPK[leadPK] = lead
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite leads: %w", err)
	}
	return leadsByPK, nil
}

func orderedLeadsFromPKMap(ctx context.Context, tx *sql.Tx, batchPK int64, leadsByPK map[int64]LeadRecord) ([]LeadRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT lead_pk, lead_position
		FROM leads
		WHERE batch_pk = ?
		ORDER BY lead_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite ordered leads: %w", err)
	}
	defer rows.Close()

	var leads []LeadRecord
	for position := 0; rows.Next(); position++ {
		var leadPK int64
		var storedPosition int
		if err := rows.Scan(&leadPK, &storedPosition); err != nil {
			return nil, fmt.Errorf("scan sqlite ordered lead: %w", err)
		}
		if err := validateSQLitePosition("lead", storedPosition, position); err != nil {
			return nil, err
		}
		lead, ok := leadsByPK[leadPK]
		if !ok {
			return nil, fmt.Errorf("%w: lead lookup", ErrInvalidStoredBatchRecord)
		}
		leads = append(leads, lead)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite ordered leads: %w", err)
	}
	return leads, nil
}

func loadSQLiteLeadSources(ctx context.Context, tx *sql.Tx, leadPK int64, postsByPK map[int64]domain.RawPost) ([]SourcePostRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT source_position, post_occurrence_pk, source_key_kind, source_key_value
		FROM lead_sources
		WHERE lead_pk = ?
		ORDER BY source_position
	`, leadPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite lead sources: %w", err)
	}
	defer rows.Close()

	var sources []SourcePostRecord
	for position := 0; rows.Next(); position++ {
		var storedPosition int
		var postPK int64
		var source SourcePostRecord
		if err := rows.Scan(&storedPosition, &postPK, &source.Key.Kind, &source.Key.Value); err != nil {
			return nil, fmt.Errorf("scan sqlite lead source: %w", err)
		}
		if err := validateSQLitePosition("lead source", storedPosition, position); err != nil {
			return nil, err
		}
		post, ok := postsByPK[postPK]
		if !ok {
			return nil, fmt.Errorf("%w: lead source post reference", ErrInvalidStoredBatchRecord)
		}
		source.Post = post
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite lead sources: %w", err)
	}
	return sources, nil
}

func loadSQLiteLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, leadsByPK map[int64]LeadRecord) ([]AllowedLeadRecord, []BlockedLeadRecord, []UnresolvedLeadRecord, error) {
	if err := validateSQLiteOutcomeBuckets(ctx, tx, batchPK); err != nil {
		return nil, nil, nil, err
	}
	allowed, err := loadSQLiteAllowedLeadOutcomes(ctx, tx, batchPK, leadsByPK)
	if err != nil {
		return nil, nil, nil, err
	}
	blocked, err := loadSQLiteBlockedLeadOutcomes(ctx, tx, batchPK, leadsByPK)
	if err != nil {
		return nil, nil, nil, err
	}
	unresolved, err := loadSQLiteUnresolvedLeadOutcomes(ctx, tx, batchPK, leadsByPK)
	if err != nil {
		return nil, nil, nil, err
	}
	return allowed, blocked, unresolved, nil
}

func validateSQLiteOutcomeBuckets(ctx context.Context, tx *sql.Tx, batchPK int64) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT outcome_bucket
		FROM lead_outcomes
		WHERE batch_pk = ?
		ORDER BY outcome_bucket
	`, batchPK)
	if err != nil {
		return fmt.Errorf("query sqlite outcome buckets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bucket string
		if err := rows.Scan(&bucket); err != nil {
			return fmt.Errorf("scan sqlite outcome bucket: %w", err)
		}
		if bucket != "allowed" && bucket != "blocked" && bucket != "unresolved" {
			return fmt.Errorf("%w: lead outcome bucket %q", ErrInvalidStoredBatchRecord, bucket)
		}
	}
	return rows.Err()
}

func loadSQLiteAllowedLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, leadsByPK map[int64]LeadRecord) ([]AllowedLeadRecord, error) {
	rows, err := querySQLiteLeadOutcomes(ctx, tx, batchPK, "allowed")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []AllowedLeadRecord
	for position := 0; rows.Next(); position++ {
		outcome, err := scanSQLiteLeadOutcome(ctx, tx, rows, leadsByPK, "allowed", position)
		if err != nil {
			return nil, err
		}
		records = append(records, AllowedLeadRecord{Lead: outcome.lead, Match: outcome.match})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite allowed lead outcomes: %w", err)
	}
	return records, nil
}

func loadSQLiteBlockedLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, leadsByPK map[int64]LeadRecord) ([]BlockedLeadRecord, error) {
	rows, err := querySQLiteLeadOutcomes(ctx, tx, batchPK, "blocked")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []BlockedLeadRecord
	for position := 0; rows.Next(); position++ {
		outcome, err := scanSQLiteLeadOutcome(ctx, tx, rows, leadsByPK, "blocked", position)
		if err != nil {
			return nil, err
		}
		records = append(records, BlockedLeadRecord{Lead: outcome.lead, Match: outcome.match})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite blocked lead outcomes: %w", err)
	}
	return records, nil
}

func loadSQLiteUnresolvedLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, leadsByPK map[int64]LeadRecord) ([]UnresolvedLeadRecord, error) {
	rows, err := querySQLiteLeadOutcomes(ctx, tx, batchPK, "unresolved")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []UnresolvedLeadRecord
	for position := 0; rows.Next(); position++ {
		outcome, err := scanSQLiteLeadOutcome(ctx, tx, rows, leadsByPK, "unresolved", position)
		if err != nil {
			return nil, err
		}
		records = append(records, UnresolvedLeadRecord{Lead: outcome.lead, Match: outcome.match, ApplicationReasons: outcome.applicationReasons})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite unresolved lead outcomes: %w", err)
	}
	return records, nil
}

type sqliteLoadedLeadOutcome struct {
	lead               LeadRecord
	match              BlocklistMatchRecord
	applicationReasons []string
}

func querySQLiteLeadOutcomes(ctx context.Context, tx *sql.Tx, batchPK int64, bucket string) (*sql.Rows, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			lead_outcome_pk,
			lead_pk,
			bucket_position,
			match_outcome,
			match_author_key_kind,
			match_author_key_value,
			matched_entry_key_kind,
			matched_entry_key_value,
			matched_entry_display_name
		FROM lead_outcomes
		WHERE batch_pk = ? AND outcome_bucket = ?
		ORDER BY bucket_position
	`, batchPK, bucket)
	if err != nil {
		return nil, fmt.Errorf("query sqlite lead outcomes: %w", err)
	}
	return rows, nil
}

func scanSQLiteLeadOutcome(ctx context.Context, tx *sql.Tx, rows *sql.Rows, leadsByPK map[int64]LeadRecord, bucket string, position int) (sqliteLoadedLeadOutcome, error) {
	var outcomePK, leadPK int64
	var storedPosition int
	var outcome sqliteLoadedLeadOutcome
	if err := rows.Scan(
		&outcomePK,
		&leadPK,
		&storedPosition,
		&outcome.match.Outcome,
		&outcome.match.AuthorKey.Kind,
		&outcome.match.AuthorKey.Value,
		&outcome.match.MatchedEntry.Key.Kind,
		&outcome.match.MatchedEntry.Key.Value,
		&outcome.match.MatchedEntry.DisplayName,
	); err != nil {
		return sqliteLoadedLeadOutcome{}, fmt.Errorf("scan sqlite lead outcome: %w", err)
	}
	if err := validateSQLitePosition(bucket+" lead outcome", storedPosition, position); err != nil {
		return sqliteLoadedLeadOutcome{}, err
	}
	lead, ok := leadsByPK[leadPK]
	if !ok {
		return sqliteLoadedLeadOutcome{}, fmt.Errorf("%w: lead outcome reference", ErrInvalidStoredBatchRecord)
	}
	if !supportedBlocklistOutcome(outcome.match.Outcome) {
		return sqliteLoadedLeadOutcome{}, ErrUnsupportedBlockOutcome
	}
	outcome.lead = lead
	var err error
	outcome.match.Reasons, err = loadSQLiteOrderedValues(ctx, tx, "lead_outcome_blocklist_reasons", "lead_outcome_pk", outcomePK, "reason_position", "reason_code")
	if err != nil {
		return sqliteLoadedLeadOutcome{}, err
	}
	outcome.applicationReasons, err = loadSQLiteOrderedValues(ctx, tx, "lead_outcome_application_reasons", "lead_outcome_pk", outcomePK, "reason_position", "reason_code")
	if err != nil {
		return sqliteLoadedLeadOutcome{}, err
	}
	if bucket != "unresolved" && len(outcome.applicationReasons) != 0 {
		return sqliteLoadedLeadOutcome{}, fmt.Errorf("%w: application reasons for %s outcome", ErrInvalidStoredBatchRecord, bucket)
	}
	return outcome, nil
}

func loadSQLiteUnaggregatedPosts(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) ([]UnaggregatedPostRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			unaggregated_pk,
			unaggregated_position,
			post_occurrence_pk,
			candidate_author_kind,
			candidate_author_value,
			candidate_need_search_profile_id,
			candidate_need_normalized_body,
			candidate_need_body_fingerprint,
			source_key_kind,
			source_key_value
		FROM unaggregated_posts
		WHERE batch_pk = ?
		ORDER BY unaggregated_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite unaggregated posts: %w", err)
	}
	defer rows.Close()

	var records []UnaggregatedPostRecord
	for position := 0; rows.Next(); position++ {
		var unaggregatedPK, postPK int64
		var storedPosition int
		var record UnaggregatedPostRecord
		if err := rows.Scan(
			&unaggregatedPK,
			&storedPosition,
			&postPK,
			&record.Candidate.Author.Kind,
			&record.Candidate.Author.Value,
			&record.Candidate.Need.SearchProfileID,
			&record.Candidate.Need.NormalizedBody,
			&record.Candidate.Need.BodyFingerprint,
			&record.Source.Kind,
			&record.Source.Value,
		); err != nil {
			return nil, fmt.Errorf("scan sqlite unaggregated post: %w", err)
		}
		if err := validateSQLitePosition("unaggregated post", storedPosition, position); err != nil {
			return nil, err
		}
		post, ok := postsByPK[postPK]
		if !ok {
			return nil, fmt.Errorf("%w: unaggregated post occurrence reference", ErrInvalidStoredBatchRecord)
		}
		record.Post = post
		var err error
		record.Candidate.Need.ProductEvidence, err = loadSQLiteOrderedValues(ctx, tx, "unaggregated_candidate_product_evidence", "unaggregated_pk", unaggregatedPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		record.Candidate.Need.BuyerIntentEvidence, err = loadSQLiteOrderedValues(ctx, tx, "unaggregated_candidate_buyer_intent_evidence", "unaggregated_pk", unaggregatedPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		record.Reasons, err = loadSQLiteOrderedValues(ctx, tx, "unaggregated_post_reasons", "unaggregated_pk", unaggregatedPK, "reason_position", "reason_code")
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite unaggregated posts: %w", err)
	}
	return records, nil
}

func loadSQLiteSourceConflicts(ctx context.Context, tx *sql.Tx, batchPK int64, postsByPK map[int64]domain.RawPost) ([]SourceConflictRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			source_conflict_pk,
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
		FROM source_conflicts
		WHERE batch_pk = ?
		ORDER BY conflict_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite source conflicts: %w", err)
	}
	defer rows.Close()

	var records []SourceConflictRecord
	for position := 0; rows.Next(); position++ {
		var conflictPK, postPK, existingPostPK int64
		var storedPosition int
		var record SourceConflictRecord
		if err := rows.Scan(
			&conflictPK,
			&storedPosition,
			&postPK,
			&existingPostPK,
			&record.ExistingSource.Key.Kind,
			&record.ExistingSource.Key.Value,
			&record.Candidate.Author.Kind,
			&record.Candidate.Author.Value,
			&record.Candidate.Need.SearchProfileID,
			&record.Candidate.Need.NormalizedBody,
			&record.Candidate.Need.BodyFingerprint,
			&record.Source.Kind,
			&record.Source.Value,
		); err != nil {
			return nil, fmt.Errorf("scan sqlite source conflict: %w", err)
		}
		if err := validateSQLitePosition("source conflict", storedPosition, position); err != nil {
			return nil, err
		}
		post, ok := postsByPK[postPK]
		if !ok {
			return nil, fmt.Errorf("%w: source conflict post reference", ErrInvalidStoredBatchRecord)
		}
		existingPost, ok := postsByPK[existingPostPK]
		if !ok {
			return nil, fmt.Errorf("%w: source conflict existing post reference", ErrInvalidStoredBatchRecord)
		}
		record.Post = post
		record.ExistingSource.Post = existingPost
		var err error
		record.Candidate.Need.ProductEvidence, err = loadSQLiteOrderedValues(ctx, tx, "source_conflict_candidate_product_evidence", "source_conflict_pk", conflictPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		record.Candidate.Need.BuyerIntentEvidence, err = loadSQLiteOrderedValues(ctx, tx, "source_conflict_candidate_buyer_intent_evidence", "source_conflict_pk", conflictPK, "evidence_position", "evidence_value")
		if err != nil {
			return nil, err
		}
		record.Reasons, err = loadSQLiteOrderedValues(ctx, tx, "source_conflict_reasons", "source_conflict_pk", conflictPK, "reason_position", "reason_code")
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite source conflicts: %w", err)
	}
	return records, nil
}

func loadSQLiteGroupSummaries(ctx context.Context, tx *sql.Tx, batchPK int64, groupPKs []int64, groups []GroupRecord) ([]GroupSummaryRecord, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			group_pk,
			group_position,
			group_id,
			input_post_count,
			evaluated_post_count,
			include_post_count,
			review_post_count,
			excluded_post_count
		FROM group_summaries
		WHERE batch_pk = ?
		ORDER BY group_position
	`, batchPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite group summaries: %w", err)
	}
	defer rows.Close()

	var summaries []GroupSummaryRecord
	for position := 0; rows.Next(); position++ {
		var groupPK int64
		var storedPosition int
		var summary GroupSummaryRecord
		if err := rows.Scan(
			&groupPK,
			&storedPosition,
			&summary.GroupID,
			&summary.InputPostCount,
			&summary.EvaluatedPostCount,
			&summary.IncludePostCount,
			&summary.ReviewPostCount,
			&summary.ExcludedPostCount,
		); err != nil {
			return nil, fmt.Errorf("scan sqlite group summary: %w", err)
		}
		if err := validateSQLitePosition("group summary", storedPosition, position); err != nil {
			return nil, err
		}
		if position >= len(groupPKs) || groupPKs[position] != groupPK || groups[position].GroupID != summary.GroupID {
			return nil, fmt.Errorf("%w: group summary alignment", ErrInvalidStoredBatchRecord)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite group summaries: %w", err)
	}
	return summaries, nil
}

func loadSQLiteCategorizedReasons(ctx context.Context, tx *sql.Tx, table string, parentColumn string, parentPK int64) ([]string, []string, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT reason_category, reason_position, reason_code
		FROM %s
		WHERE %s = ?
		ORDER BY reason_category, reason_position
	`, table, parentColumn), parentPK)
	if err != nil {
		return nil, nil, fmt.Errorf("query sqlite categorized reasons: %w", err)
	}
	defer rows.Close()

	next := map[string]int{"geographic": 0, "rule": 0}
	var ruleReasons, geographicReasons []string
	for rows.Next() {
		var category, reason string
		var position int
		if err := rows.Scan(&category, &position, &reason); err != nil {
			return nil, nil, fmt.Errorf("scan sqlite categorized reason: %w", err)
		}
		if _, ok := next[category]; !ok {
			return nil, nil, fmt.Errorf("%w: reason category %q", ErrInvalidStoredBatchRecord, category)
		}
		if err := validateSQLitePosition(category+" reason", position, next[category]); err != nil {
			return nil, nil, err
		}
		next[category]++
		switch category {
		case "rule":
			ruleReasons = append(ruleReasons, reason)
		case "geographic":
			geographicReasons = append(geographicReasons, reason)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sqlite categorized reasons: %w", err)
	}
	return ruleReasons, geographicReasons, nil
}

func loadSQLiteOrderedValues(ctx context.Context, tx *sql.Tx, table string, parentColumn string, parentPK int64, positionColumn string, valueColumn string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s, %s
		FROM %s
		WHERE %s = ?
		ORDER BY %s
	`, positionColumn, valueColumn, table, parentColumn, positionColumn), parentPK)
	if err != nil {
		return nil, fmt.Errorf("query sqlite ordered values: %w", err)
	}
	defer rows.Close()

	var values []string
	for position := 0; rows.Next(); position++ {
		var storedPosition int
		var value string
		if err := rows.Scan(&storedPosition, &value); err != nil {
			return nil, fmt.Errorf("scan sqlite ordered value: %w", err)
		}
		if err := validateSQLitePosition(table, storedPosition, position); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite ordered values: %w", err)
	}
	return values, nil
}

func validateSQLitePosition(label string, got int, want int) error {
	if got != want {
		return fmt.Errorf("%w: %s position %d, want %d", ErrInvalidStoredBatchRecord, label, got, want)
	}
	return nil
}

func decodeSQLiteBool(value int, label string) (bool, error) {
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("%w: %s boolean %d", ErrInvalidStoredBatchRecord, label, value)
	}
}

func decodeSQLiteTimeInLocation(value string, label string, loc *time.Location) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s timestamp: %v", ErrInvalidStoredBatchRecord, label, err)
	}
	if sqliteTime(parsed) != value {
		return time.Time{}, fmt.Errorf("%w: %s timestamp is not canonical", ErrInvalidStoredBatchRecord, label)
	}
	return parsed.In(loc), nil
}
