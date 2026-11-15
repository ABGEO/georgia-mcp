---
id: NAPRPARC
title: The land parcels of the NAPR cadastral map, the tools
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: parcels
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [MCP]
requirements: features/naprparc/requirements.md
---

# Specification: The land parcels of the NAPR cadastral map, the tools

The structs of `P/adapter/tool`. `spec.md` 4.3 holds the table of the tools.

`parcels_get_parcel`. Title: `Get a land parcel of Georgia`. Description:

> Returns one land parcel of the public registry of Georgia by its cadastral code
> (01.15.05.066.020): the address, the area, the land type, the ownership type, the owners,
> the registry documents with links to their PDFs, and the centre point. Without a code,
> find one with parcels_list_nearby_parcels. NAPR publishes every text in Georgian only.
> Buildings and apartments are not covered. Data from NAPR.

`parcels_list_nearby_parcels`. Title: `List the land parcels near a point in Georgia`. Description:

> Returns the land parcels nearest a point in Georgia, nearest first, with the cadastral
> code, the address, and the distance in metres. The first parcel at 0 metres holds the
> point. Get the owners and the area with parcels_get_parcel. Addresses are in Georgian.
> Data from NAPR.

`parcels_search_addresses`. Title: `Search the addresses of Georgia`. Description:

> Finds addresses in the address registry of Georgia, with the municipality and the centre
> point of each. Write the street and the number in Georgian (ჭავჭავაძის გამზირი 44) and
> give the municipality (თბილისი): the order of NAPR is poor, and other cities often come
> first. Pass a centre point to parcels_list_nearby_parcels to find the parcel. Data from NAPR.

```go
type getParcelInput struct {
    Code     string `json:"code"               jsonschema:"The cadastral code: 4 or 5 groups of digits with dots, as 01.15.05.066.020."`
    Language string `json:"language,omitempty" jsonschema:"The language of the names of the codes: ka or en. The default is en. NAPR texts stay in Georgian."`
}

type nearbyInput struct {
    Lat      float64 `json:"lat"                jsonschema:"The latitude, in degrees, 41.0 to 43.6."`
    Lon      float64 `json:"lon"                jsonschema:"The longitude, in degrees, 40.0 to 46.8."`
    Limit    int     `json:"limit,omitempty"    jsonschema:"The maximum count of parcels, 1 to 20. The default is 5."`
    Language string  `json:"language,omitempty" jsonschema:"ka or en. The default is en. NAPR texts stay in Georgian."`
}

type addressesInput struct {
    Query        string `json:"query"                  jsonschema:"The street and the number, in Georgian or Latin letters."`
    Municipality string `json:"municipality,omitempty" jsonschema:"Only addresses whose municipality holds this text, in Georgian: თბილისი."`
    Limit        int    `json:"limit,omitempty"        jsonschema:"The maximum count of addresses, 1 to 20. The default is 5."`
    Language     string `json:"language,omitempty"     jsonschema:"ka or en. The default is en. NAPR texts stay in Georgian."`
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
    Name string `json:"name" jsonschema:"The name of the code in the language. For other, the text of NAPR."`
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
```

`source` is `NAPR`. `as_of` is `ReadAt` in `kernel.TbilisiZone()`, written with
`time.RFC3339`. A `limit` of 0 asks the default. A point rounds to 6 decimals.
`map_url` is `https://maps.gov.ge/map/portal/#search/result/getinfo?lbl=<label>`.
`NAPRPARC-SC-025` checks it in a browser. The names of the codes in `en`:

| Code               | `en`                 | `ka`                             |
| ------------------ | -------------------- | -------------------------------- |
| `agricultural`     | Agricultural         | სასოფლო-სამეურნეო               |
| `non_agricultural` | Non-agricultural     | არასასოფლო-სამეურნეო            |
| `ownership`        | Ownership            | საკუთრება                        |
| `co_ownership`     | Co-ownership         | თანასაკუთრება                    |
| `registry_extract` | Registry extract     | ამონაწერი საჯარო რეესტრიდან      |
| `cadastral_plan`   | Cadastral plan       | საკადასტრო გეგმა                 |

## Retired identifiers

This file has no retired identifier.
