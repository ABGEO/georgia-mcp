package rico

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Rico Credit answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("rico: unexpected status")
	// ErrNoRateTable tells that the page holds no rate table, or a table with no row.
	ErrNoRateTable = errors.New("rico: no rate table")
	// ErrMalformedTable tells that a row lacks the code, the unit, the buy price, or the
	// sell price.
	ErrMalformedTable = errors.New("rico: malformed rate table")
)

// APIError is an answer of Rico Credit with a status that is not a success.
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
