package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewWatchedGroupWithFacebookGroupID(t *testing.T) {
	createdAt := watchedGroupCreatedAt()
	group, err := NewWatchedGroup("group-local-1", "facebook-group-1", "", "MacBook Viet Nam", createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}

	if group.ID() != "group-local-1" || group.FacebookGroupID() != "facebook-group-1" {
		t.Fatalf("unexpected identity: id=%q facebookGroupID=%q", group.ID(), group.FacebookGroupID())
	}
	if group.CanonicalURL() != "" || group.Name() != "MacBook Viet Nam" {
		t.Fatalf("unexpected group fields: canonicalURL=%q name=%q", group.CanonicalURL(), group.Name())
	}
	if !group.CreatedAt().Equal(createdAt) || !group.IsActive() {
		t.Fatalf("new group must preserve createdAt and start active: createdAt=%v active=%v", group.CreatedAt(), group.IsActive())
	}
}

func TestNewWatchedGroupWithCanonicalURLOnly(t *testing.T) {
	group, err := NewWatchedGroup(
		"group-local-2",
		"",
		"https://www.facebook.com/groups/macbook-vietnam",
		"MacBook Vietnam",
		watchedGroupCreatedAt(),
	)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}

	key := group.IdentityKey()
	if key.Kind() != WatchedGroupIdentityCanonicalURL || key.Value() != group.CanonicalURL() {
		t.Fatalf("unexpected canonical identity key: kind=%q value=%q", key.Kind(), key.Value())
	}
}

func TestNewWatchedGroupAcceptsWellFormedAbsoluteHTTPSURLWithFragment(t *testing.T) {
	canonicalURL := "https://www.facebook.com/groups/macbook-vietnam#recent"
	group, err := NewWatchedGroup("group-local-2", "", canonicalURL, "MacBook Vietnam", watchedGroupCreatedAt())
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	if group.CanonicalURL() != canonicalURL {
		t.Fatalf("CanonicalURL() = %q, want %q", group.CanonicalURL(), canonicalURL)
	}
}

func TestNewWatchedGroupValidation(t *testing.T) {
	createdAt := watchedGroupCreatedAt()
	tests := []struct {
		name         string
		id           string
		facebookID   string
		canonicalURL string
		displayName  string
		createdAt    time.Time
		want         error
	}{
		{name: "empty local id", facebookID: "facebook-group-1", displayName: "Group", createdAt: createdAt, want: ErrInvalidWatchedGroupID},
		{name: "empty name", id: "group-local-1", facebookID: "facebook-group-1", createdAt: createdAt, want: ErrInvalidWatchedGroupName},
		{name: "zero createdAt", id: "group-local-1", facebookID: "facebook-group-1", displayName: "Group", want: ErrInvalidWatchedGroupCreatedAt},
		{name: "missing source identity", id: "group-local-1", displayName: "Group", createdAt: createdAt, want: ErrMissingWatchedGroupIdentity},
		{name: "malformed canonical url", id: "group-local-1", canonicalURL: "://bad", displayName: "Group", createdAt: createdAt, want: ErrInvalidWatchedGroupCanonicalURL},
		{name: "http canonical url", id: "group-local-1", canonicalURL: "http://www.facebook.com/groups/one", displayName: "Group", createdAt: createdAt, want: ErrInvalidWatchedGroupCanonicalURL},
		{name: "relative canonical url", id: "group-local-1", canonicalURL: "/groups/one", displayName: "Group", createdAt: createdAt, want: ErrInvalidWatchedGroupCanonicalURL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewWatchedGroup(test.id, test.facebookID, test.canonicalURL, test.displayName, test.createdAt)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewWatchedGroup() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestWatchedGroupFacebookIDIsAuthoritativeWhenBothIdentitiesExist(t *testing.T) {
	group, err := NewWatchedGroup(
		"group-local-1",
		"facebook-group-1",
		"https://www.facebook.com/groups/macbook-vietnam",
		"MacBook Vietnam",
		watchedGroupCreatedAt(),
	)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}

	if group.FacebookGroupID() == "" || group.CanonicalURL() == "" {
		t.Fatal("both supplied identities must be preserved")
	}
	key := group.IdentityKey()
	if key.Kind() != WatchedGroupIdentityFacebookGroupID || key.Value() != "facebook-group-1" {
		t.Fatalf("unexpected authoritative identity key: kind=%q value=%q", key.Kind(), key.Value())
	}
}

func TestWatchedGroupMetadataUpdatePreservesIdentityAndLifecycle(t *testing.T) {
	createdAt := watchedGroupCreatedAt()
	group, err := NewWatchedGroup("group-local-1", "facebook-group-1", "", "Old name", createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	lastSuccessfulScanAt := createdAt.Add(2 * time.Hour)

	updated, err := group.WithMetadata(WatchedGroupMetadata{
		Name:                 "New name",
		Notes:                "User note",
		LastSuccessfulScanAt: lastSuccessfulScanAt,
		LastError:            "Previous scan failed",
		DisplayOrder:         17,
	})
	if err != nil {
		t.Fatalf("WithMetadata() error = %v", err)
	}

	if updated.ID() != group.ID() || updated.IdentityKey() != group.IdentityKey() {
		t.Fatal("metadata update changed group identity")
	}
	if !updated.CreatedAt().Equal(createdAt) || !updated.IsActive() {
		t.Fatal("metadata update changed createdAt or active state")
	}
	if updated.Name() != "New name" || updated.Notes() != "User note" || updated.LastError() != "Previous scan failed" || updated.DisplayOrder() != 17 {
		t.Fatalf("metadata update not preserved: %#v", updated.Metadata())
	}
	gotLastScan, ok := updated.LastSuccessfulScanAt()
	if !ok || !gotLastScan.Equal(lastSuccessfulScanAt) {
		t.Fatalf("LastSuccessfulScanAt() = %v, %v", gotLastScan, ok)
	}
	if group.Name() != "Old name" || group.Notes() != "" {
		t.Fatal("WithMetadata() mutated the original value")
	}
}

func TestWatchedGroupMetadataValidationFailsWithoutMutation(t *testing.T) {
	createdAt := watchedGroupCreatedAt()
	group, err := NewWatchedGroup("group-local-1", "facebook-group-1", "", "Original", createdAt)
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}

	if _, err := group.WithMetadata(WatchedGroupMetadata{Name: "   "}); !errors.Is(err, ErrInvalidWatchedGroupName) {
		t.Fatalf("empty name error = %v", err)
	}
	if _, err := group.WithMetadata(WatchedGroupMetadata{Name: "Changed", LastSuccessfulScanAt: createdAt.Add(-time.Second)}); !errors.Is(err, ErrWatchedGroupScanBeforeCreated) {
		t.Fatalf("impossible scan chronology error = %v", err)
	}
	if group.Name() != "Original" {
		t.Fatal("invalid metadata update mutated the original value")
	}
}

func TestWatchedGroupOptionalLastSuccessfulScanCanBeUnset(t *testing.T) {
	group, err := NewWatchedGroup("group-local-1", "facebook-group-1", "", "Group", watchedGroupCreatedAt())
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}

	updated, err := group.WithMetadata(WatchedGroupMetadata{Name: "Group"})
	if err != nil {
		t.Fatalf("WithMetadata() error = %v", err)
	}
	if value, ok := updated.LastSuccessfulScanAt(); ok || !value.IsZero() {
		t.Fatalf("LastSuccessfulScanAt() = %v, %v; want unset", value, ok)
	}
}

func TestWatchedGroupActiveStateChangePreservesIdentityCreatedAtAndMetadata(t *testing.T) {
	group, err := NewWatchedGroup("group-local-1", "facebook-group-1", "", "Group", watchedGroupCreatedAt())
	if err != nil {
		t.Fatalf("NewWatchedGroup() error = %v", err)
	}
	group, err = group.WithMetadata(WatchedGroupMetadata{Name: "Group", Notes: "Keep me", DisplayOrder: 4})
	if err != nil {
		t.Fatalf("WithMetadata() error = %v", err)
	}

	inactive := group.WithActive(false)
	active := inactive.WithActive(true)
	if inactive.IsActive() || !active.IsActive() {
		t.Fatalf("unexpected active states: inactive=%v active=%v", inactive.IsActive(), active.IsActive())
	}
	if active.ID() != group.ID() || active.IdentityKey() != group.IdentityKey() || !active.CreatedAt().Equal(group.CreatedAt()) {
		t.Fatal("active state change modified identity or createdAt")
	}
	if active.Metadata() != group.Metadata() {
		t.Fatal("active state change modified metadata")
	}
}

func watchedGroupCreatedAt() time.Time {
	return time.Date(2026, time.August, 10, 9, 30, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
}
