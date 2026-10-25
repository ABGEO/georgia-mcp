package credo

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API behind the site of Credo Bank.
	DefaultBaseURL = "https://n1.noxtton.com"

	// MethodCommercial is the kind of rate of the branches of Credo Bank.
	MethodCommercial = "FxRatesCommercial"

	dateLayout = "2006-01-02T15:04:05.000Z"
)

// Client calls the rate method of the site of Credo Bank. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// WithAPIKey sets the key of the site. The client sends it in the header x-api-key only,
// so no error text holds it. An empty key sends no header.
func WithAPIKey(apiKey string) Option {
	if apiKey == "" {
		return func(*vendorhttp.Client) {}
	}

	return vendorhttp.WithHeader("X-Api-Key", apiKey)
}

// New returns a client of the rate method of the site of Credo Bank.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CurrentRates returns the rates of the kind that method names, at the instant at.
func (c *Client) CurrentRates(ctx context.Context, method string, at time.Time) ([]Rate, error) {
	var answer envelope

	if err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Path: []string{"v2", "credo", "current-rates"},
		Query: url.Values{
			"method": {method},
			"date":   {at.UTC().Format(dateLayout)},
		},
	}, &answer); err != nil {
		return nil, fmt.Errorf("getting the rates of %s: %w", method, err)
	}

	if !answer.Success {
		return nil, fmt.Errorf("getting the rates of %s: %w", method, ErrNotSucceeded)
	}

	return answer.Rates, nil
}
