package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const day = 24 * time.Hour

// countingSource counts the calls of each method, fails with an outage once failing is
// set, and has no parcel once missing is set.
type countingSource struct {
	parcel, near, match, centre atomic.Int32
	failing, missing            atomic.Bool
}

func (c *countingSource) Parcel(
	_ context.Context,
	code domain.CadastralCode,
) (app.ParcelRead, error) {
	c.parcel.Add(1)

	if c.missing.Load() {
		return app.ParcelRead{}, app.ErrParcelNotFound
	}

	return app.ParcelRead{Parcel: domain.Parcel{Code: code}, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) ParcelsNear(context.Context, domain.Point) (app.NearbyRead, error) {
	c.near.Add(1)

	return app.NearbyRead{ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) MatchAddresses(context.Context, string) (app.MatchRead, error) {
	c.match.Add(1)

	return app.MatchRead{ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) AddressCentre(context.Context, app.AddressRef) (app.CentreRead, error) {
	c.centre.Add(1)

	return app.CentreRead{ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) fail() error {
	if c.failing.Load() {
		return fmt.Errorf("calling NAPR: %w", kernel.ErrSourceUnavailable)
	}

	return nil
}

func newCache(t *testing.T, source app.ParcelSource) *cache.SourceCache {
	t.Helper()

	sources, err := cache.NewSourceCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	return sources
}

func freedom(t *testing.T) domain.CadastralCode {
	t.Helper()

	code, err := domain.ParseCadastralCode("01.15.05.066.020")
	if err != nil {
		t.Fatal(err)
	}

	return code
}

// readAll reads each method once with the same arguments.
func readAll(t *testing.T, sources *cache.SourceCache) {
	t.Helper()

	ctx := t.Context()
	point := domain.Point{Lat: 41.6934, Lon: 44.8015}

	if _, err := sources.Parcel(ctx, freedom(t)); err != nil {
		t.Fatal(err)
	}

	if _, err := sources.ParcelsNear(ctx, point); err != nil {
		t.Fatal(err)
	}

	if _, err := sources.MatchAddresses(ctx, " ჭავჭავაძის  გამზირი 44 "); err != nil {
		t.Fatal(err)
	}

	if _, err := sources.AddressCentre(ctx, "ract_g_borders:A"); err != nil {
		t.Fatal(err)
	}
}

func counts(source *countingSource) [4]int32 {
	return [4]int32{
		source.parcel.Load(),
		source.near.Load(),
		source.match.Load(),
		source.centre.Load(),
	}
}

// NAPRPARC-SC-020: A parcel stays for 1 hour, the parcels near a point for 24 hours, and
// an address search and a centre for 7 days.
func TestSourceCacheTimesToLive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)

		steps := []struct {
			sleep time.Duration
			want  [4]int32
		}{
			{0, [4]int32{1, 1, 1, 1}},
			{59 * time.Minute, [4]int32{1, 1, 1, 1}},
			{time.Minute, [4]int32{2, 1, 1, 1}},
			{23*time.Hour - time.Minute, [4]int32{3, 1, 1, 1}},
			{time.Minute, [4]int32{3, 2, 1, 1}},
			{6 * day, [4]int32{4, 3, 2, 2}},
		}

		for _, step := range steps {
			time.Sleep(step.sleep)
			readAll(t, sources)

			if got := counts(source); got != step.want {
				t.Fatalf("after %v: calls = %v, want %v", step.sleep, got, step.want)
			}
		}
	})
}

// NAPRPARC-SC-020: The search for a text trims it and joins its spaces, and the point
// rounds to 5 decimals.
func TestSourceCacheKeys(t *testing.T) {
	t.Parallel()

	source := &countingSource{}
	sources := newCache(t, source)
	ctx := t.Context()

	for _, text := range []string{"ჭავჭავაძის გამზირი 44", " ჭავჭავაძის   გამზირი 44"} {
		if _, err := sources.MatchAddresses(ctx, text); err != nil {
			t.Fatal(err)
		}
	}

	for _, point := range []domain.Point{{Lat: 41.693401, Lon: 44.801501}, {Lat: 41.693402, Lon: 44.801502}} {
		if _, err := sources.ParcelsNear(ctx, point); err != nil {
			t.Fatal(err)
		}
	}

	if source.match.Load() != 1 || source.near.Load() != 1 {
		t.Errorf(
			"match, near = %d, %d calls, want 1 and 1",
			source.match.Load(),
			source.near.Load(),
		)
	}
}

// NAPRPARC-SC-021: While NAPR fails, a parcel stays stale until it is 7 days old. The
// parcels near a point are never stale.
func TestSourceCacheStale(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)
		readAll(t, sources)

		source.failing.Store(true)
		time.Sleep(6*day + 23*time.Hour)

		read, err := sources.Parcel(t.Context(), freedom(t))
		if err != nil || !read.Stale {
			t.Errorf(
				"Parcel() at 6 days 23 hours = stale %t, %v, want the stale parcel",
				read.Stale,
				err,
			)
		}

		_, err = sources.ParcelsNear(t.Context(), domain.Point{Lat: 41.6934, Lon: 44.8015})
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("ParcelsNear() after 24 hours error = %v, want ErrSourceUnavailable", err)
		}

		time.Sleep(time.Hour + time.Second)

		if _, err := sources.Parcel(
			t.Context(),
			freedom(t),
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf("Parcel() at 7 days error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// NAPRPARC-SC-020: A missing parcel is not kept.
func TestSourceCacheKeepsNoMissingParcel(t *testing.T) {
	t.Parallel()

	source := &countingSource{}
	source.missing.Store(true)
	sources := newCache(t, source)

	for range 2 {
		if _, err := sources.Parcel(
			t.Context(),
			freedom(t),
		); !errors.Is(
			err,
			app.ErrParcelNotFound,
		) {
			t.Fatalf("Parcel() error = %v, want ErrParcelNotFound", err)
		}
	}

	if source.parcel.Load() != 2 {
		t.Errorf("calls = %d, want 2", source.parcel.Load())
	}
}
