package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// DayQuery asks about one date. A query with no date asks about the current date.
type DayQuery struct {
	Date *kernel.Date
}

// Day is the answer of GetDay.
type Day struct {
	Date kernel.Date
	// Holiday is nil when the date is not a public holiday.
	Holiday *domain.Holiday
	ReadAt  time.Time
	Stale   bool
}

// GetDay tells whether a date is a public holiday.
type GetDay struct {
	source HolidaySource
	now    func() time.Time
}

// NewGetDay returns the use case.
func NewGetDay(source HolidaySource, now func() time.Time) *GetDay {
	return &GetDay{source: source, now: now}
}

// Get returns the date with its holiday, if it has one.
func (g *GetDay) Get(ctx context.Context, query DayQuery) (Day, error) {
	today := kernel.Today(g.now())

	date := today
	if query.Date != nil {
		date = *query.Date
	}

	year := date.Time().Year()
	if err := checkYear(year, today); err != nil {
		return Day{}, err
	}

	calendar, err := g.source.Holidays(ctx, year)
	if err != nil {
		return Day{}, fmt.Errorf("reading the holidays of %d: %w", year, err)
	}

	day := Day{Date: date, ReadAt: calendar.ReadAt, Stale: calendar.Stale}

	for _, holiday := range calendar.Holidays {
		if holiday.Date == date {
			day.Holiday = &holiday

			break
		}
	}

	return day, nil
}
