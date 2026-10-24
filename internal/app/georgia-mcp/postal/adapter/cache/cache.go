// Package cache keeps the answers of Georgian Post in memory, in front of the port of
// the context.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// addressTTL bounds the age of a list. Georgian Post changes a postal code rarely.
	addressTTL = 7 * 24 * time.Hour
	// maxAge bounds the age of a stale list.
	maxAge         = 30 * 24 * time.Hour
	addressEntries = 4096
	keySeparator   = "|"
)

// AddressCache implements app.PostalCodeSource in front of another source.
type AddressCache struct {
	source    app.PostalCodeSource
	addresses *cache.Cache[app.AddressList]
}

var _ app.PostalCodeSource = (*AddressCache)(nil)

// NewAddressCache returns a cache in front of the source.
func NewAddressCache(source app.PostalCodeSource) (*AddressCache, error) {
	addresses, err := cache.New[app.AddressList](cache.Config{
		TTL: addressTTL, MaxStale: maxAge - addressTTL, MaxEntries: addressEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of the addresses: %w", err)
	}

	return &AddressCache{source: source, addresses: addresses}, nil
}

// Addresses returns the cached addresses of the text, or reads them. The use case
// normalizes the text, so the key keeps it as it comes.
func (c *AddressCache) Addresses(
	ctx context.Context,
	text string,
	language kernel.Language,
) (app.AddressList, error) {
	key := text + keySeparator + string(language)

	result, err := c.addresses.Get(ctx, key, func(ctx context.Context) (app.AddressList, error) {
		return c.source.Addresses(ctx, text, language)
	})
	if err != nil {
		return app.AddressList{}, fmt.Errorf("searching the postal codes: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the store.
func (c *AddressCache) Close() error {
	c.addresses.Close()

	return nil
}
