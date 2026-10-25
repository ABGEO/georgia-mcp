package mbc

import "encoding/json"

// Rate is one pair of currencies of an answer. Each price is for Multiplier units of
// FromCcy, in ToCcy.
type Rate struct {
	// FromCcy is the code of the currency that MBC buys and sells: "USD".
	FromCcy string `json:"FromCcy"`
	// ToCcy is the code of the currency of the prices: "GEL".
	ToCcy string `json:"ToCcy"`
	// Multiplier is the count of units of each price: 1.
	Multiplier json.Number `json:"Multiplier"`
	// Buy is the price at which MBC buys the units, with the digits of the answer: 2.5950.
	Buy json.Number `json:"Buy"`
	// Sell is the price at which MBC sells the units, with the digits of the answer: 2.5990.
	Sell json.Number `json:"Sell"`
}

// envelope is the answer of the method. Its timestamp is the instant of the answer, not
// of a change of a price, so the client does not read it.
type envelope struct {
	FXRates []Rate `json:"FXRates"`
}
