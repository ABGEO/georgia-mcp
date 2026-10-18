package bog

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Bank of Georgia answered with a status that is not
	// a success.
	ErrUnexpectedStatus = errors.New("bog: unexpected status")

	// ErrNoRate tells that Bank of Georgia answered 204: it has no rate of the
	// currency. The bank answers so for GEL and for an unknown code.
	ErrNoRate = errors.New("bog: no rate")
)

// APIError is an answer of Bank of Georgia with a status that is not a success.
type APIError struct {
	StatusCode int
	// RetryAfter is the header Retry-After as the bank writes it, in seconds. The
	// bank sends it with 429.
	RetryAfter string
	// Body is the first 1 KiB of the answer.
	Body string
}

// Error returns the status, and the value of Retry-After when the bank sent it.
func (e *APIError) Error() string {
	text := fmt.Sprintf(
		"%s: %d: %s",
		ErrUnexpectedStatus,
		e.StatusCode,
		http.StatusText(e.StatusCode),
	)
	if e.RetryAfter != "" {
		text += " (retry after " + e.RetryAfter + ")"
	}

	return text
}

// Unwrap returns ErrUnexpectedStatus.
func (e *APIError) Unwrap() error {
	return ErrUnexpectedStatus
}
