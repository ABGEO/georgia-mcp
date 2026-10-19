package tool

type getPricesInput struct {
	Grades    []string `json:"grades,omitempty"    jsonschema:"Grades: regular, premium, super, super_100, diesel, cng, other. Empty asks every grade."`
	Companies []string `json:"companies,omitempty" jsonschema:"Fuel company codes. Empty asks every company. The description of the tool lists the codes."`
	Language  string   `json:"language,omitempty"  jsonschema:"The language of the product names: ka or en. The default is en."`
}

type pricesOutput struct {
	Meaning     string                `json:"meaning"               jsonschema:"How to read price and self_service_price."`
	Companies   []companyPricesOutput `json:"companies"             jsonschema:"The prices of each fuel company that answered, sorted by company code."`
	Unavailable []companyOutput       `json:"unavailable,omitempty" jsonschema:"The fuel companies that did not answer."`
}

type companyPricesOutput struct {
	Company string        `json:"company"           jsonschema:"The company code."`
	Source  string        `json:"source"            jsonschema:"The name of the fuel company."`
	AsOf    string        `json:"as_of"             jsonschema:"The instant at which the server read the prices, RFC 3339. The company publishes no instant."`
	Stale   bool          `json:"stale"             jsonschema:"True when the company did not answer and the server returns prices it read before."`
	Prices  []priceOutput `json:"prices"            jsonschema:"The prices of the products, sorted by grade."`
	Missing []string      `json:"missing,omitempty" jsonschema:"The requested grades that the company does not sell."`
}

type priceOutput struct {
	Product          string `json:"product"                      jsonschema:"The name of the product at the company."`
	Grade            string `json:"grade"                        jsonschema:"The grade of the product, the same for every company."`
	Unit             string `json:"unit"                         jsonschema:"The unit of each price: litre, or cubic_metre for cng."`
	Price            string `json:"price"                        jsonschema:"The standard price in GEL of one unit, as a decimal string."`
	SelfServicePrice string `json:"self_service_price,omitempty" jsonschema:"The price in GEL of one unit at a self-service station of the company, as a decimal string. Absent when the company publishes none."`
	EffectiveDate    string `json:"effective_date"               jsonschema:"The date from which the company applies the prices, YYYY-MM-DD."`
}

type companyOutput struct {
	Company string `json:"company" jsonschema:"The company code."`
	Source  string `json:"source"  jsonschema:"The name of the fuel company."`
}
