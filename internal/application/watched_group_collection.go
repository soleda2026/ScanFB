package application

import (
	"errors"
	"strings"

	"github.com/soleda2026/ScanFB/internal/domain"
)

var (
	ErrDuplicateWatchedGroupID       = errors.New("application: watched group id is duplicate")
	ErrDuplicateWatchedGroupIdentity = errors.New("application: watched group identity is duplicate")
	ErrWatchedGroupNotFound          = errors.New("application: watched group not found")
)

// WatchedGroupCollection manages watched groups in stable insertion order.
type WatchedGroupCollection struct {
	groups        []domain.WatchedGroup
	localIDIndex  map[string]int
	identityIndex map[domain.WatchedGroupIdentityKey]int
}

func NewWatchedGroupCollection() *WatchedGroupCollection {
	return &WatchedGroupCollection{}
}

// Add stores one valid group without selecting it for a scan batch.
func (c *WatchedGroupCollection) Add(group domain.WatchedGroup) error {
	if err := group.Validate(); err != nil {
		return err
	}
	c.ensureIndexes()
	if _, exists := c.localIDIndex[group.ID()]; exists {
		return ErrDuplicateWatchedGroupID
	}
	if _, exists := c.identityIndex[group.IdentityKey()]; exists {
		return ErrDuplicateWatchedGroupIdentity
	}
	if c.hasCrossKindCanonicalConflict(group) {
		return ErrDuplicateWatchedGroupIdentity
	}

	index := len(c.groups)
	c.groups = append(c.groups, group)
	c.localIDIndex[group.ID()] = index
	c.identityIndex[group.IdentityKey()] = index
	return nil
}

func (c *WatchedGroupCollection) hasCrossKindCanonicalConflict(incoming domain.WatchedGroup) bool {
	if incoming.CanonicalURL() == "" {
		return false
	}
	incomingIsURLOnly := incoming.FacebookGroupID() == ""
	for _, existing := range c.groups {
		if existing.CanonicalURL() == incoming.CanonicalURL() &&
			(existing.FacebookGroupID() == "") != incomingIsURLOnly {
			return true
		}
	}
	return false
}

func (c *WatchedGroupCollection) Count() int {
	return len(c.groups)
}

// Groups returns a defensive snapshot in stable insertion order.
func (c *WatchedGroupCollection) Groups() []domain.WatchedGroup {
	if len(c.groups) == 0 {
		return nil
	}
	groups := make([]domain.WatchedGroup, len(c.groups))
	copy(groups, c.groups)
	return groups
}

func (c *WatchedGroupCollection) GroupByID(id string) (domain.WatchedGroup, error) {
	index, ok := c.indexByLocalID(id)
	if !ok {
		return domain.WatchedGroup{}, ErrWatchedGroupNotFound
	}
	return c.groups[index], nil
}

func (c *WatchedGroupCollection) UpdateMetadata(id string, metadata domain.WatchedGroupMetadata) (domain.WatchedGroup, error) {
	index, ok := c.indexByLocalID(id)
	if !ok {
		return domain.WatchedGroup{}, ErrWatchedGroupNotFound
	}
	updated, err := c.groups[index].WithMetadata(metadata)
	if err != nil {
		return domain.WatchedGroup{}, err
	}
	c.groups[index] = updated
	return updated, nil
}

func (c *WatchedGroupCollection) Activate(id string) (domain.WatchedGroup, error) {
	return c.setActive(id, true)
}

func (c *WatchedGroupCollection) Deactivate(id string) (domain.WatchedGroup, error) {
	return c.setActive(id, false)
}

func (c *WatchedGroupCollection) setActive(id string, active bool) (domain.WatchedGroup, error) {
	index, ok := c.indexByLocalID(id)
	if !ok {
		return domain.WatchedGroup{}, ErrWatchedGroupNotFound
	}
	updated := c.groups[index].WithActive(active)
	c.groups[index] = updated
	return updated, nil
}

func (c *WatchedGroupCollection) indexByLocalID(id string) (int, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return -1, false
	}
	c.ensureIndexes()
	index, ok := c.localIDIndex[id]
	return index, ok
}

func (c *WatchedGroupCollection) ensureIndexes() {
	if c.localIDIndex == nil {
		c.localIDIndex = make(map[string]int, len(c.groups))
		for i, group := range c.groups {
			c.localIDIndex[group.ID()] = i
		}
	}
	if c.identityIndex == nil {
		c.identityIndex = make(map[domain.WatchedGroupIdentityKey]int, len(c.groups))
		for i, group := range c.groups {
			c.identityIndex[group.IdentityKey()] = i
		}
	}
}
