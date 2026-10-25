package crystal

import "encoding/json"

// Rate is one pair of currencies of an answer. Each price is for Multiplier units of
// SourceISO, in TargetISO. The type omits the official rate of the NBG.
type Rate struct {
	// SourceISO is the code of the currency that Crystal buys and sells: "USD".
	SourceISO string `json:"SourceISO"`
	// TargetISO is the code of the currency of the prices: "GEL".
	TargetISO string `json:"TargetISO"`
	// Multiplier is the count of units of each price: 1, 10, or 1000.
	Multiplier json.Number `json:"Multiplier"`
	// BuyRate is the price at which Crystal buys the units: 2.595.
	BuyRate json.Number `json:"BuyRate"`
	// SellRate is the price at which Crystal sells the units: 2.599.
	SellRate json.Number `json:"SellRate"`
}

// envelope is the outer envelope of an answer. Data is the inner envelope.
type envelope struct {
	ErrorCode string `json:"errorCode"`
	ErrorText string `json:"errorText"`
	Data      *inner `json:"data"`
}

type inner struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Success      bool   `json:"success"`
	Data         []Rate `json:"data"`
}
