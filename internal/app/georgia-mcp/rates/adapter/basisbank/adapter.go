// Package basisbank reads the commercial rates from the Basisbank client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Basisbank.
package basisbank

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
	basisclient "github.com/abgeo/georgia-mcp/pkg/basisbank"
)

const dateLayout = "2006-01-02"

var errMalformed = errors.New("basisbank: malformed answer")

// Adapter implements app.CommercialRateSource with the Basisbank client.
type Adapter struct {
	client *basisclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *basisclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Basisbank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "basisbank", Name: "Basisbank", Kind: domain.KindBank}
}

// CommercialRates reads the rates with one call, takes the newest day, and gives the
// price of each code. A code with no price in one of the two sets, or with a price of
// zero, is missing. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	days, err := a.client.Rates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	newest, err := newestDay(days)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		rate, quoted, err := toRate(code, newest.XRates)
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

// newestDay returns the day with the greatest date. No day, a date that it cannot read,
// or one date twice is an error.
func newestDay(days []basisclient.Day) (basisclient.Day, error) {
	if len(days) == 0 {
		return basisclient.Day{}, fmt.Errorf("%w: no day", errMalformed)
	}

	seen := make(map[time.Time]bool, len(days))

	var (
		newest     basisclient.Day
		newestDate time.Time
	)

	for _, day := range days {
		date, err := time.Parse(dateLayout, day.Date)
		if err != nil {
			return basisclient.Day{}, fmt.Errorf("%w: date %q", errMalformed, day.Date)
		}

		if seen[date] {
			return basisclient.Day{}, fmt.Errorf("%w: date %s twice", errMalformed, day.Date)
		}

		seen[date] = true

		if date.After(newestDate) {
			newest, newestDate = day, date
		}
	}

	return newest, nil
}

// toRate converts the prices of a code, and gives false when a price is absent or zero.
func toRate(
	code domain.CurrencyCode,
	rates basisclient.XRates,
) (domain.CommercialRate, bool, error) {
	buy, err := toPrice(code, "buy", rates.KursBuy[string(code)])
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", rates.KursSell[string(code)])
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
	return fmt.Errorf("reading the Basisbank rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
