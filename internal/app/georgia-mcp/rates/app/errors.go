package app

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
)

// MaxHistoryDays is the longest period of a history, both ends included.
const MaxHistoryDays = 31

var (
	// ErrNotPublished tells that a date comes after the newest rate set.
	ErrNotPublished = errors.New("rates: not published")
	// ErrPeriodTooLong tells that a period has more days than MaxHistoryDays.
	ErrPeriodTooLong = errors.New("rates: period too long")
	// ErrPeriodReversed tells that a period ends before it starts.
	ErrPeriodReversed = errors.New("rates: period ends before it starts")
)

// NotPublishedError names the requested date and the valid date of the newest set.
type NotPublishedError struct {
	Requested domain.Date
	Newest    domain.Date
}

// Error returns the two dates.
func (e *NotPublishedError) Error() string {
	return fmt.Sprintf("%s: %s, newest %s", ErrNotPublished, e.Requested, e.Newest)
}

// Unwrap returns ErrNotPublished.
func (e *NotPublishedError) Unwrap() error {
	return ErrNotPublished
}

// PeriodTooLongError names the count of days of a period and the limit.
type PeriodTooLongError struct {
	Days int
	Max  int
}

// Error returns the count and the limit.
func (e *PeriodTooLongError) Error() string {
	return fmt.Sprintf("%s: %d days, at most %d", ErrPeriodTooLong, e.Days, e.Max)
}

// Unwrap returns ErrPeriodTooLong.
func (e *PeriodTooLongError) Unwrap() error {
	return ErrPeriodTooLong
}
