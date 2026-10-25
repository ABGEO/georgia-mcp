package crystal

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Crystal answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("crystal: unexpected status")
	// ErrNotSucceeded tells that an envelope of the answer reports a failure, or that the
	// answer holds no list.
	ErrNotSucceeded = errors.New("crystal: not succeeded")
)

// APIError is an answer of Crystal with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
