package gulf

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
	// DefaultBaseURL is the production address of the site of Gulf Georgia.
	DefaultBaseURL = "https://gulf.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/gulf"
	defaultTimeout = 10 * time.Second
	priceTableID   = "price_archive_tbl"

	// maxBodyBytes bounds the read of an answer. The page is about 109 KiB, and holds
	// 100 rows at most.
	maxBodyBytes = 1 << 20
	maxErrorBody = 1 << 10
)

// Client reads the site of Gulf Georgia. It is safe for concurrent use.
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

// WithHTTPClient sets the HTTP client. The default has a timeout of 10 seconds and the
// transport of NewTransport. A client with another transport gets the answer of the bot
// filter of Gulf.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent sets the User-Agent header. The default is
// georgia-mcp-gulf/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the site of Gulf Georgia.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout, Transport: NewTransport()},
		userAgent:  "georgia-mcp-gulf/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// PriceArchive returns each column and each row of the table of the fuel prices page, in
// the order of the page. A table with no row gives no row.
func (c *Client) PriceArchive(ctx context.Context) (PriceTable, error) {
	body, err := c.get(ctx)
	if err != nil {
		return PriceTable{}, fmt.Errorf("getting the fuel prices page: %w", err)
	}

	table, err := parsePriceArchive(body)
	if err != nil {
		return PriceTable{}, fmt.Errorf("reading the price table: %w", err)
	}

	return table, nil
}

func parsePriceArchive(body []byte) (PriceTable, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return PriceTable{}, fmt.Errorf("parsing the page: %w", err)
	}

	table := find(doc, func(node *html.Node) bool {
		return node.DataAtom == atom.Table && attr(node, "id") == priceTableID
	})
	if table == nil {
		return PriceTable{}, ErrNoPriceTable
	}

	columns := headColumns(table)
	if len(columns) == 0 {
		return PriceTable{}, ErrNoPriceTable
	}

	rows, err := bodyRows(table, len(columns))
	if err != nil {
		return PriceTable{}, err
	}

	return PriceTable{Columns: columns, Rows: rows}, nil
}

// headColumns returns a column for each cell of the first row of the head of a table.
func headColumns(table *html.Node) []Column {
	head := find(table, isElement(atom.Thead))
	if head == nil {
		return nil
	}

	row := find(head, isElement(atom.Tr))
	if row == nil {
		return nil
	}

	var columns []Column

	for cell := range row.ChildNodes() {
		if cell.Type != html.ElementNode || (cell.DataAtom != atom.Td && cell.DataAtom != atom.Th) {
			continue
		}

		columns = append(columns, Column{Name: label(cell), Class: attr(cell, "class")})
	}

	return columns
}

// label returns the text of the first span of the class "normal" of a head cell, or the
// text of the cell. A cell of a tier with a sort link holds its label and an abbreviation:
// "G-Force Super" and "GS".
func label(cell *html.Node) string {
	normal := find(cell, func(node *html.Node) bool {
		return node.DataAtom == atom.Span &&
			slices.Contains(strings.Fields(attr(node, "class")), "normal")
	})
	if normal != nil {
		return text(normal)
	}

	return text(cell)
}

// bodyRows returns the cells of each row of the body of a table. Each row must hold one
// cell for each column of the head.
func bodyRows(table *html.Node, width int) ([][]string, error) {
	rows := [][]string{}

	tableBody := find(table, isElement(atom.Tbody))
	if tableBody == nil {
		return rows, nil
	}

	for row := range tableBody.Descendants() {
		if row.Type != html.ElementNode || row.DataAtom != atom.Tr {
			continue
		}

		cells := dataCells(row)
		if len(cells) == 0 {
			continue
		}

		if len(cells) != width {
			return nil, fmt.Errorf(
				"%w: a row with %d cells under a head of %d", ErrMalformedTable, len(cells), width,
			)
		}

		rows = append(rows, cells)
	}

	return rows, nil
}

func isElement(tag atom.Atom) func(*html.Node) bool {
	return func(node *html.Node) bool { return node.DataAtom == tag }
}

// find returns the first descendant element of a node that matches, in the order of the
// document, or nil.
func find(node *html.Node, match func(*html.Node) bool) *html.Node {
	for child := range node.Descendants() {
		if child.Type == html.ElementNode && match(child) {
			return child
		}
	}

	return nil
}

func attr(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}

	return ""
}

func dataCells(row *html.Node) []string {
	var cells []string

	for cell := range row.ChildNodes() {
		if cell.Type == html.ElementNode && cell.DataAtom == atom.Td {
			cells = append(cells, text(cell))
		}
	}

	return cells
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

// get sends a GET of the fuel prices page and returns the body of a 200. Any other status
// is an *APIError.
func (c *Client) get(ctx context.Context) ([]byte, error) {
	endpoint, err := url.JoinPath(c.baseURL, "en", "fuel_prices")
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
