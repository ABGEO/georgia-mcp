package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
)

// GridRead is a grid state with the frequency of its minute, as the server read them.
type GridRead struct {
	State domain.GridState
	// Frequency is nil when GSE has none for the minute of the slot.
	Frequency *domain.Frequency
	ReadAt    time.Time
	Stale     bool
}

// GridSource reads the grid of GSE.
type GridSource interface {
	// Newest returns the newest grid state from the slot of the read back
	// domain.SearchWindow. ErrNoGrid when GSE has none.
	Newest(ctx context.Context) (GridRead, error)
	// At returns the newest grid state from slot back domain.SearchWindow. ErrNoGrid when
	// GSE has none.
	At(ctx context.Context, slot time.Time) (GridRead, error)
}
