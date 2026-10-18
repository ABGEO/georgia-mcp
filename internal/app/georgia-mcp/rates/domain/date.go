package domain

import (
	"fmt"
	"time"
)

const (
	tbilisiName   = "Asia/Tbilisi"
	tbilisiOffset = 4 * 60 * 60
	hoursInDay    = 24
)

// TbilisiZone returns the zone of Tbilisi. Georgia keeps UTC+4 with no daylight
// saving time since 2005, and the container image holds no zone database.
func TbilisiZone() *time.Location {
	return time.FixedZone(tbilisiName, tbilisiOffset)
}

// Date is a civil date in Tbilisi. The zero value is not a valid date.
type Date struct {
	year  int
	month time.Month
	day   int
}

// ParseDate returns the date of a text in YYYY-MM-DD.
func ParseDate(text string) (Date, error) {
	parsed, err := time.Parse(time.DateOnly, text)
	if err != nil || parsed.Format(time.DateOnly) != text {
		return Date{}, fmt.Errorf("%w: %q", ErrInvalidDate, text)
	}

	return dateOf(parsed), nil
}

// Today returns the Tbilisi date of an instant.
func Today(now time.Time) Date {
	return dateOf(now.In(TbilisiZone()))
}

// DaysBetween returns the count of days from one date to another.
func DaysBetween(from, to Date) int {
	return int(to.midnight().Sub(from.midnight()).Hours() / hoursInDay)
}

func dateOf(t time.Time) Date {
	return Date{year: t.Year(), month: t.Month(), day: t.Day()}
}

// IsZero reports whether the date is the zero value.
func (d Date) IsZero() bool {
	return d == Date{}
}

// AddDays returns the date n days later. A negative n gives an earlier date.
func (d Date) AddDays(n int) Date {
	return dateOf(d.midnight().AddDate(0, 0, n))
}

// Before reports whether the date comes before another.
func (d Date) Before(other Date) bool {
	return d.midnight().Before(other.midnight())
}

// After reports whether the date comes after another.
func (d Date) After(other Date) bool {
	return d.midnight().After(other.midnight())
}

// Time returns midnight of the date in UTC, for a client that takes a time.Time.
func (d Date) Time() time.Time {
	return d.midnight()
}

// String returns the date in YYYY-MM-DD.
func (d Date) String() string {
	return d.midnight().Format(time.DateOnly)
}

// midnight is in UTC, so the arithmetic counts whole days of 24 hours.
func (d Date) midnight() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}
