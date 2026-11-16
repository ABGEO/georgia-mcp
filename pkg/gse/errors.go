package gse

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// ErrUnexpectedStatus tells that GSE answered with a status that is not a success.
var ErrUnexpectedStatus = errors.New("gse: unexpected status")

// APIError is an answer of GSE with a status that is not a success. It wraps
// ErrUnexpectedStatus. A file that GSE does not publish answers 404.
type APIError = vendorhttp.StatusError
