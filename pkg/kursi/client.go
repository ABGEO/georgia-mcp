package kursi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the rate API of Kursi.ge.
	DefaultBaseURL = "https://api-core.kursi.ge"
)

// Client calls the rate API of Kursi.ge. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the rate API of Kursi.ge.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Currencies returns every pair of currencies that Kursi.ge quotes. Kursi.ge publishes
// no limit of the calls.
func (c *Client) Currencies(ctx context.Context) ([]Currency, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{Path: []string{"api", "public", "currencies"}})
	if err != nil {
		return nil, fmt.Errorf("getting the currencies: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"getting the currencies: %w",
			&APIError{StatusCode: resp.StatusCode, Body: resp.ErrorBody()},
		)
	}

	var pairs []Currency
	if err := json.Unmarshal(resp.Body, &pairs); err != nil {
		return nil, fmt.Errorf("decoding the currencies: %w", err)
	}

	return pairs, nil
}
