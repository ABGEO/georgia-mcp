// Package credo reads the commercial rates from the Credo Bank client and converts its
// answer to the domain. It is the anti-corruption layer of the context for Credo Bank.
package credo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	credoclient "github.com/abgeo/georgia-mcp/pkg/credo"
)

const (
	keyVariable = "GEOMCP_RATES_CREDO_API_KEY"
	lari        = "GEL"

	// maxUnitExponent bounds a multiplier at 1 000 000.
	maxUnitExponent = 6
)

var errMalformed = errors.New("credo: malformed answer")

// Adapter implements app.CommercialRateSource with the Credo Bank client.
type Adapter struct {
	client *credoclient.Client
	now    func() time.Time
	logger *slog.Logger
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the instant of the
// request and the read instant. logger receives the record of a refused key.
func NewAdapter(client *credoclient.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Bank returns Credo Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "credo", Name: "Credo Bank", Kind: domain.KindBank}
}

// CommercialRates reads the rates of the branches with one call and gives the price of
// one unit of each code. A code with no item against GEL, or with a price that is
// absent or zero, is missing. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	items, err := a.client.CurrentRates(ctx, credoclient.MethodCommercial, a.now())
	if err != nil {
		return app.CommercialRatesResult{}, a.fail(ctx, err)
	}

	byCode, err := indexItems(items)
	if err != nil {
		return app.CommercialRatesResult{}, a.fail(ctx, err)
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
			return app.CommercialRatesResult{}, a.fail(ctx, err)
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

// indexItems keeps each item against GEL, and fails on two items of one code.
func indexItems(items []credoclient.Rate) (map[domain.CurrencyCode]credoclient.Rate, error) {
	byCode := make(map[domain.CurrencyCode]credoclient.Rate, len(items))

	for _, item := range items {
		if !strings.EqualFold(item.ToCcy, lari) {
			continue
		}

		code := domain.CurrencyCode(strings.ToUpper(item.FromCcy))
		if _, seen := byCode[code]; seen {
			return nil, fmt.Errorf("%w: two items of %s", errMalformed, code)
		}

		byCode[code] = item
	}

	return byCode, nil
}

// toRate converts an item to the price of one unit, and gives false when a price is
// absent or zero.
func toRate(code domain.CurrencyCode, item credoclient.Rate) (domain.CommercialRate, bool, error) {
	exponent, err := unitExponent(item.Multiplier.String())
	if err != nil {
		return domain.CommercialRate{}, false, fmt.Errorf("%w: %s", err, code)
	}

	buy, err := toPrice(code, "buy", item.Buy.String(), exponent)
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", item.Sell.String(), exponent)
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

// unitExponent reads "100" as 2. The multiplier must be a power of 10, so a division by
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

// fail wraps a failure with kernel.ErrSourceUnavailable. A refused key also tells the
// operator which variable holds it.
func (a *Adapter) fail(ctx context.Context, err error) error {
	if apiErr, ok := errors.AsType[*credoclient.APIError](err); ok &&
		apiErr.StatusCode == http.StatusUnauthorized {
		a.logger.ErrorContext(
			ctx,
			"Credo Bank refused the key",
			slog.String("variable", keyVariable),
		)
	}

	return fmt.Errorf("reading the Credo Bank rates: %w: %w", kernel.ErrSourceUnavailable, err)
}
