package gnerc

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the data of GNERC.
const DefaultBaseURL = "https://data.gnerc.org"

// maxBody bounds an answer. The balance has 454 KB and grows by about 25 KB a year.
const maxBody = 4 << 20

// Client reads the home page and the workbooks of GNERC. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the data of GNERC.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithMaxBody(maxBody),
		vendorhttp.WithHTTPClient(
			&http.Client{Timeout: vendorhttp.DefaultTimeout, Transport: NewTransport()},
		),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Links returns each link of the home page, in the order of the page.
func (c *Client) Links(ctx context.Context) ([]Link, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{Header: http.Header{"Accept": {"text/html"}}})
	if err != nil {
		return nil, fmt.Errorf("reading the home page: %w", err)
	}

	links, err := parseLinks(body)
	if err != nil {
		return nil, fmt.Errorf("reading the home page: %w", err)
	}

	return links, nil
}

// Workbook returns the workbook of the file name in the folder excel:
// "2015 - 2026 ფაქტიური ბალანსი.xlsx". The client escapes the name.
func (c *Client) Workbook(ctx context.Context, name string) (Workbook, error) {
	answer, err := c.http.Do(ctx, vendorhttp.Request{Path: []string{"excel", name}})
	if err != nil {
		return Workbook{}, fmt.Errorf("reading a workbook: %w", err)
	}

	if answer.StatusCode != http.StatusOK {
		return Workbook{}, fmt.Errorf("reading a workbook: %w", c.http.StatusError(answer))
	}

	sheets, err := readWorkbook(answer.Body)
	if err != nil {
		return Workbook{}, fmt.Errorf("reading a workbook: %w", err)
	}

	return Workbook{LastModified: answer.Header.Get("Last-Modified"), Sheets: sheets}, nil
}
