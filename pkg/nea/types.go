package nea

import "encoding/json"

// HourlyQuery selects the hourly values. An empty field asks for every value.
type HourlyQuery struct {
	// From and To are local times of Tbilisi: "2026-11-04T00:00:00".
	From, To string
	// Station is a station code: "TSRT".
	Station string
	// Municipality is the ID of a municipality: "51".
	Municipality string
	// Substance is the name of a pollutant: "PM10".
	Substance string
}

// Station is one automatic station with its equipment.
type Station struct {
	Code           string      `json:"code"`
	MunicipalityID int         `json:"municipality_id"`
	AddressGE      string      `json:"st_full_address_ge"`
	AddressEN      string      `json:"st_full_address_en"`
	Lat            float64     `json:"lat"`
	Long           float64     `json:"long"`
	Equipment      []Equipment `json:"stationequipment_set"`
}

// Equipment is the measure of one substance at a station, with its hourly values.
type Equipment struct {
	Substance Substance     `json:"substance"`
	Values    []HourlyValue `json:"data1hour_set"`
}

// Substance is a pollutant with its index levels.
type Substance struct {
	Name   string       `json:"name"`
	Levels []IndexLevel `json:"airqualityindexlevel_set"`
}

// IndexLevel holds the upper limit of each level of a substance.
type IndexLevel struct {
	GoodTo     json.Number `json:"good_to"`
	FairTo     json.Number `json:"fair_to"`
	ModerateTo json.Number `json:"moderate_to"`
	PoorTo     json.Number `json:"poor_to"`
}

// HourlyValue is the value of one hour. Value is empty when the NEA sends null.
type HourlyValue struct {
	Value    json.Number `json:"value"`
	DateTime string      `json:"date_time"`
}

// Municipality is one municipality of Georgia.
type Municipality struct {
	ID         int    `json:"id"`
	DistrictGE string `json:"district_ge"`
	DistrictEN string `json:"district_en"`
}
