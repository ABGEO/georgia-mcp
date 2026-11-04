package tool

type getQualityInput struct {
	City     string `json:"city,omitempty"     jsonschema:"A city of Georgia in English or Georgian, for example Tbilisi or რუსთავი. Leave city and station empty for every station."`
	Station  string `json:"station,omitempty"  jsonschema:"A station code of the NEA, for example TSRT. Give a city or a station, not both."`
	Language string `json:"language,omitempty" jsonschema:"The language of the addresses and the cities: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of stations, 1 to 100. The default is 25."`
}

type listHistoryInput struct {
	Station   string `json:"station"             jsonschema:"A station code of the NEA, for example TSRT."`
	From      string `json:"from,omitempty"      jsonschema:"The first date of the period, YYYY-MM-DD, at most 7 days ago. Give from and to, or neither for the last 24 hours."`
	To        string `json:"to,omitempty"        jsonschema:"The last date of the period, YYYY-MM-DD. At most 7 days with from, counted with both ends."`
	Pollutant string `json:"pollutant,omitempty" jsonschema:"One pollutant: PM10, PM2.5, NO2, O3, SO2, or CO. The default is every pollutant of the station."`
	Language  string `json:"language,omitempty"  jsonschema:"The language of the address and the city: ka or en. The default is en."`
	Limit     int    `json:"limit,omitempty"     jsonschema:"The maximum count of readings of each pollutant, newest first, 1 to 100. The default is 24."`
}

type qualityOutput struct {
	Source   string          `json:"source"   jsonschema:"The source of the data: National Environmental Agency."`
	AsOf     string          `json:"as_of"    jsonschema:"The instant at which the server read the readings, RFC 3339."`
	Stale    bool            `json:"stale"    jsonschema:"True when the NEA did not answer and the server returns readings it read before."`
	Total    int             `json:"total"    jsonschema:"The count of matching stations. More than the count of stations when the limit cut the list."`
	Stations []stationOutput `json:"stations" jsonschema:"The stations, the worst level first. A station with no level comes last."`
}

type stationOutput struct {
	Code     string          `json:"code"            jsonschema:"The station code of the NEA."`
	Address  string          `json:"address"         jsonschema:"The address of the station."`
	City     string          `json:"city"            jsonschema:"The city or the municipality of the station."`
	Lat      float64         `json:"lat"             jsonschema:"The latitude, in degrees."`
	Lon      float64         `json:"lon"             jsonschema:"The longitude, in degrees."`
	Level    string          `json:"level,omitempty" jsonschema:"The worst level of the readings: good, fair, moderate, poor, or very_poor. Empty when no reading has a level."`
	Readings []readingOutput `json:"readings"        jsonschema:"The newest reading of each pollutant, less than 24 hours old. Empty when the station sent none."`
}

type readingOutput struct {
	Pollutant string `json:"pollutant"       jsonschema:"PM10, PM2.5, NO2, O3, SO2, or CO."`
	Value     string `json:"value"           jsonschema:"The concentration as a decimal string, as the NEA publishes it. It can be negative."`
	Unit      string `json:"unit"            jsonschema:"µg/m³, or mg/m³ for CO."`
	Time      string `json:"time"            jsonschema:"The hour of the reading as the NEA publishes it, RFC 3339 with +04:00."`
	Level     string `json:"level,omitempty" jsonschema:"good, fair, moderate, poor, or very_poor. Empty for a negative value."`
}

type historyOutput struct {
	Source  string         `json:"source"  jsonschema:"The source of the data: National Environmental Agency."`
	AsOf    string         `json:"as_of"   jsonschema:"The instant at which the server read the readings, RFC 3339."`
	Stale   bool           `json:"stale"   jsonschema:"True when the NEA did not answer and the server returns readings it read before."`
	Code    string         `json:"code"    jsonschema:"The station code of the NEA."`
	Address string         `json:"address" jsonschema:"The address of the station."`
	City    string         `json:"city"    jsonschema:"The city or the municipality of the station."`
	From    string         `json:"from"    jsonschema:"The first hour of the period, RFC 3339."`
	To      string         `json:"to"      jsonschema:"The last hour of the period, RFC 3339. With no dates, the hour of the newest reading when it is ahead of the clock."`
	Series  []seriesOutput `json:"series"  jsonschema:"One series for each pollutant."`
}

type seriesOutput struct {
	Pollutant string        `json:"pollutant" jsonschema:"PM10, PM2.5, NO2, O3, SO2, or CO."`
	Unit      string        `json:"unit"      jsonschema:"µg/m³, or mg/m³ for CO."`
	Total     int           `json:"total"     jsonschema:"The count of readings in the period. More than the count of readings when the limit cut the list."`
	Readings  []pointOutput `json:"readings"  jsonschema:"The readings, newest first."`
}

type pointOutput struct {
	Value string `json:"value"           jsonschema:"The concentration as a decimal string, as the NEA publishes it."`
	Time  string `json:"time"            jsonschema:"The hour of the reading, RFC 3339 with +04:00."`
	Level string `json:"level,omitempty" jsonschema:"good, fair, moderate, poor, or very_poor. Empty for a negative value."`
}
