package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	maxPeriodDays = 7
	lastDay       = 24 * time.Hour
	lastHourOfDay = 23 * time.Hour
)

// HistoryQuery asks for the readings of one station over a period.
type HistoryQuery struct {
	Station domain.StationCode
	// From and To are both nil for the last 24 hours, or both set.
	From, To *kernel.Date
	// Pollutant is nil for every pollutant of the station.
	Pollutant *domain.Pollutant
	// Limit is the maximum count of readings of each pollutant, 1 to 100.
	Limit int
}

// History is the answer of ListHistory.
type History struct {
	// Station holds each series cut to the limit, newest first.
	Station domain.Station
	City    domain.City
	// From and To are the first and the last hour of the period.
	From, To time.Time
	// Total is the count of readings of each pollutant in the period, before the cut.
	Total  map[domain.Pollutant]int
	ReadAt time.Time
	Stale  bool
}

// ListHistory gives the hourly readings of one station over a period.
type ListHistory struct {
	readings ReadingSource
	cities   CitySource
	now      func() time.Time
}

// NewListHistory returns the use case.
func NewListHistory(readings ReadingSource, cities CitySource, now func() time.Time) *ListHistory {
	return &ListHistory{readings: readings, cities: cities, now: now}
}

// List returns the readings of the station in the period, newest first.
func (l *ListHistory) List(ctx context.Context, query HistoryQuery) (History, error) {
	from, to, keep, err := period(query.From, query.To, l.now().In(kernel.TbilisiZone()))
	if err != nil {
		return History{}, err
	}

	network, err := l.read(ctx, query, from, to)
	if err != nil {
		return History{}, fmt.Errorf("reading the history: %w", err)
	}

	if len(network.Stations) == 0 {
		return History{}, fmt.Errorf("%w: %s", ErrUnknownStation, query.Station)
	}

	list, err := l.cities.Cities(ctx)
	if err != nil {
		return History{}, fmt.Errorf("reading the cities: %w", err)
	}

	station := network.Stations[0]

	var total map[domain.Pollutant]int

	station.Series, total = cut(station.Series, query, keep, to)

	if query.From == nil {
		to = latest(station.Series, keep.Add(lastDay))
	}

	return History{
		Station: station,
		City:    cityOf(list.Cities, station.CityID),
		From:    keep,
		To:      to,
		Total:   total,
		ReadAt:  network.ReadAt,
		Stale:   network.Stale || list.Stale,
	}, nil
}

// cut keeps the readings of the query from keep to to, the last Limit of each series,
// newest first, and counts each series before the limit.
func cut(
	all []domain.Series,
	query HistoryQuery,
	keep, to time.Time,
) ([]domain.Series, map[domain.Pollutant]int) {
	total := map[domain.Pollutant]int{}

	var series []domain.Series

	for _, item := range all {
		if query.Pollutant != nil && item.Pollutant != *query.Pollutant {
			continue
		}

		readings := slices.DeleteFunc(slices.Clone(item.Readings), func(r domain.Reading) bool {
			return r.Time.Before(keep) || r.Time.After(to)
		})
		total[item.Pollutant] = len(readings)

		readings = readings[max(0, len(readings)-query.Limit):]
		slices.Reverse(readings)
		series = append(series, domain.Series{Pollutant: item.Pollutant, Readings: readings})
	}

	return series, total
}

func (l *ListHistory) read(
	ctx context.Context,
	query HistoryQuery,
	from, to time.Time,
) (Network, error) {
	if query.From == nil {
		return l.readings.Recent(ctx, query.Station)
	}

	return l.readings.History(ctx, query.Station, from, to)
}

// latest returns the time of the newest reading, or the hour when it is later.
func latest(series []domain.Series, hour time.Time) time.Time {
	for _, item := range series {
		if len(item.Readings) > 0 && item.Readings[0].Time.After(hour) {
			hour = item.Readings[0].Time
		}
	}

	return hour
}

// period returns the hours to read and the first hour to keep. With no dates, it reads
// 24 hours ahead too: the NEA stamps some pollutants ahead of the clock.
func period(from, to *kernel.Date, now time.Time) (time.Time, time.Time, time.Time, error) {
	if from == nil && to == nil {
		hour := now.Truncate(time.Hour)

		return hour.Add(-lastDay), hour.Add(lastDay), hour.Add(-lastDay), nil
	}

	if from == nil || to == nil {
		return refuse("from, to: give both dates, or neither for the last 24 hours.")
	}

	if to.Before(*from) {
		return refuse(fmt.Sprintf("to: %s is before from %s.", to, from))
	}

	earliest := kernel.Today(now).AddDays(-maxPeriodDays)
	if from.Before(earliest) {
		return refuse(fmt.Sprintf("from: %s is more than %d days ago. Give %s or later.",
			from, maxPeriodDays, earliest))
	}

	if days := kernel.DaysBetween(*from, *to) + 1; days > maxPeriodDays {
		return refuse(fmt.Sprintf("from, to: %s to %s is %d days. Give at most %d.",
			from, to, days, maxPeriodDays))
	}

	start := midnight(*from)

	return start, midnight(*to).Add(lastHourOfDay), start, nil
}

func midnight(date kernel.Date) time.Time {
	day := date.Time()

	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, kernel.TbilisiZone())
}

func refuse(message string) (time.Time, time.Time, time.Time, error) {
	return time.Time{}, time.Time{}, time.Time{}, &PeriodError{Message: message}
}
