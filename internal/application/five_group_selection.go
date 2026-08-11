package application

import (
	"errors"

	"github.com/soleda2026/ScanFB/internal/domain"
)

const fiveGroupSelectionSize = 5

var (
	ErrInvalidWatchedGroupSelectionCursor   = errors.New("application: watched group selection cursor is invalid")
	ErrEmptyWatchedGroupSelectionCollection = errors.New("application: watched group selection collection is empty")
	ErrInsufficientActiveWatchedGroups      = errors.New("application: insufficient active watched groups")
	ErrDuplicateWatchedGroupSelectionGroup  = errors.New("application: watched group selection contains a duplicate group")
)

// WatchedGroupSelectionCursor is a caller-managed collection position.
type WatchedGroupSelectionCursor struct {
	position int
}

// FiveGroupSelection is one exact-five active-group selection and its continuation cursor.
type FiveGroupSelection struct {
	groups     []domain.WatchedGroup
	nextCursor WatchedGroupSelectionCursor
}

// InitialWatchedGroupSelectionCursor starts traversal at collection position zero.
func InitialWatchedGroupSelectionCursor() WatchedGroupSelectionCursor {
	return WatchedGroupSelectionCursor{position: 0}
}

// NewWatchedGroupSelectionCursor creates a non-negative caller-managed cursor.
func NewWatchedGroupSelectionCursor(position int) (WatchedGroupSelectionCursor, error) {
	if position < 0 {
		return WatchedGroupSelectionCursor{}, ErrInvalidWatchedGroupSelectionCursor
	}
	return WatchedGroupSelectionCursor{position: position}, nil
}

func (c WatchedGroupSelectionCursor) Position() int {
	return c.position
}

// SelectNextFiveActiveGroups traverses one circular collection cycle from cursor.
func SelectNextFiveActiveGroups(groups []domain.WatchedGroup, cursor WatchedGroupSelectionCursor) (FiveGroupSelection, error) {
	if len(groups) == 0 {
		return FiveGroupSelection{}, ErrEmptyWatchedGroupSelectionCollection
	}
	if cursor.position < 0 || cursor.position >= len(groups) {
		return FiveGroupSelection{}, ErrInvalidWatchedGroupSelectionCursor
	}
	if err := validateWatchedGroupSelectionSnapshot(groups); err != nil {
		return FiveGroupSelection{}, err
	}

	selected := make([]domain.WatchedGroup, 0, fiveGroupSelectionSize)
	for inspected := 0; inspected < len(groups); inspected++ {
		index := (cursor.position + inspected) % len(groups)
		group := groups[index]
		if !group.IsActive() {
			continue
		}

		selected = append(selected, group)
		if len(selected) == fiveGroupSelectionSize {
			return FiveGroupSelection{
				groups:     copySelectedWatchedGroups(selected),
				nextCursor: WatchedGroupSelectionCursor{position: (index + 1) % len(groups)},
			}, nil
		}
	}

	return FiveGroupSelection{}, ErrInsufficientActiveWatchedGroups
}

func validateWatchedGroupSelectionSnapshot(groups []domain.WatchedGroup) error {
	localIDs := make(map[string]struct{}, len(groups))
	identities := make(map[domain.WatchedGroupIdentityKey]struct{}, len(groups))
	urlOnlyCanonicalURLs := make(map[string]struct{}, len(groups))
	facebookBackedCanonicalURLs := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if err := group.Validate(); err != nil {
			return err
		}
		if _, exists := localIDs[group.ID()]; exists {
			return ErrDuplicateWatchedGroupSelectionGroup
		}
		if _, exists := identities[group.IdentityKey()]; exists {
			return ErrDuplicateWatchedGroupSelectionGroup
		}
		if canonicalURL := group.CanonicalURL(); canonicalURL != "" {
			if group.FacebookGroupID() == "" {
				if _, exists := facebookBackedCanonicalURLs[canonicalURL]; exists {
					return ErrDuplicateWatchedGroupSelectionGroup
				}
				urlOnlyCanonicalURLs[canonicalURL] = struct{}{}
			} else {
				if _, exists := urlOnlyCanonicalURLs[canonicalURL]; exists {
					return ErrDuplicateWatchedGroupSelectionGroup
				}
				facebookBackedCanonicalURLs[canonicalURL] = struct{}{}
			}
		}
		localIDs[group.ID()] = struct{}{}
		identities[group.IdentityKey()] = struct{}{}
	}
	return nil
}

// Groups returns a defensive copy in selected traversal order.
func (s FiveGroupSelection) Groups() []domain.WatchedGroup {
	return copySelectedWatchedGroups(s.groups)
}

func (s FiveGroupSelection) NextCursor() WatchedGroupSelectionCursor {
	return s.nextCursor
}

func copySelectedWatchedGroups(groups []domain.WatchedGroup) []domain.WatchedGroup {
	if len(groups) == 0 {
		return nil
	}
	copied := make([]domain.WatchedGroup, len(groups))
	copy(copied, groups)
	return copied
}
