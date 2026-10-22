package liberty

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Liberty answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("liberty: unexpected status")
	// ErrNoSuccess tells that an answer of the API reports no success.
	ErrNoSuccess = errors.New("liberty: the answer reports no success")
	// ErrNoTerms tells that the page of an offer has no block of terms.
	ErrNoTerms = errors.New("liberty: the page has no block of terms")
)

// APIError is an answer of Liberty with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
