package domain

// GeographicMode describes the user-selected geographic scope for one scan.
type GeographicMode string

const (
	GeographicModeHoChiMinhCity          GeographicMode = "hcm"
	GeographicModeOutsideHoChiMinhCityVN GeographicMode = "non_hcm_vietnam"
	GeographicModeAllVietnam             GeographicMode = "all_vietnam"
)

// NewGeographicMode validates a geographic mode value.
func NewGeographicMode(value string) (GeographicMode, error) {
	mode := GeographicMode(value)
	if !mode.Valid() {
		return "", ErrInvalidGeographicMode
	}
	return mode, nil
}

func (m GeographicMode) Valid() bool {
	switch m {
	case GeographicModeHoChiMinhCity, GeographicModeOutsideHoChiMinhCityVN, GeographicModeAllVietnam:
		return true
	default:
		return false
	}
}
