package tbc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ErrUnexpectedStatus tells that TBC answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("tbc: unexpected status")

// APIError is an answer of TBC with a status that is not a success. TBC describes
// the failure in the problem format of RFC 9457. The fields are empty when the body
// is not in that format.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	SystemCode string `json:"systemCode"`
	TraceID    string `json:"traceId"`
}

func newAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{}
	if err := json.Unmarshal(body, apiErr); err != nil {
		apiErr = &APIError{}
	}

	apiErr.StatusCode = status

	return apiErr
}

// Error returns the status, the title, and the system code.
func (e *APIError) Error() string {
	title := e.Title
	if title == "" {
		title = http.StatusText(e.StatusCode)
	}

	if e.SystemCode == "" {
		return fmt.Sprintf("%s: %d: %s", ErrUnexpectedStatus, e.StatusCode, title)
	}

	return fmt.Sprintf("%s: %d: %s (%s)", ErrUnexpectedStatus, e.StatusCode, title, e.SystemCode)
}

// Unwrap returns ErrUnexpectedStatus.
func (e *APIError) Unwrap() error {
	return ErrUnexpectedStatus
}
