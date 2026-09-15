// Package clock centralises time zone handling: business days are Asia/Shanghai.
package clock

import "time"

var Shanghai = mustLoad("Asia/Shanghai")

func mustLoad(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return l
}

func Now() time.Time { return time.Now().UTC() }

// DateString returns YYYY-MM-DD of t in Asia/Shanghai.
func DateString(t time.Time) string { return t.In(Shanghai).Format("2006-01-02") }

// Today returns today's Shanghai date as a UTC midnight time (for DATE columns).
func Today() time.Time { return DateOnly(Now()) }

// DateOnly converts t to the Shanghai calendar date stored as UTC midnight.
func DateOnly(t time.Time) time.Time {
	s := t.In(Shanghai)
	return time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, time.UTC)
}

// SameDay reports whether the DATE column value equals today's Shanghai date.
func SameDay(dateCol time.Time, now time.Time) bool {
	return dateCol.UTC().Format("2006-01-02") == DateString(now)
}
