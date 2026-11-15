package tas

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// plainText returns the words of the HTML. A block ends a line. Each line is trimmed, with
// runs of spaces as one space, and an empty line goes.
func plainText(text string) string {
	var out strings.Builder

	tokens := html.NewTokenizer(strings.NewReader(text))

	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return lines(out.String())
		case html.TextToken:
			out.Write(tokens.Text())
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			name, _ := tokens.TagName()
			if isBlock(atom.Lookup(name)) {
				out.WriteByte('\n')
			}
		case html.CommentToken, html.DoctypeToken:
		}
	}
}

func isBlock(tag atom.Atom) bool {
	blocks := []atom.Atom{
		atom.P, atom.Br, atom.Div, atom.Li, atom.Tr,
		atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6,
	}

	return slices.Contains(blocks, tag)
}

func lines(text string) string {
	var kept []string

	for line := range strings.Lines(text) {
		if joined := strings.Join(strings.Fields(line), " "); joined != "" {
			kept = append(kept, joined)
		}
	}

	return strings.Join(kept, "\n")
}
