package rico

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	// DefaultBaseURL is the production address of the site of Rico Credit.
	DefaultBaseURL = "https://www.rico.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/rico"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. The home page is about 84 KB.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10

	classTableBody = "currencies-table-body"
	classRow       = "currency-row"
	classCurrency  = "currency-info"
	classUnit      = "currency-unit"
	classRate      = "currency-rate"

	labelBuy  = "ყიდვა"
	labelSell = "გაყიდვა"
)

// Client reads the site of Rico Credit. It is safe for concurrent use.
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
// georgia-mcp-rico/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the site of Rico Credit.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-rico/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Rates returns each row of the rate table of the home page, in the order of the page.
func (c *Client) Rates(ctx context.Context) ([]RateRow, error) {
	body, err := c.get(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting the home page: %w", err)
	}

	rows, err := parseRates(body)
	if err != nil {
		return nil, fmt.Errorf("reading the rate table: %w", err)
	}

	return rows, nil
}

func parseRates(body []byte) ([]RateRow, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the page: %w", err)
	}

	table := firstWithClass(doc, classTableBody)
	if table == nil {
		return nil, ErrNoRateTable
	}

	var rows []RateRow

	for node := range table.Descendants() {
		if !hasClass(node, classRow) {
			continue
		}

		row, err := parseRow(node)
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, ErrNoRateTable
	}

	return rows, nil
}

// parseRow reads one row. The label of each price, not its place, tells buy from sell.
func parseRow(node *html.Node) (RateRow, error) {
	row := RateRow{
		Currency: text(firstStrong(firstWithClass(node, classCurrency))),
		Unit:     text(firstStrong(firstWithClass(node, classUnit))),
	}

	for cell := range node.Descendants() {
		if !hasClass(cell, classRate) {
			continue
		}

		price := text(firstStrong(cell))

		switch text(firstElement(cell, atom.Span)) {
		case labelBuy:
			row.Buy = price
		case labelSell:
			row.Sell = price
		}
	}

	if row.Currency == "" || row.Unit == "" || row.Buy == "" || row.Sell == "" {
		return RateRow{}, fmt.Errorf("%w: a row %+v", ErrMalformedTable, row)
	}

	return row, nil
}

func firstStrong(node *html.Node) *html.Node {
	return firstElement(node, atom.Strong)
}

// firstElement returns the first descendant of a node with the tag, or nil. A nil node
// gives nil.
func firstElement(node *html.Node, tag atom.Atom) *html.Node {
	if node == nil {
		return nil
	}

	for child := range node.Descendants() {
		if child.Type == html.ElementNode && child.DataAtom == tag {
			return child
		}
	}

	return nil
}

// firstWithClass returns the first descendant of a node with the class, in the order
// of the document, or nil.
func firstWithClass(node *html.Node, class string) *html.Node {
	for child := range node.Descendants() {
		if hasClass(child, class) {
			return child
		}
	}

	return nil
}

// hasClass tells whether an element holds the class as a whole token of its class
// attribute: currency-rate does not match currency-rate-card.
func hasClass(node *html.Node, class string) bool {
	if node.Type != html.ElementNode {
		return false
	}

	for _, attr := range node.Attr {
		if attr.Namespace == "" && attr.Key == "class" {
			return slices.Contains(strings.Fields(attr.Val), class)
		}
	}

	return false
}

// text returns the text of the descendants of a node, with each run of white space as
// one space and no space at the ends. A nil node gives an empty text.
func text(node *html.Node) string {
	if node == nil {
		return ""
	}

	var builder strings.Builder

	for child := range node.Descendants() {
		if child.Type == html.TextNode {
			builder.WriteString(child.Data)
			builder.WriteByte(' ')
		}
	}

	return strings.Join(strings.Fields(builder.String()), " ")
}

// get sends a GET of the Georgian home page and returns the body of a 200. Any other
// status is an *APIError.
func (c *Client) get(ctx context.Context) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, "ka/")
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", "text/html")
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
