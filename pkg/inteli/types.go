package inteli

// RateRow is one row of the rate table of the home page, as the page shows it. The table
// names no column: the cells are in the order of the page. Each price is for one unit.
type RateRow struct {
	// Base is the code of the currency of the prices, in the first cell: "GEL".
	Base string
	// Currency is the code of the currency that the row prices, in the second cell: "USD".
	Currency string
	// Buy is the first price, at which Inteli Express buys one unit: "2.595".
	Buy string
	// Sell is the second price, at which Inteli Express sells one unit: "2.598".
	Sell string
}
