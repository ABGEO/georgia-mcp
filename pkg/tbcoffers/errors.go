package tbcoffers

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that TBC Bank answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("tbcoffers: unexpected status")

// APIError is an answer of TBC Bank with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
