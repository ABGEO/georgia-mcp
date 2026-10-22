package connect

import (
	"context"
	"fmt"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the price API of Connect.
	DefaultBaseURL = "https://connect.com.ge/wp-json/fuel-prices/v1"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/connect"
)

// Client calls the price API of Connect. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the price API of Connect.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Current returns the snapshot of the current prices: every product of Connect with its
// standard price and its self-service price. One call returns every product.
func (c *Client) Current(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot

	err := c.http.FetchJSON(ctx, vendorhttp.Request{Path: []string{"current"}}, &snapshot)
	if err != nil {
		return Snapshot{}, fmt.Errorf("getting the current prices: %w", err)
	}

	return snapshot, nil
}
