package gse

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the files of GSE.
const DefaultBaseURL = "https://admin.gse.com.ge"

var errUnknownCharset = errors.New("gse: unknown charset")

const (
	maxBody     = 64 << 10
	stampLayout = "02012006_1504"
)

// Client reads the real time exports of GSE. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the exports of GSE.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/xml, application/xml"),
		vendorhttp.WithMaxBody(maxBody),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Grid returns the export of the grid of the 3 minutes stamped at. The name of the file
// holds at as it is, in the location of at: GSE names its files in the time of Tbilisi.
func (c *Client) Grid(ctx context.Context, at time.Time) (Export, error) {
	export, err := c.export(ctx, "GSE_EXPORT_NEW_PERIOD_03M_", at)
	if err != nil {
		return Export{}, fmt.Errorf("reading the grid: %w", err)
	}

	return export, nil
}

// Frequency returns the export of the frequency of the minute stamped at.
func (c *Client) Frequency(ctx context.Context, at time.Time) (Export, error) {
	export, err := c.export(ctx, "GSE_EXPORT_PERIOD_01M_", at)
	if err != nil {
		return Export{}, fmt.Errorf("reading the frequency: %w", err)
	}

	return export, nil
}

func (c *Client) export(ctx context.Context, prefix string, at time.Time) (Export, error) {
	name := prefix + at.Format(stampLayout) + ".XML"

	answer, err := c.http.Fetch(ctx, vendorhttp.Request{
		Path: []string{"storage", "uploads", "fact", name},
	})
	if err != nil {
		return Export{}, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(answer))
	decoder.CharsetReader = charsetReader

	var export Export
	if err := decoder.Decode(&export); err != nil {
		return Export{}, fmt.Errorf("decoding the answer: %w", err)
	}

	return export, nil
}

// charsetReader converts ISO-8859-1, the encoding that GSE declares, to UTF-8. Each byte
// of ISO-8859-1 is the code point of its rune.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	if !strings.EqualFold(label, "ISO-8859-1") {
		return nil, fmt.Errorf("decoding %q: %w", label, errUnknownCharset)
	}

	latin, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("reading the answer: %w", err)
	}

	text := make([]byte, 0, len(latin))
	for _, b := range latin {
		text = utf8.AppendRune(text, rune(b))
	}

	return bytes.NewReader(text), nil
}
