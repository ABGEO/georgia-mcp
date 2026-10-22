package rompetrol

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Rompetrol answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("rompetrol: unexpected status")
	// ErrNoPriceTable tells that the page holds no anchor of the price table, or no
	// table after it.
	ErrNoPriceTable = errors.New("rompetrol: no price table")
	// ErrMalformedTable tells that a row of the price table does not hold exactly two cells.
	ErrMalformedTable = errors.New("rompetrol: malformed price table")
)

// APIError is an answer of Rompetrol with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
