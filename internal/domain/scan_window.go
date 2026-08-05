package domain

import "time"

const RequiredTimezone = "Asia/Ho_Chi_Minh"

// ScanWindow represents the allowed post creation interval for one scan.
type ScanWindow struct {
	scanDate     time.Time
	startOfDay   time.Time
	scanStarted  time.Time
	timezoneName string
}

// NewScanWindow validates a same-day scan window in Asia/Ho_Chi_Minh.
func NewScanWindow(scanDate time.Time, startOfDay time.Time, scanStarted time.Time) (ScanWindow, error) {
	if !isRequiredTimezone(scanDate) || !isRequiredTimezone(startOfDay) || !isRequiredTimezone(scanStarted) {
		return ScanWindow{}, ErrInvalidTimezone
	}
	if startOfDay.After(scanStarted) {
		return ScanWindow{}, ErrStartOfDayAfterScanStart
	}
	if startOfDay.Hour() != 0 || startOfDay.Minute() != 0 || startOfDay.Second() != 0 || startOfDay.Nanosecond() != 0 {
		return ScanWindow{}, ErrStartOfDayNotMidnight
	}
	if !sameCalendarDay(scanDate, scanStarted) || !sameCalendarDay(startOfDay, scanStarted) {
		return ScanWindow{}, ErrScanWindowCrossesDay
	}

	year, month, day := scanStarted.Date()
	return ScanWindow{
		scanDate:     time.Date(year, month, day, 0, 0, 0, 0, scanStarted.Location()),
		startOfDay:   startOfDay,
		scanStarted:  scanStarted,
		timezoneName: RequiredTimezone,
	}, nil
}

func (w ScanWindow) ScanDate() time.Time {
	return w.scanDate
}

func (w ScanWindow) StartOfDay() time.Time {
	return w.startOfDay
}

func (w ScanWindow) ScanStarted() time.Time {
	return w.scanStarted
}

func (w ScanWindow) Timezone() string {
	return w.timezoneName
}

func (w ScanWindow) valid() bool {
	return w.timezoneName == RequiredTimezone && !w.scanDate.IsZero() && !w.startOfDay.IsZero() && !w.scanStarted.IsZero()
}

func isRequiredTimezone(value time.Time) bool {
	return value.Location() != nil && value.Location().String() == RequiredTimezone
}

func sameCalendarDay(left time.Time, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}
