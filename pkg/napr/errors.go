package napr

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that NAPR answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("napr: unexpected status")
	// ErrInvalidLabel tells that a label is not a label of the map.
	ErrInvalidLabel = errors.New("napr: invalid label")
	// ErrEmptyKeyword tells that a search has no keyword.
	ErrEmptyKeyword = errors.New("napr: empty keyword")
)

// APIError is an answer of NAPR with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
