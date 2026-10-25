package mbc

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that MBC answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("mbc: unexpected status")
	// ErrNoRates tells that the answer holds no list of rates, or a null one.
	ErrNoRates = errors.New("mbc: no rates")
)

// APIError is an answer of MBC with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
