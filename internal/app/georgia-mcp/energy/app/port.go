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

// BalanceBook is the balance of GNERC from domain.FirstBalanceYear, as the server read it.
type BalanceBook struct {
	// Years are in order. Each holds a published month.
	Years []domain.YearBalance
	// Modified is when GNERC last changed the workbook. It is zero when GNERC does not say.
	Modified time.Time
	ReadAt   time.Time
	Stale    bool
}

// PriceBook is the prices of GNERC, as the server read them.
type PriceBook struct {
	// Months are the published months from domain.FirstPriceYear, in order.
	Months []domain.MonthPrices
	// Modified is when GNERC last changed a workbook of the prices. It is zero when GNERC
	// does not say.
	Modified time.Time
	ReadAt   time.Time
	Stale    bool
}

// BalanceSource reads the balance of GNERC.
type BalanceSource interface {
	Balance(ctx context.Context) (BalanceBook, error)
}

// PriceSource reads the prices of GNERC.
type PriceSource interface {
	Prices(ctx context.Context) (PriceBook, error)
}
