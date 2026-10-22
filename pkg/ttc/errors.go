package ttc

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that the TTC answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("ttc: unexpected status")

// APIError is an answer of the TTC with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
