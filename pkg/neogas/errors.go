package neogas

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Neogas answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("neogas: unexpected status")

// APIError is an answer of Neogas with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
