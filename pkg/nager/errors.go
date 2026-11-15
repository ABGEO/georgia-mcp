package nager

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Nager.Date answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("nager: unexpected status")

// APIError is an answer of Nager.Date with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
