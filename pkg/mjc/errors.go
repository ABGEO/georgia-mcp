package mjc

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that MJC answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("mjc: unexpected status")
	// ErrNoRateTable tells that the page holds no rate table, or a table with no row.
	ErrNoRateTable = errors.New("mjc: no rate table")
	// ErrMalformedTable tells that the table lacks the title of a column that the client
	// reads, or that a row lacks a cell.
	ErrMalformedTable = errors.New("mjc: malformed rate table")
)

// APIError is an answer of MJC with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
