package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// PlaceQuery asks the places in Tbilisi that match a text.
type PlaceQuery struct {
	Text     string
	Language kernel.Language
	Limit    int
}

// Places is the places of a search, in the order of the source.
type Places struct {
	Places []domain.Place
	ReadAt time.Time
	Stale  bool
}

// SearchPlaces finds the places in Tbilisi that match a text.
type SearchPlaces struct {
	finder PlaceFinder
}

// NewSearchPlaces returns the use case.
func NewSearchPlaces(finder PlaceFinder) *SearchPlaces {
	return &SearchPlaces{finder: finder}
}

// Search returns the places, up to the limit.
func (u *SearchPlaces) Search(ctx context.Context, query PlaceQuery) (Places, error) {
	text := normalize(query.Text)
	if text == "" {
		return Places{}, ErrEmptyQuery
	}

	list, err := u.finder.Places(ctx, text, query.Language)
	if err != nil {
		return Places{}, fmt.Errorf("searching the places: %w", err)
	}

	return Places{
		Places: limit(list.Places, query.Limit),
		ReadAt: list.ReadAt,
		Stale:  list.Stale,
	}, nil
}
