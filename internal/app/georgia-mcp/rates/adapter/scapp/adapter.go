// Package scapp reads the commercial rates from the Scapp client and converts its answer
// to the domain. It is the anti-corruption layer of the context for Scapp.
package scapp

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
	scappclient "github.com/abgeo/georgia-mcp/pkg/scapp"
)

// maxUnitExponent bounds a unit at 1 000 000.
const maxUnitExponent = 6

var errMalformed = errors.New("scapp: malformed rate table")

// Adapter implements app.CommercialRateSource with the Scapp client.
type Adapter struct {
	client *scappclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *scappclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Scapp, a microfinance organization.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "scapp", Name: "Scapp", Kind: domain.KindMicrofinance}
}

// CommercialRates reads the rate page with one call and gives the price of one unit of
// each code. A code with no row, or with a price that is empty or zero, is missing.
// Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	rows, err := a.client.Rates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode, err := indexRows(rows)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
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

// unitRow is a row with its unit split into the code and the exponent of the count.
type unitRow struct {
	exponent  int32
	buy, sell string
}

// indexRows splits the unit of each row, and fails on a unit that it cannot read or on
// two rows of one code.
func indexRows(rows []scappclient.RateRow) (map[domain.CurrencyCode]unitRow, error) {
	byCode := make(map[domain.CurrencyCode]unitRow, len(rows))

	for _, row := range rows {
		count, text, ok := strings.Cut(row.Unit, " ")
		if !ok {
			return nil, fmt.Errorf("%w: unit %q", errMalformed, row.Unit)
		}

		code, err := domain.ParseCurrencyCode(text)
		if err != nil {
			return nil, fmt.Errorf("%w: unit %q: %w", errMalformed, row.Unit, err)
		}

		exponent, err := unitExponent(count)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, code)
		}

		if _, seen := byCode[code]; seen {
			return nil, fmt.Errorf("%w: two rows of %s", errMalformed, code)
		}

		byCode[code] = unitRow{exponent: exponent, buy: row.Buy, sell: row.Sell}
	}

	return byCode, nil
}

// toRate converts a row to the price of one unit, and gives false when a price is
// empty or zero.
func toRate(code domain.CurrencyCode, row unitRow) (domain.CommercialRate, bool, error) {
	buy, err := toPrice(code, "buy", row.buy, row.exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", row.sell, row.exponent)
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
// trailing zeros. An empty price is zero.
func toPrice(code domain.CurrencyCode, side, text string, exponent int32) (decimal.Decimal, error) {
	if text == "" {
		return decimal.Zero, nil
	}

	price, err := decimal.NewFromString(text)
	if err != nil || price.IsNegative() {
		return decimal.Decimal{}, fmt.Errorf("%w: %s %s %q", errMalformed, code, side, text)
	}

	return decimal.RequireFromString(price.Shift(-exponent).String()), nil
}

func unavailable(err error) error {
	return fmt.Errorf("reading the Scapp rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
