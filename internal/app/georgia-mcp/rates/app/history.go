package app

import (
	"context"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const concurrentReads = 4

// OfficialHistoryQuery asks for the official rates of one currency over a period.
type OfficialHistoryQuery struct {
	Currency domain.CurrencyCode
	From, To kernel.Date
	Language kernel.Language
}

// OfficialHistory holds one point for each rate set in force in the period.
type OfficialHistory struct {
	Currency domain.CurrencyCode
	// Points ascend by ValidFrom.
	Points []HistoryPoint
	// ReadAt is the oldest read instant of the points.
	ReadAt time.Time
	// Stale is true when one point comes from a set read before an outage.
	Stale bool
}

// HistoryPoint is the official rate of one rate set.
type HistoryPoint struct {
	ValidFrom   kernel.Date
	PublishedAt time.Time
	Rate        domain.OfficialRate
}

// ListOfficialHistory returns the official rates of one currency over a period.
type ListOfficialHistory struct {
	source OfficialRateSource
	now    func() time.Time
}

// NewListOfficialHistory returns the use case.
func NewListOfficialHistory(source OfficialRateSource, now func() time.Time) *ListOfficialHistory {
	return &ListOfficialHistory{source: source, now: now}
}

// List returns each rate set in force on a day of the period, the set in force on
// From included, once for each valid date.
func (u *ListOfficialHistory) List(
	ctx context.Context,
	query OfficialHistoryQuery,
) (OfficialHistory, error) {
	if err := validatePeriod(query.From, query.To); err != nil {
		return OfficialHistory{}, err
	}

	now := u.now()
	today := kernel.Today(now)

	if query.To.After(today) {
		if _, err := readInForce(ctx, u.source, query.To, query.Language, now); err != nil {
			return OfficialHistory{}, err
		}
	}

	results, err := u.readDays(ctx, query, now, today.AddDays(1))
	if err != nil {
		return OfficialHistory{}, err
	}

	return collect(query.Currency, results), nil
}

func validatePeriod(from, to kernel.Date) error {
	if to.Before(from) {
		return ErrPeriodReversed
	}

	if days := kernel.DaysBetween(from, to) + 1; days > MaxHistoryDays {
		return &PeriodTooLongError{Days: days, Max: MaxHistoryDays}
	}

	return nil
}

func (u *ListOfficialHistory) readDays(
	ctx context.Context,
	query OfficialHistoryQuery,
	now time.Time,
	lastReadable kernel.Date,
) ([]RateSetResult, error) {
	last := query.To
	if last.After(lastReadable) {
		last = lastReadable
	}

	if last.Before(query.From) {
		return nil, nil
	}

	results := make([]RateSetResult, kernel.DaysBetween(query.From, last)+1)

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(concurrentReads)

	for i := range results {
		group.Go(func() error {
			result, err := readInForce(
				groupCtx,
				u.source,
				query.From.AddDays(i),
				query.Language,
				now,
			)
			results[i] = result

			return err
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // readInForce returns the error of the source or of this package.
	}

	return results, nil
}

func collect(currency domain.CurrencyCode, results []RateSetResult) OfficialHistory {
	history := OfficialHistory{Currency: currency}
	byDate := map[kernel.Date]HistoryPoint{}

	for _, result := range results {
		rate, ok := result.Set.Find(currency)
		if !ok {
			continue
		}

		byDate[result.Set.ValidFrom] = HistoryPoint{
			ValidFrom:   result.Set.ValidFrom,
			PublishedAt: result.Set.PublishedAt,
			Rate:        rate,
		}

		if history.ReadAt.IsZero() || result.ReadAt.Before(history.ReadAt) {
			history.ReadAt = result.ReadAt
		}

		history.Stale = history.Stale || result.Stale
	}

	for _, point := range byDate {
		history.Points = append(history.Points, point)
	}

	slices.SortFunc(history.Points, func(a, b HistoryPoint) int {
		return a.ValidFrom.Time().Compare(b.ValidFrom.Time())
	})

	return history
}
