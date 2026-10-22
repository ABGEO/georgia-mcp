package tbcoffers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the marketing API of TBC Bank.
	DefaultBaseURL = "https://apigw.tbcbank.ge/api/v1/marketing"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/tbcoffers"

	// maxBodyBytes bounds the read of an answer. The list of 518 offers is under 1 MiB.
	maxBodyBytes = 8 << 20

	entryType = "offer"
)

// Client calls the marketing API of TBC Bank. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the marketing API of TBC Bank.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Filters returns the filters of the offers of one segment, in a locale.
func (c *Client) Filters(ctx context.Context, segment, locale string) ([]Filter, error) {
	var filters []Filter

	err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Path:  []string{"filters"},
		Query: url.Values{"type": {entryType}, "locale": {locale}, "segment": {segment}},
	}, &filters)
	if err != nil {
		return nil, fmt.Errorf("getting the filters of %q: %w", segment, err)
	}

	return filters, nil
}

// Offers returns one page of the offers that match the request.
func (c *Client) Offers(ctx context.Context, request OffersRequest) (OffersPage, error) {
	if request.Filter == nil {
		request.Filter = []string{}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return OffersPage{}, fmt.Errorf("encoding the request: %w", err)
	}

	var page OffersPage

	err = c.http.FetchJSON(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"entries", entryType},
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   body,
	}, &page)
	if err != nil {
		return OffersPage{}, fmt.Errorf("getting page %d of the offers: %w", request.PageIndex, err)
	}

	return page, nil
}

// Offer returns the page of one offer, with its terms, in a locale.
func (c *Client) Offer(ctx context.Context, slug, locale string) (Entry, error) {
	var entry Entry

	err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Path: []string{"entries", entryType, url.PathEscape(slug), locale},
	}, &entry)
	if err != nil {
		return Entry{}, fmt.Errorf("getting the offer %q: %w", slug, err)
	}

	return entry, nil
}
