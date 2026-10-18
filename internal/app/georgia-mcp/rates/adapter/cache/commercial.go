package cache

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	commercialTTL      = 5 * time.Minute
	commercialMaxStale = 55 * time.Minute
	commercialEntries  = 256
)

// commercialQuote is the answer of a bank for one code, and the instant the source
// read it. A code that the bank does not quote is a quote too, with Quoted false, so
// the next request does not ask again.
type commercialQuote struct {
	Rate   domain.CommercialRate
	Quoted bool
	ReadAt time.Time
}

// CommercialRateCache implements app.CommercialRateSource in front of the source of one
// bank. It keeps each currency on its own, and asks the source for the codes it lacks.
type CommercialRateCache struct {
	source app.CommercialRateSource
	quotes *cache.Cache[commercialQuote]
}

var _ app.CommercialRateSource = (*CommercialRateCache)(nil)

// NewCommercial returns a cache in front of the source of one bank.
func NewCommercial(source app.CommercialRateSource) (*CommercialRateCache, error) {
	quotes, err := cache.New[commercialQuote](cache.Config{
		TTL:        commercialTTL,
		MaxStale:   commercialMaxStale,
		MaxEntries: commercialEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of commercial rates: %w", err)
	}

	return &CommercialRateCache{source: source, quotes: quotes}, nil
}

// Bank returns the bank of the source.
func (c *CommercialRateCache) Bank() domain.Bank {
	return c.source.Bank()
}

// CommercialRates returns the cached quote of each code, and reads the other codes in
// one call. ReadAt is the oldest read instant of the codes.
func (c *CommercialRateCache) CommercialRates(
	ctx context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	keys := make([]string, 0, len(codes))
	for _, code := range codes {
		keys = append(keys, string(code))
	}

	quotes, err := c.quotes.GetMany(ctx, keys, c.load)
	if err != nil {
		return app.CommercialRatesResult{}, err
	}

	var result app.CommercialRatesResult

	for _, code := range slices.Sorted(slices.Values(codes)) {
		quote := quotes[string(code)]

		if quote.Value.Quoted {
			result.Rates = append(result.Rates, quote.Value.Rate)
		} else {
			result.Missing = append(result.Missing, code)
		}

		if result.ReadAt.IsZero() || quote.Value.ReadAt.Before(result.ReadAt) {
			result.ReadAt = quote.Value.ReadAt
		}

		result.Stale = result.Stale || quote.Stale
	}

	return result, nil
}

// Close stops the goroutines of the store.
func (c *CommercialRateCache) Close() error {
	c.quotes.Close()

	return nil
}

func (c *CommercialRateCache) load(
	ctx context.Context,
	keys []string,
) (map[string]commercialQuote, error) {
	codes := make([]domain.CurrencyCode, 0, len(keys))
	for _, key := range keys {
		codes = append(codes, domain.CurrencyCode(key))
	}

	read, err := c.source.CommercialRates(ctx, codes)
	if err != nil {
		return nil, err
	}

	quotes := make(map[string]commercialQuote, len(keys))

	for _, rate := range read.Rates {
		quotes[string(rate.Currency)] = commercialQuote{
			Rate:   rate,
			Quoted: true,
			ReadAt: read.ReadAt,
		}
	}

	for _, code := range read.Missing {
		quotes[string(code)] = commercialQuote{ReadAt: read.ReadAt}
	}

	return quotes, nil
}
