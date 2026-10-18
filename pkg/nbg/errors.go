package nbg

import (
	"errors"
	"fmt"
)

const maxErrorBodyBytes = 1 << 10

// ErrUnexpectedStatus tells that the NBG answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("nbg: unexpected status")

// APIError is an answer of the NBG with a status that is not a success.
type APIError struct {
	StatusCode int
	// Body holds the first 1 KiB of the answer.
	Body string
}

func newAPIError(status int, body []byte) *APIError {
	if len(body) > maxErrorBodyBytes {
		body = body[:maxErrorBodyBytes]
	}

	return &APIError{StatusCode: status, Body: string(body)}
}

// Error returns the status and the body.
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %d: %s", ErrUnexpectedStatus, e.StatusCode, e.Body)
}

// Unwrap returns ErrUnexpectedStatus.
func (e *APIError) Unwrap() error {
	return ErrUnexpectedStatus
}
