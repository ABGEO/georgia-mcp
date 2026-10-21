package kursi

import "encoding/json"

// Currency is one pair of currencies, with the rates of Kursi.ge for it. For the base
// GEL, each rate is the price in GEL of one unit of the secondary currency: BuyRate is
// what Kursi.ge pays, SellRate is what it asks. Each price keeps the digits of the
// answer, four decimal places.
type Currency struct {
	BaseCurrencyCode      string      `json:"baseCurrencyCode"`
	SecondaryCurrencyCode string      `json:"secondaryCurrencyCode"`
	Name                  string      `json:"name"`
	NBGRate               json.Number `json:"nbgRate"`
	Diff                  json.Number `json:"diff"`
	BuyRate               json.Number `json:"buyRate"`
	SellRate              json.Number `json:"sellRate"`
	// BankRates holds the rates of the banks that Kursi.ge repeats, by name: TBC, BOG.
	BankRates map[string]BankRate `json:"bankRates"`
}

// BankRate is the rate of a bank that Kursi.ge repeats.
type BankRate struct {
	BuyRate  json.Number `json:"buyRate"`
	SellRate json.Number `json:"sellRate"`
}
