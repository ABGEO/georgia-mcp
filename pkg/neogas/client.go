package neogas

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Neogas.
	DefaultBaseURL = "https://neogas.ge"

	// priceClass is the class of each element that holds the gas price. The script of
	// the calculator reads it.
	priceClass = "gas_price"
)

// Client reads the site of Neogas. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Neogas.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// GasPrices returns the text of each gas price of the calculator of the home page, in the
// order of the page: ["1.60", "1.60"]. A page with none gives an empty list.
func (c *Client) GasPrices(ctx context.Context) ([]string, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Path: []string{"ka/"}})
	if err != nil {
		return nil, fmt.Errorf("getting the home page: %w", err)
	}

	prices, err := parseGasPrices(body)
	if err != nil {
		return nil, fmt.Errorf("reading the gas price: %w", err)
	}

	return prices, nil
}

func parseGasPrices(body []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the page: %w", err)
	}

	prices := []string{}

	for node := range doc.Descendants() {
		if node.Type == html.ElementNode &&
			slices.Contains(strings.Fields(attribute(node, "class")), priceClass) {
			prices = append(prices, text(node))
		}
	}

	return prices, nil
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
