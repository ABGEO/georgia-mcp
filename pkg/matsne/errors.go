package matsne

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that the Legislative Herald answered with a status that is
	// not a success.
	ErrUnexpectedStatus = errors.New("matsne: unexpected status")
	// ErrBlocked tells that the answer is the page of the bot filter. The site gives the
	// same page for an act or a version that does not exist.
	ErrBlocked = errors.New("matsne: the bot filter refused the request")
	// ErrMalformedPage tells that a page lacks a part that the client reads.
	ErrMalformedPage = errors.New("matsne: the page has an unknown shape")
)

// APIError is an answer of the Legislative Herald with a status that is not a success. It
// wraps ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
