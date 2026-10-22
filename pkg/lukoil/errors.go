package lukoil

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Lukoil answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("lukoil: unexpected status")
	// ErrNoPriceTable tells that the page holds no table, or a table with no head.
	ErrNoPriceTable = errors.New("lukoil: no price table")
	// ErrMalformedTable tells that a row of the price table does not hold one cell for
	// each cell of the head.
	ErrMalformedTable = errors.New("lukoil: malformed price table")
)

// APIError is an answer of Lukoil with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
