package application

import (
	"context"
	"time"

	"github.com/soleda2026/ScanFB/internal/blocklist"
	"github.com/soleda2026/ScanFB/internal/domain"
)

// GroupCollectionRequest is the complete input for one injected collection call.
type GroupCollectionRequest struct {
	WatchedGroup domain.WatchedGroup
	ScanWindow   domain.ScanWindow
}

// GroupCollectionResult binds ordered normalized posts to one enrolled group.
type GroupCollectionResult struct {
	WatchedGroupID string
	Posts          []domain.RawPost
}

// OrderedPosts returns a defensive copy in collector order.
func (r GroupCollectionResult) OrderedPosts() []domain.RawPost {
	if len(r.Posts) == 0 {
		return nil
	}
	posts := make([]domain.RawPost, len(r.Posts))
	copy(posts, r.Posts)
	return posts
}

// GroupPostCollector collects normalized posts for exactly one requested group.
type GroupPostCollector interface {
	CollectGroupPosts(context.Context, GroupCollectionRequest) (GroupCollectionResult, error)
}

// OneGroupScanRequest contains caller-owned identity, configuration, and transition times.
type OneGroupScanRequest struct {
	ScanID         string
	AttemptID      string
	WatchedGroup   domain.WatchedGroup
	ScanWindow     domain.ScanWindow
	SearchProfile  domain.SearchProfile
	GeographicMode domain.GeographicMode
	Blocklist      blocklist.List
	StartedAt      time.Time
	CompletedAt    time.Time
}
