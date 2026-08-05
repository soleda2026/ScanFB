package persistence

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestOpenSQLiteBatchRepositoryCreatesVersionedEmptySchema(t *testing.T) {
	path := sqliteTestPath(t)

	repo, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created at explicit path: %v", err)
	}
	assertForeignKeysEnabled(t, repo.db)
	assertSchemaVersion(t, repo.db, CurrentSQLiteSchemaVersion)
	assertSQLiteObjects(t, repo.db, "table", requiredSQLiteTables)
	assertSQLiteObjects(t, repo.db, "index", requiredSQLiteIndexes)
	assertOnlySQLiteFilesInDir(t, filepath.Dir(path), filepath.Base(path))
}

func TestOpenSQLiteBatchRepositoryRejectsInvalidPaths(t *testing.T) {
	if repo, err := OpenSQLiteBatchRepository(" \t\n "); !errors.Is(err, ErrEmptySQLitePath) {
		t.Fatalf("OpenSQLiteBatchRepository(blank) = %#v, %v; want %v", repo, err, ErrEmptySQLitePath)
	}

	path := filepath.Join(t.TempDir(), "missing", "scanfb.sqlite")
	if repo, err := OpenSQLiteBatchRepository(path); err == nil || repo != nil {
		t.Fatalf("OpenSQLiteBatchRepository(missing parent) = %#v, %v; want error", repo, err)
	}
}

func TestSQLiteBatchRepositoryClose(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestSQLiteBatchRepositoryDoesNotSatisfyBatchRepository(t *testing.T) {
	repo := &SQLiteBatchRepository{}
	if _, ok := any(repo).(BatchRepository); ok {
		t.Fatal("SQLiteBatchRepository unexpectedly satisfies BatchRepository before SaveBatch exists")
	}
	if _, ok := reflect.TypeOf(repo).MethodByName("SaveBatch"); ok {
		t.Fatal("SQLiteBatchRepository exposes SaveBatch before durable batch saving is implemented")
	}
}

func TestOpenSQLiteBatchRepositoryReopensExistingSchemaWithoutMutation(t *testing.T) {
	path := sqliteTestPath(t)
	repo, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	insertMinimalScanBatch(t, repo.db, "batch-001")
	closeSQLiteRepo(t, repo)

	reopened, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("reopen existing schema error = %v", err)
	}
	defer closeSQLiteRepo(t, reopened)

	assertSchemaVersion(t, reopened.db, CurrentSQLiteSchemaVersion)
	if got := countRows(t, reopened.db, "schema_metadata"); got != 1 {
		t.Fatalf("metadata rows after reopen = %d, want 1", got)
	}
	if got := countRows(t, reopened.db, "scan_batches"); got != 1 {
		t.Fatalf("scan batch rows after reopen = %d, want 1", got)
	}
}

func TestOpenSQLiteBatchRepositoryRejectsInvalidSchemaMetadata(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *sql.DB)
		want  error
	}{
		{
			name: "missing metadata table",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE stray_table (id INTEGER PRIMARY KEY)")
			},
			want: ErrSQLiteSchemaVersionMissing,
		},
		{
			name: "missing metadata row",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE schema_metadata (metadata_pk INTEGER PRIMARY KEY, current_schema_version INTEGER NOT NULL)")
			},
			want: ErrSQLiteSchemaVersionMissing,
		},
		{
			name: "duplicate metadata rows",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE schema_metadata (metadata_pk INTEGER PRIMARY KEY, current_schema_version INTEGER NOT NULL)")
				execSQL(t, db, "INSERT INTO schema_metadata (metadata_pk, current_schema_version) VALUES (1, 1), (2, 1)")
			},
			want: ErrInvalidSQLiteSchema,
		},
		{
			name: "malformed metadata version",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE schema_metadata (metadata_pk INTEGER PRIMARY KEY, current_schema_version INTEGER NOT NULL)")
				execSQL(t, db, "INSERT INTO schema_metadata (metadata_pk, current_schema_version) VALUES (1, 'not-an-integer')")
			},
			want: ErrInvalidSQLiteSchema,
		},
		{
			name: "unsupported older metadata version",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE schema_metadata (metadata_pk INTEGER PRIMARY KEY, current_schema_version INTEGER NOT NULL)")
				execSQL(t, db, "INSERT INTO schema_metadata (metadata_pk, current_schema_version) VALUES (1, 0)")
			},
			want: ErrUnsupportedSQLiteSchemaVersion,
		},
		{
			name: "unsupported newer metadata version",
			setup: func(t *testing.T, db *sql.DB) {
				execSQL(t, db, "CREATE TABLE schema_metadata (metadata_pk INTEGER PRIMARY KEY, current_schema_version INTEGER NOT NULL)")
				execSQL(t, db, "INSERT INTO schema_metadata (metadata_pk, current_schema_version) VALUES (1, 2)")
			},
			want: ErrUnsupportedSQLiteSchemaVersion,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := sqliteTestPath(t)
			db := openRawSQLite(t, path)
			test.setup(t, db)
			closeRawSQLite(t, db)

			repo, err := OpenSQLiteBatchRepository(path)
			if !errors.Is(err, test.want) {
				t.Fatalf("OpenSQLiteBatchRepository() error = %v, want %v", err, test.want)
			}
			if repo != nil {
				t.Fatalf("repo = %#v, want nil on invalid schema", repo)
			}
		})
	}
}

func TestOpenSQLiteBatchRepositoryRejectsPartialSchema(t *testing.T) {
	path := sqliteTestPath(t)
	repo, err := OpenSQLiteBatchRepository(path)
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	execSQL(t, repo.db, "DROP INDEX idx_lead_sources_position_unique")
	closeSQLiteRepo(t, repo)

	reopened, err := OpenSQLiteBatchRepository(path)
	if !errors.Is(err, ErrInvalidSQLiteSchema) {
		t.Fatalf("OpenSQLiteBatchRepository(partial schema) error = %v, want %v", err, ErrInvalidSQLiteSchema)
	}
	if reopened != nil {
		t.Fatalf("repo = %#v, want nil for partial schema", reopened)
	}
}

func TestInitializeSQLiteSchemaRollsBackOnFailure(t *testing.T) {
	db := openRawSQLite(t, sqliteTestPath(t))
	defer closeRawSQLite(t, db)

	err := initializeSQLiteSchema(context.Background(), db, []string{
		`CREATE TABLE schema_metadata (
			metadata_pk INTEGER PRIMARY KEY CHECK (metadata_pk = 1),
			current_schema_version INTEGER NOT NULL
		)`,
		`CREATE TABLE rolled_back (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE broken_table (`,
	})
	if err == nil {
		t.Fatal("initializeSQLiteSchema() error = nil, want failure")
	}
	if sqliteObjectExists(context.Background(), db, "table", "schema_metadata") {
		t.Fatal("schema_metadata table survived failed initialization transaction")
	}
	if sqliteObjectExists(context.Background(), db, "table", "rolled_back") {
		t.Fatal("intermediate table survived failed initialization transaction")
	}
}

func TestSQLiteSchemaEnforcesForeignKeysAndRepresentativeConstraints(t *testing.T) {
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)

	t.Run("orphan child row rejected", func(t *testing.T) {
		if _, err := repo.db.Exec(`
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (999, 0, 'group-a', 'Group A')
		`); err == nil {
			t.Fatal("orphan batch_groups row insert succeeded, want foreign-key failure")
		}
	})

	t.Run("valid parent child fixture succeeds", func(t *testing.T) {
		tx, err := repo.db.Begin()
		if err != nil {
			t.Fatalf("Begin() error = %v", err)
		}
		defer func() {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				t.Fatalf("Rollback() error = %v", err)
			}
		}()
		insertMinimalScanBatchTx(t, tx, "batch-valid")
		if _, err := tx.Exec(`
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (1, 0, 'group-a', 'Group A')
		`); err != nil {
			t.Fatalf("valid batch_groups insert error = %v", err)
		}
	})

	t.Run("duplicate batch record id rejected", func(t *testing.T) {
		insertMinimalScanBatch(t, repo.db, "batch-duplicate")
		if _, err := repo.db.Exec(minimalScanBatchInsertSQL, "batch-duplicate"); err == nil {
			t.Fatal("duplicate batch_record_id insert succeeded, want unique failure")
		}
	})

	t.Run("duplicate group position rejected", func(t *testing.T) {
		insertMinimalScanBatch(t, repo.db, "batch-group-position")
		batchPK := lastInsertRowID(t, repo.db)
		insertBatchGroup(t, repo.db, batchPK, 0, "group-a")
		if _, err := repo.db.Exec(`
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (?, 0, 'group-b', 'Group B')
		`, batchPK); err == nil {
			t.Fatal("duplicate group_position insert succeeded, want unique failure")
		}
	})

	t.Run("duplicate group id rejected", func(t *testing.T) {
		insertMinimalScanBatch(t, repo.db, "batch-group-id")
		batchPK := lastInsertRowID(t, repo.db)
		insertBatchGroup(t, repo.db, batchPK, 0, "group-a")
		if _, err := repo.db.Exec(`
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (?, 1, 'group-a', 'Group A duplicate')
		`, batchPK); err == nil {
			t.Fatal("duplicate group_id insert succeeded, want unique failure")
		}
	})

	t.Run("duplicate flattened post position rejected", func(t *testing.T) {
		batchPK, groupPK := insertBatchWithGroup(t, repo.db, "batch-flattened-position")
		insertRawPostOccurrence(t, repo.db, batchPK, groupPK, 0, 0)
		if _, err := repo.db.Exec(rawPostOccurrenceInsertSQL, batchPK, groupPK, 0, 1, 0); err == nil {
			t.Fatal("duplicate flattened_position insert succeeded, want unique failure")
		}
	})

	t.Run("duplicate reason position rejected", func(t *testing.T) {
		evaluatedPK := insertEvaluatedPostFixture(t, repo.db, "batch-reason-position")
		insertEvaluatedReason(t, repo.db, evaluatedPK, "rule", 0)
		if _, err := repo.db.Exec(`
			INSERT INTO evaluated_post_reasons (evaluated_post_pk, reason_category, reason_position, reason_code)
			VALUES (?, 'rule', 0, 'included.target_keyword')
		`, evaluatedPK); err == nil {
			t.Fatal("duplicate reason_position insert succeeded, want unique failure")
		}
	})

	t.Run("negative position rejected", func(t *testing.T) {
		insertMinimalScanBatch(t, repo.db, "batch-negative-position")
		batchPK := lastInsertRowID(t, repo.db)
		if _, err := repo.db.Exec(`
			INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
			VALUES (?, -1, 'group-negative', 'Group Negative')
		`, batchPK); err == nil {
			t.Fatal("negative group_position insert succeeded, want check failure")
		}
	})

	t.Run("null required value rejected", func(t *testing.T) {
		if _, err := repo.db.Exec(`
			INSERT INTO scan_batches (batch_record_id)
			VALUES (NULL)
		`); err == nil {
			t.Fatal("NULL required value insert succeeded, want not-null failure")
		}
	})
}

func TestSQLiteSchemaInitializationIsDeterministic(t *testing.T) {
	first := openSQLiteSchemaObjectSnapshot(t)
	second := openSQLiteSchemaObjectSnapshot(t)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("schema object snapshots differ:\nfirst  %#v\nsecond %#v", first, second)
	}
}

func sqliteTestPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "scanfb.sqlite")
}

func openRawSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db
}

func closeRawSQLite(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func closeSQLiteRepo(t *testing.T, repo *SQLiteBatchRepository) {
	t.Helper()
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func assertForeignKeysEnabled(t *testing.T, db *sql.DB) {
	t.Helper()
	var enabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys scan error = %v", err)
	}
	if enabled != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", enabled)
	}
}

func assertSchemaVersion(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT current_schema_version FROM schema_metadata").Scan(&got); err != nil {
		t.Fatalf("schema version scan error = %v", err)
	}
	if got != want {
		t.Fatalf("schema version = %d, want %d", got, want)
	}
}

func assertSQLiteObjects(t *testing.T, db *sql.DB, objectType string, want []string) {
	t.Helper()
	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = ?
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`, objectType)
	if err != nil {
		t.Fatalf("query sqlite %s objects: %v", objectType, err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan sqlite object name: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite objects: %v", err)
	}

	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if !reflect.DeepEqual(got, sortedWant) {
		t.Fatalf("sqlite %s objects = %#v, want %#v", objectType, got, sortedWant)
	}
}

func assertOnlySQLiteFilesInDir(t *testing.T, dir string, databaseFile string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != databaseFile && !strings.HasPrefix(name, databaseFile+"-") {
			t.Fatalf("unexpected file in sqlite temp dir: %s", name)
		}
	}
}

func execSQL(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatalf("Exec(%q) error = %v", statement, err)
	}
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	return count
}

const minimalScanBatchInsertSQL = `
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
	VALUES (?, 1, '2026-08-05T00:00:00Z', '2026-08-05T00:00:00Z', '2026-08-05T03:30:00Z', 'Asia/Ho_Chi_Minh', 'macbook', 'MacBook', 1, 'all_vietnam', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
`

func insertMinimalScanBatch(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	execSQL(t, db, minimalScanBatchInsertSQL, id)
}

func insertMinimalScanBatchTx(t *testing.T, tx *sql.Tx, id string) {
	t.Helper()
	if _, err := tx.Exec(minimalScanBatchInsertSQL, id); err != nil {
		t.Fatalf("insert minimal scan batch in tx: %v", err)
	}
}

func lastInsertRowID(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow("SELECT last_insert_rowid()").Scan(&id); err != nil {
		t.Fatalf("last_insert_rowid() error = %v", err)
	}
	return id
}

func insertBatchWithGroup(t *testing.T, db *sql.DB, batchID string) (int64, int64) {
	t.Helper()
	insertMinimalScanBatch(t, db, batchID)
	batchPK := lastInsertRowID(t, db)
	insertBatchGroup(t, db, batchPK, 0, "group-a")
	groupPK := lastInsertRowID(t, db)
	return batchPK, groupPK
}

func insertBatchGroup(t *testing.T, db *sql.DB, batchPK int64, position int, groupID string) {
	t.Helper()
	execSQL(t, db, `
		INSERT INTO batch_groups (batch_pk, group_position, group_id, group_name)
		VALUES (?, ?, ?, ?)
	`, batchPK, position, groupID, "Group "+groupID)
}

const rawPostOccurrenceInsertSQL = `
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
	VALUES (?, ?, ?, ?, ?, 'post-1', 'group-a', 'Group A', 'https://facebook.example/posts/post-1', 'author-1', '', '', 'Buyer One', 'can mua MacBook HCM', '2026-08-05T02:00:00Z', '2026-08-05T03:30:00Z')
`

func insertRawPostOccurrence(t *testing.T, db *sql.DB, batchPK int64, groupPK int64, groupPostPosition int, flattenedPosition int) {
	t.Helper()
	execSQL(t, db, rawPostOccurrenceInsertSQL, batchPK, groupPK, 0, groupPostPosition, flattenedPosition)
}

func insertEvaluatedPostFixture(t *testing.T, db *sql.DB, batchID string) int64 {
	t.Helper()
	batchPK, groupPK := insertBatchWithGroup(t, db, batchID)
	insertRawPostOccurrence(t, db, batchPK, groupPK, 0, 0)
	postPK := lastInsertRowID(t, db)
	execSQL(t, db, `
		INSERT INTO evaluated_posts (batch_pk, post_occurrence_pk, evaluated_position, decision, geographic_class, geographic_reason_set_present)
		VALUES (?, ?, 0, 'include', 'hcm', 0)
	`, batchPK, postPK)
	return lastInsertRowID(t, db)
}

func insertEvaluatedReason(t *testing.T, db *sql.DB, evaluatedPK int64, category string, position int) {
	t.Helper()
	execSQL(t, db, `
		INSERT INTO evaluated_post_reasons (evaluated_post_pk, reason_category, reason_position, reason_code)
		VALUES (?, ?, ?, 'included.buyer_intent')
	`, evaluatedPK, category, position)
}

type sqliteObjectSnapshot struct {
	Tables  []string
	Indexes []string
}

func openSQLiteSchemaObjectSnapshot(t *testing.T) sqliteObjectSnapshot {
	t.Helper()
	repo, err := OpenSQLiteBatchRepository(sqliteTestPath(t))
	if err != nil {
		t.Fatalf("OpenSQLiteBatchRepository() error = %v", err)
	}
	defer closeSQLiteRepo(t, repo)
	return sqliteObjectSnapshot{
		Tables:  sqliteObjectNames(t, repo.db, "table"),
		Indexes: sqliteObjectNames(t, repo.db, "index"),
	}
}

func sqliteObjectNames(t *testing.T, db *sql.DB, objectType string) []string {
	t.Helper()
	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = ?
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`, objectType)
	if err != nil {
		t.Fatalf("query sqlite %s objects: %v", objectType, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan sqlite object name: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite objects: %v", err)
	}
	return names
}
