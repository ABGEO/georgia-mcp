package app

import (
	"context"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// cellKeySeparator joins the value codes of a cell. No code of Geostat holds it.
const cellKeySeparator = "\x1f"

// Catalog is every table of Geostat in one language.
type Catalog struct {
	Entries []domain.CatalogEntry
	ReadAt  time.Time
	Stale   bool
}

// CatalogSource gives the catalog of a language.
type CatalogSource interface {
	Catalog(ctx context.Context, language kernel.Language) (Catalog, error)
}

// TableDescription is a table with its dimensions, and the instant of the read.
type TableDescription struct {
	Table  domain.Table
	ReadAt time.Time
	Stale  bool
}

// DimensionSelection is the selection of one dimension: value codes, or a count of the
// last values. The port gets value codes only.
type DimensionSelection struct {
	Dimension string
	Values    []string
	Last      int
}

// CellSet is the figures of a selection. CellKey gives the key of each figure.
type CellSet struct {
	Figures map[string]domain.Figure
	ReadAt  time.Time
	Stale   bool
}

// CellKey returns the key of the cell with these value codes, in the order of the
// dimensions of the table.
func CellKey(codes []string) string {
	return strings.Join(codes, cellKeySeparator)
}

// TableSource gives the description and the cells of a table. Cells gets a value list for
// every dimension of the table.
type TableSource interface {
	Table(
		ctx context.Context,
		code domain.TableCode,
		language kernel.Language,
	) (TableDescription, error)
	Cells(
		ctx context.Context,
		code domain.TableCode,
		selection []DimensionSelection,
	) (CellSet, error)
}

// IndexKey selects one index of a year.
type IndexKey struct {
	Region domain.Region
	Group  domain.PriceGroup
	Month  int
}

// PriceIndexYear holds the consumer price indices of one year. A month that Geostat did
// not publish holds unpublished figures. A year that the tables do not hold is empty.
type PriceIndexYear struct {
	Year int
	// Annual is the index of the same month of the previous year = 100.
	Annual map[IndexKey]domain.Figure
	// Monthly is the index of the previous month = 100.
	Monthly map[IndexKey]domain.Figure
	// Average is the index of the 12 month average. Its group is always the total.
	Average map[IndexKey]domain.Figure
	// Core is the core inflation in percent, by month from 1 to 12.
	Core   map[int]domain.Figure
	Tables []domain.CatalogEntry
	ReadAt time.Time
	Stale  bool
}

// PriceIndexSource gives the consumer price indices of a year.
type PriceIndexSource interface {
	PriceIndices(ctx context.Context, year int) (PriceIndexYear, error)
}
