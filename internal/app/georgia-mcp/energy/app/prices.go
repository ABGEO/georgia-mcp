package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// PriceQuery asks for the prices of a period of months.
type PriceQuery struct {
	// From and To are zero for no month.
	From, To domain.Month
}

// PriceAnswer holds the prices of each published month of a period.
type PriceAnswer struct {
	Months   []domain.MonthPrices
	Newest   domain.Month
	Modified time.Time
	ReadAt   time.Time
	Stale    bool
}

// ListPrices gives the prices of a period.
type ListPrices struct {
	source PriceSource
}

// NewListPrices returns the use case over source.
func NewListPrices(source PriceSource) *ListPrices {
	return &ListPrices{source: source}
}

// List returns the prices of each published month of the period. To defaults to the
// newest month, and From to 11 months before To. It refuses a From outside the prices with
// a *PeriodError, a To before From with ErrReversedPeriod, and more than
// domain.MaxPriceMonths with a *SpanError. The period ends at the newest month.
func (l *ListPrices) List(ctx context.Context, query PriceQuery) (PriceAnswer, error) {
	book, err := l.source.Prices(ctx)
	if err != nil {
		return PriceAnswer{}, fmt.Errorf("reading the prices: %w", err)
	}

	if len(book.Months) == 0 {
		return PriceAnswer{}, fmt.Errorf(
			"reading the prices: %w: %w",
			errNoData,
			kernel.ErrSourceUnavailable,
		)
	}

	newest := book.Months[len(book.Months)-1].Month

	from, to, err := period(query, newest)
	if err != nil {
		return PriceAnswer{}, err
	}

	answer := PriceAnswer{
		Newest:   newest,
		Modified: book.Modified,
		ReadAt:   book.ReadAt,
		Stale:    book.Stale,
	}

	for _, month := range book.Months {
		if !month.Month.Before(from) && !to.Before(month.Month) {
			answer.Months = append(answer.Months, month)
		}
	}

	return answer, nil
}

// period fills the missing ends of query and checks the period against the prices up to
// newest.
func period(query PriceQuery, newest domain.Month) (domain.Month, domain.Month, error) {
	first := domain.Month{Year: domain.FirstPriceYear, Month: time.January}

	to := query.To
	if to.IsZero() {
		to = newest
	}

	from := query.From
	if from.IsZero() {
		from = to.AddMonths(1 - domain.DefaultPriceMonths)
	}

	switch {
	case from.Before(first) || newest.Before(from):
		return from, to, &PeriodError{
			Field:  "from",
			Value:  from.String(),
			First:  first,
			Newest: newest,
		}
	case to.Before(from):
		return from, to, ErrReversedPeriod
	case to.MonthsFrom(from) > domain.MaxPriceMonths:
		return from, to, &SpanError{Months: to.MonthsFrom(from)}
	}

	return from, to, nil
}
