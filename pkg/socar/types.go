package socar

import "encoding/json"

type currentPricesResponse struct {
	GetCurrentPrices PriceList `json:"GetCurrentPrices"`
}

// PriceList is the answer of the current prices, with its status.
type PriceList struct {
	// Status is "1" on a success.
	Status  string      `json:"Status"`
	Message string      `json:"Message"`
	Results []FuelPrice `json:"Results"`
}

// FuelPrice is one product of SOCAR. The price is in GEL for one unit: a litre, or a
// cubic metre for CNG. FuelUnitPrice keeps the digits of the answer.
type FuelPrice struct {
	// ActionDate is a date of the record, as SOCAR writes it: "2023-09-23T10:06:54.137",
	// with no zone. It is not the date of the last change of the price.
	ActionDate  string `json:"ActionDate"`
	FuelNameGeo string `json:"FuelNameGeo"`
	// FuelNameEng is the English name as SOCAR writes it, with its spaces: "LPG ".
	FuelNameEng   string      `json:"FuelNameEng"`
	FuelUnitPrice json.Number `json:"FuelUnitPrice"`
	FuelCode      string      `json:"FuelCode"`
}
