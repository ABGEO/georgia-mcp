package bogoffers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the offers API of Bank of Georgia.
	DefaultBaseURL = "https://bankofgeorgia.ge/api/bog-b/offers-hub"

	vendorName = "bogoffers"

	// maxBodyBytes bounds the read of an answer. A page of 100 offers in Georgian is
	// under 300 KiB.
	maxBodyBytes = 4 << 20
)

// Client calls the offers API of Bank of Georgia. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the offers API of Bank of Georgia.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Offers returns one page of the offers of Bank of Georgia. A status other than 200 is
// an *APIError. A page with a Code other than 0 is not an error of the client.
func (c *Client) Offers(ctx context.Context, offersQuery OffersQuery) (OffersPage, error) {
	query := url.Values{}
	query.Set("pageInfo", "true")
	query.Set("pageNumber", strconv.Itoa(offersQuery.PageNumber))
	query.Set("pageSize", strconv.Itoa(offersQuery.PageSize))

	header := http.Header{}
	header.Set("Content-Type", "application/json")

	if offersQuery.Language != "" {
		header.Set("Accept-Language", offersQuery.Language)
	}

	var page OffersPage

	err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"get-offers"},
		Query:  query,
		Header: header,
		Body:   []byte("{}"),
	}, &page)
	if err != nil {
		return OffersPage{}, fmt.Errorf(
			"getting page %d of the offers: %w", offersQuery.PageNumber, err,
		)
	}

	return page, nil
}
