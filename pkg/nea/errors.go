package nea

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that the NEA answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("nea: unexpected status")

// APIError is an answer of the NEA with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
