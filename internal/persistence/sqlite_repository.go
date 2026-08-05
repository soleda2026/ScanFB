package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const CurrentSQLiteSchemaVersion = 1

var (
	ErrEmptySQLitePath                = errors.New("empty sqlite path")
	ErrSQLiteForeignKeysDisabled      = errors.New("sqlite foreign keys disabled")
	ErrSQLiteSchemaVersionMissing     = errors.New("sqlite schema version missing")
	ErrUnsupportedSQLiteSchemaVersion = errors.New("unsupported sqlite schema version")
	ErrInvalidSQLiteSchema            = errors.New("invalid sqlite schema")
	ErrSQLiteRepositoryClosed         = errors.New("sqlite repository closed")
)

// SQLiteBatchRepository opens, validates, and writes completed local SQLite
// batch snapshots.
type SQLiteBatchRepository struct {
	db *sql.DB
}

var _ BatchRepository = (*SQLiteBatchRepository)(nil)

func OpenSQLiteBatchRepository(path string) (*SQLiteBatchRepository, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrEmptySQLitePath
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := enableSQLiteForeignKeys(ctx, db); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, err
	}
	if err := ensureSQLiteSchema(ctx, db); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, err
	}

	return &SQLiteBatchRepository{db: db}, nil
}

func (repo *SQLiteBatchRepository) Close() error {
	if repo == nil || repo.db == nil {
		return nil
	}
	db := repo.db
	if err := db.Close(); err != nil {
		return err
	}
	repo.db = nil
	return nil
}

func closeSQLiteAfterOpenFailure(db *sql.DB) {
	_ = db.Close()
}

func enableSQLiteForeignKeys(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}

	var enabled int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return fmt.Errorf("verify sqlite foreign keys: %w", err)
	}
	if enabled != 1 {
		return ErrSQLiteForeignKeysDisabled
	}
	return nil
}

func ensureSQLiteSchema(ctx context.Context, db *sql.DB) error {
	tableCount, err := userTableCount(ctx, db)
	if err != nil {
		return err
	}
	if tableCount == 0 {
		if err := initializeSQLiteSchema(ctx, db, sqliteSchemaStatements()); err != nil {
			return err
		}
	}
	return validateSQLiteSchema(ctx, db)
}

func userTableCount(ctx context.Context, db *sql.DB) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'table'
			AND name NOT LIKE 'sqlite_%'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("inspect sqlite schema: %w", err)
	}
	return count, nil
}

func initializeSQLiteSchema(ctx context.Context, db *sql.DB, statements []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite schema transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize sqlite schema: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_metadata (metadata_pk, current_schema_version)
		VALUES (1, ?)
	`, CurrentSQLiteSchemaVersion); err != nil {
		return fmt.Errorf("initialize sqlite schema metadata: %w", err)
	}
	if err := validateSQLiteSchema(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite schema transaction: %w", err)
	}
	committed = true
	return nil
}

type sqliteSchemaQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func validateSQLiteSchema(ctx context.Context, q sqliteSchemaQueryer) error {
	if !sqliteObjectExists(ctx, q, "table", "schema_metadata") {
		return ErrSQLiteSchemaVersionMissing
	}

	var metadataRows int
	if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_metadata").Scan(&metadataRows); err != nil {
		return fmt.Errorf("%w: read metadata count: %v", ErrInvalidSQLiteSchema, err)
	}
	switch {
	case metadataRows == 0:
		return ErrSQLiteSchemaVersionMissing
	case metadataRows > 1:
		return fmt.Errorf("%w: duplicate schema metadata rows", ErrInvalidSQLiteSchema)
	}

	var version int
	if err := q.QueryRowContext(ctx, "SELECT current_schema_version FROM schema_metadata").Scan(&version); err != nil {
		return fmt.Errorf("%w: read schema version: %v", ErrInvalidSQLiteSchema, err)
	}
	if version != CurrentSQLiteSchemaVersion {
		return fmt.Errorf("%w: %d", ErrUnsupportedSQLiteSchemaVersion, version)
	}

	for _, table := range requiredSQLiteTables {
		if !sqliteObjectExists(ctx, q, "table", table) {
			return fmt.Errorf("%w: missing table %s", ErrInvalidSQLiteSchema, table)
		}
	}
	for _, index := range requiredSQLiteIndexes {
		if !sqliteObjectExists(ctx, q, "index", index) {
			return fmt.Errorf("%w: missing index %s", ErrInvalidSQLiteSchema, index)
		}
	}
	return nil
}

func sqliteObjectExists(ctx context.Context, q sqliteSchemaQueryer, objectType string, name string) bool {
	var count int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = ?
			AND name = ?
	`, objectType, name).Scan(&count)
	return err == nil && count == 1
}
