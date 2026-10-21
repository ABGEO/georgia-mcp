package kursi

import (
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
	// DefaultBaseURL is the production address of the rate API of Kursi.ge.
	DefaultBaseURL = "https://api-core.kursi.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/kursi"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. The answer of the rates is about 2 KiB.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10
)

// Client calls the rate API of Kursi.ge. It is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
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
// georgia-mcp-kursi/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the rate API of Kursi.ge.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-kursi/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Currencies returns every pair of currencies that Kursi.ge quotes. Kursi.ge publishes
// no limit of the calls.
func (c *Client) Currencies(ctx context.Context) ([]Currency, error) {
	body, err := c.get(ctx, "api", "public", "currencies")
	if err != nil {
		return nil, fmt.Errorf("getting the currencies: %w", err)
	}

	var pairs []Currency
	if err := json.Unmarshal(body, &pairs); err != nil {
		return nil, fmt.Errorf("decoding the currencies: %w", err)
	}

	return pairs, nil
}

// get sends a GET to the path under the base address, and returns the body of a 200.
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

	req.Header.Set("Accept", "application/json")
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
