package credo

import "encoding/json"

// Rate is one pair of currencies of an answer. Each price is for Multiplier units of
// FromCcy, in ToCcy.
type Rate struct {
	// FromCcy is the code of the currency that the bank buys and sells: "USD".
	FromCcy string `json:"fromCcy"`
	// ToCcy is the code of the currency of the prices: "GEL".
	ToCcy string `json:"toCcy"`
	// Multiplier is the count of units of each price: 1.
	Multiplier json.Number `json:"multiplier"`
	// Buy is the price at which the bank buys the units: 2.593.
	Buy json.Number `json:"buy"`
	// Sell is the price at which the bank sells the units: 2.607.
	Sell json.Number `json:"sell"`
}

type envelope struct {
	Success bool   `json:"success"`
	Rates   []Rate `json:"rates"`
}
