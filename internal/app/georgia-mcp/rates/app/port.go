package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// RateSetResult is a rate set and the facts of its read: the instant the server read
// it from the vendor, and whether it is a set read before an outage of the vendor.
type RateSetResult struct {
	Set    domain.RateSet
	ReadAt time.Time
	Stale  bool
}

// OfficialRateSource reads the rate set in force on a date. For a date after the
// newest set, it returns the newest set. Before the first set, it returns an empty set.
type OfficialRateSource interface {
	RateSet(ctx context.Context, date kernel.Date, language kernel.Language) (RateSetResult, error)
}

// CommercialRatesResult holds the commercial rates of one bank for the requested
// codes, and the facts of their read. The server reads each currency on its own, so
// ReadAt is the oldest read instant of the codes, and Stale is true when one code or
// more was read before an outage of the bank.
type CommercialRatesResult struct {
	// Rates holds the requested codes that the bank quotes, sorted by currency.
	Rates []domain.CommercialRate
	// Missing holds the requested codes that the bank does not quote, sorted.
	Missing []domain.CurrencyCode
	ReadAt  time.Time
	Stale   bool
}

// CommercialRateSource reads the current commercial rates of one bank.
type CommercialRateSource interface {
	Bank() domain.Bank
	// CommercialRates takes 1 to MaxCommercialCurrencies distinct codes.
	CommercialRates(ctx context.Context, codes []domain.CurrencyCode) (CommercialRatesResult, error)
}
