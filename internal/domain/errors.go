package domain

import "errors"

var (
	ErrEmptySearchProfileID            = errors.New("domain: empty search profile id")
	ErrEmptySearchProfileDisplayName   = errors.New("domain: empty search profile display name")
	ErrEmptySearchProfileProductTerm   = errors.New("domain: empty search profile product term")
	ErrNoSearchProfileProductTerms     = errors.New("domain: no search profile product terms")
	ErrInvalidGeographicMode           = errors.New("domain: invalid geographic mode")
	ErrInvalidTimezone                 = errors.New("domain: invalid timezone")
	ErrStartOfDayAfterScanStart        = errors.New("domain: start of day after scan start")
	ErrStartOfDayNotMidnight           = errors.New("domain: start of day is not midnight")
	ErrScanWindowCrossesDay            = errors.New("domain: scan window crosses calendar day")
	ErrInvalidSearchProfile            = errors.New("domain: invalid search profile")
	ErrInvalidScanWindow               = errors.New("domain: invalid scan window")
	ErrNoScanGroups                    = errors.New("domain: no scan groups")
	ErrTooManyScanGroups               = errors.New("domain: too many scan groups")
	ErrEmptyScanGroupID                = errors.New("domain: empty scan group id")
	ErrDuplicateScanGroupID            = errors.New("domain: duplicate scan group id")
	ErrInvalidWatchedGroupID           = errors.New("domain: watched group id is invalid")
	ErrInvalidWatchedGroupName         = errors.New("domain: watched group name is invalid")
	ErrInvalidWatchedGroupCreatedAt    = errors.New("domain: watched group created at is invalid")
	ErrMissingWatchedGroupIdentity     = errors.New("domain: watched group identity is missing")
	ErrInvalidWatchedGroupCanonicalURL = errors.New("domain: watched group canonical url is invalid")
	ErrWatchedGroupScanBeforeCreated   = errors.New("domain: watched group successful scan is before creation")
)
