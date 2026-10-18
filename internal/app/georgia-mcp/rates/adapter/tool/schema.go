package tool

type getOfficialRatesInput struct {
	Currencies []string `json:"currencies,omitempty" jsonschema:"ISO 4217 codes, for example USD and EUR. Empty returns every currency. At most 50."`
	Date       string   `json:"date,omitempty"       jsonschema:"The date in YYYY-MM-DD, in Tbilisi. Empty means today. At most the valid date of the newest set: tomorrow after about 17:00."`
	Language   string   `json:"language,omitempty"   jsonschema:"The language of the currency names: en or ka. Empty means en."`
}

type listOfficialHistoryInput struct {
	Currency string `json:"currency"           jsonschema:"One ISO 4217 code, for example USD."`
	From     string `json:"from"               jsonschema:"The first date of the period, YYYY-MM-DD."`
	To       string `json:"to"                 jsonschema:"The last date of the period, YYYY-MM-DD. At most 31 days after from, counted with both ends."`
	Language string `json:"language,omitempty" jsonschema:"The language of the currency name: en or ka. Empty means en."`
}

type officialRatesOutput struct {
	Source    string               `json:"source"               jsonschema:"National Bank of Georgia."`
	AsOf      string               `json:"as_of"                jsonschema:"The publication instant of the set, RFC 3339. The read instant when the set is empty."`
	ValidFrom string               `json:"valid_from,omitempty" jsonschema:"The date from which the set is in force, YYYY-MM-DD."`
	ReadAt    string               `json:"read_at"              jsonschema:"The instant the server read the set from the NBG, RFC 3339."`
	Stale     bool                 `json:"stale"                jsonschema:"True when the NBG did not answer and the server returns a set it read before."`
	Rates     []officialRateOutput `json:"rates"                jsonschema:"The official rates, sorted by currency."`
	Missing   []string             `json:"missing,omitempty"    jsonschema:"The requested codes with no official rate on the date."`
}

type officialRateOutput struct {
	Currency    string `json:"currency"      jsonschema:"The ISO 4217 code."`
	Name        string `json:"name"          jsonschema:"The name of the currency in the requested language."`
	Quantity    int64  `json:"quantity"      jsonschema:"The count of units that rate prices."`
	Rate        string `json:"rate"          jsonschema:"The price in GEL of quantity units, as a decimal string."`
	RatePerUnit string `json:"rate_per_unit" jsonschema:"The price in GEL of one unit, as a decimal string."`
	Change      string `json:"change"        jsonschema:"The change of rate from the previous set, signed, as a decimal string."`
	ValidFrom   string `json:"valid_from"    jsonschema:"The date from which this rate is in force, YYYY-MM-DD."`
}

type officialHistoryOutput struct {
	Source   string               `json:"source"   jsonschema:"National Bank of Georgia."`
	AsOf     string               `json:"as_of"    jsonschema:"The publication instant of the newest point, RFC 3339."`
	ReadAt   string               `json:"read_at"  jsonschema:"The oldest instant at which the server read a point from the NBG, RFC 3339."`
	Stale    bool                 `json:"stale"    jsonschema:"True when one point comes from a set read before an outage of the NBG."`
	Currency string               `json:"currency" jsonschema:"The ISO 4217 code."`
	Name     string               `json:"name"     jsonschema:"The name of the currency in the requested language."`
	Points   []officialRateOutput `json:"points"   jsonschema:"One official rate for each set in force in the period, ascending by valid_from."`
}
