package bog

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
	// DefaultBaseURL is the production address of the API of Bank of Georgia.
	DefaultBaseURL = "https://api.businessonline.ge"

	vendorName = "bog"
)

// Client calls the API of Bank of Georgia. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the API of Bank of Georgia.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CommercialRate returns the current commercial rate of one currency, an ISO 4217
// code. It returns ErrNoRate when the bank has no rate of the code. Bank of Georgia
// allows 60 calls in a minute, and answers 429 past the limit.
func (c *Client) CommercialRate(ctx context.Context, currency string) (*CommercialRate, error) {
	body, err := c.get(ctx, "api", "rates", "commercial", url.PathEscape(currency))
	if err != nil {
		return nil, fmt.Errorf("getting the commercial rate of %q: %w", currency, err)
	}

	var rate CommercialRate
	if err := json.Unmarshal(body, &rate); err != nil {
		return nil, fmt.Errorf("decoding the commercial rate of %q: %w", currency, err)
	}

	return &rate, nil
}

// get returns the body of a 200. A 204 is ErrNoRate. Any other status is an *APIError.
// Each element of the path is already escaped.
func (c *Client) get(ctx context.Context, path ...string) ([]byte, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{Path: path})
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusNoContent:
		return nil, ErrNoRate
	default:
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			RetryAfter: resp.Header.Get("Retry-After"),
			Body:       resp.ErrorBody(),
		}
	}
}
