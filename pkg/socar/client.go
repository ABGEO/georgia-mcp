package socar

import (
	"context"
	"fmt"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of SOCAR.
	DefaultBaseURL = "https://sgp.ge/sgp-backend/api"

	vendorName = "socar"
)

// Client calls the API of SOCAR. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the API of SOCAR.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CurrentPrices returns the list under GetCurrentPrices: every product of SOCAR with its
// current price, and the status of the answer. One call returns every product.
func (c *Client) CurrentPrices(ctx context.Context) (PriceList, error) {
	var answer currentPricesResponse

	err := c.http.FetchJSON(
		ctx,
		vendorhttp.Request{Path: []string{"integration", "info", "current-prices"}},
		&answer,
	)
	if err != nil {
		return PriceList{}, fmt.Errorf("getting the current prices: %w", err)
	}

	return answer.GetCurrentPrices, nil
}
