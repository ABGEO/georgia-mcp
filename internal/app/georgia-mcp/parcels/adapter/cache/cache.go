// Package cache keeps the parcels and the addresses in memory, in front of the port of the
// context.
package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// A sale changes the owners on the day of its registration. A parcel is stale for at
	// most 7 days in all.
	parcelTTL     = time.Hour
	parcelMaxAge  = 7 * 24 * time.Hour
	parcelEntries = 1000

	nearTTL     = 24 * time.Hour
	nearEntries = 500

	addressTTL     = 7 * 24 * time.Hour
	matchEntries   = 500
	centreEntries  = 2000
	pointDecimals  = 5
	coordinateBits = 64
)

// SourceCache implements app.ParcelSource in front of another source.
type SourceCache struct {
	source  app.ParcelSource
	parcels *cache.Cache[app.ParcelRead]
	near    *cache.Cache[app.NearbyRead]
	matches *cache.Cache[app.MatchRead]
	centres *cache.Cache[app.CentreRead]
}

var (
	_ app.ParcelSource = (*SourceCache)(nil)
	_ io.Closer        = (*SourceCache)(nil)
)

// NewSourceCache returns a cache in front of the source.
func NewSourceCache(source app.ParcelSource) (*SourceCache, error) {
	parcels, parcelsErr := cache.New[app.ParcelRead](cache.Config{
		TTL: parcelTTL, MaxStale: parcelMaxAge - parcelTTL, MaxEntries: parcelEntries,
	})
	near, nearErr := cache.New[app.NearbyRead](cache.Config{TTL: nearTTL, MaxEntries: nearEntries})
	matches, matchesErr := cache.New[app.MatchRead](
		cache.Config{TTL: addressTTL, MaxEntries: matchEntries},
	)
	centres, centresErr := cache.New[app.CentreRead](
		cache.Config{TTL: addressTTL, MaxEntries: centreEntries},
	)

	if err := errors.Join(parcelsErr, nearErr, matchesErr, centresErr); err != nil {
		for _, built := range []interface{ Close() }{parcels, near, matches, centres} {
			if built != nil {
				built.Close()
			}
		}

		return nil, fmt.Errorf("building the stores of the parcels: %w", err)
	}

	return &SourceCache{
		source:  source,
		parcels: parcels,
		near:    near,
		matches: matches,
		centres: centres,
	}, nil
}

// Parcel returns the cached parcel, or reads it. A missing parcel is not kept.
func (c *SourceCache) Parcel(
	ctx context.Context,
	code domain.CadastralCode,
) (app.ParcelRead, error) {
	result, err := c.parcels.Get(
		ctx,
		code.String(),
		func(ctx context.Context) (app.ParcelRead, error) {
			return c.source.Parcel(ctx, code)
		},
	)
	if err != nil {
		return app.ParcelRead{}, fmt.Errorf("reading the parcel: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// ParcelsNear returns the cached parcels near the point, or reads them.
func (c *SourceCache) ParcelsNear(ctx context.Context, point domain.Point) (app.NearbyRead, error) {
	key := strconv.FormatFloat(point.Lat, 'f', pointDecimals, coordinateBits) + "," +
		strconv.FormatFloat(point.Lon, 'f', pointDecimals, coordinateBits)

	result, err := c.near.Get(ctx, key, func(ctx context.Context) (app.NearbyRead, error) {
		return c.source.ParcelsNear(ctx, point)
	})
	if err != nil {
		return app.NearbyRead{}, fmt.Errorf("reading the parcels near a point: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// MatchAddresses returns the cached addresses of the text, or reads them.
func (c *SourceCache) MatchAddresses(ctx context.Context, text string) (app.MatchRead, error) {
	result, err := c.matches.Get(
		ctx,
		domain.CollapseSpaces(text),
		func(ctx context.Context) (app.MatchRead, error) {
			return c.source.MatchAddresses(ctx, text)
		},
	)
	if err != nil {
		return app.MatchRead{}, fmt.Errorf("searching the addresses: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// AddressCentre returns the cached centre of the address, or reads it.
func (c *SourceCache) AddressCentre(
	ctx context.Context,
	ref app.AddressRef,
) (app.CentreRead, error) {
	result, err := c.centres.Get(
		ctx,
		string(ref),
		func(ctx context.Context) (app.CentreRead, error) {
			return c.source.AddressCentre(ctx, ref)
		},
	)
	if err != nil {
		return app.CentreRead{}, fmt.Errorf("reading the centre of an address: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// Close stops the goroutines of the stores.
func (c *SourceCache) Close() error {
	c.parcels.Close()
	c.near.Close()
	c.matches.Close()
	c.centres.Close()

	return nil
}
