package terabank

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
	// DefaultBaseURL is the production address of the site of Terabank.
	DefaultBaseURL = "https://terabank.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/terabank"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. The answer of the rates is under 1 KiB.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10
)

// Client calls the rate API of Terabank. It is safe for concurrent use.
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
// georgia-mcp-terabank/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the rate API of Terabank.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-terabank/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// CrossRates returns the current rate of each currency against the lari. An answer
// with succeeded false is ErrNotSucceeded. Terabank publishes no limit of the calls.
func (c *Client) CrossRates(ctx context.Context) ([]CrossRate, error) {
	body, err := c.post(ctx, "_mvcapi", "CurrencyRatesApi", "GetTeraCrossRates")
	if err != nil {
		return nil, fmt.Errorf("getting the cross rates: %w", err)
	}

	var answer envelope[[]CrossRate]
	if err := json.Unmarshal(body, &answer); err != nil {
		return nil, fmt.Errorf("decoding the cross rates: %w", err)
	}

	if !answer.Succeeded {
		return nil, fmt.Errorf("getting the cross rates: %w: %q", ErrNotSucceeded, answer.Message)
	}

	return answer.Data, nil
}

// post sends a POST with an empty body to the path under the base address, and returns
// the body of a 200. Any other status is an *APIError.
func (c *Client) post(ctx context.Context, path ...string) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, path...)
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	// The site answers 411 to a POST with no Content-Length. net/http sends
	// Content-Length: 0 for a POST with http.NoBody.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
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
