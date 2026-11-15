package tool

type getParcelInput struct {
	Code string `json:"code" jsonschema:"The cadastral code: 4 or 5 groups of digits with dots, as 01.15.05.066.020."`
}

type nearbyInput struct {
	Lat   float64 `json:"lat"             jsonschema:"The latitude, in degrees, 41.0 to 43.6."`
	Lon   float64 `json:"lon"             jsonschema:"The longitude, in degrees, 40.0 to 46.8."`
	Limit int     `json:"limit,omitempty" jsonschema:"The maximum count of parcels, 1 to 20. The default is 5."`
}

type addressesInput struct {
	Query        string `json:"query"                  jsonschema:"The street and the number, in Georgian or Latin letters."`
	Municipality string `json:"municipality,omitempty" jsonschema:"Only addresses whose municipality holds this text, in Georgian: თბილისი."`
	Limit        int    `json:"limit,omitempty"        jsonschema:"The maximum count of addresses, 1 to 20. The default is 5."`
}

type parcelOutput struct {
	Source        string           `json:"source"         jsonschema:"The source of the data: NAPR."`
	AsOf          string           `json:"as_of"          jsonschema:"The instant at which the server read the parcel, RFC 3339."`
	Stale         bool             `json:"stale"          jsonschema:"True when NAPR did not answer and the server returns a parcel it read before."`
	Code          string           `json:"code"           jsonschema:"The cadastral code."`
	MapURL        string           `json:"map_url"        jsonschema:"The page of the parcel on the NAPR map."`
	Address       string           `json:"address"        jsonschema:"The address, in Georgian, as NAPR publishes it."`
	Area          string           `json:"area"           jsonschema:"The area, a decimal number."`
	AreaUnit      string           `json:"area_unit"      jsonschema:"m2, or the unit of NAPR."`
	LandType      codeOutput       `json:"land_type"      jsonschema:"agricultural, non_agricultural, or other."`
	OwnershipType codeOutput       `json:"ownership_type" jsonschema:"ownership, co_ownership, or other."`
	Owners        []string         `json:"owners"         jsonschema:"The owners, as NAPR names them. Empty when NAPR names none."`
	Documents     []documentOutput `json:"documents"      jsonschema:"The registry documents, by kind, newest first."`
	Centre        pointOutput      `json:"centre"         jsonschema:"The centre point of the parcel."`
}

type codeOutput struct {
	Code string `json:"code" jsonschema:"The code."`
	Name string `json:"name" jsonschema:"The name of the code, in Georgian. For other, the text of NAPR."`
}

type documentOutput struct {
	Kind               codeOutput `json:"kind"                jsonschema:"registry_extract, cadastral_plan, or other."`
	RegistrationNumber string     `json:"registration_number" jsonschema:"The registration number."`
	Date               string     `json:"date,omitempty"      jsonschema:"The date of the document, YYYY-MM-DD. Empty when NAPR gives none."`
	URL                string     `json:"url"                 jsonschema:"The link to the PDF."`
}

type pointOutput struct {
	Lat float64 `json:"lat" jsonschema:"The latitude, in degrees."`
	Lon float64 `json:"lon" jsonschema:"The longitude, in degrees."`
}

type nearbyOutput struct {
	Source  string               `json:"source"  jsonschema:"The source of the data: NAPR."`
	AsOf    string               `json:"as_of"   jsonschema:"The instant at which the server read the parcels, RFC 3339."`
	Stale   bool                 `json:"stale"   jsonschema:"Always false: a list near a point is never stale."`
	Parcels []nearbyParcelOutput `json:"parcels" jsonschema:"The parcels, nearest first."`
}

type nearbyParcelOutput struct {
	Code      string `json:"code"       jsonschema:"The cadastral code."`
	Address   string `json:"address"    jsonschema:"The address, in Georgian."`
	DistanceM int    `json:"distance_m" jsonschema:"The distance from the point, in whole metres. 0 when the parcel holds the point."`
}

type addressesOutput struct {
	Source    string          `json:"source"    jsonschema:"The source of the data: NAPR."`
	AsOf      string          `json:"as_of"     jsonschema:"The instant at which the server read the addresses, RFC 3339."`
	Stale     bool            `json:"stale"     jsonschema:"Always false: an address search is never stale."`
	Addresses []addressOutput `json:"addresses" jsonschema:"The addresses, in the order of NAPR."`
}

type addressOutput struct {
	Address      string      `json:"address"      jsonschema:"The address, in Georgian."`
	Municipality string      `json:"municipality" jsonschema:"The municipality, in Georgian."`
	Centre       pointOutput `json:"centre"       jsonschema:"The centre point of the address."`
}
