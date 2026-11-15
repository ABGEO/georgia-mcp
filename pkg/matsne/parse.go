package matsne

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	blockedTitle   = "Access Denied"
	resultClass    = "document-search-result-item"
	resultInfoID   = "search-result-info"
	metaTableClass = "table-info"
	switcherID     = "publication-switcher"
	documentID     = "maindoc"
	separator      = "●"
	metaCells      = 2
)

var totalPattern = regexp.MustCompile(`:\s*(\d+)\s*\(`)

func parseSearch(body []byte) (SearchPage, error) {
	doc, err := parsePage(body)
	if err != nil {
		return SearchPage{}, err
	}

	info := find(doc, func(node *html.Node) bool { return attr(node, "id") == resultInfoID })
	if info == nil {
		return SearchPage{}, fmt.Errorf("%w: no result count", ErrMalformedPage)
	}

	small := find(info, isElement(atom.Small))
	if small == nil {
		return SearchPage{}, fmt.Errorf("%w: no result count", ErrMalformedPage)
	}

	match := totalPattern.FindStringSubmatch(text(small))
	if match == nil {
		return SearchPage{}, fmt.Errorf("%w: a result count with no number", ErrMalformedPage)
	}

	page := SearchPage{Total: match[1], Items: []SearchItem{}}

	for node := range doc.Descendants() {
		if node.Type == html.ElementNode && hasClass(node, resultClass) {
			page.Items = append(page.Items, searchItem(node))
		}
	}

	return page, nil
}

func searchItem(panel *html.Node) SearchItem {
	item := SearchItem{Class: attr(panel, "class"), Fields: []string{}}

	link := find(panel, func(node *html.Node) bool {
		return node.DataAtom == atom.A && strings.Contains(attr(node, "href"), "/document/view/")
	})
	if link != nil {
		item.ID = idOf(attr(link, "href"))
		item.Title = text(link)
	}

	for node := range panel.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.Small {
			continue
		}

		if find(node, isElement(atom.A)) != nil {
			continue
		}

		if field := text(node); field != "" && field != separator {
			item.Fields = append(item.Fields, field)
		}
	}

	return item
}

// idOf returns the segment after "/document/view/" of a link, with no query.
func idOf(href string) string {
	_, rest, _ := strings.Cut(href, "/document/view/")
	id, _, _ := strings.Cut(rest, "?")
	id, _, _ = strings.Cut(id, "/")

	return id
}

func parseSuggestions(body []byte) ([]Suggestion, error) {
	if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 && trimmed[0] == '<' {
		if _, err := parsePage(body); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("%w: a page in place of the titles", ErrMalformedPage)
	}

	var titles []Suggestion
	if err := json.Unmarshal(body, &titles); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedPage, err)
	}

	return titles, nil
}

func parseDocument(body []byte) (Document, error) {
	doc, err := parsePage(body)
	if err != nil {
		return Document{}, err
	}

	table := find(doc, func(node *html.Node) bool {
		return node.DataAtom == atom.Table && hasClass(node, metaTableClass)
	})
	if table == nil {
		return Document{}, fmt.Errorf("%w: no metadata table", ErrMalformedPage)
	}

	result := Document{Meta: metaRows(table), Paragraphs: paragraphs(doc)}

	if head := find(table, isElement(atom.Th)); head != nil {
		result.Title = text(head)
	}

	result.Publications, result.Selected = publications(doc)

	return result, nil
}

// publications returns each option of the switcher of versions, and the selected value.
func publications(doc *html.Node) ([]Publication, string) {
	list := []Publication{}
	selected := ""

	switcher := find(doc, func(node *html.Node) bool { return attr(node, "id") == switcherID })
	if switcher == nil {
		return list, selected
	}

	for option := range switcher.Descendants() {
		if option.Type != html.ElementNode || option.DataAtom != atom.Option {
			continue
		}

		publication := Publication{Value: attr(option, "value"), Text: text(option)}
		list = append(list, publication)

		if hasAttr(option, "selected") {
			selected = publication.Value
		}
	}

	return list, selected
}

func paragraphs(doc *html.Node) []Paragraph {
	list := []Paragraph{}

	main := find(doc, func(node *html.Node) bool { return attr(node, "id") == documentID })
	if main == nil {
		return list
	}

	for node := range main.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.P {
			list = append(list, paragraph(node))
		}
	}

	return list
}

func metaRows(table *html.Node) []MetaRow {
	rows := []MetaRow{}

	for row := range table.Descendants() {
		if row.Type != html.ElementNode || row.DataAtom != atom.Tr {
			continue
		}

		if find(row, isElement(atom.Select)) != nil {
			continue
		}

		if label, value, ok := twoCells(row); ok {
			rows = append(rows, MetaRow{Label: label, Value: value})
		}
	}

	return rows
}

// twoCells returns the text of the two cells of a row that has exactly two.
func twoCells(row *html.Node) (string, string, bool) {
	var cells []string

	for cell := range row.ChildNodes() {
		if cell.Type == html.ElementNode && cell.DataAtom == atom.Td {
			cells = append(cells, text(cell))
		}
	}

	if len(cells) != metaCells {
		return "", "", false
	}

	return cells[0], cells[1], true
}

// paragraphWriter collects the text of a paragraph and tells whether bold covers it.
type paragraphWriter struct {
	builder  strings.Builder
	hasText  bool
	boldOnly bool
}

func paragraph(node *html.Node) Paragraph {
	writer := &paragraphWriter{boldOnly: true}

	for child := range node.ChildNodes() {
		writer.walk(child, false)
	}

	return Paragraph{
		Class: attr(node, "class"),
		Text:  strings.Join(strings.Fields(writer.builder.String()), " "),
		Bold:  writer.hasText && writer.boldOnly,
	}
}

func (w *paragraphWriter) walk(node *html.Node, bold bool) {
	switch {
	case node.Type == html.TextNode:
		w.builder.WriteString(node.Data)

		if strings.TrimSpace(node.Data) != "" {
			w.mark(bold)
		}
	case node.Type != html.ElementNode, node.DataAtom == atom.Style, node.DataAtom == atom.Script:
	case node.DataAtom == atom.Sup:
		w.superscript(node, bold)
	default:
		bold = bold || node.DataAtom == atom.B || node.DataAtom == atom.Strong

		for child := range node.ChildNodes() {
			w.walk(child, bold)
		}
	}
}

// superscript writes "^" and the text of a sup, with no space before it. A sup that holds
// only invisible characters writes nothing. A space at the end of the sup stays.
func (w *paragraphWriter) superscript(node *html.Node, bold bool) {
	sup := strings.TrimFunc(text(node), invisible)
	if sup == "" {
		return
	}

	before := strings.TrimRightFunc(w.builder.String(), unicode.IsSpace)

	w.builder.Reset()
	w.builder.WriteString(before + "^" + sup)

	if raw := rawText(node); strings.TrimRightFunc(raw, unicode.IsSpace) != raw {
		w.builder.WriteByte(' ')
	}

	w.mark(bold)
}

func invisible(r rune) bool {
	return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r)
}

// rawText returns the text of the descendants of a node as written.
func rawText(node *html.Node) string {
	var builder strings.Builder

	for descendant := range node.Descendants() {
		if descendant.Type == html.TextNode {
			builder.WriteString(descendant.Data)
		}
	}

	return builder.String()
}

func (w *paragraphWriter) mark(bold bool) {
	w.hasText = true
	w.boldOnly = w.boldOnly && bold
}

// parsePage parses an HTML page and returns ErrBlocked for the page of the bot filter.
func parsePage(body []byte) (*html.Node, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedPage, err)
	}

	if title := find(doc, isElement(atom.Title)); title != nil && text(title) == blockedTitle {
		return nil, ErrBlocked
	}

	return doc, nil
}

func isElement(tag atom.Atom) func(*html.Node) bool {
	return func(node *html.Node) bool { return node.DataAtom == tag }
}

// find returns the first descendant element of a node that matches, in the order of the
// document, or nil.
func find(node *html.Node, match func(*html.Node) bool) *html.Node {
	for child := range node.Descendants() {
		if child.Type == html.ElementNode && match(child) {
			return child
		}
	}

	return nil
}

func attr(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}

	return ""
}

func hasAttr(node *html.Node, key string) bool {
	return slices.ContainsFunc(node.Attr, func(a html.Attribute) bool { return a.Key == key })
}

func hasClass(node *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attr(node, "class")), class)
}

// text returns the text of the descendants of a node, with each run of white space as one
// space and no space at the ends. Inline elements join with no space, so a highlighted
// part of a word stays in the word. A block element or a line break separates. It leaves
// out style and script.
func text(node *html.Node) string {
	var builder strings.Builder

	writeText(&builder, node)

	return strings.Join(strings.Fields(builder.String()), " ")
}

func writeText(builder *strings.Builder, node *html.Node) {
	switch {
	case node.Type == html.TextNode:
		builder.WriteString(node.Data)

		return
	case node.DataAtom == atom.Style || node.DataAtom == atom.Script:
		return
	}

	block := isBlock(node)
	if block {
		builder.WriteByte(' ')
	}

	for child := range node.ChildNodes() {
		writeText(builder, child)
	}

	if block {
		builder.WriteByte(' ')
	}
}

func isBlock(node *html.Node) bool {
	return slices.Contains(
		[]atom.Atom{atom.Br, atom.P, atom.Div, atom.Li, atom.Td, atom.Th, atom.Tr, atom.Option},
		node.DataAtom,
	)
}
