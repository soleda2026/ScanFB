package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestSQLiteBatchRepositorySaveBatchBasicSaves(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	oneGroup := buildEmptySQLiteBatchRecord(t, "batch-one", 1)
	fiveGroups := buildEmptySQLiteBatchRecord(t, "batch-five", 5)

	if err := repo.SaveBatch(oneGroup); err != nil {
		t.Fatalf("SaveBatch(one group) error = %v", err)
	}
	if err := repo.SaveBatch(fiveGroups); err != nil {
		t.Fatalf("SaveBatch(five groups) error = %v", err)
	}

	assertSchemaVersion(t, repo.db, CurrentSQLiteSchemaVersion)
	if got, want := queryStrings(t, repo.db, "SELECT batch_record_id FROM scan_batches ORDER BY batch_pk"), []string{"batch-one", "batch-five"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved batch IDs = %#v, want %#v", got, want)
	}
	if got := countRows(t, repo.db, "batch_groups"); got != 6 {
		t.Fatalf("batch_groups rows = %d, want 6", got)
	}
	if got := countRows(t, repo.db, "group_summaries"); got != 6 {
		t.Fatalf("group_summaries rows = %d, want 6", got)
	}
}

func TestSQLiteBatchRepositorySaveBatchCompleteMappingAndOrdering(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	record := buildRichSQLiteBatchRecord(t, "batch-rich")

	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}

	wantCounts := map[string]int{
		"scan_batches":                                    1,
		"batch_search_profile_terms":                      len(record.searchProfile.ProductTerms) + len(record.searchProfile.BuyerIntentTerms) + len(record.searchProfile.NoiseTerms),
		"batch_groups":                                    2,
		"raw_post_occurrences":                            6,
		"evaluated_posts":                                 6,
		"evaluated_post_reasons":                          16,
		"bucketed_posts":                                  6,
		"bucketed_post_reasons":                           16,
		"leads":                                           3,
		"lead_key_need_product_evidence":                  3,
		"lead_key_need_buyer_intent_evidence":             3,
		"lead_need_product_evidence":                      3,
		"lead_need_buyer_intent_evidence":                 3,
		"lead_sources":                                    4,
		"lead_outcomes":                                   3,
		"lead_outcome_blocklist_reasons":                  3,
		"lead_outcome_application_reasons":                1,
		"unaggregated_posts":                              1,
		"unaggregated_candidate_product_evidence":         1,
		"unaggregated_candidate_buyer_intent_evidence":    1,
		"unaggregated_post_reasons":                       1,
		"source_conflicts":                                1,
		"source_conflict_candidate_product_evidence":      1,
		"source_conflict_candidate_buyer_intent_evidence": 1,
		"source_conflict_reasons":                         1,
		"group_summaries":                                 2,
	}
	for table, want := range wantCounts {
		if got := countRows(t, repo.db, table); got != want {
			t.Fatalf("%s rows = %d, want %d", table, got, want)
		}
	}

	if got, want := queryStrings(t, repo.db, "SELECT term_value FROM batch_search_profile_terms WHERE term_kind = 'product' ORDER BY term_position"), record.searchProfile.ProductTerms; !reflect.DeepEqual(got, want) {
		t.Fatalf("product term order = %#v, want %#v", got, want)
	}
	if got, want := queryStrings(t, repo.db, "SELECT group_id FROM batch_groups ORDER BY group_position"), []string{"group-a", "group-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group order = %#v, want %#v", got, want)
	}
	if got, want := queryStrings(t, repo.db, "SELECT post_id FROM raw_post_occurrences ORDER BY flattened_position"), []string{"post-1", "post-2", "post-3", "post-4", "post-5", "post-6"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("raw post order = %#v, want %#v", got, want)
	}
	if got, want := queryInts(t, repo.db, "SELECT group_post_position FROM raw_post_occurrences WHERE group_id = 'group-b' ORDER BY group_post_position"), []int{0, 1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group-relative positions = %#v, want %#v", got, want)
	}
	if got, want := queryStrings(t, repo.db, "SELECT reason_code FROM evaluated_post_reasons WHERE evaluated_post_pk = 1 ORDER BY reason_category, reason_position"), []string{"geo.hcm", "included.buyer_intent", "included.target_keyword"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("evaluated reasons = %#v, want %#v", got, want)
	}
	if got, want := queryStrings(t, repo.db, "SELECT post_id FROM raw_post_occurrences WHERE body = 'Bán MacBook Pro HCM'"), []string{"post-4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Vietnamese body preservation lookup = %#v, want %#v", got, want)
	}
	if got := queryOneString(t, repo.db, "SELECT created_at FROM raw_post_occurrences WHERE post_id = 'post-1'"); got != "2026-08-05T02:00:00Z" {
		t.Fatalf("created_at = %q, want UTC RFC3339Nano instant", got)
	}
	if got := queryOneInt(t, repo.db, "SELECT search_profile_is_enabled FROM scan_batches WHERE batch_record_id = 'batch-rich'"); got != 1 {
		t.Fatalf("search_profile_is_enabled = %d, want 1", got)
	}
	if got, want := queryStrings(t, repo.db, "SELECT outcome_bucket || ':' || match_outcome FROM lead_outcomes ORDER BY outcome_bucket, bucket_position"), []string{"allowed:not_blocked", "blocked:blocked", "unresolved:insufficient_identity"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lead outcomes = %#v, want %#v", got, want)
	}
	if got, want := queryStrings(t, repo.db, "SELECT reason_code FROM lead_outcome_application_reasons ORDER BY reason_position"), []string{"application.blocklist_evaluation_unsupported"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("application reasons = %#v, want %#v", got, want)
	}
	if got, want := queryInts(t, repo.db, "SELECT lead_position FROM leads ORDER BY lead_position"), []int{0, 1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lead positions = %#v, want %#v", got, want)
	}
	if got, want := queryInts(t, repo.db, "SELECT source_position FROM lead_sources WHERE lead_pk = 1 ORDER BY source_position"), []int{0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lead source positions = %#v, want %#v", got, want)
	}
	if got, want := queryInts(t, repo.db, "SELECT bucket_position FROM bucketed_posts WHERE bucket = 'include' ORDER BY bucket_position"), []int{0, 1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("include bucket positions = %#v, want %#v", got, want)
	}
	if got, want := queryInts(t, repo.db, "SELECT group_position FROM group_summaries ORDER BY group_position"), []int{0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group summary positions = %#v, want %#v", got, want)
	}
}

func TestSQLiteBatchRepositorySaveBatchValidationFailuresDoNotMutate(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	valid := buildRichSQLiteBatchRecord(t, "batch-valid")
	if err := repo.SaveBatch(valid); err != nil {
		t.Fatalf("SaveBatch(valid) error = %v", err)
	}
	before := sqliteTableCounts(t, repo.db)

	tests := []struct {
		name string
		edit func(*BatchRecord)
		want error
	}{
		{name: "zero record", edit: func(record *BatchRecord) { *record = BatchRecord{} }, want: ErrEmptyBatchRecordID},
		{name: "empty id", edit: func(record *BatchRecord) { record.id = "" }, want: ErrEmptyBatchRecordID},
		{name: "malformed scan window", edit: func(record *BatchRecord) { record.scanWindow = ScanWindowRecord{} }, want: ErrInvalidBatchRecord},
		{name: "invalid profile", edit: func(record *BatchRecord) { record.searchProfile.ID = "" }, want: ErrInvalidBatchRecord},
		{name: "invalid geographic mode", edit: func(record *BatchRecord) { record.geographicMode = "foreign" }, want: ErrInvalidBatchRecord},
		{name: "invalid group", edit: func(record *BatchRecord) { record.groups[0].GroupID = "" }, want: ErrInvalidBatchRecord},
		{name: "invalid decision", edit: func(record *BatchRecord) { record.evaluatedPosts[0].Decision = "hold" }, want: ErrUnsupportedDecision},
		{name: "invalid outcome", edit: func(record *BatchRecord) { record.blockedLeads[0].Match.Outcome = "not_blocked" }, want: ErrUnsupportedLeadOutcome},
		{name: "inconsistent summary", edit: func(record *BatchRecord) { record.summary.InputPostCount++ }, want: ErrInconsistentBatchSummary},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := buildRichSQLiteBatchRecord(t, "batch-invalid-"+strings.ReplaceAll(test.name, " ", "-"))
			test.edit(&record)
			if err := repo.SaveBatch(record); !errors.Is(err, test.want) {
				t.Fatalf("SaveBatch() error = %v, want %v", err, test.want)
			}
			if after := sqliteTableCounts(t, repo.db); !reflect.DeepEqual(after, before) {
				t.Fatalf("failed validation mutated database:\nbefore %#v\nafter  %#v", before, after)
			}
		})
	}
}

func TestSQLiteBatchRepositorySaveBatchDuplicateIDFailsWithoutOverwrite(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	original := buildRichSQLiteBatchRecord(t, "batch-duplicate")
	duplicate := buildRichSQLiteBatchRecord(t, "batch-duplicate")
	duplicate.groups[0].GroupName = "Mutated Duplicate"

	if err := repo.SaveBatch(original); err != nil {
		t.Fatalf("SaveBatch(original) error = %v", err)
	}
	before := sqliteTableCounts(t, repo.db)
	if err := repo.SaveBatch(duplicate); !errors.Is(err, ErrBatchRecordAlreadyExists) {
		t.Fatalf("SaveBatch(duplicate) error = %v, want %v", err, ErrBatchRecordAlreadyExists)
	}
	if after := sqliteTableCounts(t, repo.db); !reflect.DeepEqual(after, before) {
		t.Fatalf("duplicate save mutated database:\nbefore %#v\nafter  %#v", before, after)
	}
	if got := queryOneString(t, repo.db, "SELECT group_name FROM batch_groups WHERE group_position = 0"); got != "Group A" {
		t.Fatalf("duplicate overwrote original group name: %q", got)
	}

	differentID := buildRichSQLiteBatchRecord(t, "batch-different")
	if err := repo.SaveBatch(differentID); err != nil {
		t.Fatalf("SaveBatch(different ID same content) error = %v", err)
	}
	if got := countRows(t, repo.db, "scan_batches"); got != 2 {
		t.Fatalf("scan_batches after distinct ID = %d, want 2", got)
	}
}

func TestSQLiteBatchRepositorySaveBatchRollsBackChildFailure(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	if err := repo.SaveBatch(buildEmptySQLiteBatchRecord(t, "batch-before", 1)); err != nil {
		t.Fatalf("SaveBatch(before) error = %v", err)
	}
	before := sqliteTableCounts(t, repo.db)
	execSQL(t, repo.db, `
		CREATE TRIGGER fail_batch_group_insert
		BEFORE INSERT ON batch_groups
		BEGIN
			SELECT RAISE(ABORT, 'test induced batch group failure');
		END
	`)

	err = repo.SaveBatch(buildEmptySQLiteBatchRecord(t, "batch-triggered", 1))
	if err == nil {
		t.Fatal("SaveBatch() error = nil, want triggered child insert failure")
	}
	if after := sqliteTableCounts(t, repo.db); !reflect.DeepEqual(after, before) {
		t.Fatalf("triggered failure did not rollback all batch rows:\nbefore %#v\nafter  %#v", before, after)
	}
	if got := countRows(t, repo.db, "schema_metadata"); got != 1 {
		t.Fatalf("schema metadata rows = %d, want 1", got)
	}
}

func TestSQLiteBatchRepositorySaveBatchAfterCloseFailsSafely(t *testing.T) {
	path := sqliteTestPath(t)
	repo, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := repo.SaveBatch(buildEmptySQLiteBatchRecord(t, "batch-closed", 1)); !errors.Is(err, ErrSQLiteRepositoryClosed) {
		t.Fatalf("SaveBatch(after Close) error = %v, want %v", err, ErrSQLiteRepositoryClosed)
	}

	reopened, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("reopen after closed SaveBatch error = %v", err)
	}
	defer closeSQLiteRepo(t, reopened)
	if got := countRows(t, reopened.db, "scan_batches"); got != 0 {
		t.Fatalf("scan_batches after closed SaveBatch = %d, want 0", got)
	}
}

func TestSQLiteBatchRepositorySaveBatchDoesNotMutateRecord(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	record := buildRichSQLiteBatchRecord(t, "batch-immutable")
	before := copyBatchRecord(record)

	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if !reflect.DeepEqual(record, before) {
		t.Fatalf("SaveBatch mutated BatchRecord:\ngot  %#v\nwant %#v", record, before)
	}

	record.groups[0].Posts[0].Body = "mutated after save"
	if got := queryOneString(t, repo.db, "SELECT body FROM raw_post_occurrences WHERE flattened_position = 0"); got != before.groups[0].Posts[0].Body {
		t.Fatalf("stored body changed after caller mutation: %q", got)
	}
}

func TestSQLiteBatchRepositorySaveBatchDeterminismAndNoLoadAPI(t *testing.T) {
	first := savedSQLiteLogicalSnapshot(t, buildRichSQLiteBatchRecord(t, "batch-first"))
	second := savedSQLiteLogicalSnapshot(t, buildRichSQLiteBatchRecord(t, "batch-second"))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("logical snapshots differ:\nfirst  %#v\nsecond %#v", first, second)
	}

	repoType := reflect.TypeOf(&SQLiteBatchRepository{})
	for _, method := range []string{"LoadBatch", "ListBatches", "UpdateBatch", "DeleteBatch", "SearchBatches"} {
		if _, ok := repoType.MethodByName(method); ok {
			t.Fatalf("SQLiteBatchRepository exposes deferred API %s", method)
		}
	}
}

func buildEmptySQLiteBatchRecord(t *testing.T, id string, groupCount int) BatchRecord {
	t.Helper()
	recordID, err := NewBatchRecordID(id)
	if err != nil {
		t.Fatalf("NewBatchRecordID() error = %v", err)
	}
	groups := make([]GroupRecord, groupCount)
	groupSummaries := make([]GroupSummaryRecord, groupCount)
	for i := range groups {
		groupID := fmt.Sprintf("group-%d", i+1)
		groups[i] = GroupRecord{GroupID: groupID, GroupName: fmt.Sprintf("Group %d", i+1)}
		groupSummaries[i] = GroupSummaryRecord{GroupID: groupID}
	}
	record := BatchRecord{
		id:             recordID,
		scanWindow:     scanWindowRecord(validScanWindow(t)),
		searchProfile:  searchProfileRecord(domain.MacBookSearchProfile()),
		geographicMode: string(domain.GeographicModeAllVietnam),
		groups:         groups,
		summary: BatchSummaryRecord{
			GroupCount: groupCount,
		},
		groupSummaries: groupSummaries,
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("empty sqlite batch record invalid: %v", err)
	}
	return record
}

func buildRichSQLiteBatchRecord(t *testing.T, id string) BatchRecord {
	t.Helper()
	record := buildTestBatchRecordWithID(t, BatchRecordID(id))
	for i := range record.evaluatedPosts {
		record.evaluatedPosts[i].GeographicClass = "hcm"
		record.evaluatedPosts[i].GeographicReasons = []string{"geo.hcm"}
		if record.evaluatedPosts[i].Decision == "include" {
			record.evaluatedPosts[i].Reasons = []string{"included.buyer_intent", "included.target_keyword"}
		}
	}
	for i := range record.includedPosts {
		record.includedPosts[i].GeographicClass = "hcm"
		record.includedPosts[i].GeographicReasons = []string{"geo.hcm"}
		record.includedPosts[i].Reasons = []string{"included.buyer_intent", "included.target_keyword"}
	}
	for i := range record.reviewPosts {
		record.reviewPosts[i].GeographicClass = "unknown"
		record.reviewPosts[i].GeographicReasons = []string{"geo.unknown"}
	}
	for i := range record.excludedPosts {
		record.excludedPosts[i].GeographicClass = "hcm"
		record.excludedPosts[i].GeographicReasons = []string{"geo.hcm"}
	}

	unresolvedLead := leadRecordFromLeadParts(
		"lead-unresolved",
		"display_name",
		"Needs Human",
		"macbook",
		"can mua macbook hcm",
		"fingerprint-unresolved",
		[]string{"macbook"},
		[]string{"can mua"},
		[]SourcePostRecord{{
			Key:  SourceIdentityRecord{Kind: "post_id", Value: "post-6"},
			Post: record.unaggregated[0].Post,
		}},
	)
	record.leads = append(record.leads, unresolvedLead)
	record.unresolvedLeads = []UnresolvedLeadRecord{{
		Lead: unresolvedLead,
		Match: BlocklistMatchRecord{
			Outcome:   "insufficient_identity",
			Reasons:   []string{"blocklist.stable_author_identity_missing"},
			AuthorKey: IdentityRecord{Kind: "display_name", Value: "Needs Human"},
		},
		ApplicationReasons: []string{"application.blocklist_evaluation_unsupported"},
	}}
	record.conflicts = []SourceConflictRecord{{
		Post:           record.groups[0].Posts[1],
		ExistingSource: SourcePostRecord{Key: SourceIdentityRecord{Kind: "post_id", Value: "post-1"}, Post: record.groups[0].Posts[0]},
		Candidate: candidateRecord(
			"facebook_user_id",
			"allowed-author",
			"macbook",
			"can mua macbook hcm",
			"fingerprint-conflict",
			[]string{"macbook"},
			[]string{"can mua"},
		),
		Source:  SourceIdentityRecord{Kind: "post_id", Value: "post-2"},
		Reasons: []string{"dedup.source_identity_conflict"},
	}}
	record.summary.AggregatedLeadCount = len(record.leads)
	record.summary.UnresolvedLeadCount = len(record.unresolvedLeads)
	record.summary.SourceConflictCount = len(record.conflicts)
	if err := record.Validate(); err != nil {
		t.Fatalf("rich sqlite batch record invalid: %v", err)
	}
	return record
}

func sqliteTableCounts(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	counts := make(map[string]int, len(requiredSQLiteTables))
	for _, table := range requiredSQLiteTables {
		counts[table] = countRows(t, db, table)
	}
	return counts
}

func savedSQLiteLogicalSnapshot(t *testing.T, record BatchRecord) []string {
	t.Helper()
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	return []string{
		strings.Join(queryStrings(t, repo.db, "SELECT post_id || ':' || body FROM raw_post_occurrences ORDER BY flattened_position"), "|"),
		strings.Join(queryStrings(t, repo.db, "SELECT term_kind || ':' || term_position || ':' || term_value FROM batch_search_profile_terms ORDER BY term_kind, term_position"), "|"),
		strings.Join(queryStrings(t, repo.db, "SELECT decision || ':' || geographic_class FROM evaluated_posts ORDER BY evaluated_position"), "|"),
		strings.Join(queryStrings(t, repo.db, "SELECT lead_key_value || ':' || lead_position FROM leads ORDER BY lead_position"), "|"),
		strings.Join(queryStrings(t, repo.db, "SELECT outcome_bucket || ':' || match_outcome FROM lead_outcomes ORDER BY outcome_bucket, bucket_position"), "|"),
		strings.Join(queryStrings(t, repo.db, "SELECT group_id || ':' || group_position FROM group_summaries ORDER BY group_position"), "|"),
	}
}

func queryStrings(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("query strings: %v", err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("scan string: %v", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate strings: %v", err)
	}
	return values
}

func queryInts(t *testing.T, db *sql.DB, query string, args ...any) []int {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("query ints: %v", err)
	}
	defer rows.Close()
	var values []int
	for rows.Next() {
		var value int
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("scan int: %v", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate ints: %v", err)
	}
	return values
}

func queryOneString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var value string
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("query one string: %v", err)
	}
	return value
}

func queryOneInt(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var value int
	if err := db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("query one int: %v", err)
	}
	return value
}
