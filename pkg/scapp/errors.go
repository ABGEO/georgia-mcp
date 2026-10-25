package scapp

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Scapp answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("scapp: unexpected status")
	// ErrNoRateTable tells that the page holds no rate table, or a table with no row.
	ErrNoRateTable = errors.New("scapp: no rate table")
	// ErrMalformedTable tells that the table lacks the title of a price, or that a row
	// lacks a cell.
	ErrMalformedTable = errors.New("scapp: malformed rate table")
)

// APIError is an answer of Scapp with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
