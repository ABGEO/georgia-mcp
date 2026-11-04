package app

import (
	"errors"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
)

var (
	// ErrUnknownCity tells that a text names no city with a station.
	ErrUnknownCity = errors.New("city: unknown")
	// ErrUnknownStation tells that a code names no station of the NEA.
	ErrUnknownStation = errors.New("station: unknown")
	// ErrInvalidPeriod tells that a period breaks a rule of the history.
	ErrInvalidPeriod = errors.New("period: invalid")
	// ErrBothFilters tells that a query names a city and a station.
	ErrBothFilters = errors.New("filter: a city and a station")
)

// CityError tells that a text names no city with a station. Accepted holds each city
// with a station, by English name.
type CityError struct {
	Accepted []domain.City
}

// Error returns the text of ErrUnknownCity.
func (e *CityError) Error() string {
	return ErrUnknownCity.Error()
}

// Unwrap returns ErrUnknownCity.
func (e *CityError) Unwrap() error {
	return ErrUnknownCity
}

// PeriodError tells which rule of the history a period breaks.
type PeriodError struct {
	Message string
}

// Error returns the message.
func (e *PeriodError) Error() string {
	return e.Message
}

// Unwrap returns ErrInvalidPeriod.
func (e *PeriodError) Unwrap() error {
	return ErrInvalidPeriod
}
