package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// bookTTL bounds the age of a book. GNERC changes its workbooks once a month.
	bookTTL = 6 * time.Hour
	// bookMaxAge bounds the age of a stale book.
	bookMaxAge = 7 * 24 * time.Hour

	bookKey = "book"
)

// BookCache implements app.BalanceSource and app.PriceSource in front of other sources.
// Each source keeps its whole book under one key.
type BookCache struct {
	balanceSource app.BalanceSource
	priceSource   app.PriceSource
	balance       *cache.Cache[app.BalanceBook]
	prices        *cache.Cache[app.PriceBook]
}

var (
	_ app.BalanceSource = (*BookCache)(nil)
	_ app.PriceSource   = (*BookCache)(nil)
)

// NewBookCache returns a cache in front of the sources.
func NewBookCache(balance app.BalanceSource, prices app.PriceSource) (*BookCache, error) {
	balanceStore, err := newBookStore[app.BalanceBook]()
	if err != nil {
		return nil, err
	}

	priceStore, err := newBookStore[app.PriceBook]()
	if err != nil {
		balanceStore.Close()

		return nil, err
	}

	return &BookCache{
		balanceSource: balance, priceSource: prices,
		balance: balanceStore, prices: priceStore,
	}, nil
}

func newBookStore[V any]() (*cache.Cache[V], error) {
	store, err := cache.New[V](
		cache.Config{TTL: bookTTL, MaxStale: bookMaxAge - bookTTL, MaxEntries: 1},
	)
	if err != nil {
		return nil, fmt.Errorf("building a store of GNERC: %w", err)
	}

	return store, nil
}

// Balance returns the cached balance, or reads it.
func (c *BookCache) Balance(ctx context.Context) (app.BalanceBook, error) {
	result, err := c.balance.Get(ctx, bookKey, c.balanceSource.Balance)
	if err != nil {
		return app.BalanceBook{}, fmt.Errorf("reading the balance: %w", err)
	}

	book := result.Value
	book.Stale = result.Stale

	return book, nil
}

// Prices returns the cached prices, or reads them.
func (c *BookCache) Prices(ctx context.Context) (app.PriceBook, error) {
	result, err := c.prices.Get(ctx, bookKey, c.priceSource.Prices)
	if err != nil {
		return app.PriceBook{}, fmt.Errorf("reading the prices: %w", err)
	}

	book := result.Value
	book.Stale = result.Stale

	return book, nil
}

// Close stops the goroutines of the stores.
func (c *BookCache) Close() error {
	c.balance.Close()
	c.prices.Close()

	return nil
}
