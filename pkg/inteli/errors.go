package inteli

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Inteli Express answered with a status that is not a
	// success.
	ErrUnexpectedStatus = errors.New("inteli: unexpected status")
	// ErrNoRateTable tells that the page holds no table with the title of the rates, or a
	// table with no row.
	ErrNoRateTable = errors.New("inteli: no rate table")
	// ErrMalformedTable tells that a row of the table does not have four cells.
	ErrMalformedTable = errors.New("inteli: malformed rate table")
)

// APIError is an answer of Inteli Express with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
