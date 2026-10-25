package credo

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Credo Bank answered with a status that is not a
	// success. A wrong key or no key gives 401.
	ErrUnexpectedStatus = errors.New("credo: unexpected status")
	// ErrNotSucceeded tells that the answer holds a false success.
	ErrNotSucceeded = errors.New("credo: not succeeded")
)

// APIError is an answer of Credo Bank with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
