package bogoffers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	// DefaultBaseURL is the production address of the offers API of Bank of Georgia.
	DefaultBaseURL = "https://bankofgeorgia.ge/api/bog-b/offers-hub"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/bogoffers"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. A page of 100 offers in Georgian is
	// under 300 KiB.
	maxBodyBytes = 4 << 20
	maxErrorBody = 1 << 10
)

// Client calls the offers API of Bank of Georgia. It is safe for concurrent use.
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
// georgia-mcp-bogoffers/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the offers API of Bank of Georgia.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-bogoffers/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Offers returns one page of the offers of Bank of Georgia. A status other than 200 is
// an *APIError. A page with a Code other than 0 is not an error of the client.
func (c *Client) Offers(ctx context.Context, offersQuery OffersQuery) (OffersPage, error) {
	endpoint, err := url.JoinPath(c.baseURL, "get-offers")
	if err != nil {
		return OffersPage{}, fmt.Errorf("building the address: %w", err)
	}

	values := url.Values{}
	values.Set("pageInfo", "true")
	values.Set("pageNumber", strconv.Itoa(offersQuery.PageNumber))
	values.Set("pageSize", strconv.Itoa(offersQuery.PageSize))

	body, err := c.post(ctx, endpoint+"?"+values.Encode(), offersQuery.Language)
	if err != nil {
		return OffersPage{}, fmt.Errorf(
			"getting page %d of the offers: %w", offersQuery.PageNumber, err,
		)
	}

	var page OffersPage
	if err := json.Unmarshal(body, &page); err != nil {
		return OffersPage{}, fmt.Errorf(
			"decoding page %d of the offers: %w", offersQuery.PageNumber, err,
		)
	}

	return page, nil
}

// post sends a POST with the body {} and returns the body of a 200.
func (c *Client) post(ctx context.Context, endpoint, language string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader([]byte("{}")),
	)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if language != "" {
		req.Header.Set("Accept-Language", language)
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
