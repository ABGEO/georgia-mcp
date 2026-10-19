package ttc

// Stop is a stop of the TTC: a bus stop, a metro station, or a cable car station.
type Stop struct {
	ID string `json:"id"`
	// Code is the number on the sign of a bus stop. It is nil for a station of metro
	// or cable car.
	Code *string `json:"code"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	// VehicleMode is BUS, SUBWAY, or GONDOLA. A minibus stops at a BUS stop.
	VehicleMode string `json:"vehicleMode"`
}

// Route is one line of the TTC. The TTC gives a minibus route the mode BUS.
type Route struct {
	ID        string `json:"id"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
	Color     string `json:"color"`
	Mode      string `json:"mode"`
}

// RouteDetail is a route with its patterns: one for each direction.
type RouteDetail struct {
	ID                   string    `json:"id"`
	ShortName            string    `json:"shortName"`
	Color                string    `json:"color"`
	Mode                 string    `json:"mode"`
	Patterns             []Pattern `json:"patterns"`
	DefaultPatternSuffix string    `json:"defaultPatternSuffix"`
}

// Pattern is one direction of a route. PatternSuffix names it in the other calls of
// the route.
type Pattern struct {
	PatternSuffix string  `json:"patternSuffix"`
	DirectionID   int     `json:"directionId"`
	FirstStop     StopRef `json:"firstStop"`
	LastStop      StopRef `json:"lastStop"`
	Headsign      string  `json:"headsign"`
}

// StopRef names a stop inside another answer.
type StopRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ScheduleGroup is the timetable of the days from FromDay to ToDay, which share the
// same times.
type ScheduleGroup struct {
	// FromDay and ToDay are English names of a weekday in uppercase: MONDAY.
	FromDay string `json:"fromDay"`
	ToDay   string `json:"toDay"`
	// ServiceDates holds each date of the group in the current week, as YYYY-MM-DD.
	ServiceDates []string       `json:"serviceDates"`
	Stops        []ScheduleStop `json:"stops"`
}

// ScheduleStop is the times of one stop of a pattern.
type ScheduleStop struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Position int    `json:"position"`
	// ArrivalTimes is a list of H:MM separated by commas: "6:45,6:55". An hour of 24
	// or more is after midnight.
	ArrivalTimes string `json:"arrivalTimes"`
}

// ArrivalTimesQuery is the query of the arrivals at a stop.
type ArrivalTimesQuery struct {
	Locale string
	// RouteID asks the arrivals of one route. Empty asks the next arrival of each route.
	RouteID string
	// Max is the maximum count of arrivals. 0 asks the default of the TTC.
	Max int
}

// ArrivalTime is the expected arrival of one vehicle at a stop.
type ArrivalTime struct {
	ShortName     string `json:"shortName"`
	Color         string `json:"color"`
	Headsign      string `json:"headsign"`
	PatternSuffix string `json:"patternSuffix"`
	VehicleMode   string `json:"vehicleMode"`
	// Realtime is true when the TTC tracks the vehicle.
	Realtime               bool `json:"realtime"`
	RealtimeArrivalMinutes int  `json:"realtimeArrivalMinutes"`
	// ScheduledArrivalMinutes comes from the timetable. It can be negative while
	// Realtime is true.
	ScheduledArrivalMinutes int `json:"scheduledArrivalMinutes"`
}

// PlanQuery is the query of a trip plan.
type PlanQuery struct {
	FromLat float64
	FromLon float64
	ToLat   float64
	ToLon   float64
	// DepartMode is leaveNow, departAt, or arriveBy.
	DepartMode string
	// Date is YYYY-MM-DD and Time is HH:MM. The client sends them only when DepartMode
	// is not leaveNow.
	Date string
	Time string
	// Modes holds WALK, BUS, SUBWAY, and GONDOLA.
	Modes []string
	// Optimize is quick, lessWalking, or fewerTransfers.
	Optimize string
	Locale   string
}

// Plan is the answer of a trip plan.
type Plan struct {
	Itineraries []Itinerary `json:"itineraries"`
}

// Itinerary is one way to travel between the two points.
type Itinerary struct {
	// StartTime and EndTime are RFC 3339 in UTC: "2026-10-19T17:42:48.000+00:00".
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	// Duration and WalkTime are in seconds.
	Duration int `json:"duration"`
	WalkTime int `json:"walkTime"`
	// WalkDistance is in metres.
	WalkDistance float64 `json:"walkDistance"`
	Legs         []Leg   `json:"legs"`
}

// Leg is a walk, or a ride on one route.
type Leg struct {
	From      Place  `json:"from"`
	To        Place  `json:"to"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	// Duration is in seconds.
	Duration int `json:"duration"`
	// Mode is WALK, BUS, SUBWAY, or GONDOLA.
	Mode string `json:"mode"`
	// Distance is in metres.
	Distance float64 `json:"distance"`
	// Route is nil for a walk.
	Route             *LegRoute `json:"route"`
	IntermediateStops []Stop    `json:"intermediateStops"`
	RealTime          bool      `json:"realTime"`
}

// Place is an end of a leg: a stop, or a point of the query.
type Place struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

// LegRoute is the route of a ride. The TTC gives no identifier.
type LegRoute struct {
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
	Color     string `json:"color"`
}

// GeocodeQuery is the query of a place search.
type GeocodeQuery struct {
	Query  string
	Locale string
	// BBox is the box of the search: west,north,east,south.
	BBox string
}

// FeatureCollection is the answer of a place search, in GeoJSON.
type FeatureCollection struct {
	Features []Feature `json:"features"`
}

// Feature is one place of OpenStreetMap.
type Feature struct {
	Geometry   Geometry          `json:"geometry"`
	Properties FeatureProperties `json:"properties"`
}

// Geometry is the point of a feature.
type Geometry struct {
	// Coordinates is [longitude, latitude].
	Coordinates []float64 `json:"coordinates"`
}

// FeatureProperties is the address of a feature.
type FeatureProperties struct {
	Name        string `json:"name"`
	Street      string `json:"street"`
	HouseNumber string `json:"housenumber"`
	District    string `json:"district"`
	Locality    string `json:"locality"`
	City        string `json:"city"`
}
