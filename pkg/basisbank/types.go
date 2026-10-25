package basisbank

// Day is the rates of Basisbank for one date.
type Day struct {
	// Date is the date of the rates: "2026-10-25".
	Date string
	// XRates is the text of the field xrates of the answer, decoded.
	XRates XRates
}

// XRates is the three sets of prices of a date. Each set maps a code to a price, as the
// text of the answer: "USD": "2.5600".
type XRates struct {
	// Kurs is the official rate of the NBG, for the unit of the NBG: 10 for CNY.
	Kurs map[string]string `json:"kurs"`
	// KursBuy is the price in GEL at which Basisbank buys one unit.
	KursBuy map[string]string `json:"kursBuy"`
	// KursSell is the price in GEL at which Basisbank sells one unit.
	KursSell map[string]string `json:"kursSell"`
}

// wireDay is a date of the answer, with its rates as a JSON text.
type wireDay struct {
	Date   string `json:"date"`
	XRates string `json:"xrates"`
}
