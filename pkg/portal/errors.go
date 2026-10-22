package portal

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Portal answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("portal: unexpected status")

// APIError is an answer of Portal with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError

// ErrNoPriceList tells that the data of the page holds no price list.
var ErrNoPriceList = errors.New("portal: no price list")

// ErrMalformedPriceList tells that the price list of the page does not decode.
var ErrMalformedPriceList = errors.New("portal: malformed price list")
