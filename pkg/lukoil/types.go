package lukoil

// PriceTable is the table of the price history page, as the page shows it. Each price is
// in GEL for one litre.
type PriceTable struct {
	// Header is the text of each cell of the head: "#", one name for each product, and
	// "შეცვლის თარიღი", the instant of the change.
	Header []string
	// Rows holds the text of each cell of each row, in the order of the page:
	// ["106", "0.00", "4.60", "4.25", "3.95", "4.97", "2026-10-05 10:01:46"].
	Rows [][]string
}
