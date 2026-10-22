package socar

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that SOCAR answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("socar: unexpected status")

// APIError is an answer of SOCAR with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
