package tas

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that TAS answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("tas: unexpected status")
	// ErrInvalidFilter tells that a filter holds a text that the body cannot carry.
	ErrInvalidFilter = errors.New("tas: invalid filter")
	// ErrInvalidReply tells that the reply is not a reply of DWR that the client reads.
	ErrInvalidReply = errors.New("tas: invalid reply")
	// ErrRemote tells that TAS answered with an exception or a failure.
	ErrRemote = errors.New("tas: remote exception")
)

// APIError is an answer of TAS with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
