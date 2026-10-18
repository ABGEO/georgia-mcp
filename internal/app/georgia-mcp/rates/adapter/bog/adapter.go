// Package bog reads the commercial rates from the Bank of Georgia client and converts
// its answer to the domain. It is the anti-corruption layer of the context for Bank of
// Georgia.
package bog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	bogclient "github.com/abgeo/georgia-mcp/pkg/bog"
)

var errMalformed = errors.New("bog: malformed answer")

// Adapter implements app.CommercialRateSource with the Bank of Georgia client.
type Adapter struct {
	client *bogclient.Client
	now    func() time.Time
}

var _ app.CommercialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *bogclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Bank returns Bank of Georgia.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "bog", Name: "Bank of Georgia"}
}

// CommercialRates reads each code with its own call, all at once. A code with no rate
// is missing. The first failure cancels the other calls. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	sorted := slices.Sorted(slices.Values(codes))
	rates := make([]domain.CommercialRate, len(sorted))
	quoted := make([]bool, len(sorted))

	group, ctx := errgroup.WithContext(ctx)

	for i, code := range sorted {
		group.Go(func() error {
			rate, ok, err := a.read(ctx, code)
			rates[i], quoted[i] = rate, ok

			return err
		})
	}

	if err := group.Wait(); err != nil {
		return app.CommercialRatesResult{}, fmt.Errorf(
			"reading the Bank of Georgia rates: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	var result app.CommercialRatesResult

	for i, code := range sorted {
		if quoted[i] {
			result.Rates = append(result.Rates, rates[i])
		} else {
			result.Missing = append(result.Missing, code)
		}
	}

	result.ReadAt = a.now()

	return result, nil
}

// read returns the rate of one code, and false when the bank has no rate of it.
func (a *Adapter) read(
	ctx context.Context,
	code domain.CurrencyCode,
) (domain.CommercialRate, bool, error) {
	quote, err := a.client.CommercialRate(ctx, string(code))
	if errors.Is(err, bogclient.ErrNoRate) {
		return domain.CommercialRate{}, false, nil
	}

	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	buy, err := toPrice(code, "buy", quote.Buy.String())
	if err != nil {
		return domain.CommercialRate{}, false, err
	}

	sell, err := toPrice(code, "sell", quote.Sell.String())
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

// toPrice parses a price and drops the zeros that pad it to eight decimal places. An
// absent price is zero.
func toPrice(code domain.CurrencyCode, side, text string) (decimal.Decimal, error) {
	if text == "" {
		return decimal.Zero, nil
	}

	price, err := decimal.NewFromString(text)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("%w: %s %s %q", errMalformed, code, side, text)
	}

	if price.IsNegative() {
		return decimal.Decimal{}, fmt.Errorf("%w: %s %s %q", errMalformed, code, side, text)
	}

	return decimal.RequireFromString(price.String()), nil
}
