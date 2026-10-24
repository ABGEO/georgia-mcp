package gpost

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Georgian Post.
	DefaultBaseURL = "https://www.gpost.ge"

	modulePath = "github.com/abgeo/georgia-mcp/pkg/gpost"

	tokenName     = "__RequestVerificationToken" //nolint:gosec // The name of a field, not a token.
	tokenFormID   = "__AjaxAntiForgeryForm"      //nolint:gosec // The id of a form, not a token.
	districtClass = "com-district"
	rowClass      = "com-cell-postcode"
	codeClass     = "com-postcode-name"
	addressClass  = "com-location-name"

	// The page sends this value in a hidden field and offers no other.
	regionType = "1"
	// The form of the page sends the address of the page itself, on the production site.
	referer = DefaultBaseURL + "/help/postal-codes"
)

// Client reads the search of postal codes of Georgian Post. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the site of Georgian Post.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "text/html"),
	}

	return &Client{
		http: vendorhttp.New(modulePath, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// PostalCodes visits the page of postal codes, then searches text. lang is ka or en. It
// returns each address of the answer in the order of the page. An answer with no match
// gives an empty list.
func (c *Client) PostalCodes(ctx context.Context, text, lang string) ([]PostalCodeRow, error) {
	formToken, cookieToken, err := c.visit(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("visiting the page of postal codes: %w", err)
	}

	form := url.Values{
		tokenName:               {formToken},
		"regionTypeId":          {regionType},
		"postalCodeSearchValue": {text},
	}

	body, err := c.http.Fetch(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{"Help", "FindPostalCode"},
		Header: http.Header{
			"Content-Type":     {"application/x-www-form-urlencoded"},
			"X-Requested-With": {"XMLHttpRequest"},
			"Referer":          {referer},
			"Cookie":           {tokenName + "=" + cookieToken + "; " + languageCookie(lang)},
		},
		Body: []byte(form.Encode()),
	})
	if err != nil {
		return nil, fmt.Errorf("searching the postal codes: %w", err)
	}

	rows, err := parseRows(body)
	if err != nil {
		return nil, fmt.Errorf("reading the search answer: %w", err)
	}

	return rows, nil
}

// visit returns the form token of the page and the cookie token of its answer. The
// search needs both, from the same visit.
func (c *Client) visit(ctx context.Context, lang string) (string, string, error) {
	resp, err := c.http.Do(ctx, vendorhttp.Request{
		Path:   []string{"help", "postal-codes"},
		Header: http.Header{"Cookie": {languageCookie(lang)}},
	})
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", c.http.StatusError(resp)
	}

	cookieToken := tokenCookie(resp.Header)
	if cookieToken == "" {
		return "", "", fmt.Errorf("%w: no cookie", ErrNoToken)
	}

	formToken, err := parseFormToken(resp.Body)
	if err != nil {
		return "", "", err
	}

	return formToken, cookieToken, nil
}

// languageCookie returns the cookie that the link of the site to a language sets.
func languageCookie(lang string) string {
	return "lang=" + lang
}

func tokenCookie(header http.Header) string {
	for _, line := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		if err == nil && cookie.Name == tokenName {
			return cookie.Value
		}
	}

	return ""
}

func parseFormToken(body []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parsing the page: %w", err)
	}

	for form := range doc.Descendants() {
		if form.Type != html.ElementNode || form.DataAtom != atom.Form ||
			attribute(form, "id") != tokenFormID {
			continue
		}

		if token := inputValue(form, tokenName); token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("%w: no form token", ErrNoToken)
}

func inputValue(form *html.Node, name string) string {
	for input := range form.Descendants() {
		if input.Type == html.ElementNode && input.DataAtom == atom.Input &&
			attribute(input, "name") == name {
			return attribute(input, "value")
		}
	}

	return ""
}

// parseRows reads the rows of a search answer in the order of the document. A district
// starts a group, and each row after it belongs to that group. An answer with no row is
// a valid empty answer only when it holds the note of the page, a p element. An answer
// with a title or a form element is a whole page, not a search answer.
func parseRows(body []byte) ([]PostalCodeRow, error) {
	doc, err := parseAnswer(body)
	if err != nil {
		return nil, err
	}

	var (
		rows     []PostalCodeRow
		district string
		hasNote  bool
	)

	for node := range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}

		switch {
		case node.DataAtom == atom.P:
			hasNote = true
		case hasClass(node, districtClass):
			district = text(node)
		case hasClass(node, rowClass):
			row, err := readRow(node, district, len(rows)+1)
			if err != nil {
				return nil, err
			}

			rows = append(rows, row)
		}
	}

	if len(rows) == 0 && !hasNote {
		return nil, fmt.Errorf("%w: no row and no note", ErrMalformedResult)
	}

	return rows, nil
}

func parseAnswer(body []byte) (*html.Node, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the answer: %w", err)
	}

	if isPage(doc) {
		return nil, fmt.Errorf("%w: a whole page", ErrMalformedResult)
	}

	return doc, nil
}

func isPage(doc *html.Node) bool {
	for node := range doc.Descendants() {
		if node.Type == html.ElementNode &&
			(node.DataAtom == atom.Title || node.DataAtom == atom.Form) {
			return true
		}
	}

	return false
}

func readRow(node *html.Node, district string, number int) (PostalCodeRow, error) {
	row := PostalCodeRow{
		District: district,
		Code:     text(descendantWithClass(node, codeClass)),
		Address:  text(descendantWithClass(node, addressClass)),
	}
	if row.District == "" || row.Code == "" || row.Address == "" {
		return PostalCodeRow{}, fmt.Errorf("%w: row %d has no district, code, or address",
			ErrMalformedResult, number)
	}

	return row, nil
}

func descendantWithClass(node *html.Node, class string) *html.Node {
	for child := range node.Descendants() {
		if child.Type == html.ElementNode && hasClass(child, class) {
			return child
		}
	}

	return nil
}

func hasClass(node *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attribute(node, "class")), class)
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
// one space and no space at the ends. A nil node has no text.
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
