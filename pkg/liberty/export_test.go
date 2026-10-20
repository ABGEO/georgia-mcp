package liberty

// ParsePage exposes the parse of the page of an offer to the fuzz test.
func ParsePage(body []byte) (Page, error) {
	return parsePage(body)
}
