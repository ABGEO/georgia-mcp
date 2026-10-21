package libertyrates

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrUnexpectedStatus tells that Liberty answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("libertyrates: unexpected status")
	// ErrNoRateBlock tells that the page holds no rate block, or a block with no currency.
	ErrNoRateBlock = errors.New("libertyrates: no rate block")
	// ErrMalformedBlock tells that the block has no commercial column, a currency with
	// no code, or a column that does not hold 0 or 2 prices.
	ErrMalformedBlock = errors.New("libertyrates: malformed rate block")
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
