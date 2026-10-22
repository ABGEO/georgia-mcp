package gulf

// PriceTable is the table of the fuel prices page, as the page shows it. Each price is in
// GEL for one litre.
type PriceTable struct {
	// Columns holds each cell of the head: the date of the change, then one column for
	// each product of each tier.
	Columns []Column
	// Rows holds the text of each cell of each row, in the order of the page:
	// ["2026-10-19", "4.69", "4.39", "4.09", "5.05", "4.99", "4.59", ...].
	Rows [][]string
}

// Column is one cell of the head of the price table.
type Column struct {
	// Name is the label of the cell: "Date", "G-Force Super".
	Name string
	// Class is the class attribute of the cell, as the page writes it. The page marks
	// the date "first", a price of the tier Gulf "g_col", and of the tier Gulf+ "gp_col".
	Class string
}
