package app

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
)

var (
	// ErrUnknownBank tells that a bank code names no bank of the server.
	ErrUnknownBank = errors.New("offers: unknown bank")
	// ErrOfferNotFound tells that a bank lists no current offer with an id.
	ErrOfferNotFound = errors.New("offers: offer not found")
	// ErrInvalidRange tells that the end of a range of dates is before its start, or
	// more than 366 days after it.
	ErrInvalidRange = errors.New("offers: invalid range of dates")
)

// UnknownBankError names the unknown code and the codes of the server, sorted.
type UnknownBankError struct {
	Code  domain.BankCode
	Known []domain.BankCode
}

// Error returns the code and the known codes.
func (e *UnknownBankError) Error() string {
	return fmt.Sprintf("%s: %q, known %v", ErrUnknownBank, e.Code, e.Known)
}

// Unwrap returns ErrUnknownBank.
func (e *UnknownBankError) Unwrap() error {
	return ErrUnknownBank
}
