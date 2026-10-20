// Package cache keeps the catalog and the terms of each bank in memory, in front of its port.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// ttl bounds the delay of a new offer. A bank adds offers a few times in a week.
	ttl = time.Hour
	// maxStale keeps a catalog or terms for 24 hours after the time to live, while the
	// bank fails.
	maxStale = 24 * time.Hour
	// catalogEntries holds one catalog for each language.
	catalogEntries = 2
	termsEntries   = 500
)

// OfferCache implements app.OfferSource in front of another source.
type OfferCache struct {
	source   app.OfferSource
	catalogs *cache.Cache[app.Catalog]
	terms    *cache.Cache[app.Terms]
}

var _ app.OfferSource = (*OfferCache)(nil)

// New returns a cache in front of the source.
func New(source app.OfferSource) (*OfferCache, error) {
	catalogs, err := cache.New[app.Catalog](
		cache.Config{TTL: ttl, MaxStale: maxStale, MaxEntries: catalogEntries},
	)
	if err != nil {
		return nil, fmt.Errorf("building the store of the catalogs: %w", err)
	}

	terms, err := cache.New[app.Terms](
		cache.Config{TTL: ttl, MaxStale: maxStale, MaxEntries: termsEntries},
	)
	if err != nil {
		catalogs.Close()

		return nil, fmt.Errorf("building the store of the terms: %w", err)
	}

	return &OfferCache{source: source, catalogs: catalogs, terms: terms}, nil
}

// Bank returns the bank of the source.
func (c *OfferCache) Bank() domain.Bank {
	return c.source.Bank()
}

// Cards returns the cards of the source.
func (c *OfferCache) Cards() []domain.Card {
	return c.source.Cards()
}

// Offers returns the cached catalog of the language, or reads it. ReadAt is the read
// instant that the source gave.
func (c *OfferCache) Offers(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	result, err := c.catalogs.Get(
		ctx,
		string(language),
		func(ctx context.Context) (app.Catalog, error) {
			return c.source.Offers(ctx, language)
		},
	)
	if err != nil {
		return app.Catalog{}, fmt.Errorf("reading the offers of %s: %w", c.source.Bank().Name, err)
	}

	catalog := result.Value
	catalog.Stale = result.Stale

	return catalog, nil
}

// Terms returns the cached terms of the offer in the language, or reads them.
func (c *OfferCache) Terms(
	ctx context.Context,
	offer domain.Offer,
	language kernel.Language,
) (app.Terms, error) {
	key := offer.ID + "/" + string(language)

	result, err := c.terms.Get(ctx, key, func(ctx context.Context) (app.Terms, error) {
		return c.source.Terms(ctx, offer, language)
	})
	if err != nil {
		return app.Terms{}, fmt.Errorf(
			"reading the terms of %s %q: %w", c.source.Bank().Name, offer.ID, err,
		)
	}

	terms := result.Value
	terms.Stale = result.Stale

	return terms, nil
}

// Close stops the goroutines of the stores.
func (c *OfferCache) Close() error {
	c.catalogs.Close()
	c.terms.Close()

	return nil
}
