package wissol

import (
	"context"
	"fmt"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of Wissol.
	DefaultBaseURL = "https://api.wissol.ge"
)

// Client calls the API of Wissol. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the API of Wissol.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// FuelPrices returns every product of Wissol with its current prices, the products
// that are not active included. One call returns every product.
func (c *Client) FuelPrices(ctx context.Context) ([]FuelPrice, error) {
	var prices []FuelPrice

	err := c.http.FetchJSON(ctx, vendorhttp.Request{Path: []string{"FuelPrice"}}, &prices)
	if err != nil {
		return nil, fmt.Errorf("getting the fuel prices: %w", err)
	}

	return prices, nil
}
