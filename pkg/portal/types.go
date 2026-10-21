package portal

import "encoding/json"

// FuelPrice is one entry of the price list "fuelPricesData" of the page. Each price is
// in GEL for one litre, as text: "3.94".
type FuelPrice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Type is "PETROL" or "DIESEL".
	Type string `json:"type"`
	// Octane is the octane number of a petrol, "93.7", and 0 for a diesel.
	Octane      json.Number `json:"octane"`
	LatestPrice string      `json:"latestPrice"`
	// LatestOnlinePrice is the price of a QR liter that a customer buys online. The page
	// does not show it.
	LatestOnlinePrice string   `json:"latestOnlinePrice"`
	FuelType          FuelType `json:"fuelType"`
}

// FuelType is the product of an entry.
type FuelType struct {
	// Slug is the stable name of the product: "euro-regular".
	Slug string `json:"slug"`
}
