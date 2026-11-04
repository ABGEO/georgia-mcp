// Package nea reads the stations and the municipalities from the NEA client and converts
// them to the domain. It is the anti-corruption layer of the context for the NEA.
package nea

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	neaclient "github.com/abgeo/georgia-mcp/pkg/nea"
)

const (
	timeLayout = "2006-01-02T15:04:05"
	window     = 24 * time.Hour
)

var errNoStation = errors.New("no station in the answer")

// Adapter implements app.ReadingSource and app.CitySource with the NEA client.
type Adapter struct {
	client *neaclient.Client
	now    func() time.Time
}

var (
	_ app.ReadingSource = (*Adapter)(nil)
	_ app.CitySource    = (*Adapter)(nil)
)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *neaclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Current reads every station from 24 hours before the read hour to 24 hours after it:
// the NEA stamps some pollutants ahead of the clock. The NEA always lists its stations,
// so an answer with none is an outage.
func (a *Adapter) Current(ctx context.Context) (app.Network, error) {
	network, err := a.read(ctx, a.window(""))
	if err != nil {
		return app.Network{}, err
	}

	if len(network.Stations) == 0 {
		return app.Network{}, unavailable(errNoStation)
	}

	return network, nil
}

// Recent reads one station over the window of Current.
func (a *Adapter) Recent(ctx context.Context, code domain.StationCode) (app.Network, error) {
	return a.read(ctx, a.window(code.String()))
}

// History reads one station between the two hours.
func (a *Adapter) History(
	ctx context.Context,
	code domain.StationCode,
	from, to time.Time,
) (app.Network, error) {
	zone := kernel.TbilisiZone()

	return a.read(ctx, neaclient.HourlyQuery{
		From:    from.In(zone).Format(timeLayout),
		To:      to.In(zone).Format(timeLayout),
		Station: code.String(),
	})
}

// Cities reads the municipalities.
func (a *Adapter) Cities(ctx context.Context) (app.CityList, error) {
	municipalities, err := a.client.Municipalities(ctx)
	if err != nil {
		return app.CityList{}, unavailable(err)
	}

	cities := make([]domain.City, 0, len(municipalities))
	for _, municipality := range municipalities {
		cities = append(cities, domain.City{
			ID:   municipality.ID,
			Name: domain.Name{Georgian: municipality.DistrictGE, English: municipality.DistrictEN},
		})
	}

	return app.CityList{Cities: cities, ReadAt: a.now().In(kernel.TbilisiZone())}, nil
}

func (a *Adapter) window(station string) neaclient.HourlyQuery {
	hour := a.now().In(kernel.TbilisiZone()).Truncate(time.Hour)

	return neaclient.HourlyQuery{
		From:    hour.Add(-window).Format(timeLayout),
		To:      hour.Add(window).Format(timeLayout),
		Station: station,
	}
}

// read fails the whole answer for one field that does not convert: a broken field means
// a change of the source, and a partial list looks complete to the model.
func (a *Adapter) read(ctx context.Context, query neaclient.HourlyQuery) (app.Network, error) {
	answer, err := a.client.Hourly(ctx, query)
	if err != nil {
		return app.Network{}, unavailable(err)
	}

	stations := make([]domain.Station, 0, len(answer))

	for _, item := range answer {
		station, err := convertStation(item)
		if err != nil {
			return app.Network{}, unavailable(err)
		}

		stations = append(stations, station)
	}

	return app.Network{Stations: stations, ReadAt: a.now().In(kernel.TbilisiZone())}, nil
}

func convertStation(item neaclient.Station) (domain.Station, error) {
	code, err := domain.ParseStationCode(item.Code)
	if err != nil {
		return domain.Station{}, err
	}

	byPollutant := map[domain.Pollutant]domain.Series{}

	for _, equipment := range item.Equipment {
		pollutant, err := domain.ParsePollutant(equipment.Substance.Name)
		if err != nil {
			continue
		}

		series, err := convertSeries(pollutant, equipment)
		if err != nil {
			return domain.Station{}, fmt.Errorf("station %s: %w", item.Code, err)
		}

		byPollutant[pollutant] = series
	}

	var series []domain.Series

	for _, pollutant := range domain.Pollutants() {
		if found, ok := byPollutant[pollutant]; ok {
			series = append(series, found)
		}
	}

	return domain.Station{
		Code:    code,
		CityID:  item.MunicipalityID,
		Address: domain.Name{Georgian: item.AddressGE, English: item.AddressEN},
		Lat:     item.Lat,
		Lon:     item.Long,
		Series:  series,
	}, nil
}

func convertSeries(
	pollutant domain.Pollutant,
	equipment neaclient.Equipment,
) (domain.Series, error) {
	bands, err := convertBands(equipment.Substance.Levels)
	if err != nil {
		return domain.Series{}, fmt.Errorf("%s: %w", pollutant, err)
	}

	readings := make([]domain.Reading, 0, len(equipment.Values))

	for _, value := range equipment.Values {
		if value.Value == "" {
			continue
		}

		number, err := decimal.NewFromString(value.Value.String())
		if err != nil {
			return domain.Series{}, fmt.Errorf("%s value %q: %w", pollutant, value.Value, err)
		}

		at, err := time.ParseInLocation(timeLayout, value.DateTime, kernel.TbilisiZone())
		if err != nil {
			return domain.Series{}, fmt.Errorf("%s time: %w", pollutant, err)
		}

		readings = append(
			readings,
			domain.Reading{Value: number, Time: at, Level: bands.Level(number)},
		)
	}

	slices.SortStableFunc(readings, func(a, b domain.Reading) int {
		return a.Time.Compare(b.Time)
	})

	return domain.Series{Pollutant: pollutant, Readings: readings}, nil
}

func convertBands(levels []neaclient.IndexLevel) (domain.Bands, error) {
	if len(levels) == 0 {
		return domain.Bands{}, fmt.Errorf("%w: no limits", domain.ErrInvalidBands)
	}

	var limits [4]decimal.Decimal

	for index, text := range []string{
		levels[0].GoodTo.String(), levels[0].FairTo.String(),
		levels[0].ModerateTo.String(), levels[0].PoorTo.String(),
	} {
		limit, err := decimal.NewFromString(text)
		if err != nil {
			return domain.Bands{}, fmt.Errorf("%w: %q", domain.ErrInvalidBands, text)
		}

		limits[index] = limit
	}

	return domain.NewBands(limits[0], limits[1], limits[2], limits[3])
}

func unavailable(err error) error {
	return fmt.Errorf("reading the NEA: %w: %w", kernel.ErrSourceUnavailable, err)
}
