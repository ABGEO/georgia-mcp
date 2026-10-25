package halyk

// ParseRates exposes the parse of a page to the fuzz test.
func ParseRates(body []byte) ([]RateRow, error) {
	return parseRates(body)
}
