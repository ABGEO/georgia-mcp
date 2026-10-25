// Package crystal reads the commercial rates from the Crystal client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Crystal.
package crystal

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
	crystalclient "github.com/abgeo/georgia-mcp/pkg/crystal"
)

const (
	lari = "GEL"

	// maxUnitExponent bounds a multiplier at 1 000 000.
	maxUnitExponent = 6
)

var errMalformed = errors.New("crystal: malformed answer")

// Adapter implements app.CommercialRateSource with the Crystal client.
type Adapter struct {
	client *crystalclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *crystalclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Crystal, a microfinance organization.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "crystal", Name: "Crystal", Kind: domain.KindMicrofinance}
}

// CommercialRates reads the default cash rates with one call and gives the price of one
// unit of each code. A code with no item against GEL, or with a price that is absent or
// zero, is missing. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	items, err := a.client.CashRates(ctx)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
	}

	byCode, err := indexItems(items)
	if err != nil {
		return app.CommercialRatesResult{}, unavailable(err)
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

// indexItems keeps each item against GEL, which drops the cross rate, and fails on two
// items of one code.
func indexItems(items []crystalclient.Rate) (map[domain.CurrencyCode]crystalclient.Rate, error) {
	byCode := make(map[domain.CurrencyCode]crystalclient.Rate, len(items))

	for _, item := range items {
		if !strings.EqualFold(item.TargetISO, lari) {
			continue
		}

		code := domain.CurrencyCode(strings.ToUpper(item.SourceISO))
		if _, seen := byCode[code]; seen {
			return nil, fmt.Errorf("%w: two items of %s", errMalformed, code)
		}

		byCode[code] = item
	}

	return byCode, nil
}

// toRate converts an item to the price of one unit, and gives false when a price is
// absent or zero.
func toRate(
	code domain.CurrencyCode,
	item crystalclient.Rate,
) (domain.CommercialRate, bool, error) {
	exponent, err := unitExponent(item.Multiplier.String())
	if err != nil {
		return domain.CommercialRate{}, false, fmt.Errorf("%w: %s", err, code)
	}

	buy, err := toPrice(code, "buy", item.BuyRate.String(), exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", item.SellRate.String(), exponent)
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

// unitExponent reads "1000" as 3. The multiplier must be a power of 10, so a division by
// it is a shift of the exponent and rounds no digit.
func unitExponent(multiplier string) (int32, error) {
	exponent := len(multiplier) - 1

	if strings.TrimRight(multiplier, "0") != "1" || exponent > maxUnitExponent {
		return 0, fmt.Errorf("%w: multiplier %q", errMalformed, multiplier)
	}

	return int32(exponent), nil
}

// toPrice parses a price for the units, divides it by 10^exponent, and drops the
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
	return fmt.Errorf("reading the Crystal rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
