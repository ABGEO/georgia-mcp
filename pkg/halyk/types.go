package halyk

// RateRow is one row of the table of the current rates, as the page shows it. Each
// price is for one unit.
type RateRow struct {
	// Currency is the code that the page shows: "USD".
	Currency string
	// Buy is the price in GEL at which the bank buys one unit: "2.5870".
	Buy string
	// Sell is the price in GEL at which the bank sells one unit: "2.6120".
	Sell string
}
