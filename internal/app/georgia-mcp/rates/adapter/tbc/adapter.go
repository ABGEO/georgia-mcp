// Package tbc reads the commercial rates from the TBC client and converts its answer
// to the domain. It is the anti-corruption layer of the context for TBC Bank.
package tbc

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
	tbcclient "github.com/abgeo/georgia-mcp/pkg/tbc"
)

const baseCurrency = "GEL"

var errMalformed = errors.New("tbc: malformed answer")

// Adapter implements app.CommercialRateSource with the TBC client.
type Adapter struct {
	client *tbcclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *tbcclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns TBC Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "tbc", Name: "TBC Bank"}
}

// CommercialRates reads the codes in one call. A requested code that the answer lacks
// is missing. Every failure of TBC wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	requested := make([]string, 0, len(codes))
	for _, code := range codes {
		requested = append(requested, string(code))
	}

	answer, err := a.client.CommercialRates(ctx, requested)
	if err != nil {
		return app.CommercialRatesResult{}, fmt.Errorf(
			"reading the TBC rates: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	result, err := toResult(answer, codes)
	if err != nil {
		return app.CommercialRatesResult{}, fmt.Errorf(
			"reading the TBC rates: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	result.ReadAt = a.now()

	return result, nil
}

// toResult maps the rates of the requested codes, and ignores any other code.
func toResult(
	answer *tbcclient.CommercialRates,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	if answer.Base != baseCurrency {
		return app.CommercialRatesResult{}, fmt.Errorf("%w: base %q", errMalformed, answer.Base)
	}

	quoted := make(map[domain.CurrencyCode]domain.CommercialRate, len(codes))

	for _, quote := range answer.CommercialRatesList {
		code := domain.CurrencyCode(quote.Currency)
		if !slices.Contains(codes, code) {
			continue
		}

		rate, err := toCommercialRate(code, quote)
		if err != nil {
			return app.CommercialRatesResult{}, err
		}

		quoted[code] = rate
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		if rate, ok := quoted[code]; ok {
			result.Rates = append(result.Rates, rate)
		} else {
			result.Missing = append(result.Missing, code)
		}
	}

	return result, nil
}

func toCommercialRate(
	code domain.CurrencyCode,
	quote tbcclient.CommercialRate,
) (domain.CommercialRate, error) {
	buy, err := decimal.NewFromString(quote.Buy.String())
	if err != nil {
		return domain.CommercialRate{}, fmt.Errorf("%w: %s buy %q", errMalformed, code, quote.Buy)
	}

	sell, err := decimal.NewFromString(quote.Sell.String())
	if err != nil {
		return domain.CommercialRate{}, fmt.Errorf("%w: %s sell %q", errMalformed, code, quote.Sell)
	}

	rate, err := domain.NewCommercialRate(code, buy, sell)
	if err != nil {
		return domain.CommercialRate{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return rate, nil
}
