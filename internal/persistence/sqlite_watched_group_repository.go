package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

const (
	CurrentSQLiteWatchedGroupSchemaVersion = 1
	ScanFBApplicationDirectoryName         = "com.soleda.ScanFB"
	WatchedGroupDatabaseFilename           = "watched-groups.sqlite3"
)

var (
	ErrEmptySQLiteWatchedGroupPath          = errors.New("empty sqlite watched group path")
	ErrInvalidSQLiteWatchedGroupSchema      = errors.New("invalid sqlite watched group schema")
	ErrSQLiteWatchedGroupSchemaVersion      = errors.New("unsupported sqlite watched group schema version")
	ErrInvalidStoredWatchedGroupState       = errors.New("invalid stored watched group state")
	ErrSQLiteWatchedGroupRepositoryClosed   = errors.New("sqlite watched group repository closed")
	ErrWatchedGroupApplicationSupportPath   = errors.New("watched group application support path unavailable")
	ErrWatchedGroupApplicationSupportCreate = errors.New("watched group application support directory creation failed")
)

// SQLiteWatchedGroupRepository owns the independent schema-v1 group state DB.
type SQLiteWatchedGroupRepository struct {
	db *sql.DB
}

func ResolveWatchedGroupDatabasePath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrWatchedGroupApplicationSupportPath, err)
	}
	return watchedGroupDatabasePath(root)
}

func OpenProductionSQLiteWatchedGroupRepository() (*SQLiteWatchedGroupRepository, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWatchedGroupApplicationSupportPath, err)
	}
	return openSQLiteWatchedGroupRepositoryAtApplicationSupport(root)
}

func watchedGroupDatabasePath(applicationSupportRoot string) (string, error) {
	root := strings.TrimSpace(applicationSupportRoot)
	if root == "" {
		return "", ErrWatchedGroupApplicationSupportPath
	}
	return filepath.Join(root, ScanFBApplicationDirectoryName, WatchedGroupDatabaseFilename), nil
}

func openSQLiteWatchedGroupRepositoryAtApplicationSupport(root string) (*SQLiteWatchedGroupRepository, error) {
	path, err := watchedGroupDatabasePath(root)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWatchedGroupApplicationSupportCreate, err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWatchedGroupApplicationSupportCreate, err)
	}
	return OpenSQLiteWatchedGroupRepository(path)
}

func OpenSQLiteWatchedGroupRepository(path string) (*SQLiteWatchedGroupRepository, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrEmptySQLiteWatchedGroupPath
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite watched group database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, fmt.Errorf("open sqlite watched group database: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, fmt.Errorf("set sqlite watched group database permissions: %w", err)
	}
	if err := configureSQLiteWatchedGroupJournal(ctx, db); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, err
	}
	if err := ensureSQLiteWatchedGroupSchema(ctx, db); err != nil {
		closeSQLiteAfterOpenFailure(db)
		return nil, err
	}

	return &SQLiteWatchedGroupRepository{db: db}, nil
}

func (repo *SQLiteWatchedGroupRepository) Close() error {
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

func (repo *SQLiteWatchedGroupRepository) Load() (WatchedGroupState, error) {
	if repo == nil || repo.db == nil {
		return WatchedGroupState{}, ErrSQLiteWatchedGroupRepositoryClosed
	}
	tx, err := repo.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return WatchedGroupState{}, fmt.Errorf("begin sqlite watched group load: %w", err)
	}
	defer tx.Rollback()

	state, err := loadSQLiteWatchedGroupState(context.Background(), tx)
	if err != nil {
		return WatchedGroupState{}, err
	}
	if err := tx.Commit(); err != nil {
		return WatchedGroupState{}, fmt.Errorf("commit sqlite watched group load: %w", err)
	}
	return state, nil
}

func (repo *SQLiteWatchedGroupRepository) Add(group domain.WatchedGroup) (WatchedGroupState, error) {
	if err := group.Validate(); err != nil {
		return WatchedGroupState{}, err
	}
	return repo.mutate(func(ctx context.Context, tx *sql.Tx, state WatchedGroupState) (WatchedGroupState, error) {
		collection, err := watchedGroupCollectionFromState(state)
		if err != nil {
			return WatchedGroupState{}, err
		}
		if err := collection.Add(group); err != nil {
			return WatchedGroupState{}, err
		}
		if err := insertSQLiteWatchedGroup(ctx, tx, len(state.groups), group); err != nil {
			return WatchedGroupState{}, err
		}
		return newWatchedGroupState(collection.Groups(), state.cursor), nil
	})
}

func (repo *SQLiteWatchedGroupRepository) SetActive(id string, active bool) (WatchedGroupState, error) {
	return repo.mutate(func(ctx context.Context, tx *sql.Tx, state WatchedGroupState) (WatchedGroupState, error) {
		collection, err := watchedGroupCollectionFromState(state)
		if err != nil {
			return WatchedGroupState{}, err
		}
		if active {
			_, err = collection.Activate(id)
		} else {
			_, err = collection.Deactivate(id)
		}
		if err != nil {
			return WatchedGroupState{}, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE watched_groups SET active = ? WHERE local_id = ?`, watchedGroupSQLiteBool(active), strings.TrimSpace(id))
		if err != nil {
			return WatchedGroupState{}, fmt.Errorf("update sqlite watched group active state: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return WatchedGroupState{}, application.ErrWatchedGroupNotFound
		}
		return newWatchedGroupState(collection.Groups(), state.cursor), nil
	})
}

func (repo *SQLiteWatchedGroupRepository) AdvanceCursor() (WatchedGroupState, error) {
	return repo.mutate(func(ctx context.Context, tx *sql.Tx, state WatchedGroupState) (WatchedGroupState, error) {
		selection, err := application.SelectNextFiveActiveGroups(state.Groups(), state.cursor)
		if err != nil {
			return WatchedGroupState{}, err
		}
		next := selection.NextCursor()
		if _, err := tx.ExecContext(ctx, `UPDATE watched_group_selection_state SET cursor_position = ? WHERE state_pk = 1`, next.Position()); err != nil {
			return WatchedGroupState{}, fmt.Errorf("update sqlite watched group cursor: %w", err)
		}
		return newWatchedGroupState(state.groups, next), nil
	})
}

func (repo *SQLiteWatchedGroupRepository) mutate(operation func(context.Context, *sql.Tx, WatchedGroupState) (WatchedGroupState, error)) (WatchedGroupState, error) {
	if repo == nil || repo.db == nil {
		return WatchedGroupState{}, ErrSQLiteWatchedGroupRepositoryClosed
	}
	ctx := context.Background()
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return WatchedGroupState{}, fmt.Errorf("begin sqlite watched group mutation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	state, err := loadSQLiteWatchedGroupState(ctx, tx)
	if err != nil {
		return WatchedGroupState{}, err
	}
	updated, err := operation(ctx, tx, state)
	if err != nil {
		return WatchedGroupState{}, err
	}
	if err := tx.Commit(); err != nil {
		return WatchedGroupState{}, fmt.Errorf("commit sqlite watched group mutation: %w", err)
	}
	committed = true
	return updated, nil
}

func configureSQLiteWatchedGroupJournal(ctx context.Context, db *sql.DB) error {
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode = DELETE").Scan(&mode); err != nil {
		return fmt.Errorf("configure sqlite watched group journal: %w", err)
	}
	if !strings.EqualFold(mode, "delete") {
		return fmt.Errorf("%w: journal mode %s", ErrInvalidSQLiteWatchedGroupSchema, mode)
	}
	return nil
}

func ensureSQLiteWatchedGroupSchema(ctx context.Context, db *sql.DB) error {
	count, err := watchedGroupUserTableCount(ctx, db)
	if err != nil {
		return err
	}
	if count == 0 {
		if err := initializeSQLiteWatchedGroupSchema(ctx, db); err != nil {
			return err
		}
	}
	return validateSQLiteWatchedGroupSchema(ctx, db)
}

func watchedGroupUserTableCount(ctx context.Context, db *sql.DB) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&count); err != nil {
		return 0, fmt.Errorf("inspect sqlite watched group schema: %w", err)
	}
	return count, nil
}

func initializeSQLiteWatchedGroupSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sqlite watched group schema: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	for _, statement := range sqliteWatchedGroupSchemaV1Statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize sqlite watched group schema: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO watched_group_schema_metadata (metadata_pk, current_schema_version) VALUES (1, ?)`, CurrentSQLiteWatchedGroupSchemaVersion); err != nil {
		return fmt.Errorf("initialize sqlite watched group metadata: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO watched_group_selection_state (state_pk, cursor_position) VALUES (1, 0)`); err != nil {
		return fmt.Errorf("initialize sqlite watched group cursor: %w", err)
	}
	if err := validateSQLiteWatchedGroupSchema(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite watched group schema: %w", err)
	}
	committed = true
	return nil
}

type watchedGroupSchemaQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func validateSQLiteWatchedGroupSchema(ctx context.Context, q watchedGroupSchemaQueryer) error {
	var metadataRows int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'watched_group_schema_metadata'`).Scan(&metadataRows); err != nil || metadataRows != 1 {
		return ErrInvalidSQLiteWatchedGroupSchema
	}
	var version int
	if err := q.QueryRowContext(ctx, `SELECT current_schema_version FROM watched_group_schema_metadata WHERE metadata_pk = 1`).Scan(&version); err != nil {
		return fmt.Errorf("%w: metadata", ErrInvalidSQLiteWatchedGroupSchema)
	}
	if version != CurrentSQLiteWatchedGroupSchemaVersion {
		return fmt.Errorf("%w: %d", ErrSQLiteWatchedGroupSchemaVersion, version)
	}
	tables, err := sqliteWatchedGroupObjectNames(ctx, q, "table")
	if err != nil || !reflect.DeepEqual(tables, sortedStrings(requiredSQLiteWatchedGroupTables)) {
		return fmt.Errorf("%w: table inventory", ErrInvalidSQLiteWatchedGroupSchema)
	}
	indexes, err := sqliteWatchedGroupObjectNames(ctx, q, "index")
	if err != nil || !reflect.DeepEqual(indexes, sortedStrings(requiredSQLiteWatchedGroupIndexes)) {
		return fmt.Errorf("%w: index inventory", ErrInvalidSQLiteWatchedGroupSchema)
	}
	return nil
}

func sqliteWatchedGroupObjectNames(ctx context.Context, q watchedGroupSchemaQueryer, objectType string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = ? AND name NOT LIKE 'sqlite_%' ORDER BY name`, objectType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func sortedStrings(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

type watchedGroupStateQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadSQLiteWatchedGroupState(ctx context.Context, q watchedGroupStateQueryer) (WatchedGroupState, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT position, local_id, facebook_group_id, canonical_url, name, created_at,
			active, notes, last_successful_scan_at, last_error, display_order
		FROM watched_groups ORDER BY position
	`)
	if err != nil {
		return WatchedGroupState{}, fmt.Errorf("load sqlite watched groups: %w", err)
	}
	defer rows.Close()

	collection := application.NewWatchedGroupCollection()
	position := 0
	for rows.Next() {
		group, storedPosition, err := scanSQLiteWatchedGroup(rows)
		if err != nil {
			return WatchedGroupState{}, err
		}
		if storedPosition != position {
			return WatchedGroupState{}, fmt.Errorf("%w: insertion position", ErrInvalidStoredWatchedGroupState)
		}
		if err := collection.Add(group); err != nil {
			return WatchedGroupState{}, fmt.Errorf("%w: %v", ErrInvalidStoredWatchedGroupState, err)
		}
		position++
	}
	if err := rows.Err(); err != nil {
		return WatchedGroupState{}, fmt.Errorf("iterate sqlite watched groups: %w", err)
	}

	var cursorPosition int
	var cursorRows int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM watched_group_selection_state`).Scan(&cursorRows); err != nil || cursorRows != 1 {
		return WatchedGroupState{}, fmt.Errorf("%w: cursor row", ErrInvalidStoredWatchedGroupState)
	}
	if err := q.QueryRowContext(ctx, `SELECT cursor_position FROM watched_group_selection_state WHERE state_pk = 1`).Scan(&cursorPosition); err != nil {
		return WatchedGroupState{}, fmt.Errorf("%w: cursor", ErrInvalidStoredWatchedGroupState)
	}
	groups := collection.Groups()
	if (len(groups) == 0 && cursorPosition != 0) || (len(groups) > 0 && (cursorPosition < 0 || cursorPosition >= len(groups))) {
		return WatchedGroupState{}, fmt.Errorf("%w: cursor", ErrInvalidStoredWatchedGroupState)
	}
	cursor, err := application.NewWatchedGroupSelectionCursor(cursorPosition)
	if err != nil {
		return WatchedGroupState{}, fmt.Errorf("%w: cursor", ErrInvalidStoredWatchedGroupState)
	}
	return newWatchedGroupState(groups, cursor), nil
}

type watchedGroupRowScanner interface {
	Scan(...any) error
}

func scanSQLiteWatchedGroup(row watchedGroupRowScanner) (domain.WatchedGroup, int, error) {
	var (
		position           int
		id                 string
		facebookGroupID    sql.NullString
		canonicalURL       sql.NullString
		name               string
		createdAtText      string
		active             int
		notes              string
		lastSuccessfulText sql.NullString
		lastError          string
		displayOrder       int
	)
	if err := row.Scan(&position, &id, &facebookGroupID, &canonicalURL, &name, &createdAtText, &active, &notes, &lastSuccessfulText, &lastError, &displayOrder); err != nil {
		return domain.WatchedGroup{}, 0, fmt.Errorf("%w: group row", ErrInvalidStoredWatchedGroupState)
	}
	if active != 0 && active != 1 {
		return domain.WatchedGroup{}, 0, fmt.Errorf("%w: active", ErrInvalidStoredWatchedGroupState)
	}
	createdAt, err := parseWatchedGroupSQLiteTime(createdAtText)
	if err != nil {
		return domain.WatchedGroup{}, 0, err
	}
	group, err := domain.NewWatchedGroup(id, nullableStringValue(facebookGroupID), nullableStringValue(canonicalURL), name, createdAt)
	if err != nil {
		return domain.WatchedGroup{}, 0, fmt.Errorf("%w: %v", ErrInvalidStoredWatchedGroupState, err)
	}
	var lastSuccessful time.Time
	if lastSuccessfulText.Valid {
		lastSuccessful, err = parseWatchedGroupSQLiteTime(lastSuccessfulText.String)
		if err != nil {
			return domain.WatchedGroup{}, 0, err
		}
	}
	group, err = group.WithMetadata(domain.WatchedGroupMetadata{
		Name:                 name,
		Notes:                notes,
		LastSuccessfulScanAt: lastSuccessful,
		LastError:            lastError,
		DisplayOrder:         displayOrder,
	})
	if err != nil {
		return domain.WatchedGroup{}, 0, fmt.Errorf("%w: %v", ErrInvalidStoredWatchedGroupState, err)
	}
	return group.WithActive(active == 1), position, nil
}

func watchedGroupCollectionFromState(state WatchedGroupState) (*application.WatchedGroupCollection, error) {
	collection := application.NewWatchedGroupCollection()
	for _, group := range state.Groups() {
		if err := collection.Add(group); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidStoredWatchedGroupState, err)
		}
	}
	return collection, nil
}

func insertSQLiteWatchedGroup(ctx context.Context, tx *sql.Tx, position int, group domain.WatchedGroup) error {
	lastSuccessful, hasLastSuccessful := group.LastSuccessfulScanAt()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO watched_groups (
			position, local_id, facebook_group_id, canonical_url, name, created_at,
			active, notes, last_successful_scan_at, last_error, display_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, position, group.ID(), nullableSQLiteString(group.FacebookGroupID()), nullableSQLiteString(group.CanonicalURL()), group.Name(), watchedGroupSQLiteTime(group.CreatedAt()), watchedGroupSQLiteBool(group.IsActive()), group.Notes(), nullableSQLiteTime(lastSuccessful, hasLastSuccessful), group.LastError(), group.DisplayOrder())
	if err != nil {
		return fmt.Errorf("insert sqlite watched group: %w", err)
	}
	return nil
}

func parseWatchedGroupSQLiteTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || watchedGroupSQLiteTime(parsed) != value {
		return time.Time{}, fmt.Errorf("%w: timestamp", ErrInvalidStoredWatchedGroupState)
	}
	return parsed, nil
}

func watchedGroupSQLiteTime(value time.Time) string {
	return value.Format(time.RFC3339Nano)
}

func watchedGroupSQLiteBool(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableSQLiteString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableSQLiteTime(value time.Time, valid bool) any {
	if !valid {
		return nil
	}
	return watchedGroupSQLiteTime(value)
}

func nullableStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
