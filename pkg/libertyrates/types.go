package libertyrates

// RateRow is one currency of the rate block of the home page, as the page shows it.
// Each price is for Unit, not for one unit.
type RateRow struct {
	// Currency is the code that the page shows: "AMD".
	Currency string
	// Unit is the count and the code that the prices are for: "1000 AMD".
	Unit string
	// Official is the official rate of the NBG for Unit.
	Official string
	// Commercial is the rate of the bank at a branch.
	Commercial Quote
	// InternetBank is the rate of the internet bank. It is empty when the page shows none.
	InternetBank Quote
	// Transfer is the special rate for money transfers. It is empty when the page shows none.
	Transfer Quote
}

// Quote is a buy price and a sell price, as the page shows them.
type Quote struct {
	Buy  string
	Sell string
}
