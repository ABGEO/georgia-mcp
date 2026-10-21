// Package rico reads the commercial rates from the Rico Credit client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Rico Credit.
package rico

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	ricoclient "github.com/abgeo/georgia-mcp/pkg/rico"
)

// maxUnitExponent bounds a unit at 1 000 000.
const maxUnitExponent = 6

var errMalformed = errors.New("rico: malformed rate table")

// Adapter implements app.CommercialRateSource with the Rico Credit client.
type Adapter struct {
	client *ricoclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *ricoclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Rico Credit, a microfinance organization.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "rico", Name: "Rico Credit", Kind: domain.KindMicrofinance}
}

// CommercialRates reads the home page with one call and gives the price of one unit of
// each code. A row that is not a currency code, such as the cross rate USD-EUR, is
// skipped. A code with no row, or with a price of zero, is missing. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	rows, err := a.client.Rates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode := make(map[domain.CurrencyCode]ricoclient.RateRow, len(rows))

	for _, row := range rows {
		code, err := domain.ParseCurrencyCode(row.Currency)
		if err != nil {
			continue
		}

		byCode[code] = row
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		row, ok := byCode[code]
		if !ok {
			result.Missing = append(result.Missing, code)

			continue
		}

		rate, quoted, err := toRate(code, row)
		if err != nil {
			return app.CommercialRatesResult{}, unavailable(err)
		}

		if quoted {
			result.Rates = append(result.Rates, rate)
		} else {
			result.Missing = append(result.Missing, code)
		}
	}

	result.ReadAt = a.now()

	return result, nil
}

// toRate converts a row to the price of one unit, and gives false when a price is zero.
func toRate(code domain.CurrencyCode, row ricoclient.RateRow) (domain.CommercialRate, bool, error) {
	exponent, err := unitExponent(row.Unit)
	if err != nil {
		return domain.CommercialRate{}, false, fmt.Errorf("%w: %s", err, code)
	}

	buy, err := toPrice(code, "buy", row.Buy, exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", row.Sell, exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	if buy.IsZero() || sell.IsZero() {
		return domain.CommercialRate{}, false, nil
	}

	rate, err := domain.NewCommercialRate(code, buy, sell)
	if err != nil {
		return domain.CommercialRate{}, false, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return rate, true, nil
}

// unitExponent reads "1000" as 3. The unit must be a power of 10, so a division by it
// is a shift of the exponent and rounds no digit.
func unitExponent(unit string) (int32, error) {
	exponent := len(unit) - 1

	if strings.TrimRight(unit, "0") != "1" || exponent > maxUnitExponent {
		return 0, fmt.Errorf("%w: unit %q", errMalformed, unit)
	}

	return int32(exponent), nil
}

// toPrice parses a price for the unit, divides it by 10^exponent, and drops the
// trailing zeros.
func toPrice(code domain.CurrencyCode, side, text string, exponent int32) (decimal.Decimal, error) {
	price, err := decimal.NewFromString(text)
	if err != nil || price.IsNegative() {
		return decimal.Decimal{}, fmt.Errorf("%w: %s %s %q", errMalformed, code, side, text)
	}

	return decimal.RequireFromString(price.Shift(-exponent).String()), nil
}

func unavailable(err error) error {
	return fmt.Errorf("reading the Rico Credit rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
