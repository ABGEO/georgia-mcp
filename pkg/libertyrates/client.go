package libertyrates

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
)

const (
	// DefaultBaseURL is the production address of the site of Liberty Bank.
	DefaultBaseURL = "https://www.libertybank.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/libertyrates"
	defaultTimeout = 10 * time.Second

	// maxBodyBytes bounds the read of an answer. The home page is about 124 KB.
	maxBodyBytes = 2 << 20
	maxErrorBody = 1 << 10

	classBlock       = "currency-rates"
	classHead        = "currency-rates__row-head"
	classRow         = "currency-rates__row"
	classCurrencyRow = "js-homepage__currency-item"
	classItem        = "currency-rates__item"
	classMobileItem  = "currency-rates__item--showonmobile"
	classItemTitle   = "currency-rates__item-title"
	className        = "currency-rates__currency-name"
	classPrice       = "currency-rates__currency"
	classUnit        = "currency-rates__currency-description"

	titleCommercial   = "კომერციული"
	titleInternetBank = "ინტერნეტ ბანკი"
	titleTransfer     = "გზავნილების სპეც კურსი"

	pricesInQuote = 2
)

// Client reads the site of Liberty Bank. It is safe for concurrent use.
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
// georgia-mcp-libertyrates/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the site of Liberty Bank.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-libertyrates/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Rates returns each currency of the rate block of the home page, in the order of the
// page.
func (c *Client) Rates(ctx context.Context) ([]RateRow, error) {
	body, err := c.get(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting the home page: %w", err)
	}

	rows, err := parseRates(body)
	if err != nil {
		return nil, fmt.Errorf("reading the rate block: %w", err)
	}

	return rows, nil
}

func parseRates(body []byte) ([]RateRow, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the page: %w", err)
	}

	// The page holds the block twice, the second time for a narrow screen.
	block := firstWithClass(doc, classBlock)
	if block == nil {
		return nil, ErrNoRateBlock
	}

	titles, err := columnTitles(block)
	if err != nil {
		return nil, err
	}

	var rows []RateRow

	for node := range block.Descendants() {
		if !hasClass(node, classCurrencyRow) {
			continue
		}

		row, err := parseRow(node, titles)
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, ErrNoRateBlock
	}

	return rows, nil
}

// columnTitles returns the title of each column of the head, in the order of the page.
func columnTitles(block *html.Node) ([]string, error) {
	var titles []string

	if head := firstWithClass(block, classHead); head != nil {
		if row := firstWithClass(head, classRow); row != nil {
			for _, item := range items(row) {
				titles = append(titles, text(firstWithClass(item, classItemTitle)))
			}
		}
	}

	if !slices.Contains(titles, titleCommercial) {
		return nil, fmt.Errorf("%w: no column %q", ErrMalformedBlock, titleCommercial)
	}

	return titles, nil
}

// parseRow reads one currency. The first item holds the code, the official rate, and
// the unit. Each other item is the column with the title at the same position.
func parseRow(node *html.Node, titles []string) (RateRow, error) {
	cells := items(node)
	if len(cells) == 0 {
		return RateRow{}, fmt.Errorf("%w: a currency with no item", ErrMalformedBlock)
	}

	row := RateRow{
		Currency: text(firstWithClass(cells[0], className)),
		Unit:     text(firstWithClass(cells[0], classUnit)),
		Official: text(firstWithClass(cells[0], classPrice)),
	}
	if row.Currency == "" {
		return RateRow{}, fmt.Errorf("%w: a currency with no code", ErrMalformedBlock)
	}

	for i, cell := range cells[1:min(len(cells), len(titles))] {
		quote, err := parseQuote(cell)
		if err != nil {
			return RateRow{}, fmt.Errorf("%w: %s %q", err, row.Currency, titles[i+1])
		}

		switch titles[i+1] {
		case titleCommercial:
			row.Commercial = quote
		case titleInternetBank:
			row.InternetBank = quote
		case titleTransfer:
			row.Transfer = quote
		}
	}

	return row, nil
}

func parseQuote(cell *html.Node) (Quote, error) {
	var prices []string

	for node := range cell.Descendants() {
		if hasClass(node, classPrice) {
			prices = append(prices, text(node))
		}
	}

	switch len(prices) {
	case 0:
		return Quote{}, nil
	case pricesInQuote:
		return Quote{Buy: prices[0], Sell: prices[1]}, nil
	default:
		return Quote{}, fmt.Errorf("%w: %d prices in a column", ErrMalformedBlock, len(prices))
	}
}

// items returns the children of a row that are columns, with the copy for a narrow
// screen left out.
func items(row *html.Node) []*html.Node {
	var cells []*html.Node

	for child := range row.ChildNodes() {
		if hasClass(child, classItem) && !hasClass(child, classMobileItem) {
			cells = append(cells, child)
		}
	}

	return cells
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
// attribute: currency-rates__currency does not match currency-rates__currency-change.
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
