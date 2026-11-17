package gnerc

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func parseLinks(body []byte) ([]Link, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing the page: %w", err)
	}

	var links []Link

	for node := range doc.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.A {
			continue
		}

		for _, attribute := range node.Attr {
			if attribute.Key == "href" {
				links = append(links, Link{Href: attribute.Val, Text: text(node)})
			}
		}
	}

	return links, nil
}

func text(node *html.Node) string {
	var parts strings.Builder

	for child := range node.Descendants() {
		if child.Type == html.TextNode {
			parts.WriteString(child.Data)
			parts.WriteByte(' ')
		}
	}

	return strings.Join(strings.Fields(parts.String()), " ")
}
