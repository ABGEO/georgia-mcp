package app

import (
	"errors"
	"fmt"
)

var (
	// ErrApplicationNotFound tells that the source has no application with the number.
	ErrApplicationNotFound = errors.New("application: not found")
	// ErrNoFilter tells that a search has no code and no period.
	ErrNoFilter = errors.New("search: no code and no period")
	// ErrTooManyMatches tells that a search matches more applications than the server sorts.
	ErrTooManyMatches = errors.New("search: too many matches")
)

// TooManyMatchesError gives the count of the matches. It wraps ErrTooManyMatches.
type TooManyMatchesError struct {
	Count int
}

func (e *TooManyMatchesError) Error() string {
	return fmt.Sprintf("%v: %d", ErrTooManyMatches, e.Count)
}

func (e *TooManyMatchesError) Unwrap() error {
	return ErrTooManyMatches
}
