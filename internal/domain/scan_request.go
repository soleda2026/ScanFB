package domain

import "strings"

const MaxScanRequestGroups = 5

// ScanRequest represents the domain configuration for exactly one scan batch.
type ScanRequest struct {
	searchProfile  SearchProfile
	geographicMode GeographicMode
	scanWindow     ScanWindow
	groupIDs       []string
	dryRun         bool
}

// NewScanRequest validates the configuration for one user-triggered scan batch.
func NewScanRequest(profile SearchProfile, mode GeographicMode, window ScanWindow, groupIDs []string, dryRun bool) (ScanRequest, error) {
	if !profile.valid() {
		return ScanRequest{}, ErrInvalidSearchProfile
	}
	if !mode.Valid() {
		return ScanRequest{}, ErrInvalidGeographicMode
	}
	if !window.valid() {
		return ScanRequest{}, ErrInvalidScanWindow
	}

	normalizedGroupIDs, err := normalizeGroupIDs(groupIDs)
	if err != nil {
		return ScanRequest{}, err
	}

	return ScanRequest{
		searchProfile:  profile,
		geographicMode: mode,
		scanWindow:     window,
		groupIDs:       normalizedGroupIDs,
		dryRun:         dryRun,
	}, nil
}

func (r ScanRequest) SearchProfile() SearchProfile {
	return r.searchProfile
}

func (r ScanRequest) GeographicMode() GeographicMode {
	return r.geographicMode
}

func (r ScanRequest) ScanWindow() ScanWindow {
	return r.scanWindow
}

func (r ScanRequest) GroupIDs() []string {
	return copyStrings(r.groupIDs)
}

func (r ScanRequest) DryRun() bool {
	return r.dryRun
}

func normalizeGroupIDs(groupIDs []string) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, ErrNoScanGroups
	}
	if len(groupIDs) > MaxScanRequestGroups {
		return nil, ErrTooManyScanGroups
	}

	normalized := make([]string, len(groupIDs))
	seen := make(map[string]struct{}, len(groupIDs))
	for i, groupID := range groupIDs {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			return nil, ErrEmptyScanGroupID
		}
		if _, exists := seen[groupID]; exists {
			return nil, ErrDuplicateScanGroupID
		}
		seen[groupID] = struct{}{}
		normalized[i] = groupID
	}

	return normalized, nil
}
