package liberty

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The classes of the page of an offer, as they were on 2026-10-20.
const (
	titleClass = "featured-hero__title"
	timerClass = "featured-card__timer-text"
	termsClass = "featured-hero__text"
)

func parsePage(body []byte) (Page, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Page{}, fmt.Errorf("parsing the page: %w", err)
	}

	terms := firstWithClass(doc, termsClass)
	if terms == nil {
		return Page{}, ErrNoTerms
	}

	page := Page{
		Title: text(firstWithClass(doc, titleClass)),
		Timer: text(firstWithClass(doc, timerClass)),
	}

	for node := range terms.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.P {
			page.Paragraphs = append(page.Paragraphs, text(node))
		}
	}

	return page, nil
}

// firstWithClass returns the first element of the document whose class list holds the
// class, or nil.
func firstWithClass(doc *html.Node, class string) *html.Node {
	for node := range doc.Descendants() {
		if node.Type == html.ElementNode &&
			slices.Contains(strings.Fields(attribute(node, "class")), class) {
			return node
		}
	}

	return nil
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
