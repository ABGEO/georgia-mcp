package basisbank

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of the site of Basisbank.
	DefaultBaseURL = "https://static.bb.ge/api/view"

	maxBody = 64 << 10
)

// Client calls the rate method of the site of Basisbank. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the rate method of the site of Basisbank.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithMaxBody(maxBody),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Rates returns the rates of each date of the answer, in the order of the answer: two
// dates on 2026-10-25, the newest first. The method answers without the token that the
// page sends.
func (c *Client) Rates(ctx context.Context) ([]Day, error) {
	var wire []wireDay

	if err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"bb", "websiteXrates"},
	}, &wire); err != nil {
		return nil, fmt.Errorf("getting the rates: %w", err)
	}

	days := make([]Day, 0, len(wire))

	for _, day := range wire {
		var rates XRates
		if err := json.Unmarshal([]byte(day.XRates), &rates); err != nil {
			return nil, fmt.Errorf("%w: the rates of %s: %w", ErrMalformedRates, day.Date, err)
		}

		days = append(days, Day{Date: day.Date, XRates: rates})
	}

	return days, nil
}
