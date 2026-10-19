package wissol

import "encoding/json"

// FuelPrice is one product of Wissol. Each price is in GEL for one unit: a litre, or
// a cubic metre for the gas. Price and PriceSelfService keep the digits of the answer.
type FuelPrice struct {
	ID          int    `json:"id"`
	ProductCode string `json:"productCode"`
	// FuelType is the Georgian name.
	FuelType    string `json:"fuelType"`
	FuelTypeEng string `json:"fuelTypeEng"`
	FuelTypeRus string `json:"fuelTypeRus"`
	IsActive    bool   `json:"isActive"`
	IsEco       bool   `json:"isEco"`
	// Price is the price at a station with an attendant.
	Price json.Number `json:"price"`
	// PriceSelfService is the price at a self-service station. It is empty when
	// Wissol sends no value: a product that the self-service stations do not sell.
	PriceSelfService json.Number `json:"priceSelfService"`
	AdditionalInfo   string      `json:"additionalInfo"`
	// Date is the day of the last change of the price, as Wissol writes it:
	// "2026-10-08T00:00:00", with no zone.
	Date    string `json:"date"`
	OrderID int    `json:"orderId"`
}
