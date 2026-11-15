package parliament

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that the Parliament answered with a status that is not a
// success.
var ErrUnexpectedStatus = errors.New("parliament: unexpected status")

// APIError is an answer of the Parliament with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
