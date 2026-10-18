package nbg

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the production address of the API of the NBG.
	DefaultBaseURL = "https://nbg.gov.ge/gw/api/ct/monetarypolicy"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/nbg"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. A rate set is about 10 KiB.
	maxBodyBytes = 1 << 20

	acceptJSON = "application/json"
	acceptRSS  = "application/rss+xml, application/xml;q=0.9, text/xml;q=0.8"
)

// Language is the language of the names of the currencies.
type Language string

const (
	// LanguageEnglish gives the names in English.
	LanguageEnglish Language = "en"
	// LanguageGeorgian gives the names in Georgian.
	LanguageGeorgian Language = "ka"
)

// Client calls the API of the NBG. It is safe for concurrent use.
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
// georgia-mcp-nbg/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the API of the NBG.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-nbg/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// RateSetsQuery selects the rate sets of a request.
type RateSetsQuery struct {
	// Date is zero to ask for the newest set.
	Date time.Time
	// Currencies is empty to ask for every currency. The client sends the list joined
	// with commas, as the NBG takes it. The NBG answers most lists of two or more codes
	// with an empty list.
	Currencies []string
}

func (q RateSetsQuery) values() url.Values {
	values := url.Values{}

	if !q.Date.IsZero() {
		values.Set("date", q.Date.Format(time.DateOnly))
	}

	if len(q.Currencies) > 0 {
		values.Set("currencies", strings.Join(q.Currencies, ","))
	}

	return values
}

// RateSets returns the rate set in force on the date, as a list of zero or one set.
// The NBG answers a date after its newest set, and a date that it cannot read, with
// its newest set.
func (c *Client) RateSets(
	ctx context.Context,
	language Language,
	query RateSetsQuery,
) ([]RateSet, error) {
	var sets []RateSet
	if err := c.getJSON(ctx, ratesPath(language, "json"), query.values(), &sets); err != nil {
		return nil, fmt.Errorf("getting the rate sets: %w", err)
	}

	return sets, nil
}

// RSS returns the RSS feed of the rate set in force on the date. Each item holds the
// rates as the HTML table of the NBG.
func (c *Client) RSS(ctx context.Context, language Language, query RateSetsQuery) (*RSS, error) {
	body, err := c.get(ctx, ratesPath(language, "rss"), query.values(), acceptRSS)
	if err != nil {
		return nil, fmt.Errorf("getting the feed: %w", err)
	}

	var feed RSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("decoding the feed: %w", err)
	}

	return &feed, nil
}

// Codes returns every currency code that the NBG knows, with its name in Georgian.
func (c *Client) Codes(ctx context.Context) ([]Code, error) {
	var codes []Code
	if err := c.getJSON(ctx, []string{"currencies", "codes"}, nil, &codes); err != nil {
		return nil, fmt.Errorf("getting the codes: %w", err)
	}

	return codes, nil
}

// ratesPath gives the path of the rates in a format. The NBG serves it with a
// trailing slash.
func ratesPath(language Language, format string) []string {
	return []string{"currencies", url.PathEscape(string(language)), format + "/"}
}

func (c *Client) getJSON(ctx context.Context, path []string, query url.Values, target any) error {
	body, err := c.get(ctx, path, query, acceptJSON)
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
func (c *Client) get(
	ctx context.Context,
	path []string,
	query url.Values,
	accept string,
) ([]byte, error) {
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

	req.Header.Set("Accept", accept)
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
