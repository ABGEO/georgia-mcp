package tas

import "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"

// PlainText exposes the HTML reader to the tests.
func PlainText(html string) string {
	return plainText(html)
}

// StatusOf exposes the table of statuses to the tests.
func StatusOf(id int) (domain.Status, error) {
	return statusOf(id)
}

// ClassOf exposes the reading of a class to the tests.
func ClassOf(name string) string {
	return classOf(name)
}
