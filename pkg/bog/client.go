package bog

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
	// DefaultBaseURL is the production address of the API of Bank of Georgia.
	DefaultBaseURL = "https://api.businessonline.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/bog"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. An answer of one rate is under 64 bytes.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10
)

// Client calls the API of Bank of Georgia. It is safe for concurrent use.
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
// georgia-mcp-bog/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the API of Bank of Georgia.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-bog/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
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

// get sends a GET to the path under the base address and returns the body of a 200.
// A 204 is ErrNoRate. Any other status is an *APIError. Each element of the path is
// already escaped.
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

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNoContent:
		return nil, ErrNoRate
	default:
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			RetryAfter: resp.Header.Get("Retry-After"),
			Body:       string(body[:min(len(body), maxErrorBody)]),
		}
	}
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
