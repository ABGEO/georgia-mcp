package tool

type listHolidaysInput struct {
	Year     int    `json:"year,omitempty"     jsonschema:"The year, from 2025 to the next year. The default is the current year. Give a year or from, not both."`
	From     string `json:"from,omitempty"     jsonschema:"A date, YYYY-MM-DD. The tool returns the next holidays on or after it, up to limit."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of holidays with from, 1 to 20. The default is 1. A year returns every holiday."`
	Language string `json:"language,omitempty" jsonschema:"The language of the holiday names: ka or en. The default is en."`
}

type getDayInput struct {
	Date     string `json:"date,omitempty"     jsonschema:"The date, YYYY-MM-DD. The default is today in Georgia."`
	Language string `json:"language,omitempty" jsonschema:"The language of the holiday name: ka or en. The default is en."`
}

type listOutput struct {
	Source   string          `json:"source"   jsonschema:"The source of the data: Nager.Date."`
	AsOf     string          `json:"as_of"    jsonschema:"The instant at which the server read the holidays, RFC 3339."`
	Stale    bool            `json:"stale"    jsonschema:"True when Nager.Date did not answer and the server returns holidays it read before."`
	Holidays []holidayOutput `json:"holidays" jsonschema:"The public holidays, in date order."`
}

type holidayOutput struct {
	Date    string `json:"date"    jsonschema:"The date of the holiday, YYYY-MM-DD."`
	Weekday string `json:"weekday" jsonschema:"The day of the week: monday to sunday."`
	Name    string `json:"name"    jsonschema:"The holiday name."`
}

type dayOutput struct {
	Source  string `json:"source"         jsonschema:"The source of the data: Nager.Date."`
	AsOf    string `json:"as_of"          jsonschema:"The instant at which the server read the holidays, RFC 3339."`
	Stale   bool   `json:"stale"          jsonschema:"True when Nager.Date did not answer and the server returns holidays it read before."`
	Date    string `json:"date"           jsonschema:"The date, YYYY-MM-DD."`
	Weekday string `json:"weekday"        jsonschema:"The day of the week: monday to sunday."`
	Holiday bool   `json:"holiday"        jsonschema:"True when the date is a public holiday."`
	Name    string `json:"name,omitempty" jsonschema:"The holiday name. Empty when the date is not a public holiday."`
}
