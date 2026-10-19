package app

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
)

// ErrUnknownCompany tells that a company code names no fuel company of the server.
var ErrUnknownCompany = errors.New("fuel: unknown company")

// UnknownCompanyError names the unknown code and the codes of the server, sorted.
type UnknownCompanyError struct {
	Code  domain.CompanyCode
	Known []domain.CompanyCode
}

// Error returns the code and the known codes.
func (e *UnknownCompanyError) Error() string {
	return fmt.Sprintf("%s: %q, known %v", ErrUnknownCompany, e.Code, e.Known)
}

// Unwrap returns ErrUnknownCompany.
func (e *UnknownCompanyError) Unwrap() error {
	return ErrUnknownCompany
}
