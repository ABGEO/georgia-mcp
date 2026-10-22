package gulf

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Gulf answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("gulf: unexpected status")
	// ErrNoPriceTable tells that the page holds no price table, or a table with no head.
	ErrNoPriceTable = errors.New("gulf: no price table")
	// ErrMalformedTable tells that a row of the price table does not hold one cell for
	// each cell of the head.
	ErrMalformedTable = errors.New("gulf: malformed price table")
)

// APIError is an answer of Gulf with a status that is not a success.
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
