// Package schooltime defines the application's authoritative school calendar timezone.
package schooltime

import (
	_ "time/tzdata"
	"time"
)

const TimeZone = "Asia/Kolkata"

var location = func() *time.Location {
	loc, err := time.LoadLocation(TimeZone)
	if err != nil {
		// India has no daylight-saving transitions; this keeps date rules
		// deterministic even when the host has no timezone database installed.
		return time.FixedZone(TimeZone, 5*60*60+30*60)
	}
	return loc
}()

// DateAt returns the calendar date at the supplied instant in the school timezone.
func DateAt(instant time.Time) string {
	return instant.In(location).Format("2006-01-02")
}

// Today returns midnight at the start of the current school-local calendar day.
func Today() time.Time {
	now := time.Now().In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
}

// TodayDate returns today's school-local calendar date in YYYY-MM-DD form.
func TodayDate() string {
	return Today().Format("2006-01-02")
}
