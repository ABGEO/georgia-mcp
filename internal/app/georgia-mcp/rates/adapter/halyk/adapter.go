// Package halyk reads the commercial rates from the Halyk Bank client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Halyk Bank.
package halyk

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	halykclient "github.com/abgeo/georgia-mcp/pkg/halyk"
)

var errMalformed = errors.New("halyk: malformed rate table")

// Adapter implements app.CommercialRateSource with the Halyk Bank client.
type Adapter struct {
	client *halykclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *halykclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Halyk Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "halyk", Name: "Halyk Bank", Kind: domain.KindBank}
}

// CommercialRates reads the rate page with one call and gives the price of each code. A
// code with no row, or with a price that is empty or zero, is missing. Every failure
// wraps kernel.ErrSourceUnavailable.
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

// indexRows reads the code of each row, and fails on a code that it cannot read or on
// two rows of one code.
func indexRows(rows []halykclient.RateRow) (map[domain.CurrencyCode]halykclient.RateRow, error) {
	byCode := make(map[domain.CurrencyCode]halykclient.RateRow, len(rows))

	for _, row := range rows {
		code, err := domain.ParseCurrencyCode(row.Currency)
		if err != nil {
			return nil, fmt.Errorf("%w: currency %q: %w", errMalformed, row.Currency, err)
		}

		if _, seen := byCode[code]; seen {
			return nil, fmt.Errorf("%w: two rows of %s", errMalformed, code)
		}

		byCode[code] = row
	}

	return byCode, nil
}

// toRate converts a row, and gives false when a price is empty or zero.
func toRate(
	code domain.CurrencyCode,
	row halykclient.RateRow,
) (domain.CommercialRate, bool, error) {
	buy, err := toPrice(code, "buy", row.Buy)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", row.Sell)
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

// toPrice parses a price and drops the zeros that pad it to four decimal places. An
// empty price is zero.
func toPrice(code domain.CurrencyCode, side, text string) (decimal.Decimal, error) {
	if text == "" {
		return decimal.Zero, nil
	}

	price, err := decimal.NewFromString(text)
	if err != nil || price.IsNegative() {
		return decimal.Decimal{}, fmt.Errorf("%w: %s %s %q", errMalformed, code, side, text)
	}

	return decimal.RequireFromString(price.String()), nil
}

func unavailable(err error) error {
	return fmt.Errorf("reading the Halyk Bank rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
