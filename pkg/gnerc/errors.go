package gnerc

import (
	"errors"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var (
	// ErrUnexpectedStatus tells that GNERC answered with a status that is not a success.
	ErrUnexpectedStatus = errors.New("gnerc: unexpected status")
	// ErrNotWorkbook tells that a body is not a workbook that the client can read.
	ErrNotWorkbook = errors.New("gnerc: not a workbook")
)

// APIError is an answer of GNERC with a status that is not a success. It wraps
// ErrUnexpectedStatus.
type APIError = vendorhttp.StatusError
