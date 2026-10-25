package halyk

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
	// DefaultBaseURL is the production address of the site of Halyk Bank Georgia.
	DefaultBaseURL = "https://halykbank.ge"

	idCurrentTab = "currentexchangerate"

	titleCurrency = "ვალუტა"
	titleBuy      = "ყიდვა"
	titleSell     = "გაყიდვა"
)

// Client reads the site of Halyk Bank Georgia. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Halyk Bank Georgia.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Rates returns each row of the table of the current rates, in the order of the page.
func (c *Client) Rates(ctx context.Context) ([]RateRow, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{
		Path: []string{"ka", "individuals", "currency"},
	})
	if err != nil {
		return nil, fmt.Errorf("getting the rate page: %w", err)
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

	table := firstElement(withID(doc, idCurrentTab), atom.Table)
	if table == nil {
		return nil, ErrNoRateTable
	}

	columns, err := parseTitles(firstElement(table, atom.Thead))
	if err != nil {
		return nil, err
	}

	var rows []RateRow

	for body := range table.Descendants() {
		if !isElement(body, atom.Tbody) {
			continue
		}

		for tr := range body.ChildNodes() {
			if !isElement(tr, atom.Tr) {
				continue
			}

			row, err := parseRow(tr, columns)
			if err != nil {
				return nil, err
			}

			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		return nil, ErrNoRateTable
	}

	return rows, nil
}

// columns holds the place of each column that the client reads.
type columns struct {
	currency, buy, sell int
}

// parseTitles finds each column by its title, so a swap of two columns stays visible.
func parseTitles(thead *html.Node) (columns, error) {
	place := map[string]int{}

	for i, th := range cells(thead, atom.Th) {
		place[text(th)] = i
	}

	currency, okCurrency := place[titleCurrency]
	buy, okBuy := place[titleBuy]
	sell, okSell := place[titleSell]

	if !okCurrency || !okBuy || !okSell {
		return columns{}, fmt.Errorf("%w: titles %v", ErrMalformedTable, place)
	}

	return columns{currency: currency, buy: buy, sell: sell}, nil
}

func parseRow(tr *html.Node, cols columns) (RateRow, error) {
	tds := cells(tr, atom.Td)
	if len(tds) <= max(cols.currency, cols.buy, cols.sell) {
		return RateRow{}, fmt.Errorf("%w: a row of %d cells", ErrMalformedTable, len(tds))
	}

	return RateRow{
		Currency: text(tds[cols.currency]),
		Buy:      text(tds[cols.buy]),
		Sell:     text(tds[cols.sell]),
	}, nil
}

// cells returns the descendants of a node with the tag, in the order of the document. A
// nil node gives none.
func cells(node *html.Node, tag atom.Atom) []*html.Node {
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

// firstElement returns the first descendant of a node with the tag, or nil. A nil node
// gives nil.
func firstElement(node *html.Node, tag atom.Atom) *html.Node {
	if found := cells(node, tag); len(found) > 0 {
		return found[0]
	}

	return nil
}

// withID returns the first element of the document with the id, or nil.
func withID(doc *html.Node, id string) *html.Node {
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}

		for _, attr := range node.Attr {
			if attr.Namespace == "" && attr.Key == "id" && attr.Val == id {
				return node
			}
		}
	}

	return nil
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
