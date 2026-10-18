package tbc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the production address of the API of TBC.
	DefaultBaseURL = "https://api.tbcbank.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/tbc"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. An answer of 10 rates is under 1 KiB.
	maxBodyBytes = 1 << 20
)

// Client calls the API of TBC. It is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	apiKey     string
}

// Option sets one value of a Client.
type Option func(*Client)

// WithBaseURL sets the address of the API. The default is DefaultBaseURL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client. The default has a timeout of 10 seconds.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent sets the User-Agent header. The default is
// georgia-mcp-tbc/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// WithAPIKey sets the key of the developer account. The client sends it in the
// header apikey only, so no error text holds it.
func WithAPIKey(apiKey string) Option {
	return func(c *Client) { c.apiKey = apiKey }
}

// New returns a client of the API of TBC.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-tbc/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// CommercialRates returns the current commercial rates of the currencies. TBC answers
// at most 10 currencies, and leaves out a code that it does not quote. With no
// currencies, it answers 10 of its currencies, and which 10 changes over time.
func (c *Client) CommercialRates(
	ctx context.Context,
	currencies []string,
) (*CommercialRates, error) {
	query := url.Values{}
	if len(currencies) > 0 {
		query.Set("currency", strings.Join(currencies, ","))
	}

	var rates CommercialRates
	if err := c.getJSON(
		ctx,
		[]string{"v1", "exchange-rates", "commercial"},
		query,
		&rates,
	); err != nil {
		return nil, fmt.Errorf("getting the commercial rates: %w", err)
	}

	return &rates, nil
}

func (c *Client) getJSON(ctx context.Context, path []string, query url.Values, target any) error {
	body, err := c.get(ctx, path, query)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}

// get sends a GET to the path under the base address and returns the body of a
// successful answer. Any other status is an *APIError.
func (c *Client) get(ctx context.Context, path []string, query url.Values) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, path...)
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if c.apiKey != "" {
		req.Header.Set("Apikey", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading the answer: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newAPIError(resp.StatusCode, body)
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
