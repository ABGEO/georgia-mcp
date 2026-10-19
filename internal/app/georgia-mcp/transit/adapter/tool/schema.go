package tool

type searchStopsInput struct {
	Query    string `json:"query"              jsonschema:"A stop name or a stop code, in Georgian or English."`
	Kind     string `json:"kind,omitempty"     jsonschema:"Only stops of this kind: bus, metro, or cable_car. Empty gives every kind."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of stops, 1 to 50. The default is 10."`
}

type nearbyStopsInput struct {
	Lat      float64 `json:"lat"                jsonschema:"The latitude of the point, in degrees."`
	Lon      float64 `json:"lon"                jsonschema:"The longitude of the point, in degrees."`
	RadiusM  int     `json:"radius_m,omitempty" jsonschema:"The distance from the point in metres, 1 to 2000. The default is 500."`
	Kind     string  `json:"kind,omitempty"     jsonschema:"Only stops of this kind: bus, metro, or cable_car. Empty gives every kind."`
	Language string  `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
	Limit    int     `json:"limit,omitempty"    jsonschema:"The maximum count of stops, 1 to 50. The default is 10."`
}

type stopsOutput struct {
	Source string       `json:"source" jsonschema:"The source of the data: TTC."`
	AsOf   string       `json:"as_of"  jsonschema:"The instant at which the server read the stops, RFC 3339."`
	Stale  bool         `json:"stale"  jsonschema:"True when the TTC did not answer and the server returns stops it read before."`
	Stops  []stopOutput `json:"stops"  jsonschema:"The matching stops, best match first."`
}

type nearbyOutput struct {
	Source string             `json:"source" jsonschema:"The source of the data: TTC."`
	AsOf   string             `json:"as_of"  jsonschema:"The instant at which the server read the stops, RFC 3339."`
	Stale  bool               `json:"stale"  jsonschema:"True when the TTC did not answer and the server returns stops it read before."`
	Stops  []nearbyStopOutput `json:"stops"  jsonschema:"The stops within the distance, nearest first."`
}

type stopOutput struct {
	ID   string  `json:"id"             jsonschema:"The stop id, for example 1:1000. Every transit tool that takes a stop accepts it."`
	Code string  `json:"code,omitempty" jsonschema:"The number on the sign of a bus stop. Absent for a station of metro or cable car."`
	Name string  `json:"name"           jsonschema:"The name of the stop."`
	Kind string  `json:"kind"           jsonschema:"bus, metro, or cable_car. A minibus stops at a bus stop."`
	Lat  float64 `json:"lat"            jsonschema:"The latitude of the stop, in degrees."`
	Lon  float64 `json:"lon"            jsonschema:"The longitude of the stop, in degrees."`
}

type nearbyStopOutput struct {
	Stop      stopOutput `json:"stop"       jsonschema:"The stop."`
	DistanceM int        `json:"distance_m" jsonschema:"The distance from the point in metres, rounded."`
}

type listArrivalsInput struct {
	Stop     string `json:"stop"               jsonschema:"The stop id (1:1000) or the stop code (1000)."`
	Route    string `json:"route,omitempty"    jsonschema:"Only this route: its number (385) or its id. Empty gives the next vehicle of each route."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of arrivals, 1 to 30. The default is 10."`
}

type arrivalsOutput struct {
	Source   string          `json:"source"         jsonschema:"The source of the data: TTC."`
	AsOf     string          `json:"as_of"          jsonschema:"The instant at which the server read the arrivals, RFC 3339. minutes count from it."`
	Stop     stopOutput      `json:"stop"           jsonschema:"The stop."`
	Arrivals []arrivalOutput `json:"arrivals"       jsonschema:"The next arrivals, soonest first."`
	Note     string          `json:"note,omitempty" jsonschema:"Why the list is empty."`
}

type arrivalOutput struct {
	Route    string `json:"route"    jsonschema:"The route number."`
	Kind     string `json:"kind"     jsonschema:"bus, minibus, metro, or cable_car."`
	Headsign string `json:"headsign" jsonschema:"The destination that the vehicle shows."`
	Minutes  int    `json:"minutes"  jsonschema:"The minutes from as_of to the arrival."`
	Expected string `json:"expected" jsonschema:"The expected instant of the arrival, RFC 3339."`
	Live     bool   `json:"live"     jsonschema:"True when the TTC tracks the vehicle. False when the time comes from the timetable."`
}

type searchRoutesInput struct {
	Query    string `json:"query,omitempty"    jsonschema:"A route number or a part of a route name. Empty gives every route."`
	Kind     string `json:"kind,omitempty"     jsonschema:"Only routes of this kind: bus, minibus, metro, or cable_car."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of routes, 1 to 100. The default is 20."`
}

type routesOutput struct {
	Source string        `json:"source" jsonschema:"The source of the data: TTC."`
	AsOf   string        `json:"as_of"  jsonschema:"The instant at which the server read the routes, RFC 3339."`
	Stale  bool          `json:"stale"  jsonschema:"True when the TTC did not answer and the server returns routes it read before."`
	Routes []routeOutput `json:"routes" jsonschema:"The matching routes, best match first."`
}

type routeOutput struct {
	ID     string `json:"id"     jsonschema:"The route id."`
	Number string `json:"number" jsonschema:"The route number that the vehicle shows."`
	Name   string `json:"name"   jsonschema:"The name of the route, usually its two ends."`
	Kind   string `json:"kind"   jsonschema:"bus, minibus, metro, or cable_car."`
}

type getRouteInput struct {
	Route    string `json:"route"              jsonschema:"The route number (385) or the route id."`
	Kind     string `json:"kind,omitempty"     jsonschema:"The kind, when two routes share the number: bus, minibus, metro, or cable_car."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
}

type routeDetailOutput struct {
	Source     string            `json:"source"     jsonschema:"The source of the data: TTC."`
	AsOf       string            `json:"as_of"      jsonschema:"The instant at which the server read the route, RFC 3339."`
	Stale      bool              `json:"stale"      jsonschema:"True when the TTC did not answer and the server returns a route it read before."`
	Route      routeOutput       `json:"route"      jsonschema:"The route."`
	Directions []directionOutput `json:"directions" jsonschema:"Each direction of the route. A loop has one."`
}

type directionOutput struct {
	Direction int          `json:"direction" jsonschema:"The direction number, for transit_get_timetable."`
	Headsign  string       `json:"headsign"  jsonschema:"The destination that the vehicle shows."`
	Stops     []stopOutput `json:"stops"     jsonschema:"The stops in the order of travel."`
}

type getTimetableInput struct {
	Route     string `json:"route"               jsonschema:"The route number (385) or the route id."`
	Stop      string `json:"stop"                jsonschema:"The stop id or the stop code. The stop must be on the direction."`
	Kind      string `json:"kind,omitempty"      jsonschema:"The kind, when two routes share the number: bus, minibus, metro, or cable_car."`
	Direction *int   `json:"direction,omitempty" jsonschema:"The direction number from transit_get_route. Required when the route has two directions."`
	Date      string `json:"date,omitempty"      jsonschema:"The date, YYYY-MM-DD, in the current week. The default is today in Tbilisi."`
	Language  string `json:"language,omitempty"  jsonschema:"The language of the names: ka or en. The default is en."`
}

type timetableOutput struct {
	Source    string      `json:"source"    jsonschema:"The source of the data: TTC."`
	AsOf      string      `json:"as_of"     jsonschema:"The instant at which the server read the timetable, RFC 3339."`
	Stale     bool        `json:"stale"     jsonschema:"True when the TTC did not answer and the server returns a timetable it read before."`
	Route     routeOutput `json:"route"     jsonschema:"The route."`
	Direction int         `json:"direction" jsonschema:"The direction number."`
	Headsign  string      `json:"headsign"  jsonschema:"The destination of the direction."`
	Stop      stopOutput  `json:"stop"      jsonschema:"The stop."`
	Date      string      `json:"date"      jsonschema:"The date of the times, YYYY-MM-DD."`
	Times     []string    `json:"times"     jsonschema:"The planned departure times, HH:MM, in order. 24:00 or later is after midnight."`
}

type pointInput struct {
	Lat float64 `json:"lat" jsonschema:"The latitude, in degrees."`
	Lon float64 `json:"lon" jsonschema:"The longitude, in degrees."`
}

type planTripInput struct {
	From       pointInput `json:"from"                 jsonschema:"The start point in Tbilisi."`
	To         pointInput `json:"to"                   jsonschema:"The end point in Tbilisi."`
	Timing     string     `json:"timing,omitempty"     jsonschema:"leave_now, depart_at, or arrive_by. The default is leave_now."`
	Date       string     `json:"date,omitempty"       jsonschema:"The date of depart_at or arrive_by, YYYY-MM-DD. The default is today in Tbilisi."`
	Time       string     `json:"time,omitempty"       jsonschema:"The time of depart_at or arrive_by, HH:MM in Tbilisi. Required with them."`
	Kinds      []string   `json:"kinds,omitempty"      jsonschema:"The kinds to ride: bus, minibus, metro, cable_car. bus and minibus go together. Empty gives every kind. A walk is always allowed."`
	Preference string     `json:"preference,omitempty" jsonschema:"fastest, less_walking, or fewer_transfers. The default is fastest."`
	Language   string     `json:"language,omitempty"   jsonschema:"The language of the names: ka or en. The default is en."`
}

type tripOutput struct {
	Source      string            `json:"source"         jsonschema:"The source of the data: TTC."`
	AsOf        string            `json:"as_of"          jsonschema:"The instant at which the server read the plan, RFC 3339."`
	Itineraries []itineraryOutput `json:"itineraries"    jsonschema:"Up to 3 ways to travel, in the order of the TTC."`
	Note        string            `json:"note,omitempty" jsonschema:"Why the list is empty."`
}

type itineraryOutput struct {
	Start           string      `json:"start"            jsonschema:"The instant of departure, RFC 3339."`
	End             string      `json:"end"              jsonschema:"The instant of arrival, RFC 3339."`
	DurationMinutes int         `json:"duration_minutes" jsonschema:"The duration in minutes, rounded up."`
	WalkM           int         `json:"walk_m"           jsonschema:"The total distance of walk in metres, rounded."`
	Transfers       int         `json:"transfers"        jsonschema:"The count of changes from one vehicle to another."`
	Legs            []legOutput `json:"legs"             jsonschema:"The parts of the trip, in order."`
}

type legOutput struct {
	Kind        string         `json:"kind"                   jsonschema:"walk, bus, minibus, metro, or cable_car."`
	From        legPointOutput `json:"from"                   jsonschema:"Where the leg starts."`
	To          legPointOutput `json:"to"                     jsonschema:"Where the leg ends."`
	Start       string         `json:"start"                  jsonschema:"The instant at which the leg starts, RFC 3339."`
	End         string         `json:"end"                    jsonschema:"The instant at which the leg ends, RFC 3339."`
	DistanceM   int            `json:"distance_m"             jsonschema:"The distance of the leg in metres, rounded."`
	RouteNumber string         `json:"route_number,omitempty" jsonschema:"The route number of a ride."`
	RouteName   string         `json:"route_name,omitempty"   jsonschema:"The route name of a ride."`
	Stops       int            `json:"stops,omitempty"        jsonschema:"The count of stops of a ride after the boarding stop."`
}

type legPointOutput struct {
	Name   string  `json:"name"              jsonschema:"The name of the stop or the point."`
	StopID string  `json:"stop_id,omitempty" jsonschema:"The stop id. Absent when the point is not a stop."`
	Lat    float64 `json:"lat"               jsonschema:"The latitude, in degrees."`
	Lon    float64 `json:"lon"               jsonschema:"The longitude, in degrees."`
}

type searchPlacesInput struct {
	Query    string `json:"query"              jsonschema:"An address, a street, or a building in Tbilisi."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of places, 1 to 20. The default is 5."`
}

type placesOutput struct {
	Source string        `json:"source"         jsonschema:"The source of the data: TTC, from OpenStreetMap."`
	AsOf   string        `json:"as_of"          jsonschema:"The instant at which the server read the places, RFC 3339."`
	Stale  bool          `json:"stale"          jsonschema:"True when the TTC did not answer and the server returns places it read before."`
	Places []placeOutput `json:"places"         jsonschema:"The matching places, in the order of the TTC."`
	Note   string        `json:"note,omitempty" jsonschema:"Why the list is empty."`
}

type placeOutput struct {
	Name        string  `json:"name"                   jsonschema:"The name of the place."`
	Street      string  `json:"street,omitempty"       jsonschema:"The street."`
	HouseNumber string  `json:"house_number,omitempty" jsonschema:"The house number."`
	District    string  `json:"district,omitempty"     jsonschema:"The district of Tbilisi."`
	Lat         float64 `json:"lat"                    jsonschema:"The latitude, in degrees."`
	Lon         float64 `json:"lon"                    jsonschema:"The longitude, in degrees."`
}
