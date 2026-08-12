package persistence

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

func TestSQLiteWatchedGroupRepositoryInitializesIndependentSchemaV1(t *testing.T) {
	path := watchedGroupTestPath(t)
	repo := openWatchedGroupTestRepository(t, path)
	defer closeWatchedGroupTestRepository(t, repo)

	state, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(state.Groups()) != 0 || state.Cursor().Position() != 0 {
		t.Fatalf("empty state = groups %d cursor %d", len(state.Groups()), state.Cursor().Position())
	}

	var version int
	if err := repo.db.QueryRow(`SELECT current_schema_version FROM watched_group_schema_metadata WHERE metadata_pk = 1`).Scan(&version); err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	if version != CurrentSQLiteWatchedGroupSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, CurrentSQLiteWatchedGroupSchemaVersion)
	}
	assertWatchedGroupJournalMode(t, repo, "delete")
	assertWatchedGroupFileMode(t, path, 0o600)
	assertNoCompletedBatchTables(t, repo)
}

func TestProductionWatchedGroupPathAndPermissionsUseApplicationSupportRoot(t *testing.T) {
	root := t.TempDir()
	path, err := watchedGroupDatabasePath(root)
	if err != nil {
		t.Fatalf("watchedGroupDatabasePath() error = %v", err)
	}
	want := filepath.Join(root, "com.soleda.ScanFB", "watched-groups.sqlite3")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	repo, err := openSQLiteWatchedGroupRepositoryAtApplicationSupport(root)
	if err != nil {
		t.Fatalf("open production-shaped repository: %v", err)
	}
	closeWatchedGroupTestRepository(t, repo)
	assertWatchedGroupFileMode(t, filepath.Dir(path), 0o700)
	assertWatchedGroupFileMode(t, path, 0o600)
}

func TestSQLiteWatchedGroupRepositoryPersistsFullValuesAndInsertionOrderAcrossReopen(t *testing.T) {
	path := watchedGroupTestPath(t)
	repo := openWatchedGroupTestRepository(t, path)
	first := watchedGroupFixture(t, "local-a", "facebook-a", "https://www.facebook.com/groups/a", 3, false)
	second := watchedGroupFixture(t, "local-b", "", "https://www.facebook.com/groups/b", 7, true)

	if _, err := repo.Add(first); err != nil {
		t.Fatalf("Add(first) error = %v", err)
	}
	if _, err := repo.Add(second); err != nil {
		t.Fatalf("Add(second) error = %v", err)
	}
	closeWatchedGroupTestRepository(t, repo)

	reopened := openWatchedGroupTestRepository(t, path)
	defer closeWatchedGroupTestRepository(t, reopened)
	state, err := reopened.Load()
	if err != nil {
		t.Fatalf("Load() after reopen error = %v", err)
	}
	want := []domain.WatchedGroup{first, second}
	if !watchedGroupsEqual(state.Groups(), want) {
		t.Fatalf("restored groups = %#v, want %#v", state.Groups(), want)
	}
	if state.Cursor().Position() != 0 {
		t.Fatalf("cursor = %d, want 0", state.Cursor().Position())
	}
}

func TestSQLiteWatchedGroupRepositoryPersistsDeactivateAndReactivate(t *testing.T) {
	path := watchedGroupTestPath(t)
	repo := openWatchedGroupTestRepository(t, path)
	defer closeWatchedGroupTestRepository(t, repo)
	group := watchedGroupFixture(t, "local-a", "facebook-a", "", 0, true)
	if _, err := repo.Add(group); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	state, err := repo.SetActive(group.ID(), false)
	if err != nil || state.Groups()[0].IsActive() {
		t.Fatalf("SetActive(false) state=%#v error=%v", state, err)
	}
	state, err = repo.SetActive(group.ID(), true)
	if err != nil || !state.Groups()[0].IsActive() {
		t.Fatalf("SetActive(true) state=%#v error=%v", state, err)
	}
}

func TestSQLiteWatchedGroupRepositoryRejectsIdentityConflictsWithoutMutation(t *testing.T) {
	tests := []struct {
		name     string
		existing domain.WatchedGroup
		incoming domain.WatchedGroup
		want     error
	}{
		{
			name:     "duplicate local id",
			existing: watchedGroupFixture(t, "same", "facebook-a", "", 0, true),
			incoming: watchedGroupFixture(t, "same", "facebook-b", "", 1, true),
			want:     application.ErrDuplicateWatchedGroupID,
		},
		{
			name:     "duplicate facebook identity",
			existing: watchedGroupFixture(t, "local-a", "facebook-a", "", 0, true),
			incoming: watchedGroupFixture(t, "local-b", "facebook-a", "", 1, true),
			want:     application.ErrDuplicateWatchedGroupIdentity,
		},
		{
			name:     "duplicate url-only identity",
			existing: watchedGroupFixture(t, "local-a", "", "https://www.facebook.com/groups/shared", 0, true),
			incoming: watchedGroupFixture(t, "local-b", "", "https://www.facebook.com/groups/shared", 1, true),
			want:     application.ErrDuplicateWatchedGroupIdentity,
		},
		{
			name:     "cross-kind canonical conflict",
			existing: watchedGroupFixture(t, "local-a", "", "https://www.facebook.com/groups/shared", 0, true),
			incoming: watchedGroupFixture(t, "local-b", "facebook-b", "https://www.facebook.com/groups/shared", 1, true),
			want:     application.ErrDuplicateWatchedGroupIdentity,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := openWatchedGroupTestRepository(t, watchedGroupTestPath(t))
			defer closeWatchedGroupTestRepository(t, repo)
			if _, err := repo.Add(test.existing); err != nil {
				t.Fatalf("Add(existing) error = %v", err)
			}
			before, err := repo.Load()
			if err != nil {
				t.Fatalf("Load(before) error = %v", err)
			}
			if _, err := repo.Add(test.incoming); !errors.Is(err, test.want) {
				t.Fatalf("Add(conflict) error = %v, want %v", err, test.want)
			}
			after, err := repo.Load()
			if err != nil {
				t.Fatalf("Load(after) error = %v", err)
			}
			if !watchedGroupsEqual(after.Groups(), before.Groups()) {
				t.Fatalf("failed add mutated state: before=%#v after=%#v", before.Groups(), after.Groups())
			}
		})
	}
}

func TestSQLiteWatchedGroupRepositoryAllowsDifferentFacebookIDsWithSharedSecondaryURL(t *testing.T) {
	repo := openWatchedGroupTestRepository(t, watchedGroupTestPath(t))
	defer closeWatchedGroupTestRepository(t, repo)
	url := "https://www.facebook.com/groups/shared"
	if _, err := repo.Add(watchedGroupFixture(t, "local-a", "facebook-a", url, 0, true)); err != nil {
		t.Fatalf("Add(first) error = %v", err)
	}
	if _, err := repo.Add(watchedGroupFixture(t, "local-b", "facebook-b", url, 1, true)); err != nil {
		t.Fatalf("Add(second) error = %v", err)
	}
}

func TestSQLiteWatchedGroupRepositoryAdvanceCursorPersistsAndAffectsReopenSelection(t *testing.T) {
	path := watchedGroupTestPath(t)
	repo := openWatchedGroupTestRepository(t, path)
	for i := 0; i < 6; i++ {
		id := "local-" + string(rune('a'+i))
		if _, err := repo.Add(watchedGroupFixture(t, id, "facebook-"+id, "", i, true)); err != nil {
			t.Fatalf("Add(%s) error = %v", id, err)
		}
	}
	state, err := repo.AdvanceCursor()
	if err != nil {
		t.Fatalf("AdvanceCursor() error = %v", err)
	}
	if state.Cursor().Position() != 5 {
		t.Fatalf("advanced cursor = %d, want 5", state.Cursor().Position())
	}
	closeWatchedGroupTestRepository(t, repo)

	reopened := openWatchedGroupTestRepository(t, path)
	defer closeWatchedGroupTestRepository(t, reopened)
	restored, err := reopened.Load()
	if err != nil {
		t.Fatalf("Load() after reopen error = %v", err)
	}
	selection, err := application.SelectNextFiveActiveGroups(restored.Groups(), restored.Cursor())
	if err != nil {
		t.Fatalf("SelectNextFiveActiveGroups() error = %v", err)
	}
	want := []string{"local-f", "local-a", "local-b", "local-c", "local-d"}
	for i, group := range selection.Groups() {
		if group.ID() != want[i] {
			t.Fatalf("selection[%d] = %q, want %q", i, group.ID(), want[i])
		}
	}
}

func TestSQLiteWatchedGroupRepositoryFailedMutationsPreserveState(t *testing.T) {
	repo := openWatchedGroupTestRepository(t, watchedGroupTestPath(t))
	defer closeWatchedGroupTestRepository(t, repo)
	for i := 0; i < 4; i++ {
		id := "local-" + string(rune('a'+i))
		if _, err := repo.Add(watchedGroupFixture(t, id, "facebook-"+id, "", i, true)); err != nil {
			t.Fatalf("Add(%s) error = %v", id, err)
		}
	}
	before, err := repo.Load()
	if err != nil {
		t.Fatalf("Load(before) error = %v", err)
	}
	if _, err := repo.SetActive("missing", false); !errors.Is(err, application.ErrWatchedGroupNotFound) {
		t.Fatalf("SetActive(missing) error = %v", err)
	}
	if _, err := repo.AdvanceCursor(); !errors.Is(err, application.ErrInsufficientActiveWatchedGroups) {
		t.Fatalf("AdvanceCursor(insufficient) error = %v", err)
	}
	after, err := repo.Load()
	if err != nil {
		t.Fatalf("Load(after) error = %v", err)
	}
	if !watchedGroupsEqual(after.Groups(), before.Groups()) || after.Cursor().Position() != before.Cursor().Position() {
		t.Fatalf("failed mutation changed state: before=%#v after=%#v", before, after)
	}
}

func TestSQLiteWatchedGroupRepositoryFailsClosedOnMalformedStoredState(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, repo *SQLiteWatchedGroupRepository)
		want    error
	}{
		{
			name: "invalid group",
			prepare: func(t *testing.T, repo *SQLiteWatchedGroupRepository) {
				if _, err := repo.Add(watchedGroupFixture(t, "local-a", "facebook-a", "", 0, true)); err != nil {
					t.Fatalf("Add() error = %v", err)
				}
				execWatchedGroupSQL(t, repo, `UPDATE watched_groups SET name = '' WHERE position = 0`)
			},
			want: ErrInvalidStoredWatchedGroupState,
		},
		{
			name: "zero groups nonzero cursor",
			prepare: func(t *testing.T, repo *SQLiteWatchedGroupRepository) {
				execWatchedGroupSQL(t, repo, `UPDATE watched_group_selection_state SET cursor_position = 1 WHERE state_pk = 1`)
			},
			want: ErrInvalidStoredWatchedGroupState,
		},
		{
			name: "nonempty out of range cursor",
			prepare: func(t *testing.T, repo *SQLiteWatchedGroupRepository) {
				if _, err := repo.Add(watchedGroupFixture(t, "local-a", "facebook-a", "", 0, true)); err != nil {
					t.Fatalf("Add() error = %v", err)
				}
				execWatchedGroupSQL(t, repo, `UPDATE watched_group_selection_state SET cursor_position = 1 WHERE state_pk = 1`)
			},
			want: ErrInvalidStoredWatchedGroupState,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := openWatchedGroupTestRepository(t, watchedGroupTestPath(t))
			defer closeWatchedGroupTestRepository(t, repo)
			test.prepare(t, repo)
			if state, err := repo.Load(); !errors.Is(err, test.want) || len(state.Groups()) != 0 {
				t.Fatalf("Load() state=%#v error=%v, want %v", state, err, test.want)
			}
		})
	}
}

func TestOpenSQLiteWatchedGroupRepositoryRejectsSchemaCorruptionWithoutRecreate(t *testing.T) {
	tests := []struct {
		name   string
		mutate string
		want   error
	}{
		{name: "unsupported version", mutate: `UPDATE watched_group_schema_metadata SET current_schema_version = 2 WHERE metadata_pk = 1`, want: ErrSQLiteWatchedGroupSchemaVersion},
		{name: "missing metadata", mutate: `DELETE FROM watched_group_schema_metadata`, want: ErrInvalidSQLiteWatchedGroupSchema},
		{name: "extra table", mutate: `CREATE TABLE unexpected_state (id INTEGER PRIMARY KEY)`, want: ErrInvalidSQLiteWatchedGroupSchema},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := watchedGroupTestPath(t)
			repo := openWatchedGroupTestRepository(t, path)
			execWatchedGroupSQL(t, repo, test.mutate)
			closeWatchedGroupTestRepository(t, repo)
			reopened, err := OpenSQLiteWatchedGroupRepository(path)
			if !errors.Is(err, test.want) || reopened != nil {
				t.Fatalf("reopen = %#v, %v; want nil, %v", reopened, err, test.want)
			}
		})
	}
}

func TestSQLiteWatchedGroupRepositoryReopenIsDeterministic(t *testing.T) {
	path := watchedGroupTestPath(t)
	repo := openWatchedGroupTestRepository(t, path)
	for i := 0; i < 5; i++ {
		id := "local-" + string(rune('a'+i))
		if _, err := repo.Add(watchedGroupFixture(t, id, "facebook-"+id, "", i, i%2 == 0)); err != nil {
			t.Fatalf("Add(%s) error = %v", id, err)
		}
	}
	closeWatchedGroupTestRepository(t, repo)

	var first WatchedGroupState
	for i := 0; i < 3; i++ {
		reopened := openWatchedGroupTestRepository(t, path)
		state, err := reopened.Load()
		if err != nil {
			t.Fatalf("reopen %d Load() error = %v", i, err)
		}
		closeWatchedGroupTestRepository(t, reopened)
		if i == 0 {
			first = state
		} else if !watchedGroupsEqual(state.Groups(), first.Groups()) || state.Cursor().Position() != first.Cursor().Position() {
			t.Fatalf("reopen %d state differs", i)
		}
	}
}

func TestSQLiteWatchedGroupDeleteJournalSidecarsStayOwnerOnlyAndDoNotPersist(t *testing.T) {
	root := t.TempDir()
	repo, err := openSQLiteWatchedGroupRepositoryAtApplicationSupport(root)
	if err != nil {
		t.Fatalf("open production-shaped repository: %v", err)
	}
	path, _ := watchedGroupDatabasePath(root)

	tx, err := repo.db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if _, err := tx.Exec(`UPDATE watched_group_selection_state SET cursor_position = 0 WHERE state_pk = 1`); err != nil {
		t.Fatalf("write transaction error = %v", err)
	}
	journal := path + "-journal"
	if _, err := os.Stat(journal); err == nil {
		assertWatchedGroupFileMode(t, journal, 0o600)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat journal: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	closeWatchedGroupTestRepository(t, repo)

	for _, suffix := range []string{"-journal", "-wal", "-shm"} {
		if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
			t.Fatalf("sidecar %s persists after close: %v", suffix, err)
		}
	}
}

func watchedGroupFixture(t *testing.T, id string, facebookGroupID string, canonicalURL string, displayOrder int, active bool) domain.WatchedGroup {
	t.Helper()
	createdAt := time.Date(2026, time.August, 12, 9, 0, 0, displayOrder, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
	group, err := domain.NewWatchedGroup(id, facebookGroupID, canonicalURL, "Name "+id, createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup(%s) error = %v", id, err)
	}
	group, err = group.WithMetadata(domain.WatchedGroupMetadata{
		Name:                 "Name " + id,
		Notes:                "notes " + id,
		LastSuccessfulScanAt: createdAt.Add(time.Hour),
		LastError:            "last error " + id,
		DisplayOrder:         displayOrder,
	})
	if err != nil {
		t.Fatalf("WithMetadata(%s) error = %v", id, err)
	}
	return group.WithActive(active)
}

func watchedGroupTestPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "watched-groups.sqlite3")
}

func openWatchedGroupTestRepository(t *testing.T, path string) *SQLiteWatchedGroupRepository {
	t.Helper()
	repo, err := OpenSQLiteWatchedGroupRepository(path)
	if err != nil {
		t.Fatalf("OpenSQLiteWatchedGroupRepository() error = %v", err)
	}
	return repo
}

func closeWatchedGroupTestRepository(t *testing.T, repo *SQLiteWatchedGroupRepository) {
	t.Helper()
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func execWatchedGroupSQL(t *testing.T, repo *SQLiteWatchedGroupRepository, statement string) {
	t.Helper()
	if _, err := repo.db.Exec(statement); err != nil {
		t.Fatalf("Exec(%q) error = %v", statement, err)
	}
}

func assertWatchedGroupJournalMode(t *testing.T, repo *SQLiteWatchedGroupRepository, want string) {
	t.Helper()
	var got string
	if err := repo.db.QueryRow(`PRAGMA journal_mode`).Scan(&got); err != nil {
		t.Fatalf("query journal mode: %v", err)
	}
	if !strings.EqualFold(got, want) {
		t.Fatalf("journal mode = %q, want %q", got, want)
	}
}

func assertWatchedGroupFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode(%s) = %04o, want %04o", path, got, want)
	}
}

func assertNoCompletedBatchTables(t *testing.T, repo *SQLiteWatchedGroupRepository) {
	t.Helper()
	rows, err := repo.db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("query table names: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		if strings.HasPrefix(name, "scan_") || strings.HasPrefix(name, "batch_") || name == "leads" {
			t.Fatalf("watched-group database contains completed-batch table %q", name)
		}
	}
}

func watchedGroupsEqual(left []domain.WatchedGroup, right []domain.WatchedGroup) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		leftLast, leftHasLast := left[i].LastSuccessfulScanAt()
		rightLast, rightHasLast := right[i].LastSuccessfulScanAt()
		_, leftCreatedOffset := left[i].CreatedAt().Zone()
		_, rightCreatedOffset := right[i].CreatedAt().Zone()
		_, leftLastOffset := leftLast.Zone()
		_, rightLastOffset := rightLast.Zone()
		if left[i].ID() != right[i].ID() ||
			left[i].FacebookGroupID() != right[i].FacebookGroupID() ||
			left[i].CanonicalURL() != right[i].CanonicalURL() ||
			left[i].Name() != right[i].Name() ||
			!left[i].CreatedAt().Equal(right[i].CreatedAt()) ||
			leftCreatedOffset != rightCreatedOffset ||
			left[i].IsActive() != right[i].IsActive() ||
			left[i].Notes() != right[i].Notes() ||
			leftHasLast != rightHasLast ||
			(leftHasLast && (!leftLast.Equal(rightLast) || leftLastOffset != rightLastOffset)) ||
			left[i].LastError() != right[i].LastError() ||
			left[i].DisplayOrder() != right[i].DisplayOrder() {
			return false
		}
	}
	return true
}
