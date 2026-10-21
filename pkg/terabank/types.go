package terabank

import "encoding/json"

// CrossRate is the price in GEL at which Terabank buys and sells one unit of a
// currency. The bank calls it a cross rate. Each price keeps the digits of the answer,
// four decimal places.
type CrossRate struct {
	ISO             string      `json:"iso"`
	Buy             json.Number `json:"teraCrossRateBuy"`
	Sell            json.Number `json:"teraCrossRateSell"`
	PreviousDayBuy  json.Number `json:"previousDayTeraCrossRateBuy"`
	PreviousDaySell json.Number `json:"previousDayTeraCrossRateSell"`
	// ShowPreviousDayRates tells the page to show the prices of the day before. The
	// bank spells the field showPreviosDayRates.
	ShowPreviousDayRates bool `json:"showPreviosDayRates"`
}

// envelope is the shape of every answer of the API.
type envelope[T any] struct {
	Data      T      `json:"data"`
	Succeeded bool   `json:"succeeded"`
	Message   string `json:"message"`
}
