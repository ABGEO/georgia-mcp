package nbg

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the API of the NBG.
	DefaultBaseURL = "https://nbg.gov.ge/gw/api/ct/monetarypolicy"

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
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the API of the NBG.
func New(opts ...Option) *Client {
	defaults := []Option{vendorhttp.WithBaseURL(DefaultBaseURL)}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
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

// get returns the body of an answer with a 2xx status. Any other status is an *APIError.
func (c *Client) get(
	ctx context.Context,
	path []string,
	query url.Values,
	accept string,
) ([]byte, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{
		Path:   path,
		Query:  query,
		Header: http.Header{"Accept": {accept}},
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: resp.ErrorBody()}
	}

	return resp.Body, nil
}
