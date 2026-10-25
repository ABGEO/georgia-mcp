package mjc

// RateRow is one row of the rate table of the page, as the page shows it. Each price is
// for the count of units that Unit gives, not for one unit.
type RateRow struct {
	// Unit is the count of units and the code, in one cell: "1 USD".
	Unit string
	// Buy is the price in GEL at which MJC buys the unit: "2.59500".
	Buy string
	// Sell is the price in GEL at which MJC sells the unit: "2.59800".
	Sell string
}
