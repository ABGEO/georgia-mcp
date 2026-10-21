package portal

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrUnexpectedStatus tells that Portal answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("portal: unexpected status")

// APIError is an answer of Portal with a status that is not a success.
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

// ErrNoPriceList tells that the data of the page holds no price list.
var ErrNoPriceList = errors.New("portal: no price list")

// ErrMalformedPriceList tells that the price list of the page does not decode.
var ErrMalformedPriceList = errors.New("portal: malformed price list")
