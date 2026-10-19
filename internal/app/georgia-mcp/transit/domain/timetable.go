package domain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	minutesInHour = 60
	// maxServiceHour bounds the hour of a time. A service date ends after midnight, so
	// the TTC writes 00:05 of the next date as 24:05.
	maxServiceHour = 47
	minuteDigits   = 2
)

// ClockTime is a time of a service date, in minutes after the midnight that starts it.
type ClockTime struct {
	minutes int
}

// ParseClockTime returns the time of a text in H:MM or HH:MM, with an hour of 0 to 47.
func ParseClockTime(text string) (ClockTime, error) {
	hourText, minuteText, found := strings.Cut(text, ":")
	if !found || len(hourText) < 1 || len(hourText) > 2 || len(minuteText) != minuteDigits ||
		!digits(hourText) || !digits(minuteText) {
		return ClockTime{}, fmt.Errorf("%w: %q", ErrInvalidClockTime, text)
	}

	hour, _ := strconv.Atoi(hourText)
	minute, _ := strconv.Atoi(minuteText)

	if hour > maxServiceHour || minute >= minutesInHour {
		return ClockTime{}, fmt.Errorf("%w: %q", ErrInvalidClockTime, text)
	}

	return ClockTime{minutes: hour*minutesInHour + minute}, nil
}

// String returns the time in HH:MM. An hour of 24 or more is after midnight.
func (c ClockTime) String() string {
	return fmt.Sprintf("%02d:%02d", c.minutes/minutesInHour, c.minutes%minutesInHour)
}

// Minutes returns the minutes after the midnight that starts the service date.
func (c ClockTime) Minutes() int {
	return c.minutes
}

// StopTimes is the times of one stop of a direction.
type StopTimes struct {
	Stop StopID
	// Position is 1 for the first stop of the direction.
	Position int
	Times    []ClockTime
}

// TimetableDay is the times of each stop of a direction on one date.
type TimetableDay struct {
	Date kernel.Date
	// Stops is in the order of Position.
	Stops []StopTimes
}

// Timetable is the times of one direction of a route, one day for each date.
type Timetable struct {
	// Days is sorted by date.
	Days []TimetableDay
}

// Span returns the first and the last date of the timetable. Both are zero when the
// timetable has no day.
func (t Timetable) Span() (kernel.Date, kernel.Date) {
	if len(t.Days) == 0 {
		return kernel.Date{}, kernel.Date{}
	}

	return t.Days[0].Date, t.Days[len(t.Days)-1].Date
}

// Times returns the times of a stop on a date, from its first position on the
// direction. The second result is false when the timetable has no such date or no such
// stop.
func (t Timetable) Times(date kernel.Date, stop StopID) ([]ClockTime, bool) {
	for _, day := range t.Days {
		if day.Date != date {
			continue
		}

		for _, times := range day.Stops {
			if times.Stop == stop {
				return times.Times, true
			}
		}

		return nil, false
	}

	return nil, false
}

// HasDate reports whether the timetable has a day for the date.
func (t Timetable) HasDate(date kernel.Date) bool {
	for _, day := range t.Days {
		if day.Date == date {
			return true
		}
	}

	return false
}

func digits(text string) bool {
	for _, char := range text {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
