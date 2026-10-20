package tbc

import (
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
)

// treeText returns the text of a tree of the terms. A paragraph and a list item end a
// line, and a list item starts with "- ". A blank line goes away.
func treeText(root *tbcoffers.Node) string {
	if root == nil {
		return ""
	}

	writer := &lineWriter{}
	writer.walk(*root)
	writer.flush()

	return strings.Join(writer.lines, "\n")
}

type lineWriter struct {
	lines   []string
	current strings.Builder
	prefix  string
}

func (w *lineWriter) walk(node tbcoffers.Node) {
	switch node.NodeType {
	case "text":
		w.current.WriteString(node.Value)
	case "list-item":
		w.flush()
		w.prefix = "- "

		w.walkChildren(node)
		w.flush()
	case "paragraph":
		w.walkChildren(node)
		w.flush()
	default:
		w.walkChildren(node)
	}
}

func (w *lineWriter) walkChildren(node tbcoffers.Node) {
	for _, child := range node.Content {
		w.walk(child)
	}
}

// flush ends the current line. A list item keeps its prefix until it has a text.
func (w *lineWriter) flush() {
	line := strings.TrimSpace(w.current.String())
	w.current.Reset()

	if line == "" {
		return
	}

	w.lines = append(w.lines, w.prefix+line)
	w.prefix = ""
}
