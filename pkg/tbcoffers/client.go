package tbcoffers

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
	// DefaultBaseURL is the production address of the marketing API of TBC Bank.
	DefaultBaseURL = "https://apigw.tbcbank.ge/api/v1/marketing"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/tbcoffers"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. The list of 518 offers is under 1 MiB.
	maxBodyBytes = 8 << 20
	maxErrorBody = 1 << 10
)

// Client calls the marketing API of TBC Bank. It is safe for concurrent use.
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
// georgia-mcp-tbcoffers/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the marketing API of TBC Bank.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-tbcoffers/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Filters returns the filters of the offers of one segment, in a locale.
func (c *Client) Filters(ctx context.Context, segment, locale string) ([]Filter, error) {
	endpoint, err := url.JoinPath(c.baseURL, "filters")
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	values := url.Values{"type": {"offer"}, "locale": {locale}, "segment": {segment}}

	var filters []Filter
	if err := c.send(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil, &filters); err != nil {
		return nil, fmt.Errorf("getting the filters of %q: %w", segment, err)
	}

	return filters, nil
}

// Offers returns one page of the offers that match the request.
func (c *Client) Offers(ctx context.Context, request OffersRequest) (OffersPage, error) {
	endpoint, err := url.JoinPath(c.baseURL, "entries", "offer")
	if err != nil {
		return OffersPage{}, fmt.Errorf("building the address: %w", err)
	}

	if request.Filter == nil {
		request.Filter = []string{}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return OffersPage{}, fmt.Errorf("encoding the request: %w", err)
	}

	var page OffersPage
	if err := c.send(ctx, http.MethodPost, endpoint, body, &page); err != nil {
		return OffersPage{}, fmt.Errorf("getting page %d of the offers: %w", request.PageIndex, err)
	}

	return page, nil
}

// Offer returns the page of one offer, with its terms, in a locale.
func (c *Client) Offer(ctx context.Context, slug, locale string) (Entry, error) {
	endpoint, err := url.JoinPath(c.baseURL, "entries", "offer", url.PathEscape(slug), locale)
	if err != nil {
		return Entry{}, fmt.Errorf("building the address: %w", err)
	}

	var entry Entry
	if err := c.send(ctx, http.MethodGet, endpoint, nil, &entry); err != nil {
		return Entry{}, fmt.Errorf("getting the offer %q: %w", slug, err)
	}

	return entry, nil
}

// send sends a request, with a JSON body when body is not nil, and decodes the answer
// of a 200 into out. Any other status is an *APIError.
func (c *Client) send(ctx context.Context, method, endpoint string, body []byte, out any) error {
	var reader io.Reader = http.NoBody
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	answer, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("reading the answer: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(answer[:min(len(answer), maxErrorBody)]),
		}
	}

	if err := json.Unmarshal(answer, out); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
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
