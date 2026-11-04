package domain

import "errors"

var (
	// ErrUnknownPollutant tells that a text is not a pollutant of the NEA.
	ErrUnknownPollutant = errors.New("pollutant: unknown")
	// ErrInvalidStationCode tells that a text is not a station code.
	ErrInvalidStationCode = errors.New("station code: invalid")
	// ErrInvalidBands tells that the limits of the levels do not rise above zero.
	ErrInvalidBands = errors.New("bands: invalid")
)
