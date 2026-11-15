// Package nager reads the public holidays from the Nager.Date client and converts them to
// the domain. It is the anti-corruption layer of the context for Nager.Date.
package nager

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	nagerclient "github.com/abgeo/georgia-mcp/pkg/nager"
)

const (
	country    = "GE"
	publicType = "Public"
)

var (
	errNoHoliday = errors.New("no public holiday in the answer")
	errOtherYear = errors.New("a holiday of another year")
)

// Adapter implements app.HolidaySource with the Nager.Date client.
type Adapter struct {
	client *nagerclient.Client
	now    func() time.Time
}

var _ app.HolidaySource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *nagerclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Holidays reads the public holidays of Georgia in the year. Georgia has public holidays
// each year, so an answer with none is an outage.
func (a *Adapter) Holidays(ctx context.Context, year int) (app.Calendar, error) {
	items, err := a.client.PublicHolidays(ctx, year, country)
	if err != nil {
		return app.Calendar{}, unavailable(err)
	}

	holidays := make([]domain.Holiday, 0, len(items))

	for _, item := range items {
		if !slices.Contains(item.Types, publicType) {
			continue
		}

		holiday, err := toHoliday(item, year)
		if err != nil {
			return app.Calendar{}, unavailable(err)
		}

		holidays = append(holidays, holiday)
	}

	if len(holidays) == 0 {
		return app.Calendar{}, unavailable(errNoHoliday)
	}

	slices.SortStableFunc(holidays, func(x, y domain.Holiday) int {
		return x.Date.Time().Compare(y.Date.Time())
	})

	return app.Calendar{
		Year:     year,
		Holidays: holidays,
		ReadAt:   a.now().In(kernel.TbilisiZone()),
	}, nil
}

func toHoliday(item nagerclient.PublicHoliday, year int) (domain.Holiday, error) {
	date, err := kernel.ParseDate(item.Date)
	if err != nil {
		return domain.Holiday{}, fmt.Errorf("reading the date of a holiday: %w", err)
	}

	if date.Time().Year() != year {
		return domain.Holiday{}, fmt.Errorf("%w: %s", errOtherYear, date)
	}

	name, err := domain.NewHolidayName(item.LocalName, item.Name)
	if err != nil {
		return domain.Holiday{}, fmt.Errorf("reading the name of %s: %w", date, err)
	}

	return domain.Holiday{Date: date, Name: name}, nil
}

func unavailable(err error) error {
	return fmt.Errorf("reading Nager.Date: %w: %w", kernel.ErrSourceUnavailable, err)
}
