package persistence

var requiredSQLiteWatchedGroupTables = []string{
	"watched_group_schema_metadata",
	"watched_groups",
	"watched_group_selection_state",
}

var requiredSQLiteWatchedGroupIndexes = []string{
	"idx_watched_groups_local_id_unique",
	"idx_watched_groups_facebook_id_unique",
	"idx_watched_groups_url_only_unique",
}

var sqliteWatchedGroupSchemaV1Statements = []string{
	`CREATE TABLE watched_group_schema_metadata (
		metadata_pk INTEGER PRIMARY KEY CHECK (metadata_pk = 1),
		current_schema_version INTEGER NOT NULL
	)`,
	`CREATE TABLE watched_groups (
		position INTEGER PRIMARY KEY CHECK (position >= 0),
		local_id TEXT NOT NULL,
		facebook_group_id TEXT,
		canonical_url TEXT,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL,
		active INTEGER NOT NULL CHECK (active IN (0, 1)),
		notes TEXT NOT NULL,
		last_successful_scan_at TEXT,
		last_error TEXT NOT NULL,
		display_order INTEGER NOT NULL,
		CHECK (
			(facebook_group_id IS NOT NULL AND length(facebook_group_id) > 0)
			OR (canonical_url IS NOT NULL AND length(canonical_url) > 0)
		)
	)`,
	`CREATE TABLE watched_group_selection_state (
		state_pk INTEGER PRIMARY KEY CHECK (state_pk = 1),
		cursor_position INTEGER NOT NULL CHECK (cursor_position >= 0)
	)`,
	`CREATE UNIQUE INDEX idx_watched_groups_local_id_unique
		ON watched_groups(local_id)`,
	`CREATE UNIQUE INDEX idx_watched_groups_facebook_id_unique
		ON watched_groups(facebook_group_id)
		WHERE facebook_group_id IS NOT NULL`,
	`CREATE UNIQUE INDEX idx_watched_groups_url_only_unique
		ON watched_groups(canonical_url)
		WHERE facebook_group_id IS NULL AND canonical_url IS NOT NULL`,
}
