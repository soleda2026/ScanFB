package persistence

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestSQLiteBatchRepositoryLoadBatchRoundTripRichOneGroupAndFiveGroups(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	oneGroup := buildOneGroupRichSQLiteBatchRecord(t, "batch-one-rich")
	fiveGroups := buildEmptySQLiteBatchRecord(t, "batch-five", 5)
	if err := repo.SaveBatch(oneGroup); err != nil {
		t.Fatalf("SaveBatch(oneGroup) error = %v", err)
	}
	if err := repo.SaveBatch(fiveGroups); err != nil {
		t.Fatalf("SaveBatch(fiveGroups) error = %v", err)
	}

	loadedOne, err := repo.LoadBatch(oneGroup.ID())
	if err != nil {
		t.Fatalf("LoadBatch(oneGroup) error = %v", err)
	}
	assertBatchRecordsDeeplyEqual(t, loadedOne, oneGroup)
	loadedFive, err := repo.LoadBatch(fiveGroups.ID())
	if err != nil {
		t.Fatalf("LoadBatch(fiveGroups) error = %v", err)
	}
	assertBatchRecordsDeeplyEqual(t, loadedFive, fiveGroups)

	loadedOneAgain, err := repo.LoadBatch(oneGroup.ID())
	if err != nil {
		t.Fatalf("LoadBatch(oneGroup again) error = %v", err)
	}
	assertBatchRecordsDeeplyEqual(t, loadedOneAgain, loadedOne)
	if loadedOneAgain.ID() == loadedFive.ID() {
		t.Fatal("different batch IDs loaded as the same record")
	}
}

func TestSQLiteBatchRepositoryLoadBatchPreservesFullSnapshotDetails(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	record := buildOneGroupRichSQLiteBatchRecord(t, "batch-details")
	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	beforeCounts := sqliteTableCounts(t, repo.db)
	loaded, err := repo.LoadBatch(record.ID())
	if err != nil {
		t.Fatalf("LoadBatch() error = %v", err)
	}
	afterCounts := sqliteTableCounts(t, repo.db)
	if !reflect.DeepEqual(afterCounts, beforeCounts) {
		t.Fatalf("LoadBatch mutated row counts:\nbefore %#v\nafter  %#v", beforeCounts, afterCounts)
	}

	assertBatchRecordsDeeplyEqual(t, loaded, record)
	if got, want := loaded.ScanWindow().ScanStarted, record.ScanWindow().ScanStarted; !got.Equal(want) || got.Location().String() != domain.RequiredTimezone {
		t.Fatalf("loaded ScanStarted = %v (%s), want instant %v in %s", got, got.Location(), want, domain.RequiredTimezone)
	}
	if got, want := loaded.SearchProfile().ProductTerms, record.SearchProfile().ProductTerms; !reflect.DeepEqual(got, want) {
		t.Fatalf("product terms = %#v, want %#v", got, want)
	}
	if got, want := loaded.FlattenedPosts()[3].Body, "Bán MacBook Pro HCM"; got != want {
		t.Fatalf("Vietnamese body = %q, want %q", got, want)
	}
	if got, want := loaded.Groups()[0].Posts[3].PostID, loaded.FlattenedPosts()[3].PostID; got != want {
		t.Fatalf("group-relative post order mismatch: %q vs %q", got, want)
	}
	if got, want := loaded.EvaluatedPosts()[0].Reasons, []string{"included.buyer_intent", "included.target_keyword"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("evaluated reason order = %#v, want %#v", got, want)
	}
	if got, want := len(loaded.AllowedLeads()[0].Lead.Sources), 2; got != want {
		t.Fatalf("allowed source count = %d, want %d", got, want)
	}
	if got, want := loaded.UnresolvedLeads()[0].ApplicationReasons, []string{"application.blocklist_evaluation_unsupported"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("application reasons = %#v, want %#v", got, want)
	}
	if got, want := loaded.Conflicts()[0].ExistingSource.Post.PostID, "post-1"; got != want {
		t.Fatalf("conflict existing source post = %q, want %q", got, want)
	}
}

func TestSQLiteBatchRepositoryLoadBatchNotFoundAndLifecycle(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	missingID, err := NewBatchRecordID("missing")
	if err != nil {
		t.Fatalf("NewBatchRecordID() error = %v", err)
	}
	got, err := repo.LoadBatch(missingID)
	if !errors.Is(err, ErrBatchRecordNotFound) {
		t.Fatalf("LoadBatch(missing) error = %v, want %v", err, ErrBatchRecordNotFound)
	}
	assertZeroBatchRecord(t, got)

	got, err = repo.LoadBatch("")
	if !errors.Is(err, ErrEmptyBatchRecordID) {
		t.Fatalf("LoadBatch(empty) error = %v, want %v", err, ErrEmptyBatchRecordID)
	}
	assertZeroBatchRecord(t, got)

	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	got, err = repo.LoadBatch(missingID)
	if !errors.Is(err, ErrSQLiteRepositoryClosed) {
		t.Fatalf("LoadBatch(after Close) error = %v, want %v", err, ErrSQLiteRepositoryClosed)
	}
	assertZeroBatchRecord(t, got)

	var nilRepo *SQLiteBatchRepository
	got, err = nilRepo.LoadBatch(missingID)
	if !errors.Is(err, ErrSQLiteRepositoryClosed) {
		t.Fatalf("LoadBatch(nil repo) error = %v, want %v", err, ErrSQLiteRepositoryClosed)
	}
	assertZeroBatchRecord(t, got)
}

func TestSQLiteBatchRepositoryLoadBatchFailsClosedForCorruption(t *testing.T) {
	tests := []struct {
		name    string
		corrupt func(t *testing.T, db *sql.DB)
	}{
		{name: "malformed root timestamp", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE scan_batches SET scan_started_at = 'not-time'")
		}},
		{name: "invalid boolean integer", corrupt: func(t *testing.T, db *sql.DB) {
			withIgnoredSQLiteChecks(t, db, func() { execSQL(t, db, "UPDATE scan_batches SET search_profile_is_enabled = 2") })
		}},
		{name: "unknown geographic mode", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE scan_batches SET geographic_mode = 'foreign'")
		}},
		{name: "unknown rule decision", corrupt: func(t *testing.T, db *sql.DB) {
			withIgnoredSQLiteChecks(t, db, func() { execSQL(t, db, "UPDATE evaluated_posts SET decision = 'hold' WHERE evaluated_position = 0") })
		}},
		{name: "unknown lead outcome", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE lead_outcomes SET match_outcome = 'unknown' WHERE outcome_bucket = 'allowed'")
		}},
		{name: "wrong block outcome for bucket", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE lead_outcomes SET match_outcome = 'blocked' WHERE outcome_bucket = 'allowed'")
		}},
		{name: "missing required group", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "DELETE FROM batch_groups WHERE group_position = 0") })
		}},
		{name: "blank group id", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE batch_groups SET group_id = '' WHERE group_position = 0")
		}},
		{name: "duplicate group position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_batch_groups_position_unique")
			execSQL(t, db, "UPDATE batch_groups SET group_position = 0 WHERE group_id = 'group-b'")
		}},
		{name: "gapped group position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE batch_groups SET group_position = 2 WHERE group_id = 'group-b'")
		}},
		{name: "missing raw post occurrence", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "DELETE FROM raw_post_occurrences WHERE flattened_position = 0") })
		}},
		{name: "duplicate flattened post position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_raw_post_occurrences_flattened_position_unique")
			execSQL(t, db, "UPDATE raw_post_occurrences SET flattened_position = 0 WHERE flattened_position = 1")
		}},
		{name: "gapped flattened post position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE raw_post_occurrences SET flattened_position = 10 WHERE flattened_position = 1")
		}},
		{name: "invalid group reference", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "UPDATE raw_post_occurrences SET group_pk = 999 WHERE flattened_position = 0") })
		}},
		{name: "missing evaluated-post reference", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() {
				execSQL(t, db, "UPDATE evaluated_posts SET post_occurrence_pk = 999 WHERE evaluated_position = 0")
			})
		}},
		{name: "duplicate rule reason position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_evaluated_post_reasons_position_unique")
			execSQL(t, db, "UPDATE evaluated_post_reasons SET reason_position = 0 WHERE reason_category = 'rule' AND reason_position = 1")
		}},
		{name: "gapped rule reason position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE evaluated_post_reasons SET reason_position = 10 WHERE reason_category = 'rule' AND reason_position = 1")
		}},
		{name: "unknown bucket value", corrupt: func(t *testing.T, db *sql.DB) {
			withIgnoredSQLiteChecks(t, db, func() { execSQL(t, db, "UPDATE bucketed_posts SET bucket = 'hold' WHERE bucket = 'review'") })
		}},
		{name: "duplicate bucket position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_bucketed_posts_position_unique")
			execSQL(t, db, "UPDATE bucketed_posts SET bucket_position = 0 WHERE bucket = 'include' AND bucket_position = 1")
		}},
		{name: "gapped bucket position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE bucketed_posts SET bucket_position = 10 WHERE bucket = 'include' AND bucket_position = 1")
		}},
		{name: "missing lead source reference", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "UPDATE lead_sources SET post_occurrence_pk = 999 WHERE lead_source_pk = 1") })
		}},
		{name: "duplicate lead position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_leads_position_unique")
			execSQL(t, db, "UPDATE leads SET lead_position = 0 WHERE lead_position = 1")
		}},
		{name: "gapped lead position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE leads SET lead_position = 10 WHERE lead_position = 1")
		}},
		{name: "duplicate source position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_lead_sources_position_unique")
			execSQL(t, db, "UPDATE lead_sources SET source_position = 0 WHERE lead_pk = 1 AND source_position = 1")
		}},
		{name: "gapped source position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE lead_sources SET source_position = 10 WHERE lead_pk = 1 AND source_position = 1")
		}},
		{name: "missing outcome lead reference", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "UPDATE lead_outcomes SET lead_pk = 999 WHERE lead_outcome_pk = 1") })
		}},
		{name: "duplicate outcome position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_lead_outcomes_position_unique")
			execSQL(t, db, "UPDATE lead_outcomes SET outcome_bucket = 'allowed', bucket_position = 0 WHERE outcome_bucket = 'blocked'")
		}},
		{name: "gapped outcome position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE lead_outcomes SET bucket_position = 10 WHERE outcome_bucket = 'allowed'")
		}},
		{name: "duplicate blocklist reason position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_lead_outcome_blocklist_reasons_position_unique")
			execSQL(t, db, `
				INSERT INTO lead_outcome_blocklist_reasons (lead_outcome_pk, reason_position, reason_code)
				SELECT lead_outcome_pk, 0, 'blocklist.duplicate'
				FROM lead_outcomes
				WHERE outcome_bucket = 'allowed'
			`)
		}},
		{name: "gapped blocklist reason position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE lead_outcome_blocklist_reasons SET reason_position = 10 WHERE reason_pk = 1")
		}},
		{name: "missing unaggregated occurrence", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() {
				execSQL(t, db, "UPDATE unaggregated_posts SET post_occurrence_pk = 999 WHERE unaggregated_pk = 1")
			})
		}},
		{name: "duplicate unaggregated position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_unaggregated_posts_position_unique")
			execSQL(t, db, `
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
				SELECT batch_pk, 0, post_occurrence_pk, candidate_author_kind, candidate_author_value, candidate_need_search_profile_id, candidate_need_normalized_body, candidate_need_body_fingerprint, source_key_kind, source_key_value
				FROM unaggregated_posts
			`)
		}},
		{name: "gapped unaggregated position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE unaggregated_posts SET unaggregated_position = 10")
		}},
		{name: "missing conflict reference", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() {
				execSQL(t, db, "UPDATE source_conflicts SET existing_post_occurrence_pk = 999 WHERE source_conflict_pk = 1")
			})
		}},
		{name: "duplicate conflict position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_source_conflicts_position_unique")
			execSQL(t, db, `
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
				SELECT batch_pk, 0, post_occurrence_pk, existing_post_occurrence_pk, existing_source_key_kind, existing_source_key_value, candidate_author_kind, candidate_author_value, candidate_need_search_profile_id, candidate_need_normalized_body, candidate_need_body_fingerprint, source_key_kind, source_key_value
				FROM source_conflicts
			`)
		}},
		{name: "gapped conflict position", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE source_conflicts SET conflict_position = 10")
		}},
		{name: "missing group summary", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DELETE FROM group_summaries WHERE group_position = 0")
		}},
		{name: "inconsistent stored summary", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "UPDATE scan_batches SET summary_input_post_count = 99")
		}},
		{name: "unsupported root schema version", corrupt: func(t *testing.T, db *sql.DB) {
			withIgnoredSQLiteChecks(t, db, func() { execSQL(t, db, "UPDATE scan_batches SET schema_version = 2") })
		}},
		{name: "missing schema metadata", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DELETE FROM schema_metadata")
		}},
		{name: "missing required table", corrupt: func(t *testing.T, db *sql.DB) {
			withSQLiteForeignKeysOff(t, db, func() { execSQL(t, db, "DROP TABLE source_conflicts") })
		}},
		{name: "missing required index", corrupt: func(t *testing.T, db *sql.DB) {
			execSQL(t, db, "DROP INDEX idx_lead_sources_position_unique")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
			if err != nil {
				t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
			}
			defer closeSQLiteRepo(t, repo)
			record := buildRichSQLiteBatchRecord(t, "batch-corrupt")
			if err := repo.SaveBatch(record); err != nil {
				t.Fatalf("SaveBatch() error = %v", err)
			}
			test.corrupt(t, repo.db)

			got, err := repo.LoadBatch(record.ID())
			if err == nil {
				t.Fatal("LoadBatch() error = nil, want fail-closed error")
			}
			assertZeroBatchRecord(t, got)
		})
	}
}

func TestSQLiteBatchRepositoryLoadBatchDefensiveAndDeterministic(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	record := buildRichSQLiteBatchRecord(t, "batch-deterministic")
	if err := repo.SaveBatch(record); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}

	first, err := repo.LoadBatch(record.ID())
	if err != nil {
		t.Fatalf("LoadBatch(first) error = %v", err)
	}
	first.groups[0].Posts[0].Body = "mutated loaded record"
	first.allowedLeads[0].Lead.Sources[0].Post.Body = "mutated source"
	first.allowedLeads[0].Match.Reasons[0] = "mutated reason"

	second, err := repo.LoadBatch(record.ID())
	if err != nil {
		t.Fatalf("LoadBatch(second) error = %v", err)
	}
	assertBatchRecordsDeeplyEqual(t, second, record)
	if got := record.groups[0].Posts[0].Body; got == "mutated loaded record" {
		t.Fatal("LoadBatch mutated originally saved record")
	}
}

func TestSQLiteBatchRepositoryLoadBatchConcreteOnly(t *testing.T) {
	repositoryType := reflect.TypeOf((*BatchRepository)(nil)).Elem()
	if got, want := repositoryType.NumMethod(), 1; got != want {
		t.Fatalf("BatchRepository method count = %d, want %d", got, want)
	}
	if method, ok := repositoryType.MethodByName("SaveBatch"); !ok || method.Name != "SaveBatch" {
		t.Fatalf("BatchRepository SaveBatch method missing: %#v", repositoryType)
	}
	if _, ok := repositoryType.MethodByName("LoadBatch"); ok {
		t.Fatal("BatchRepository exposes LoadBatch")
	}

	sqliteType := reflect.TypeOf(&SQLiteBatchRepository{})
	if _, ok := sqliteType.MethodByName("LoadBatch"); !ok {
		t.Fatal("SQLiteBatchRepository does not expose concrete LoadBatch")
	}
	for _, method := range []string{"ListBatches", "UpdateBatch", "DeleteBatch", "SearchBatches"} {
		if _, ok := sqliteType.MethodByName(method); ok {
			t.Fatalf("SQLiteBatchRepository exposes deferred API %s", method)
		}
	}
}

func buildOneGroupRichSQLiteBatchRecord(t *testing.T, id string) BatchRecord {
	t.Helper()
	record := buildRichSQLiteBatchRecord(t, id)
	groupID := "group-all"
	groupName := "Group All"
	rewriter := func(post domain.RawPost) domain.RawPost {
		post.GroupID = groupID
		post.GroupName = groupName
		return post
	}
	for i := range record.flattenedPosts {
		record.flattenedPosts[i] = rewriter(record.flattenedPosts[i])
	}
	for i := range record.evaluatedPosts {
		record.evaluatedPosts[i].Post = rewriter(record.evaluatedPosts[i].Post)
	}
	for i := range record.includedPosts {
		record.includedPosts[i].Post = rewriter(record.includedPosts[i].Post)
	}
	for i := range record.reviewPosts {
		record.reviewPosts[i].Post = rewriter(record.reviewPosts[i].Post)
	}
	for i := range record.excludedPosts {
		record.excludedPosts[i].Post = rewriter(record.excludedPosts[i].Post)
	}
	rewriteLeadPosts := func(lead *LeadRecord) {
		for i := range lead.Sources {
			lead.Sources[i].Post = rewriter(lead.Sources[i].Post)
		}
	}
	for i := range record.leads {
		rewriteLeadPosts(&record.leads[i])
	}
	for i := range record.allowedLeads {
		rewriteLeadPosts(&record.allowedLeads[i].Lead)
	}
	for i := range record.blockedLeads {
		rewriteLeadPosts(&record.blockedLeads[i].Lead)
	}
	for i := range record.unresolvedLeads {
		rewriteLeadPosts(&record.unresolvedLeads[i].Lead)
	}
	for i := range record.unaggregated {
		record.unaggregated[i].Post = rewriter(record.unaggregated[i].Post)
	}
	for i := range record.conflicts {
		record.conflicts[i].Post = rewriter(record.conflicts[i].Post)
		record.conflicts[i].ExistingSource.Post = rewriter(record.conflicts[i].ExistingSource.Post)
	}
	record.groups = []GroupRecord{{GroupID: groupID, GroupName: groupName, Posts: copyRawPosts(record.flattenedPosts)}}
	record.summary.GroupCount = 1
	record.groupSummaries = []GroupSummaryRecord{{
		GroupID:            groupID,
		InputPostCount:     record.summary.InputPostCount,
		EvaluatedPostCount: record.summary.EvaluatedPostCount,
		IncludePostCount:   record.summary.IncludePostCount,
		ReviewPostCount:    record.summary.ReviewPostCount,
		ExcludedPostCount:  record.summary.ExcludedPostCount,
	}}
	if err := record.Validate(); err != nil {
		t.Fatalf("one-group rich sqlite batch record invalid: %v", err)
	}
	return record
}

func assertBatchRecordsDeeplyEqual(t *testing.T, got BatchRecord, want BatchRecord) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BatchRecord mismatch:\ngot  %#v\nwant %#v", got, want)
	}
	if !reflect.DeepEqual(got.ScanWindow(), want.ScanWindow()) ||
		!reflect.DeepEqual(got.SearchProfile(), want.SearchProfile()) ||
		got.GeographicMode() != want.GeographicMode() ||
		!reflect.DeepEqual(got.Groups(), want.Groups()) ||
		!reflect.DeepEqual(got.FlattenedPosts(), want.FlattenedPosts()) ||
		!reflect.DeepEqual(got.EvaluatedPosts(), want.EvaluatedPosts()) ||
		!reflect.DeepEqual(got.IncludedPosts(), want.IncludedPosts()) ||
		!reflect.DeepEqual(got.ReviewPosts(), want.ReviewPosts()) ||
		!reflect.DeepEqual(got.ExcludedPosts(), want.ExcludedPosts()) ||
		!reflect.DeepEqual(got.Leads(), want.Leads()) ||
		!reflect.DeepEqual(got.AllowedLeads(), want.AllowedLeads()) ||
		!reflect.DeepEqual(got.BlockedLeads(), want.BlockedLeads()) ||
		!reflect.DeepEqual(got.UnresolvedLeads(), want.UnresolvedLeads()) ||
		!reflect.DeepEqual(got.Unaggregated(), want.Unaggregated()) ||
		!reflect.DeepEqual(got.Conflicts(), want.Conflicts()) ||
		!reflect.DeepEqual(got.Summary(), want.Summary()) ||
		!reflect.DeepEqual(got.GroupSummaries(), want.GroupSummaries()) {
		t.Fatal("one or more BatchRecord accessors did not match")
	}
}

func assertZeroBatchRecord(t *testing.T, got BatchRecord) {
	t.Helper()
	if !reflect.DeepEqual(got, BatchRecord{}) {
		t.Fatalf("LoadBatch returned partial record on failure: %#v", got)
	}
}

func withIgnoredSQLiteChecks(t *testing.T, db *sql.DB, fn func()) {
	t.Helper()
	execSQL(t, db, "PRAGMA ignore_check_constraints = ON")
	defer execSQL(t, db, "PRAGMA ignore_check_constraints = OFF")
	fn()
}

func withSQLiteForeignKeysOff(t *testing.T, db *sql.DB, fn func()) {
	t.Helper()
	execSQL(t, db, "PRAGMA foreign_keys = OFF")
	defer execSQL(t, db, "PRAGMA foreign_keys = ON")
	fn()
}
