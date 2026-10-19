package rompetrol

// PriceRow is one row of the body of the price table, as the page shows it. Each run
// of spaces in a cell is one space.
type PriceRow struct {
	// Product is the name in the first cell: "efix ევრო რეგულარი". A row of a
	// self-service price starts with "- ": "- ევრო რეგულარი (თვითმომსახურების სადგურებზე)".
	Product string
	// Price is the price in GEL for one litre in the second cell: "4.02".
	Price string
}
