package connect

import "encoding/json"

// Snapshot is the answer of the current prices. Connect writes one snapshot each day at
// 04:00, and one more at each change that it enters by hand.
type Snapshot struct {
	// Date is the day of the snapshot, "2026-10-21". It is not the day of the last
	// change of a price.
	Date     string `json:"date"`
	IsManual bool   `json:"is_manual"`
	// Updated is the instant of the snapshot, as Connect writes it: "2026-10-21 04:00:02".
	Updated string `json:"updated"`
	// Prices holds each product under its key: "super_ron98".
	Prices map[string]FuelPrice `json:"prices"`
}

// FuelPrice is one product of Connect, with its prices in GEL for one litre. Each price
// keeps the digits of the answer: the full digits of a binary float,
// "4.3499999999999996447286321199499070644378662109375" for 4.35.
type FuelPrice struct {
	// Label is the Georgian name of the product: "სუპერი RON 98".
	Label    string      `json:"label"`
	Standard json.Number `json:"standard"`
	// SelfService is the price at a self-service station. It is "" when the answer holds
	// null or no field.
	SelfService json.Number `json:"self_service"`
}
