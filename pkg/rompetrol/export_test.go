package rompetrol

// ParsePrices exposes the parse of a page to the fuzz test.
func ParsePrices(body []byte) ([]PriceRow, error) {
	return parsePrices(body)
}
