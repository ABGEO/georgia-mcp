package wissol

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that Wissol answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("wissol: unexpected status")

// APIError is an answer of Wissol with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
