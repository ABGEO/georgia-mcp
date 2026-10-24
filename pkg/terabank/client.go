package terabank

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Terabank.
	DefaultBaseURL = "https://terabank.ge"
)

// Client calls the rate API of Terabank. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the rate API of Terabank.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CrossRates returns the current rate of each currency against the lari. An answer
// with succeeded false is ErrNotSucceeded. Terabank publishes no limit of the calls.
func (c *Client) CrossRates(ctx context.Context) ([]CrossRate, error) {
	// The site answers 411 to a POST with no Content-Length. A request with no body sends
	// Content-Length: 0.
	resp, err := c.http.Do(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"_mvcapi", "CurrencyRatesApi", "GetTeraCrossRates"},
	})
	if err != nil {
		return nil, fmt.Errorf("getting the cross rates: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"getting the cross rates: %w",
			&APIError{StatusCode: resp.StatusCode, Body: resp.ErrorBody()},
		)
	}

	var answer envelope[[]CrossRate]
	if err := json.Unmarshal(resp.Body, &answer); err != nil {
		return nil, fmt.Errorf("decoding the cross rates: %w", err)
	}

	if !answer.Succeeded {
		return nil, fmt.Errorf("getting the cross rates: %w: %q", ErrNotSucceeded, answer.Message)
	}

	return answer.Data, nil
}
