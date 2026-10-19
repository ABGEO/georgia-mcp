package domain

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// ProductName is the name of a product in Georgian and in English.
type ProductName struct {
	Georgian string
	English  string
}

// In returns the name in a language, or the name in the other language when the
// product lacks one.
func (n ProductName) In(language kernel.Language) string {
	first, second := n.English, n.Georgian
	if language == kernel.LanguageGeorgian {
		first, second = second, first
	}

	if first != "" {
		return first
	}

	return second
}

// ProductPrice is the price in lari of one unit of a product of a fuel company.
// Standard is the price at a station with an attendant. SelfService is the price at
// a self-service station, valid only when the company publishes one.
type ProductPrice struct {
	Code          string
	Name          ProductName
	Grade         Grade
	Standard      decimal.Decimal
	SelfService   decimal.NullDecimal
	EffectiveDate kernel.Date
}

// NewProductPrice returns the price of a product. It needs a code, a name, a known
// grade, a date, a standard price more than zero, and a self-service price more than
// zero when it is valid.
func NewProductPrice(
	code string,
	name ProductName,
	grade Grade,
	standard decimal.Decimal,
	selfService decimal.NullDecimal,
	effective kernel.Date,
) (ProductPrice, error) {
	if _, err := ParseGrade(string(grade)); err != nil {
		return ProductPrice{}, fmt.Errorf("%w: %q: %w", ErrInvalidProductPrice, code, err)
	}

	switch {
	case code == "":
		return ProductPrice{}, fmt.Errorf("%w: no code", ErrInvalidProductPrice)
	case name.Georgian == "" && name.English == "":
		return ProductPrice{}, fmt.Errorf("%w: %q has no name", ErrInvalidProductPrice, code)
	case effective.IsZero():
		return ProductPrice{}, fmt.Errorf("%w: %q has no date", ErrInvalidProductPrice, code)
	case !standard.IsPositive():
		return ProductPrice{}, fmt.Errorf(
			"%w: %q standard %s", ErrInvalidProductPrice, code, standard,
		)
	case selfService.Valid && !selfService.Decimal.IsPositive():
		return ProductPrice{}, fmt.Errorf(
			"%w: %q self-service %s", ErrInvalidProductPrice, code, selfService.Decimal,
		)
	}

	return ProductPrice{
		Code:          code,
		Name:          name,
		Grade:         grade,
		Standard:      standard,
		SelfService:   selfService,
		EffectiveDate: effective,
	}, nil
}
