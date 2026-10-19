package rompetrol

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	// DefaultBaseURL is the production address of the site of Rompetrol Georgia.
	DefaultBaseURL = "https://www.rompetrol.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/rompetrol"
	defaultTimeout = 10 * time.Second

	// tableAnchor is the id of the element directly before the price table. Two links
	// of the page name it, so it outlives a change of the style of the table.
	tableAnchor = "pricelist"
	cellsInRow  = 2

	// maxBodyBytes bounds the read of an answer. The home page is about 120 KiB.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10
)

// Client reads the site of Rompetrol Georgia. It is safe for concurrent use.
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
// georgia-mcp-rompetrol/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the site of Rompetrol Georgia.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-rompetrol/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Prices returns each row of the body of the price table of the home page, in the
// order of the page. A table with no row of data gives an empty list.
func (c *Client) Prices(ctx context.Context) ([]PriceRow, error) {
	body, err := c.get(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting the home page: %w", err)
	}

	rows, err := parsePrices(body)
	if err != nil {
		return nil, fmt.Errorf("reading the price table: %w", err)
	}

	return rows, nil
}

func parsePrices(body []byte) ([]PriceRow, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the page: %w", err)
	}

	table := priceTable(doc)
	if table == nil {
		return nil, ErrNoPriceTable
	}

	var rows []PriceRow

	for row := range table.Descendants() {
		if row.Type != html.ElementNode || row.DataAtom != atom.Tr {
			continue
		}

		cells := dataCells(row)
		if len(cells) == 0 {
			continue
		}

		if len(cells) != cellsInRow {
			return nil, fmt.Errorf("%w: a row with %d cells", ErrMalformedTable, len(cells))
		}

		rows = append(rows, PriceRow{Product: text(cells[0]), Price: text(cells[1])})
	}

	return rows, nil
}

// priceTable returns the first table after the anchor of the price table, in the
// order of the document, or nil.
func priceTable(doc *html.Node) *html.Node {
	anchored := false

	for node := range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}

		if !anchored {
			anchored = attribute(node, "id") == tableAnchor

			continue
		}

		if node.DataAtom == atom.Table {
			return node
		}
	}

	return nil
}

func dataCells(row *html.Node) []*html.Node {
	var cells []*html.Node

	for cell := range row.ChildNodes() {
		if cell.Type == html.ElementNode && cell.DataAtom == atom.Td {
			cells = append(cells, cell)
		}
	}

	return cells
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Namespace == "" && attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

// text returns the text of the descendants of a node, with each run of white space as
// one space and no space at the ends.
func text(node *html.Node) string {
	var builder strings.Builder

	for child := range node.Descendants() {
		if child.Type == html.TextNode {
			builder.WriteString(child.Data)
			builder.WriteByte(' ')
		}
	}

	return strings.Join(strings.Fields(builder.String()), " ")
}

// get sends a GET of the home page and returns the body of a 200. Any other status is
// an *APIError.
func (c *Client) get(ctx context.Context) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, "/")
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
