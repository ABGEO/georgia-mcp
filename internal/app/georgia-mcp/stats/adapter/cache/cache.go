// Package cache keeps the tables, the cells, and the price indices of Geostat in memory,
// in front of the ports of the context. The catalog has its own holder.
package cache

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// ttl bounds the age of an answer. Geostat publishes at most once a day.
	ttl = time.Hour
	// maxAge bounds the age of a stale answer.
	maxAge = 24 * time.Hour

	tableEntries = 1024
	cellEntries  = 256
	yearEntries  = 32

	// The key separators hold control characters that no code of Geostat holds.
	partSeparator  = "\x1d"
	valueSeparator = "\x1f"
)

func newStore[V any](entries int) (*cache.Cache[V], error) {
	store, err := cache.New[V](
		cache.Config{TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: entries},
	)
	if err != nil {
		return nil, fmt.Errorf("building a store of Geostat: %w", err)
	}

	return store, nil
}

// TableCache implements app.TableSource in front of another source.
type TableCache struct {
	source app.TableSource
	tables *cache.Cache[app.TableDescription]
	cells  *cache.Cache[app.CellSet]
}

var _ app.TableSource = (*TableCache)(nil)

// NewTableCache returns a cache in front of the source.
func NewTableCache(source app.TableSource) (*TableCache, error) {
	tables, err := newStore[app.TableDescription](tableEntries)
	if err != nil {
		return nil, err
	}

	cells, err := newStore[app.CellSet](cellEntries)
	if err != nil {
		tables.Close()

		return nil, err
	}

	return &TableCache{source: source, tables: tables, cells: cells}, nil
}

// Table returns the cached description of a table in a language, or reads it.
func (c *TableCache) Table(
	ctx context.Context,
	code domain.TableCode,
	language kernel.Language,
) (app.TableDescription, error) {
	result, err := c.tables.Get(
		ctx,
		code.String()+partSeparator+string(language),
		func(ctx context.Context) (app.TableDescription, error) {
			return c.source.Table(ctx, code, language)
		},
	)
	if err != nil {
		return app.TableDescription{}, fmt.Errorf("describing the table: %w", err)
	}

	description := result.Value
	description.Stale = result.Stale

	return description, nil
}

// Cells returns the cached cells of a selection, or reads them. The key sorts the
// dimensions and the values, so the same selection in another order hits the same entry.
func (c *TableCache) Cells(
	ctx context.Context,
	code domain.TableCode,
	selection []app.DimensionSelection,
) (app.CellSet, error) {
	result, err := c.cells.Get(
		ctx,
		cellsKey(code, selection),
		func(ctx context.Context) (app.CellSet, error) {
			return c.source.Cells(ctx, code, selection)
		},
	)
	if err != nil {
		return app.CellSet{}, fmt.Errorf("reading the cells: %w", err)
	}

	set := result.Value
	set.Stale = result.Stale

	return set, nil
}

// Close stops the goroutines of the stores.
func (c *TableCache) Close() error {
	c.tables.Close()
	c.cells.Close()

	return nil
}

func cellsKey(code domain.TableCode, selection []app.DimensionSelection) string {
	parts := make([]string, 0, len(selection))

	for _, item := range selection {
		values := slices.Clone(item.Values)
		slices.Sort(values)
		parts = append(parts, item.Dimension+valueSeparator+strings.Join(values, valueSeparator))
	}

	slices.Sort(parts)

	return code.String() + partSeparator + strings.Join(parts, partSeparator)
}

// PriceIndexCache implements app.PriceIndexSource in front of another source.
type PriceIndexCache struct {
	source app.PriceIndexSource
	years  *cache.Cache[app.PriceIndexYear]
}

var _ app.PriceIndexSource = (*PriceIndexCache)(nil)

// NewPriceIndexCache returns a cache in front of the source.
func NewPriceIndexCache(source app.PriceIndexSource) (*PriceIndexCache, error) {
	years, err := newStore[app.PriceIndexYear](yearEntries)
	if err != nil {
		return nil, err
	}

	return &PriceIndexCache{source: source, years: years}, nil
}

// PriceIndices returns the cached indices of a year, or reads them.
func (c *PriceIndexCache) PriceIndices(ctx context.Context, year int) (app.PriceIndexYear, error) {
	result, err := c.years.Get(
		ctx,
		strconv.Itoa(year),
		func(ctx context.Context) (app.PriceIndexYear, error) {
			return c.source.PriceIndices(ctx, year)
		},
	)
	if err != nil {
		return app.PriceIndexYear{}, fmt.Errorf("reading the price indices: %w", err)
	}

	indices := result.Value
	indices.Stale = result.Stale

	return indices, nil
}

// Close stops the goroutines of the store.
func (c *PriceIndexCache) Close() error {
	c.years.Close()

	return nil
}
