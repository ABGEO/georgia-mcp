package app

import (
	"errors"
	"time"
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
