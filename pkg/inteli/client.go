package inteli

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
	// DefaultBaseURL is the production address of the site of Inteli Express.
	DefaultBaseURL = "https://inteliexpress.com"

	title = "ვალუტის სტანდარტული კურსები"

	cellsOfARow = 4
)

// Client reads the site of Inteli Express. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Inteli Express.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Rates returns each row of the rate table of the home page, in the order of the page.
func (c *Client) Rates(ctx context.Context) ([]RateRow, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"ka", "main-page-geo/"}})
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

	table := rateTable(doc)
	if table == nil {
		return nil, ErrNoRateTable
	}

	var rows []RateRow

	for body := range table.ChildNodes() {
		if !isElement(body, atom.Tbody) {
			continue
		}

		for tr := range body.ChildNodes() {
			if !isElement(tr, atom.Tr) {
				continue
			}

			tds := elements(tr, atom.Td)
			if len(tds) != cellsOfARow {
				return nil, fmt.Errorf("%w: a row of %d cells", ErrMalformedTable, len(tds))
			}

			rows = append(rows, RateRow{
				Base:     text(tds[0]),
				Currency: text(tds[1]),
				Buy:      text(tds[2]),
				Sell:     text(tds[3]),
			})
		}
	}

	if len(rows) == 0 {
		return nil, ErrNoRateTable
	}

	return rows, nil
}

// rateTable returns the first table whose head holds the title, or nil. The table names
// no column, so the title is the only sign of the rate table.
func rateTable(doc *html.Node) *html.Node {
	for _, table := range elements(doc, atom.Table) {
		for _, thead := range elements(table, atom.Thead) {
			if strings.Contains(text(thead), title) {
				return table
			}
		}
	}

	return nil
}

// elements returns the descendants of a node with the tag, in the order of the
// document. A nil node gives none.
func elements(node *html.Node, tag atom.Atom) []*html.Node {
	if node == nil {
		return nil
	}

	var found []*html.Node

	for child := range node.Descendants() {
		if isElement(child, tag) {
			found = append(found, child)
		}
	}

	return found
}

func isElement(node *html.Node, tag atom.Atom) bool {
	return node.Type == html.ElementNode && node.DataAtom == tag
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
