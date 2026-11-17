package app

import (
	"errors"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
)

var (
	// ErrNoGrid tells that GSE published no grid state in the window of the search.
	ErrNoGrid = errors.New("grid: no data")
	// ErrMomentOutOfRange tells that a moment is in the future or older than GSE keeps.
	ErrMomentOutOfRange = errors.New("moment: out of range")
)

// MomentError tells that a moment is out of range. Earliest is the request instant minus
// domain.Retention.
type MomentError struct {
	Moment, Earliest time.Time
	Future           bool
}

// Error returns the text of ErrMomentOutOfRange.
func (e *MomentError) Error() string {
	return ErrMomentOutOfRange.Error()
}

// Unwrap returns ErrMomentOutOfRange.
func (e *MomentError) Unwrap() error {
	return ErrMomentOutOfRange
}

// NoGridError tells that GSE published no grid state in the window that starts at Slot
// and runs back domain.SearchWindow.
type NoGridError struct {
	Slot time.Time
}

// Error returns the text of ErrNoGrid.
func (e *NoGridError) Error() string {
	return ErrNoGrid.Error()
}

// Unwrap returns ErrNoGrid.
func (e *NoGridError) Unwrap() error {
	return ErrNoGrid
}

var (
	// ErrMonthAndYear tells that a request for the balance names a month and a year.
	ErrMonthAndYear = errors.New("balance: month and year")
	// ErrPeriodOutOfRange tells that a month or a year is outside the data of GNERC.
	ErrPeriodOutOfRange = errors.New("period: out of range")
	// ErrReversedPeriod tells that a period ends before it starts.
	ErrReversedPeriod = errors.New("period: reversed")
	// ErrPeriodTooLong tells that a period holds more than domain.MaxPriceMonths.
	ErrPeriodTooLong = errors.New("period: too long")
	// errNoData tells that a book holds no published month.
	errNoData = errors.New("book: no published month")
)

// PeriodError tells that a month or a year is outside the data, from First to Newest.
type PeriodError struct {
	// Field is "month", "year", "from", or "to".
	Field string
	// Value is the month or the year of the request: "2021-12", "2021".
	Value         string
	First, Newest domain.Month
}

// Error returns the text of ErrPeriodOutOfRange.
func (e *PeriodError) Error() string {
	return ErrPeriodOutOfRange.Error()
}

// Unwrap returns ErrPeriodOutOfRange.
func (e *PeriodError) Unwrap() error {
	return ErrPeriodOutOfRange
}

// SpanError tells that a period holds more than domain.MaxPriceMonths.
type SpanError struct {
	Months int
}

// Error returns the text of ErrPeriodTooLong.
func (e *SpanError) Error() string {
	return ErrPeriodTooLong.Error()
}

// Unwrap returns ErrPeriodTooLong.
func (e *SpanError) Unwrap() error {
	return ErrPeriodTooLong
}
