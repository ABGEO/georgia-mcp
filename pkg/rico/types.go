package rico

// RateRow is one row of the rate table of the home page, as the page shows it. Each
// price is for Unit units, not for one unit.
type RateRow struct {
	// Currency is the code that the page shows: "PLN", or "USD-EUR" for a cross rate.
	Currency string
	// Unit is the count of units that the prices are for: "10".
	Unit string
	Buy  string
	Sell string
}
