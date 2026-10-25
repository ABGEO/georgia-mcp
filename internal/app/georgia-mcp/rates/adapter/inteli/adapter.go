// Package inteli reads the commercial rates from the Inteli Express client and converts
// its answer to the domain. It is the anti-corruption layer of the context for Inteli
// Express.
package inteli

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
	inteliclient "github.com/abgeo/georgia-mcp/pkg/inteli"
)

const lari = "GEL"

var errMalformed = errors.New("inteli: malformed rate table")

// Adapter implements app.CommercialRateSource with the Inteli Express client.
type Adapter struct {
	client *inteliclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *inteliclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Inteli Express, a microfinance organization.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "inteli", Name: "Inteli Express", Kind: domain.KindMicrofinance}
}

// CommercialRates reads the home page with one call and gives the price of one unit of
// each code. A code with no row against GEL, or with a price that is empty or zero, is
// missing. Every failure wraps kernel.ErrSourceUnavailable.
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
		rate, ok := byCode[code]
		if !ok {
			result.Missing = append(result.Missing, code)

			continue
		}

		result.Rates = append(result.Rates, rate)
	}

	result.ReadAt = a.now()

	return result, nil
}

// indexRows converts each row against GEL, and fails on a row that it cannot read, on a
// buy price above its sell price, or on two rows of one code. A row with a price that is
// empty or zero gives no rate.
func indexRows(rows []inteliclient.RateRow) (map[domain.CurrencyCode]domain.CommercialRate, error) {
	byCode := make(map[domain.CurrencyCode]domain.CommercialRate, len(rows))
	seen := make(map[domain.CurrencyCode]bool, len(rows))

	for _, row := range rows {
		if !strings.EqualFold(row.Base, lari) {
			continue
		}

		code, err := domain.ParseCurrencyCode(row.Currency)
		if err != nil {
			return nil, fmt.Errorf("%w: currency %q: %w", errMalformed, row.Currency, err)
		}

		if seen[code] {
			return nil, fmt.Errorf("%w: two rows of %s", errMalformed, code)
		}

		seen[code] = true

		rate, quoted, err := toRate(code, row)
		if err != nil {
			return nil, err
		}

		if quoted {
			byCode[code] = rate
		}
	}

	return byCode, nil
}

// toRate converts a row to a rate, and gives false when a price is empty or zero. The
// table names no column, so a buy price above the sell price is a sign of a swap.
func toRate(
	code domain.CurrencyCode,
	row inteliclient.RateRow,
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

	if buy.GreaterThan(sell) {
		return domain.CommercialRate{}, false, fmt.Errorf(
			"%w: %s buys at %s above its sell price %s", errMalformed, code, buy, sell,
		)
	}

	rate, err := domain.NewCommercialRate(code, buy, sell)
	if err != nil {
		return domain.CommercialRate{}, false, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return rate, true, nil
}

// toPrice parses a price of one unit and drops the trailing zeros. An empty price is zero.
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
	return fmt.Errorf("reading the Inteli Express rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
