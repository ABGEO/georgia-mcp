package tool

type getGridInput struct {
	Moment string `json:"moment,omitempty" jsonschema:"A moment in the last 31 days: 2026-11-16T13:00 for a time of Tbilisi, or RFC 3339 with an offset, for example 2026-11-16T09:00:00Z. Leave it empty for the newest grid state."`
}

type gridOutput struct {
	Source         string           `json:"source"              jsonschema:"The source of the data: Georgian State Electrosystem (GSE)."`
	AsOf           string           `json:"as_of"               jsonschema:"The instant at which the server read the grid state, RFC 3339."`
	Stale          bool             `json:"stale"               jsonschema:"True when GSE did not answer and the server returns a grid state it read before."`
	Time           string           `json:"time"                jsonschema:"The time of the grid slot as GSE stamps it, RFC 3339 with +04:00. It can be before the moment of the request when GSE published no newer slot."`
	AverageMinutes int              `json:"average_minutes"     jsonschema:"The length of the average of each power value, in minutes: 3."`
	Generation     generationOutput `json:"generation"          jsonschema:"The generation by type of power plant, in MW."`
	Consumption    valueOutput      `json:"consumption"         jsonschema:"The consumption of Georgia, in MW."`
	Flows          []flowOutput     `json:"flows"               jsonschema:"The flow on the lines to each country, in MW. A positive flow is an import."`
	NetImport      netOutput        `json:"net_import"          jsonschema:"The sum of every flow, in MW. A negative net import is an export."`
	Measured       int              `json:"measured"            jsonschema:"The count of the values with the quality code ACT."`
	Values         int              `json:"values"              jsonschema:"The count of the values that GSE published for the slot."`
	Frequency      *frequencyOutput `json:"frequency,omitempty" jsonschema:"The frequency of the grid at the minute of the slot. Absent when GSE published none."`
}

type generationOutput struct {
	Hydro   valueOutput `json:"hydro"   jsonschema:"The hydro power plants."`
	Thermal valueOutput `json:"thermal" jsonschema:"The thermal power plants."`
	Wind    valueOutput `json:"wind"    jsonschema:"The wind power plants."`
	Solar   valueOutput `json:"solar"   jsonschema:"The solar power plants."`
	Total   valueOutput `json:"total"   jsonschema:"The total generation, as GSE publishes it."`
}

type valueOutput struct {
	Value   string `json:"value"   jsonschema:"The power in MW as a decimal string, as GSE publishes it."`
	Quality string `json:"quality" jsonschema:"The quality code of GSE: ACT, SUB, NRE, BLS, or another code."`
	Meaning string `json:"meaning" jsonschema:"The meaning of the code: measured, substituted by the operator, not received, not documented by GSE, or unknown."`
}

type flowOutput struct {
	Country   string       `json:"country"           jsonschema:"The country at the other end of the lines, as GSE names it."`
	Value     string       `json:"value"             jsonschema:"The flow in MW as a decimal string. The sum of the lines when the country has more than one."`
	Direction string       `json:"direction"         jsonschema:"import, export, or none."`
	Quality   string       `json:"quality,omitempty" jsonschema:"The quality code of a country with one line. Absent for a sum."`
	Meaning   string       `json:"meaning,omitempty" jsonschema:"The meaning of the quality code. Absent for a sum."`
	Lines     []lineOutput `json:"lines,omitempty"   jsonschema:"Each line of a country with more than one, as GSE names it."`
}

type lineOutput struct {
	Name      string `json:"name"      jsonschema:"The name of the line at GSE, for example Salkhino."`
	Value     string `json:"value"     jsonschema:"The flow in MW as a decimal string, as GSE publishes it."`
	Direction string `json:"direction" jsonschema:"import, export, or none."`
	Quality   string `json:"quality"   jsonschema:"The quality code of GSE."`
	Meaning   string `json:"meaning"   jsonschema:"The meaning of the quality code."`
}

type netOutput struct {
	Value     string `json:"value"     jsonschema:"The net import in MW as a decimal string."`
	Direction string `json:"direction" jsonschema:"import, export, or none."`
}

type frequencyOutput struct {
	Value   string `json:"value"   jsonschema:"The frequency in Hz as a decimal string, as GSE publishes it. The nominal value is 50."`
	Time    string `json:"time"    jsonschema:"The minute of the frequency, RFC 3339 with +04:00."`
	Quality string `json:"quality" jsonschema:"The quality code of GSE."`
	Meaning string `json:"meaning" jsonschema:"The meaning of the quality code."`
}
