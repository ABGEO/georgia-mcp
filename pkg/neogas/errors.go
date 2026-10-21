package neogas

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrUnexpectedStatus tells that Neogas answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("neogas: unexpected status")

// APIError is an answer of Neogas with a status that is not a success.
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
