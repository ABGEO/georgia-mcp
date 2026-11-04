package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingGeostat answers empty values read at the current instant, counts each call,
// and fails with an outage once failing is set.
type countingGeostat struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (c *countingGeostat) Table(
	context.Context,
	domain.TableCode,
	kernel.Language,
) (app.TableDescription, error) {
	readAt, err := c.read()

	return app.TableDescription{ReadAt: readAt}, err
}

func (c *countingGeostat) Cells(
	context.Context,
	domain.TableCode,
	[]app.DimensionSelection,
) (app.CellSet, error) {
	readAt, err := c.read()

	return app.CellSet{ReadAt: readAt}, err
}

func (c *countingGeostat) PriceIndices(context.Context, int) (app.PriceIndexYear, error) {
	readAt, err := c.read()

	return app.PriceIndexYear{ReadAt: readAt}, err
}

func (c *countingGeostat) read() (time.Time, error) {
	c.calls.Add(1)

	if c.failing.Load() {
		return time.Time{}, fmt.Errorf("calling Geostat: %w", kernel.ErrSourceUnavailable)
	}

	return time.Now(), nil
}

// read is one cached method. The flag picks the second form of the same request.
type read func(ctx context.Context, second bool) (bool, error)

func methods(t *testing.T, source *countingGeostat) map[string]read {
	t.Helper()

	tables, err := cache.NewTableCache(source)
	if err != nil {
		t.Fatal(err)
	}

	indices, err := cache.NewPriceIndexCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = tables.Close()
		_ = indices.Close()
	})

	code, err := domain.ParseTableCode("Social Statistics/Labour/Wages_Region.px")
	if err != nil {
		t.Fatal(err)
	}

	first := []app.DimensionSelection{
		{Dimension: "Region", Values: []string{"0", "1"}},
		{Dimension: "Year", Values: []string{"16"}},
	}
	second := []app.DimensionSelection{
		{Dimension: "Year", Values: []string{"16"}},
		{Dimension: "Region", Values: []string{"1", "0"}},
	}

	return map[string]read{
		"Table": func(ctx context.Context, _ bool) (bool, error) {
			got, err := tables.Table(ctx, code, kernel.LanguageEnglish)

			return got.Stale, err
		},
		"Cells": func(ctx context.Context, other bool) (bool, error) {
			selection := first
			if other {
				selection = second
			}

			got, err := tables.Cells(ctx, code, selection)

			return got.Stale, err
		},
		"PriceIndices": func(ctx context.Context, _ bool) (bool, error) {
			got, err := indices.PriceIndices(ctx, 2026)

			return got.Stale, err
		},
	}
}

func methodNames() []string {
	return []string{"Table", "Cells", "PriceIndices"}
}

// GEOSTAT-SC-022: The same request within 1 hour sends 1 load, the same selection in
// another order too. At 1 hour, the cache loads again.
func TestEachAnswerStaysForAnHour(t *testing.T) {
	t.Parallel()

	for _, name := range methodNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingGeostat{}
				call := methods(t, source)[name]

				if _, err := call(t.Context(), false); err != nil {
					t.Fatal(err)
				}

				time.Sleep(time.Hour - time.Second)

				if _, err := call(t.Context(), true); err != nil || source.calls.Load() != 1 {
					t.Fatalf("at 59:59 = %v with %d calls, want 1 call", err, source.calls.Load())
				}

				time.Sleep(time.Second)

				if _, err := call(t.Context(), false); err != nil || source.calls.Load() != 2 {
					t.Fatalf("at 1 hour = %v with %d calls, want 2 calls", err, source.calls.Load())
				}
			})
		})
	}
}

// GEOSTAT-SC-022: While Geostat fails, an answer stays as stale until it is 24 hours old.
// After that, the cache returns the error.
func TestEachAnswerStaysStaleForADay(t *testing.T) {
	t.Parallel()

	for _, name := range methodNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingGeostat{}
				call := methods(t, source)[name]

				if _, err := call(t.Context(), false); err != nil {
					t.Fatal(err)
				}

				source.failing.Store(true)
				time.Sleep(23 * time.Hour)

				if stale, err := call(t.Context(), false); err != nil || !stale {
					t.Fatalf("at 23 hours = stale %t, error %v, want the stale answer", stale, err)
				}

				time.Sleep(time.Hour + time.Second)

				if _, err := call(
					t.Context(),
					false,
				); !errors.Is(
					err,
					kernel.ErrSourceUnavailable,
				) {
					t.Errorf("at 24 hours error = %v, want kernel.ErrSourceUnavailable", err)
				}
			})
		})
	}
}
