package liberty

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Liberty answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("liberty: unexpected status")
	// ErrNoSuccess tells that an answer of the API reports no success.
	ErrNoSuccess = errors.New("liberty: the answer reports no success")
	// ErrNoTerms tells that the page of an offer has no block of terms.
	ErrNoTerms = errors.New("liberty: the page has no block of terms")
)

// APIError is an answer of Liberty with a status that is not a success.
type APIError struct {
	StatusCode int
	// Body is the first 1 KiB of the answer.
	Body string
}

// Error returns the status.
func (e *APIError) Error() string {
	return fmt.Sprintf(
		"%s: %d: %s",
		ErrUnexpectedStatus,
		e.StatusCode,
		http.StatusText(e.StatusCode),
	)
}

// Unwrap returns ErrUnexpectedStatus.
func (e *APIError) Unwrap() error {
	return ErrUnexpectedStatus
}
