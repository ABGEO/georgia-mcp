package tbc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of TBC.
	DefaultBaseURL = "https://api.tbcbank.ge"

	vendorName = "tbc"
)

// Client calls the API of TBC. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// WithAPIKey sets the key of the developer account. The client sends it in the
// header apikey only, so no error text holds it. An empty key sends no header.
func WithAPIKey(apiKey string) Option {
	if apiKey == "" {
		return func(*vendorhttp.Client) {}
	}

	return vendorhttp.WithHeader("Apikey", apiKey)
}

// New returns a client of the API of TBC.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CommercialRates returns the current commercial rates of the currencies. TBC answers
// at most 10 currencies, and leaves out a code that it does not quote. With no
// currencies, it answers 10 of its currencies, and which 10 changes over time.
func (c *Client) CommercialRates(
	ctx context.Context,
	currencies []string,
) (*CommercialRates, error) {
	query := url.Values{}
	if len(currencies) > 0 {
		query.Set("currency", strings.Join(currencies, ","))
	}

	var rates CommercialRates
	if err := c.getJSON(
		ctx,
		[]string{"v1", "exchange-rates", "commercial"},
		query,
		&rates,
	); err != nil {
		return nil, fmt.Errorf("getting the commercial rates: %w", err)
	}

	return &rates, nil
}

// getJSON decodes the body of an answer with a 2xx status into target. Any other
// status is an *APIError.
func (c *Client) getJSON(ctx context.Context, path []string, query url.Values, target any) error {
	resp, err := c.http.Do(ctx, vendorhttp.Request{Path: path, Query: query})
	if err != nil {
		return err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(resp.StatusCode, resp.Body)
	}

	if err := json.Unmarshal(resp.Body, target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}
