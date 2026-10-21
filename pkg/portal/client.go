package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"
)

const (
	// DefaultBaseURL is the production address of the site of Portal.
	DefaultBaseURL = "https://portal.com.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/portal"
	defaultTimeout = 10 * time.Second

	// priceListKey starts the price list in the data stream of the page.
	priceListKey = `"fuelPricesData":`

	// maxBodyBytes bounds the read of an answer. The stream of the page is about 112 KiB.
	maxBodyBytes = 2 << 20
	maxErrorBody = 1 << 10
)

// Client reads the data of the price page of Portal. It is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// Option sets one value of a Client.
type Option func(*Client)

// WithBaseURL sets the address of the site. The default is DefaultBaseURL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client. The default has a timeout of 10 seconds.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent sets the User-Agent header. The default is
// georgia-mcp-portal/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the price page of Portal.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-portal/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// FuelPrices returns the price list of the page, in the order of the page. A list with
// no entry gives an empty list.
func (c *Client) FuelPrices(ctx context.Context) ([]FuelPrice, error) {
	body, err := c.get(ctx, "ka", "fuel-prices")
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

// get sends a GET to the path under the base address and returns the body of a 200.
// Any other status is an *APIError.
func (c *Client) get(ctx context.Context, path ...string) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, path...)
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", "text/x-component")
	req.Header.Set("Rsc", "1")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading the answer: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body[:min(len(body), maxErrorBody)]),
		}
	}

	return body, nil
}

func version() string {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if build.Main.Path == modulePath && build.Main.Version != "" {
		return build.Main.Version
	}

	for _, dep := range build.Deps {
		if dep.Path == modulePath {
			return dep.Version
		}
	}

	return "dev"
}
