package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// ListQuery asks for the holidays of a year, or the next holidays from a date. A query
// with neither asks for the current year.
type ListQuery struct {
	Year *int
	From *kernel.Date
	// Limit is the maximum count of holidays with From. A year gives every holiday.
	Limit int
}

// HolidayList is the answer of ListHolidays.
type HolidayList struct {
	Holidays []domain.Holiday
	// ReadAt is the oldest read of the calendars of the list.
	ReadAt time.Time
	// Stale is true when any calendar of the list is stale.
	Stale bool
}

// ListHolidays gives the public holidays of a year, or the next ones from a date.
type ListHolidays struct {
	source HolidaySource
	now    func() time.Time
}

// NewListHolidays returns the use case.
func NewListHolidays(source HolidaySource, now func() time.Time) *ListHolidays {
	return &ListHolidays{source: source, now: now}
}

// List returns the holidays of the query in date order.
func (l *ListHolidays) List(ctx context.Context, query ListQuery) (HolidayList, error) {
	if query.Year != nil && query.From != nil {
		return HolidayList{}, ErrBothFilters
	}

	today := kernel.Today(l.now())

	if query.From != nil {
		return l.next(ctx, *query.From, max(query.Limit, 1), today)
	}

	year := today.Time().Year()
	if query.Year != nil {
		year = *query.Year
	}

	if err := checkYear(year, today); err != nil {
		return HolidayList{}, err
	}

	calendar, err := l.source.Holidays(ctx, year)
	if err != nil {
		return HolidayList{}, fmt.Errorf("reading the holidays of %d: %w", year, err)
	}

	return HolidayList{
		Holidays: calendar.Holidays,
		ReadAt:   calendar.ReadAt,
		Stale:    calendar.Stale,
	}, nil
}

func (l *ListHolidays) next(
	ctx context.Context,
	from kernel.Date,
	limit int,
	today kernel.Date,
) (HolidayList, error) {
	start := from.Time().Year()
	if err := checkYear(start, today); err != nil {
		return HolidayList{}, err
	}

	_, last := SupportedYears(today)

	var list HolidayList

	for year := start; year <= last && len(list.Holidays) < limit; year++ {
		calendar, err := l.source.Holidays(ctx, year)
		if err != nil {
			return HolidayList{}, fmt.Errorf("reading the holidays of %d: %w", year, err)
		}

		list.add(calendar, from, limit)
	}

	return list, nil
}

func (l *HolidayList) add(calendar Calendar, from kernel.Date, limit int) {
	if l.ReadAt.IsZero() || calendar.ReadAt.Before(l.ReadAt) {
		l.ReadAt = calendar.ReadAt
	}

	l.Stale = l.Stale || calendar.Stale

	for _, holiday := range calendar.Holidays {
		if len(l.Holidays) == limit {
			return
		}

		if !holiday.Date.Before(from) {
			l.Holidays = append(l.Holidays, holiday)
		}
	}
}
