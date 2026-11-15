package matsne

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the Legislative Herald.
	DefaultBaseURL = "https://matsne.gov.ge"

	// maxBody holds the page of the Tax Code: 4.9 MB in Georgian on 2026-11-09.
	maxBody = 16 << 20

	documentSegment = "document"
)

// Client reads the pages of the Legislative Herald. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the Legislative Herald.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
		vendorhttp.WithMaxBody(maxBody),
		vendorhttp.WithHTTPClient(
			&http.Client{Timeout: vendorhttp.DefaultTimeout, Transport: NewTransport()},
		),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Search returns one page of results of the search page in a language: "ka" or "en".
func (c *Client) Search(
	ctx context.Context,
	language string,
	query SearchQuery,
) (SearchPage, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{
		Path:  []string{language, documentSegment, "search"},
		Query: query.values(),
	})
	if err != nil {
		return SearchPage{}, fmt.Errorf("getting the search page: %w", err)
	}

	page, err := parseSearch(body)
	if err != nil {
		return SearchPage{}, fmt.Errorf("reading the search page: %w", err)
	}

	return page, nil
}

// Suggest returns the titles that the site suggests for a Georgian text.
func (c *Client) Suggest(ctx context.Context, text string) ([]Suggestion, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{documentSegment, "suggest"},
		Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:   []byte(url.Values{"text": {text}}.Encode()),
	})
	if err != nil {
		return nil, fmt.Errorf("getting the suggested titles: %w", err)
	}

	titles, err := parseSuggestions(body)
	if err != nil {
		return nil, fmt.Errorf("reading the suggested titles: %w", err)
	}

	return titles, nil
}

// Document returns the page of an act in a language. A publication of 0 or more asks that
// version. A negative publication asks the page with no version: the newest version, or
// in English the newest translated version.
func (c *Client) Document(
	ctx context.Context,
	language, id string,
	publication int,
) (Document, error) {
	req := vendorhttp.Request{Path: []string{language, documentSegment, "view", id}}
	if publication >= 0 {
		req.Query = url.Values{"publication": {strconv.Itoa(publication)}}
	}

	body, err := c.http.Fetch(ctx, req)
	if err != nil {
		return Document{}, fmt.Errorf("getting the page of the act: %w", err)
	}

	doc, err := parseDocument(body)
	if err != nil {
		return Document{}, fmt.Errorf("reading the page of the act: %w", err)
	}

	return doc, nil
}

func (q SearchQuery) values() url.Values {
	values := url.Values{}

	set := func(key, value string) {
		if value != "" {
			values.Set(key, value)
		}
	}

	set("query", q.Query)
	set("query_target", q.QueryTarget)
	set("type", q.Type)
	set("group", q.Group)
	set("status", q.Status)
	set("issuer", q.Issuer)
	set("number", q.Number)
	set("signing_date_fr[date]", q.SigningFrom)
	set("signing_date_to[date]", q.SigningTo)
	set("registration_code", q.RegistrationCode)
	set("sort", q.Sort)

	// The page counts its pages from 0 in the address and from 1 on the screen.
	if q.Page > 1 {
		values.Set("page", strconv.Itoa(q.Page-1))
	}

	return values
}
