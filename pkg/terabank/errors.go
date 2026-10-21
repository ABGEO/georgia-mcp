package terabank

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Terabank answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("terabank: unexpected status")
	// ErrNotSucceeded tells that Terabank answered 200 with succeeded false.
	ErrNotSucceeded = errors.New("terabank: not succeeded")
)

// APIError is an answer of Terabank with a status that is not a success.
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
