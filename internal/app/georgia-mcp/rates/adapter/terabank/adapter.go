// Package terabank reads the commercial rates from the Terabank client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Terabank.
package terabank

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
	terabankclient "github.com/abgeo/georgia-mcp/pkg/terabank"
)

var errMalformed = errors.New("terabank: malformed answer")

// Adapter implements app.CommercialRateSource with the Terabank client.
type Adapter struct {
	client *terabankclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *terabankclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Terabank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "terabank", Name: "Terabank", Kind: domain.KindBank}
}

// CommercialRates reads every currency with one call and gives the price of each code.
// A code with no item, or with no price, is missing. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	items, err := a.client.CrossRates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode := make(map[domain.CurrencyCode]terabankclient.CrossRate, len(items))
	for _, item := range items {
		byCode[domain.CurrencyCode(strings.ToUpper(item.ISO))] = item
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		item, ok := byCode[code]
		if !ok {
			result.Missing = append(result.Missing, code)

			continue
		}

		rate, quoted, err := toRate(code, item)
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

// toRate converts an item, and gives false when a price is absent or zero.
func toRate(
	code domain.CurrencyCode,
	item terabankclient.CrossRate,
) (domain.CommercialRate, bool, error) {
	buy, err := toPrice(code, "buy", item.Buy.String())
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", item.Sell.String())
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
// absent price is zero.
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
	return fmt.Errorf("reading the Terabank rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
