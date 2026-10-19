package domain

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const decimalBase = 10

// OfficialRate is the price in lari of Quantity units of a currency, set by the NBG.
type OfficialRate struct {
	Currency CurrencyCode
	Name     string
	Quantity int64
	Rate     decimal.Decimal
	Change   decimal.Decimal
}

// NewOfficialRate returns an official rate. The quantity must be a power of ten.
func NewOfficialRate(
	code CurrencyCode,
	name string,
	quantity int64,
	rate, change decimal.Decimal,
) (OfficialRate, error) {
	if _, ok := powerOfTen(quantity); !ok {
		return OfficialRate{}, fmt.Errorf("%w: %d", ErrInvalidQuantity, quantity)
	}

	return OfficialRate{
		Currency: code,
		Name:     name,
		Quantity: quantity,
		Rate:     rate,
		Change:   change,
	}, nil
}

// RatePerUnit returns the price in lari of one unit. A shift by the power of ten of
// the quantity is exact.
func (r OfficialRate) RatePerUnit() decimal.Decimal {
	exponent, _ := powerOfTen(r.Quantity)

	return r.Rate.Shift(-exponent)
}

func powerOfTen(quantity int64) (int32, bool) {
	if quantity < 1 {
		return 0, false
	}

	var exponent int32

	for quantity%decimalBase == 0 {
		quantity /= decimalBase
		exponent++
	}

	return exponent, quantity == 1
}

// RateSet is every official rate that the NBG publishes for one valid date.
type RateSet struct {
	ValidFrom   kernel.Date
	PublishedAt time.Time
	Rates       []OfficialRate
}

// NewRateSet returns a rate set with its rates sorted by currency.
func NewRateSet(validFrom kernel.Date, publishedAt time.Time, rates []OfficialRate) RateSet {
	sorted := slices.Clone(rates)
	slices.SortFunc(
		sorted,
		func(a, b OfficialRate) int { return cmp.Compare(a.Currency, b.Currency) },
	)

	return RateSet{ValidFrom: validFrom, PublishedAt: publishedAt, Rates: sorted}
}

// Empty reports whether the set holds no rate.
func (s RateSet) Empty() bool {
	return len(s.Rates) == 0
}

// Find returns the rate of a currency.
func (s RateSet) Find(code CurrencyCode) (OfficialRate, bool) {
	i, found := slices.BinarySearchFunc(s.Rates, code, func(r OfficialRate, c CurrencyCode) int {
		return cmp.Compare(r.Currency, c)
	})
	if !found {
		return OfficialRate{}, false
	}

	return s.Rates[i], true
}
