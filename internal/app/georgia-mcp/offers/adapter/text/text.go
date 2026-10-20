// Package text turns the HTML fragment of a bank into plain text with the standard
// library. The server module holds no HTML parser.
package text

import (
	"html"
	"regexp"
	"strings"
)

// itemMark stands for the start of a list item until the lines exist. No text of a
// bank holds the character.
const itemMark = "\x00"

var (
	lineEnd     = regexp.MustCompile(`(?i)<br\s*/?>|</p\s*>|</li\s*>`)
	listItem    = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	anyTag      = regexp.MustCompile(`<[^>]*>`)
	midLineItem = regexp.MustCompile(`([^\n])` + itemMark)
	emptyRuns   = regexp.MustCompile(`\n{3,}`)
)

// FromHTML returns the text of an HTML fragment. A paragraph, a break, and a list item
// end a line. A list item starts with "- ". Every other tag goes away, and each entity
// becomes its character. A run of empty lines becomes one empty line.
func FromHTML(fragment string) string {
	plain := strings.ReplaceAll(fragment, itemMark, "")
	plain = lineEnd.ReplaceAllString(plain, "\n")
	plain = listItem.ReplaceAllString(plain, itemMark)
	plain = anyTag.ReplaceAllString(plain, "")
	plain = midLineItem.ReplaceAllString(plain, "$1\n"+itemMark)
	plain = strings.ReplaceAll(html.UnescapeString(plain), " ", " ")

	lines := strings.Split(plain, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if item, ok := strings.CutPrefix(line, itemMark); ok {
			line = "- " + strings.TrimSpace(item)
		}

		lines[i] = line
	}

	return strings.TrimSpace(emptyRuns.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
