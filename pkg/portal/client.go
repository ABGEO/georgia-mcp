package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Portal.
	DefaultBaseURL = "https://portal.com.ge"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/portal"

	// priceListKey starts the price list in the data stream of the page.
	priceListKey = `"fuelPricesData":`

	// maxBodyBytes bounds the read of an answer. The stream of the page is about 112 KiB.
	maxBodyBytes = 2 << 20
)

// Client reads the data of the price page of Portal. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the price page of Portal.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/x-component"),
		vendorhttp.WithHeader("Rsc", "1"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// FuelPrices returns the price list of the page, in the order of the page. A list with
// no entry gives an empty list.
func (c *Client) FuelPrices(ctx context.Context) ([]FuelPrice, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"ka", "fuel-prices"}})
	if err != nil {
		return nil, fmt.Errorf("getting the price page: %w", err)
	}

	prices, err := parseFuelPrices(body)
	if err != nil {
		return nil, fmt.Errorf("reading the price list: %w", err)
	}

	return prices, nil
}

// parseFuelPrices decodes the JSON value after the key of the price list. The decoder
// stops at the end of the value, so the rest of the stream needs no parse.
func parseFuelPrices(body []byte) ([]FuelPrice, error) {
	_, list, found := bytes.Cut(body, []byte(priceListKey))
	if !found {
		return nil, ErrNoPriceList
	}

	prices := []FuelPrice{}

	decoder := json.NewDecoder(bytes.NewReader(list))
	if err := decoder.Decode(&prices); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedPriceList, err)
	}

	return prices, nil
}
