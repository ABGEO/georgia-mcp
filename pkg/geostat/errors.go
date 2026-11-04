package geostat

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Geostat answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("geostat: unexpected status")

// APIError is an answer of Geostat with a status that is not a success. It wraps
// ErrUnexpectedStatus. Geostat answers 404 to an unknown path and to a selection with an
// unknown code.
type APIError = vendorhttp.StatusError
