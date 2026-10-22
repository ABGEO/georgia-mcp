package connect

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Connect answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("connect: unexpected status")

// APIError is an answer of Connect with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
