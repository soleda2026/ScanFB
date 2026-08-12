package persistence

import (
	"github.com/soleda2026/ScanFB/internal/application"
	"github.com/soleda2026/ScanFB/internal/domain"
)

// WatchedGroupState is one authoritative ordered group snapshot and cursor.
type WatchedGroupState struct {
	groups []domain.WatchedGroup
	cursor application.WatchedGroupSelectionCursor
}

func newWatchedGroupState(groups []domain.WatchedGroup, cursor application.WatchedGroupSelectionCursor) WatchedGroupState {
	copied := make([]domain.WatchedGroup, len(groups))
	copy(copied, groups)
	return WatchedGroupState{groups: copied, cursor: cursor}
}

func (s WatchedGroupState) Groups() []domain.WatchedGroup {
	if len(s.groups) == 0 {
		return nil
	}
	groups := make([]domain.WatchedGroup, len(s.groups))
	copy(groups, s.groups)
	return groups
}

func (s WatchedGroupState) Cursor() application.WatchedGroupSelectionCursor {
	return s.cursor
}
