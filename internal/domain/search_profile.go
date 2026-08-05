package domain

import "strings"

const macBookSearchProfileID = "macbook"

// SearchProfile describes one buyer-intent product search target.
type SearchProfile struct {
	id               string
	displayName      string
	productTerms     []string
	buyerIntentTerms []string
	noiseTerms       []string
	enabled          bool
}

// NewSearchProfile creates a SearchProfile while preserving value semantics for term slices.
func NewSearchProfile(id string, displayName string, productTerms []string, buyerIntentTerms []string, noiseTerms []string, enabled bool) (SearchProfile, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SearchProfile{}, ErrEmptySearchProfileID
	}

	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return SearchProfile{}, ErrEmptySearchProfileDisplayName
	}

	productTerms = copyStrings(productTerms)
	for i := range productTerms {
		productTerms[i] = strings.TrimSpace(productTerms[i])
		if productTerms[i] == "" {
			return SearchProfile{}, ErrEmptySearchProfileProductTerm
		}
	}
	if len(productTerms) == 0 {
		return SearchProfile{}, ErrNoSearchProfileProductTerms
	}

	return SearchProfile{
		id:               id,
		displayName:      displayName,
		productTerms:     productTerms,
		buyerIntentTerms: copyStrings(buyerIntentTerms),
		noiseTerms:       copyStrings(noiseTerms),
		enabled:          enabled,
	}, nil
}

// MacBookSearchProfile returns the built-in MVP buyer SearchProfile.
func MacBookSearchProfile() SearchProfile {
	return SearchProfile{
		id:               macBookSearchProfileID,
		displayName:      "MacBook",
		productTerms:     []string{"MacBook", "MacBook Pro"},
		buyerIntentTerms: []string{"can mua", "cần mua", "tim mua", "muon mua", "can tim", "đang tìm", "co ai ban", "có ai bán", "can may", "can MacBook gap"},
		noiseTerms:       []string{"Bán MacBook Pro", "có sẵn", "quảng cáo", "can tien nen ban", "shop can thu mua"},
		enabled:          true,
	}
}

func (p SearchProfile) ID() string {
	return p.id
}

func (p SearchProfile) DisplayName() string {
	return p.displayName
}

func (p SearchProfile) ProductTerms() []string {
	return copyStrings(p.productTerms)
}

func (p SearchProfile) BuyerIntentTerms() []string {
	return copyStrings(p.buyerIntentTerms)
}

func (p SearchProfile) NoiseTerms() []string {
	return copyStrings(p.noiseTerms)
}

func (p SearchProfile) IsEnabled() bool {
	return p.enabled
}

func (p SearchProfile) valid() bool {
	return p.id != "" && p.displayName != "" && len(p.productTerms) > 0
}

func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}
