package nea

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the map of the NEA.
const DefaultBaseURL = "https://air.gov.ge"

const every = "all"

// Client reads the data of the map of the NEA. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the map of the NEA.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Hourly returns the stations that match the query, each with the hourly values of its
// substances from From to To. An unknown station code gives an empty list.
func (c *Client) Hourly(ctx context.Context, query HourlyQuery) ([]Station, error) {
	values := url.Values{
		"from_date_time":  {query.From},
		"to_date_time":    {query.To},
		"station_code":    {cmp.Or(query.Station, every)},
		"municipality_id": {cmp.Or(query.Municipality, every)},
		"substance":       {cmp.Or(query.Substance, every)},
		"last_data":       {"false"},
		"chart":           {"false"},
		"format":          {"json"},
	}

	var stations []Station
	if err := c.get(ctx, "get_data_1hour/", values, &stations); err != nil {
		return nil, fmt.Errorf("reading the hourly values: %w", err)
	}

	return stations, nil
}

// Municipalities returns every municipality that the map knows.
func (c *Client) Municipalities(ctx context.Context) ([]Municipality, error) {
	var municipalities []Municipality
	if err := c.get(ctx, "municipalities/", nil, &municipalities); err != nil {
		return nil, fmt.Errorf("reading the municipalities: %w", err)
	}

	return municipalities, nil
}

// get ends each path with a slash: the NEA redirects a path without it.
func (c *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	answer, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"api", path}, Query: query})
	if err != nil {
		return err
	}

	if err := json.Unmarshal(answer, target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}
