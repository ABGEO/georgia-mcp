package geostat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the PxWeb API of Geostat.
const DefaultBaseURL = "https://pc-axis.geostat.ge/PXWeb/api/v1"

const database = "Database"

// byteOrderMark starts the answer of a query.
const byteOrderMark = "\xef\xbb\xbf"

// Client calls the PxWeb API of Geostat. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the PxWeb API of Geostat.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Folder lists the folders and the tables of a folder. lang is ka or en. path is "" for
// the root, or the IDs of the folders joined with "/".
func (c *Client) Folder(ctx context.Context, lang, path string) ([]Node, error) {
	var nodes []Node
	if err := c.send(ctx, http.MethodGet, lang, path, nil, &nodes); err != nil {
		return nil, fmt.Errorf("listing the folder: %w", err)
	}

	return nodes, nil
}

// Table returns the title and the dimensions of a table. path ends with the file name
// of the table.
func (c *Client) Table(ctx context.Context, lang, path string) (*TableMetadata, error) {
	var metadata TableMetadata
	if err := c.send(ctx, http.MethodGet, lang, path, nil, &metadata); err != nil {
		return nil, fmt.Errorf("describing the table: %w", err)
	}

	return &metadata, nil
}

// Query returns the cells of a table for a selection of value codes. A dimension that
// the selection omits takes the default of Geostat.
func (c *Client) Query(
	ctx context.Context,
	lang, path string,
	selection []Selection,
) (*Dataset, error) {
	request := queryBody{Query: []querySelection{}, Response: queryResponse{Format: "json"}}
	for _, item := range selection {
		request.Query = append(request.Query, querySelection{
			Code: item.Code, Selection: valueList{Filter: "item", Values: item.Values},
		})
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encoding the query: %w", err)
	}

	var dataset Dataset
	if err := c.send(ctx, http.MethodPost, lang, path, body, &dataset); err != nil {
		return nil, fmt.Errorf("querying the table: %w", err)
	}

	return &dataset, nil
}

// send ends the path with a slash: the server of Geostat answers 404 to a path whose
// last segment holds a dot.
func (c *Client) send(
	ctx context.Context,
	method, lang, path string,
	body []byte,
	target any,
) error {
	answer, err := c.http.Fetch(ctx, vendorhttp.Request{
		Method: method,
		Path:   []string{lang, database, path + "/"},
		Body:   body,
	})
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes.TrimPrefix(answer, []byte(byteOrderMark)), target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}
