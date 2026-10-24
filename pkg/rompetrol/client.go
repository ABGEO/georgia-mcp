package rompetrol

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Rompetrol Georgia.
	DefaultBaseURL = "https://www.rompetrol.ge"

	// tableAnchor is the id of the element directly before the price table. Two links
	// of the page name it, so it outlives a change of the style of the table.
	tableAnchor = "pricelist"
	cellsInRow  = 2
)

// Client reads the site of Rompetrol Georgia. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Rompetrol Georgia.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Prices returns each row of the body of the price table of the home page, in the
// order of the page. A table with no row of data gives an empty list.
func (c *Client) Prices(ctx context.Context) ([]PriceRow, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"/"}})
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
