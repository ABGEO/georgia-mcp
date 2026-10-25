package mbc

import (
	"context"
	"fmt"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the rate method behind the site of MBC. The
// method listens on port 8022.
const DefaultBaseURL = "https://fxrates.mbc.com.ge:8022"

// Client calls the rate method of the site of MBC. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the rate method of the site of MBC.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CommercialRates returns each item of the commercial rates, in the order of the answer.
func (c *Client) CommercialRates(ctx context.Context) ([]Rate, error) {
	var answer envelope

	if err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Path: []string{"api", "fxrates", "mbc", "commercial"},
	}, &answer); err != nil {
		return nil, fmt.Errorf("getting the commercial rates: %w", err)
	}

	if answer.FXRates == nil {
		return nil, fmt.Errorf("getting the commercial rates: %w", ErrNoRates)
	}

	return answer.FXRates, nil
}
