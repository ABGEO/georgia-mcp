package gpost

// ParseRows exposes the parse of a search answer to the fuzz test.
func ParseRows(body []byte) ([]PostalCodeRow, error) {
	return parseRows(body)
}
