package bog

import "encoding/json"

// CommercialRate is the price in GEL at which Bank of Georgia buys and sells one unit
// of a currency. Buy and Sell keep the digits of the answer, eight decimal places.
type CommercialRate struct {
	Buy  json.Number `json:"Buy"`
	Sell json.Number `json:"Sell"`
}
