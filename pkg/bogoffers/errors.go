package bogoffers

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Bank of Georgia answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("bogoffers: unexpected status")

// APIError is an answer of Bank of Georgia with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
