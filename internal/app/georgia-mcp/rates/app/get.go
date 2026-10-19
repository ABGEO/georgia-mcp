package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// OfficialRatesQuery asks for the official rates of a date.
type OfficialRatesQuery struct {
	// Currencies is empty to ask for every currency.
	Currencies []domain.CurrencyCode
	// Date is nil to ask for the Tbilisi date of the request.
	Date     *kernel.Date
	Language kernel.Language
}

// OfficialRates holds the requested rates and the requested codes with no rate.
type OfficialRates struct {
	RateSetResult

	Missing []domain.CurrencyCode
}

// GetOfficialRates returns the official rates of a date.
type GetOfficialRates struct {
	source OfficialRateSource
	now    func() time.Time
}

// NewGetOfficialRates returns the use case.
func NewGetOfficialRates(source OfficialRateSource, now func() time.Time) *GetOfficialRates {
	return &GetOfficialRates{source: source, now: now}
}

// Get returns the requested rates of the set in force on the date.
func (u *GetOfficialRates) Get(
	ctx context.Context,
	query OfficialRatesQuery,
) (OfficialRates, error) {
	date := kernel.Today(u.now())
	if query.Date != nil {
		date = *query.Date
	}

	result, err := readInForce(ctx, u.source, date, query.Language, u.now())
	if err != nil {
		return OfficialRates{}, err
	}

	if len(query.Currencies) == 0 {
		return OfficialRates{RateSetResult: result}, nil
	}

	found := make([]domain.OfficialRate, 0, len(query.Currencies))

	var missing []domain.CurrencyCode

	for _, code := range query.Currencies {
		if rate, ok := result.Set.Find(code); ok {
			found = append(found, rate)
		} else {
			missing = append(missing, code)
		}
	}

	result.Set = domain.NewRateSet(result.Set.ValidFrom, result.Set.PublishedAt, found)

	return OfficialRates{RateSetResult: result, Missing: missing}, nil
}

// readInForce reads the set in force on the date. The vendor answers a date after its
// newest set with the newest set and no warning, so a date after today with an older
// set is not published. A date of today or earlier accepts an older set: it is the set
// in force on a weekend or a holiday.
func readInForce(
	ctx context.Context,
	source OfficialRateSource,
	date kernel.Date,
	language kernel.Language,
	now time.Time,
) (RateSetResult, error) {
	result, err := source.RateSet(ctx, date, language)
	if err != nil {
		return RateSetResult{}, err
	}

	if result.Set.Empty() {
		return result, nil
	}

	if date.After(kernel.Today(now)) && result.Set.ValidFrom.Before(date) {
		return RateSetResult{}, &NotPublishedError{Requested: date, Newest: result.Set.ValidFrom}
	}

	if result.Set.ValidFrom.After(date) {
		return RateSetResult{}, fmt.Errorf(
			"%w: the set of %s is valid from %s, after the date",
			kernel.ErrSourceUnavailable, date, result.Set.ValidFrom,
		)
	}

	return result, nil
}
