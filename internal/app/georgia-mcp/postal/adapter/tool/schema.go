package tool

type searchCodesInput struct {
	Query    string `json:"query"              jsonschema:"An address, a street, a town, or a village of Georgia, in Georgian or English. 3 to 100 characters."`
	Language string `json:"language,omitempty" jsonschema:"The language of the addresses: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of addresses, 1 to 100. The default is 20."`
}

type codesOutput struct {
	Source    string          `json:"source"         jsonschema:"The source of the data: Georgian Post."`
	AsOf      string          `json:"as_of"          jsonschema:"The instant at which the server read the postal codes, RFC 3339."`
	Stale     bool            `json:"stale"          jsonschema:"True when Georgian Post did not answer and the server returns postal codes it read before."`
	Total     int             `json:"total"          jsonschema:"The count of addresses that Georgian Post matched. More than the count of addresses when the limit cut the list."`
	Addresses []addressOutput `json:"addresses"      jsonschema:"The matching addresses, in the order of Georgian Post."`
	Note      string          `json:"note,omitempty" jsonschema:"How to search again, when the list is empty."`
}

type addressOutput struct {
	PostalCode string `json:"postal_code" jsonschema:"The postal code: 4 digits as a string. It can start with 0."`
	Address    string `json:"address"     jsonschema:"The address as Georgian Post writes it: the settlement, then the street. A part in parentheses limits the postal code to some house numbers."`
	District   string `json:"district"    jsonschema:"The city or the municipality under which Georgian Post groups the address."`
}
