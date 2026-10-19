package domain

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Name is a name in Georgian and in English.
type Name struct {
	Georgian string
	English  string
}

// In returns the name in the language, or in the other language when the TTC gives
// none in the requested one.
func (n Name) In(language kernel.Language) string {
	if language == kernel.LanguageGeorgian && n.Georgian != "" || n.English == "" {
		return n.Georgian
	}

	return n.English
}

// StopID is the identifier of a stop: 1:1000.
type StopID string

// RouteID is the identifier of a route: 1:R29925.
type RouteID string

// Stop is a place where a vehicle of the TTC takes passengers.
type Stop struct {
	ID StopID
	// Code is the number on the sign of a bus stop. It is empty for a station.
	Code     string
	Name     Name
	Kind     Kind
	Location Point
}

// Route is one line of the TTC.
type Route struct {
	ID     RouteID
	Number string
	Name   Name
	Kind   Kind
}

// Direction is one way of a route.
type Direction struct {
	// ID is the direction of the TTC: 0 or 1.
	ID       int
	Headsign Name
	// Stops is in the order of travel. The first stop of a loop is also its last.
	Stops []StopID
}

// Arrival is the expected arrival of a vehicle of one route at one stop.
type Arrival struct {
	RouteNumber string
	Kind        Kind
	// Headsign is in the language of the request.
	Headsign string
	// Minutes counts from the read instant. It is 0 or more.
	Minutes int
	// Live is true when the TTC computes the arrival from the position of the vehicle.
	Live     bool
	Expected time.Time
}

// Place is a named point in Tbilisi.
type Place struct {
	Name        string
	Street      string
	HouseNumber string
	District    string
	Location    Point
}
