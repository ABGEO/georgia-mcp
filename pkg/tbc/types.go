package tbc

import "encoding/json"

// CommercialRates is the answer of TBC to a request of commercial rates.
type CommercialRates struct {
	// Base is the currency of the prices: GEL.
	Base                string           `json:"base"`
	CommercialRatesList []CommercialRate `json:"commercialRatesList"`
}

// CommercialRate is the price in the base currency at which TBC buys and sells one
// unit of a currency. Buy and Sell keep the digits of the answer.
type CommercialRate struct {
	Currency string      `json:"currency"`
	Buy      json.Number `json:"buy"`
	Sell     json.Number `json:"sell"`
}
