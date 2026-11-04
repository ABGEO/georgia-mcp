package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
)

// QualityQuery asks for the newest readings of a city, a station, or every station.
type QualityQuery struct {
	// City is a name in Georgian or English, or "" for every city.
	City string
	// Station is nil for every station.
	Station *domain.StationCode
	// Limit is the maximum count of stations, 1 to 100.
	Limit int
}

// StationQuality is a station with its newest readings and its level.
type StationQuality struct {
	Station  domain.Station
	City     domain.City
	Readings []domain.PollutantReading
	Level    domain.Level
}

// Quality is the answer of GetQuality.
type Quality struct {
	// Stations holds the first Limit stations, the worst first.
	Stations []StationQuality
	// Total is the count of stations that match the query.
	Total  int
	ReadAt time.Time
	Stale  bool
}

// GetQuality gives the newest readings of the stations.
type GetQuality struct {
	readings ReadingSource
	cities   CitySource
	now      func() time.Time
}

// NewGetQuality returns the use case.
func NewGetQuality(readings ReadingSource, cities CitySource, now func() time.Time) *GetQuality {
	return &GetQuality{readings: readings, cities: cities, now: now}
}

// Get returns the stations of the query with their newest readings, the worst first.
func (g *GetQuality) Get(ctx context.Context, query QualityQuery) (Quality, error) {
	if query.City != "" && query.Station != nil {
		return Quality{}, ErrBothFilters
	}

	network, err := g.readings.Current(ctx)
	if err != nil {
		return Quality{}, fmt.Errorf("reading the stations: %w", err)
	}

	list, err := g.cities.Cities(ctx)
	if err != nil {
		return Quality{}, fmt.Errorf("reading the cities: %w", err)
	}

	stations, err := selectStations(network.Stations, list.Cities, query)
	if err != nil {
		return Quality{}, err
	}

	now := g.now()
	items := make([]StationQuality, 0, len(stations))

	for _, station := range stations {
		readings := station.Newest(now)
		items = append(items, StationQuality{
			Station:  station,
			City:     cityOf(list.Cities, station.CityID),
			Readings: readings,
			Level:    domain.StationLevel(readings),
		})
	}

	slices.SortFunc(items, compareQuality)

	return Quality{
		Stations: items[:min(query.Limit, len(items))],
		Total:    len(items),
		ReadAt:   network.ReadAt,
		Stale:    network.Stale || list.Stale,
	}, nil
}

func selectStations(
	stations []domain.Station,
	cities []domain.City,
	query QualityQuery,
) ([]domain.Station, error) {
	switch {
	case query.Station != nil:
		index := slices.IndexFunc(stations, func(s domain.Station) bool {
			return s.Code == *query.Station
		})
		if index < 0 {
			return nil, fmt.Errorf("%w: %s", ErrUnknownStation, query.Station)
		}

		return stations[index : index+1], nil
	case query.City != "":
		served := citiesWithStations(stations, cities)

		index := slices.IndexFunc(served, func(c domain.City) bool {
			text := strings.TrimSpace(query.City)

			return strings.EqualFold(c.Name.English, text) ||
				strings.EqualFold(c.Name.Georgian, text)
		})
		if index < 0 {
			return nil, &CityError{Accepted: served}
		}

		return slices.DeleteFunc(slices.Clone(stations), func(s domain.Station) bool {
			return s.CityID != served[index].ID
		}), nil
	default:
		return stations, nil
	}
}

// citiesWithStations returns the cities that hold a station, by English name.
func citiesWithStations(stations []domain.Station, cities []domain.City) []domain.City {
	served := slices.DeleteFunc(slices.Clone(cities), func(c domain.City) bool {
		return !slices.ContainsFunc(
			stations,
			func(s domain.Station) bool { return s.CityID == c.ID },
		)
	})

	slices.SortFunc(served, func(a, b domain.City) int {
		return cmp.Compare(a.Name.English, b.Name.English)
	})

	return served
}

func cityOf(cities []domain.City, id int) domain.City {
	index := slices.IndexFunc(cities, func(c domain.City) bool { return c.ID == id })
	if index < 0 {
		return domain.City{}
	}

	return cities[index]
}

// compareQuality puts the worst level first and no level last.
func compareQuality(a, b StationQuality) int {
	return cmp.Or(
		cmp.Compare(rank(a.Level), rank(b.Level)),
		cmp.Compare(a.City.Name.English, b.City.Name.English),
		cmp.Compare(a.Station.Code.String(), b.Station.Code.String()),
	)
}

func rank(level domain.Level) int {
	if level == 0 {
		return 0
	}

	return -int(level)
}
