// Package liberty reads the commercial rates from the Liberty Bank client and converts
// its answer to the domain. It is the anti-corruption layer of the context for Liberty.
package liberty

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
	libertyclient "github.com/abgeo/georgia-mcp/pkg/libertyrates"
)

// maxUnitExponent bounds a unit at 1 000 000.
const maxUnitExponent = 6

var errMalformed = errors.New("liberty: malformed rate block")

// Adapter implements app.CommercialRateSource with the Liberty Bank client.
type Adapter struct {
	client *libertyclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *libertyclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Liberty Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "liberty", Name: "Liberty Bank", Kind: domain.KindBank}
}

// CommercialRates reads the home page with one call and gives the commercial price of
// one unit of each code. A code with no row, or with no price, is missing. Every
// failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	rows, err := a.client.Rates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode := make(map[domain.CurrencyCode]libertyclient.RateRow, len(rows))

	for _, row := range rows {
		code, err := domain.ParseCurrencyCode(row.Currency)
		if err != nil {
			return app.CommercialRatesResult{}, unavailable(fmt.Errorf("%w: %w", errMalformed, err))
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

// toRate converts the commercial column of a row to the price of one unit, and gives
// false when a price is absent or zero.
func toRate(
	code domain.CurrencyCode,
	row libertyclient.RateRow,
) (domain.CommercialRate, bool, error) {
	exponent, err := unitExponent(row)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	buy, err := toPrice(code, "buy", row.Commercial.Buy, exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", row.Commercial.Sell, exponent)
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

// unitExponent reads "1000 AMD" as 3. The count must be a power of 10, so a division
// by it is a shift of the exponent and rounds no digit.
func unitExponent(row libertyclient.RateRow) (int32, error) {
	fields := strings.Fields(row.Unit)
	if len(fields) != 2 || fields[1] != row.Currency {
		return 0, fmt.Errorf("%w: %s unit %q", errMalformed, row.Currency, row.Unit)
	}

	count := fields[0]
	exponent := len(count) - 1

	if strings.TrimRight(count, "0") != "1" || exponent > maxUnitExponent {
		return 0, fmt.Errorf("%w: %s unit %q", errMalformed, row.Currency, row.Unit)
	}

	return int32(exponent), nil
}

// toPrice parses a price for the unit, divides it by 10^exponent, and drops the
// trailing zeros. An absent price is zero.
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
	return fmt.Errorf("reading the Liberty Bank rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
