package lukoil

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
	// DefaultBaseURL is the production address of the site of Lukoil Georgia.
	DefaultBaseURL = "https://www.lukoil.ge"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/lukoil"

	// maxBodyBytes bounds the read of an answer. The page is about 97 KiB, and a change
	// of the prices adds about 1 KiB.
	maxBodyBytes = 2 << 20
)

// Client reads the site of Lukoil Georgia. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Lukoil Georgia.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// PriceHistory returns the head and each row of the table of the price history page, in
// the order of the page. A table with no row gives no row.
func (c *Client) PriceHistory(ctx context.Context) (PriceTable, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"prices-history"}})
	if err != nil {
		return PriceTable{}, fmt.Errorf("getting the price history page: %w", err)
	}

	table, err := parsePriceHistory(body)
	if err != nil {
		return PriceTable{}, fmt.Errorf("reading the price table: %w", err)
	}

	return table, nil
}

func parsePriceHistory(body []byte) (PriceTable, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return PriceTable{}, fmt.Errorf("parsing the page: %w", err)
	}

	table := first(doc, atom.Table)
	if table == nil {
		return PriceTable{}, ErrNoPriceTable
	}

	header := headCells(table)
	if len(header) == 0 {
		return PriceTable{}, ErrNoPriceTable
	}

	rows, err := bodyRows(table, len(header))
	if err != nil {
		return PriceTable{}, err
	}

	return PriceTable{Header: header, Rows: rows}, nil
}

// headCells returns the text of each th of the head of a table.
func headCells(table *html.Node) []string {
	head := first(table, atom.Thead)
	if head == nil {
		return nil
	}

	var header []string

	for cell := range head.Descendants() {
		if cell.Type == html.ElementNode && cell.DataAtom == atom.Th {
			header = append(header, text(cell))
		}
	}

	return header
}

// bodyRows returns the cells of each row of the body of a table. Each row must hold one
// cell for each cell of the head.
func bodyRows(table *html.Node, width int) ([][]string, error) {
	rows := [][]string{}

	tableBody := first(table, atom.Tbody)
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

// first returns the first descendant element of a node with the tag, in the order of the
// document, or nil.
func first(node *html.Node, tag atom.Atom) *html.Node {
	for child := range node.Descendants() {
		if child.Type == html.ElementNode && child.DataAtom == tag {
			return child
		}
	}

	return nil
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
