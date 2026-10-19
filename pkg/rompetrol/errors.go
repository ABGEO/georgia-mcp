package rompetrol

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Rompetrol answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("rompetrol: unexpected status")
	// ErrNoPriceTable tells that the page holds no anchor of the price table, or no
	// table after it.
	ErrNoPriceTable = errors.New("rompetrol: no price table")
	// ErrMalformedTable tells that a row of the price table does not hold exactly two cells.
	ErrMalformedTable = errors.New("rompetrol: malformed price table")
)

// APIError is an answer of Rompetrol with a status that is not a success.
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
