package app

import (
	"errors"
	"fmt"
)

var (
	// ErrUnsupportedYear tells that a year or a date is outside the supported years.
	ErrUnsupportedYear = errors.New("year: unsupported")
	// ErrBothFilters tells that a query names a year and a from date.
	ErrBothFilters = errors.New("filter: a year and a from date")
)

// YearError tells that a year is outside the supported years, from First to Last.
type YearError struct {
	Year        int
	First, Last int
}

// Error returns the text of ErrUnsupportedYear with the year.
func (e *YearError) Error() string {
	return fmt.Sprintf("%s: %d", ErrUnsupportedYear, e.Year)
}

// Unwrap returns ErrUnsupportedYear.
func (e *YearError) Unwrap() error {
	return ErrUnsupportedYear
}
