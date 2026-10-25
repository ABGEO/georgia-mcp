package halyk

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Halyk Bank answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("halyk: unexpected status")
	// ErrNoRateTable tells that the page holds no table of the current rates, or a table
	// with no row.
	ErrNoRateTable = errors.New("halyk: no rate table")
	// ErrMalformedTable tells that the table lacks the title of a column, or that a row
	// lacks a cell.
	ErrMalformedTable = errors.New("halyk: malformed rate table")
)

// APIError is an answer of Halyk Bank with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
