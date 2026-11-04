package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
)

// Network is a set of stations with their readings, and the instant of the read.
type Network struct {
	Stations []domain.Station
	ReadAt   time.Time
	Stale    bool
}

// CityList is the cities of Georgia, and the instant of the read.
type CityList struct {
	Cities []domain.City
	ReadAt time.Time
	Stale  bool
}

// ReadingSource gives the readings of the stations.
type ReadingSource interface {
	// Current returns every station with its readings from 24 hours before the read to 24
	// hours after it.
	Current(ctx context.Context) (Network, error)
	// Recent returns one station with its readings over the window of Current. An unknown
	// code gives an empty network.
	Recent(ctx context.Context, code domain.StationCode) (Network, error)
	// History returns one station with its readings between the two hours, both included.
	// An unknown code gives an empty network.
	History(ctx context.Context, code domain.StationCode, from, to time.Time) (Network, error)
}

// CitySource gives the cities of Georgia.
type CitySource interface {
	Cities(ctx context.Context) (CityList, error)
}
