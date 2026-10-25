package basisbank

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Basisbank answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("basisbank: unexpected status")
	// ErrMalformedRates tells that the text of the rates of a date is not a JSON object.
	ErrMalformedRates = errors.New("basisbank: malformed rates")
)

// APIError is an answer of Basisbank with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
