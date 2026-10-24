package gpost

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that Georgian Post answered with a status that is not a
	// success.
	ErrUnexpectedStatus = errors.New("gpost: unexpected status")
	// ErrNoToken tells that the page holds no form token, or that its answer sets no
	// cookie token.
	ErrNoToken = errors.New("gpost: no anti-forgery token")
	// ErrMalformedResult tells that the search answer holds a row that the client cannot
	// read, or neither a row nor the note of an empty search.
	ErrMalformedResult = errors.New("gpost: malformed search result")
)

// APIError is an answer of Georgian Post with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
