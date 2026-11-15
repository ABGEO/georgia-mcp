package nager

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of Nager.Date.
const DefaultBaseURL = "https://date.nager.at"

// Client reads the public holidays of Nager.Date. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of Nager.Date.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// PublicHolidays returns the public holidays of one country in one year, in the order of
// the answer. countryCode is ISO 3166-1 alpha-2: "GE".
func (c *Client) PublicHolidays(
	ctx context.Context,
	year int,
	countryCode string,
) ([]PublicHoliday, error) {
	var holidays []PublicHoliday

	request := vendorhttp.Request{
		Path: []string{"api", "v3", "PublicHolidays", strconv.Itoa(year), countryCode},
	}
	if err := c.http.FetchJSON(ctx, request, &holidays); err != nil {
		return nil, fmt.Errorf("reading the public holidays: %w", err)
	}

	return holidays, nil
}
