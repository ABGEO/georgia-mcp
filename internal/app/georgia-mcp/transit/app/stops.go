package app

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// defaultRadius is the distance of a nearby query with no distance, in metres.
const defaultRadius = 500

// StopSearch asks the stops that match a text.
type StopSearch struct {
	Text string
	// Kind is empty for any kind.
	Kind     domain.Kind
	Language kernel.Language
	Limit    int
}

// StopMatches is the stops of a search, best match first.
type StopMatches struct {
	Stops  []domain.Stop
	ReadAt time.Time
	Stale  bool
}

// SearchStops finds the stops whose name or stop code matches a text.
type SearchStops struct {
	network NetworkSource
}

// NewSearchStops returns the use case.
func NewSearchStops(network NetworkSource) *SearchStops {
	return &SearchStops{network: network}
}

// Search returns the matching stops, up to the limit.
func (u *SearchStops) Search(ctx context.Context, query StopSearch) (StopMatches, error) {
	text := normalize(query.Text)
	if text == "" {
		return StopMatches{}, ErrEmptyQuery
	}

	list, err := u.network.Stops(ctx)
	if err != nil {
		return StopMatches{}, fmt.Errorf("searching the stops: %w", err)
	}

	var matches []ranked[domain.Stop]

	for _, stop := range list.Stops {
		if query.Kind != "" && stop.Kind != query.Kind {
			continue
		}

		rank := rankName(stop.Name, text)
		if stop.Code != "" && stop.Code == text {
			rank = rankEqual
		}

		if rank != noMatch {
			matches = append(matches, ranked[domain.Stop]{item: stop, rank: rank})
		}
	}

	stops := sortRanked(
		matches,
		func(s domain.Stop) domain.Name { return s.Name },
		func(s domain.Stop) string { return string(s.ID) },
		query.Language,
	)

	return StopMatches{
		Stops:  limit(stops, query.Limit),
		ReadAt: list.ReadAt,
		Stale:  list.Stale,
	}, nil
}

// NearbyQuery asks the stops within a distance of a point.
type NearbyQuery struct {
	Point domain.Point
	// Radius is in metres. 0 gives 500.
	Radius float64
	// Kind is empty for any kind.
	Kind  domain.Kind
	Limit int
}

// NearbyStop is a stop and its distance from the point, in metres.
type NearbyStop struct {
	Stop   domain.Stop
	Metres int
}

// NearbyStops is the stops within the distance, nearest first.
type NearbyStops struct {
	Stops  []NearbyStop
	ReadAt time.Time
	Stale  bool
}

// ListNearbyStops finds the stops within a distance of a point.
type ListNearbyStops struct {
	network NetworkSource
}

// NewListNearbyStops returns the use case.
func NewListNearbyStops(network NetworkSource) *ListNearbyStops {
	return &ListNearbyStops{network: network}
}

// List returns the stops within the distance, nearest first, up to the limit.
func (u *ListNearbyStops) List(ctx context.Context, query NearbyQuery) (NearbyStops, error) {
	if !query.Point.InTbilisi() {
		return NearbyStops{}, &OutsideTbilisiError{Field: "point", Point: query.Point}
	}

	radius := query.Radius
	if radius <= 0 {
		radius = defaultRadius
	}

	list, err := u.network.Stops(ctx)
	if err != nil {
		return NearbyStops{}, fmt.Errorf("listing the nearby stops: %w", err)
	}

	type measured struct {
		stop   domain.Stop
		metres float64
	}

	var near []measured

	for _, stop := range list.Stops {
		if query.Kind != "" && stop.Kind != query.Kind {
			continue
		}

		if metres := query.Point.MetresTo(stop.Location); metres <= radius {
			near = append(near, measured{stop: stop, metres: metres})
		}
	}

	slices.SortFunc(near, func(a, b measured) int {
		return cmp.Or(cmp.Compare(a.metres, b.metres), cmp.Compare(a.stop.ID, b.stop.ID))
	})

	stops := make([]NearbyStop, 0, len(near))
	for _, item := range limit(near, query.Limit) {
		stops = append(stops, NearbyStop{Stop: item.stop, Metres: int(math.Round(item.metres))})
	}

	return NearbyStops{Stops: stops, ReadAt: list.ReadAt, Stale: list.Stale}, nil
}
