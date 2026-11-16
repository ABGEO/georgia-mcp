package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
)

// GridQuery selects a grid state.
type GridQuery struct {
	// Moment is nil for the newest grid state.
	Moment *time.Time
}

// GetGrid returns the grid state of the newest slot, or of the slot of a moment.
type GetGrid struct {
	source GridSource
	now    func() time.Time
}

// NewGetGrid returns the use case over the source.
func NewGetGrid(source GridSource, now func() time.Time) *GetGrid {
	return &GetGrid{source: source, now: now}
}

// Get returns the grid state of the query.
func (g *GetGrid) Get(ctx context.Context, query GridQuery) (GridRead, error) {
	now := g.now()

	if query.Moment == nil {
		read, err := g.source.Newest(ctx)

		return read, noGrid(err, domain.SlotOf(now))
	}

	moment := *query.Moment
	earliest := now.Add(-domain.Retention)

	if moment.After(now) || moment.Before(earliest) {
		return GridRead{}, &MomentError{
			Moment:   moment,
			Earliest: earliest,
			Future:   moment.After(now),
		}
	}

	slot := domain.SlotOf(moment)
	read, err := g.source.At(ctx, slot)

	return read, noGrid(err, slot)
}

func noGrid(err error, slot time.Time) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNoGrid):
		return &NoGridError{Slot: slot}
	default:
		return fmt.Errorf("reading the grid: %w", err)
	}
}
