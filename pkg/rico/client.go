package rico

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Rico Credit.
	DefaultBaseURL = "https://www.rico.ge"

	vendorName = "rico"

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
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Rico Credit.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
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
// status is an *APIError. The APIError of this client compares by value, so it stays
// its own type.
func (c *Client) get(ctx context.Context) ([]byte, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{Path: []string{"ka/"}})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: resp.ErrorBody()}
	}

	return resp.Body, nil
}
