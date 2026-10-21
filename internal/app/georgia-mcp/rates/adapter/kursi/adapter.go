// Package kursi reads the commercial rates from the Kursi.ge client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Kursi.ge.
package kursi

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
	kursiclient "github.com/abgeo/georgia-mcp/pkg/kursi"
)

const lari = "GEL"

var errMalformed = errors.New("kursi: malformed answer")

// Adapter implements app.CommercialRateSource with the Kursi.ge client.
type Adapter struct {
	client *kursiclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *kursiclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Kursi.ge, an online exchange bureau.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "kursi", Name: "Kursi.ge", Kind: domain.KindExchangeBureau}
}

// CommercialRates reads every pair with one call and gives the rate of each code
// against the lari. A pair between two foreign currencies is left out. A code with no
// pair, or with a price of zero, is missing. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	all, err := a.client.Currencies(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode, err := pairsOfTheLari(all)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		pair, ok := byCode[code]
		if !ok {
			result.Missing = append(result.Missing, code)

			continue
		}

		rate, quoted, err := toRate(code, pair)
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

// pairsOfTheLari keeps each pair with the base GEL, by its secondary code. Two pairs
// for one code is a broken answer.
func pairsOfTheLari(
	all []kursiclient.Currency,
) (map[domain.CurrencyCode]kursiclient.Currency, error) {
	byCode := make(map[domain.CurrencyCode]kursiclient.Currency, len(all))

	for _, pair := range all {
		if pair.BaseCurrencyCode != lari {
			continue
		}

		code, err := domain.ParseCurrencyCode(pair.SecondaryCurrencyCode)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errMalformed, err)
		}

		if _, seen := byCode[code]; seen {
			return nil, fmt.Errorf("%w: two pairs of GEL and %s", errMalformed, code)
		}

		byCode[code] = pair
	}

	return byCode, nil
}

// toRate converts a pair, and gives false when a price is absent or zero.
func toRate(
	code domain.CurrencyCode,
	pair kursiclient.Currency,
) (domain.CommercialRate, bool, error) {
	buy, err := toPrice(code, "buy", pair.BuyRate.String())
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", pair.SellRate.String())
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
	return fmt.Errorf("reading the Kursi.ge rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
