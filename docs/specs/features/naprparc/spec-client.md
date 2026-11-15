---
id: NAPRPARC
title: The land parcels of the NAPR cadastral map, the client
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: parcels
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [CLT, DOM]
requirements: features/naprparc/requirements.md
---

# Specification: The land parcels of the NAPR cadastral map, the client

## 1. `pkg/napr`

The shape of `pkg/nager`. The package comment and the README say that the methods serve
the map page of NAPR and have no documentation.

```go
const DefaultBaseURL = "https://maps.gov.ge"

var (
    ErrUnexpectedStatus = errors.New("napr: unexpected status")
    ErrInvalidLabel     = errors.New("napr: invalid label")
    ErrEmptyKeyword     = errors.New("napr: empty keyword")
)

type APIError = vendorhttp.StatusError

// Search finds parcels by cadastral code and addresses by text.
func (c *Client) Search(ctx context.Context, keyword string) ([]SearchResult, error)

// ParcelsAt returns the parcels near a point, nearest first.
func (c *Client) ParcelsAt(ctx context.Context, lat, lon float64) ([]NearbyParcel, error)

// Parcel returns the detail of the parcel with a label: "lr_parcels:AAArWyAATAAIs9BAAB".
func (c *Client) Parcel(ctx context.Context, label string) ([]ParcelInfo, error)

// ParcelShape returns the shape of the parcel with a label.
func (c *Client) ParcelShape(ctx context.Context, label string) ([]Shape, error)

// AddressShape returns the shape of the address with a label: "ract_g_borders:AAAlnvAAyAAEGARAAD".
func (c *Client) AddressShape(ctx context.Context, label string) ([]Shape, error)

// Label returns the value of lbl in a link of the source, and false when it has none
// or the value is not a label.
func Label(link string) (string, bool)

type SearchResult struct {
    Name     string        `json:"name"`     // "01.15.05.066.020", or " ილია ჭავჭავაძის გამზირი, N 44"
    Descript string        `json:"descript"` // the address of a parcel, the municipality of an address
    Details  SearchDetails `json:"details"`
}

type SearchDetails struct {
    Type         string `json:"type"`          // "address" for both kinds
    InfoLink     string `json:"info_link"`     // "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:..."
    GeometryLink string `json:"geometry_link"`
}

type NearbyParcel struct {
    Name       string `json:"name"`       // "01.15.05.066.020"
    Descript   string `json:"descript"`
    ResultLink string `json:"resultlink"` // "#search/result/getinfo?lbl=lr_parcels:..."
    Distance   string `json:"distance"`   // "91.775768836483", metres
}

type ParcelInfo struct {
    ObjTypeID string        `json:"_obj_type_id_"` // "1002"
    ObjName   string        `json:"_obj_name_"`
    Detail    ParcelDetail  `json:"_obj_detail_"`
    Owners    []Owner       `json:"_owners_"`
    Docs      []DocumentSet `json:"_docs_"`
}

type ParcelDetail struct {
    CadCode      string `json:"_cadcode_"`
    AreaMeasured string `json:"_area_measured_"` // "618"
    Unit         string `json:"_unit_"`          // "კვ.მ."
    Address      string `json:"_address_"`
    LandType     string `json:"_land_type_"`
    OwnTypeName  string `json:"_owntype_name_"`
}

type Owner struct {
    Person string `json:"_person_"`
}

type DocumentSet struct {
    DocName string     `json:"_doc_name_"`
    List    []Document `json:"_list_"`
}

type Document struct {
    BlobURI   string `json:"_blob_uri_"`
    RegNumber string `json:"_regnumber_"`
    DocDate   string `json:"_doc_date_"` // "13.08.2009", or ""
}

type Shape struct {
    Name        string `json:"name"`
    Proj        string `json:"proj"`         // "EPSG:4326"
    Shape       string `json:"shape"`        // WKT, "POLYGON ((44.80166 41.69339, ...))"
    ShapeFormat string `json:"shape_format"` // "WKT"
}
```

| Method         | Request                                                                         |
| -------------- | ------------------------------------------------------------------------------- |
| `Search`       | `POST /map/portal/search`, `application/x-www-form-urlencoded`, `keyword=<text>&keyword_description=`. An empty keyword gives `ErrEmptyKeyword` and no call. |
| `ParcelsAt`    | `GET /lr/bo/mg/getinfo2?LON=<lon>&LAT=<lat>&C=0&FRAME_NAME=BY_COORDS_AND_COMPASS.GETINFO.MAPGOV`. 6 decimals. |
| `Parcel`       | `GET /lr/bo/mg/getinfo.alpha`, `RawQuery` `lbl=<label>&fmt=json&lang=ka`.       |
| `ParcelShape`  | `GET /lr/bo/mg/getinfo.alpha`, `RawQuery` `lbl=<label>&res=shp`.                |
| `AddressShape` | `GET /ar/mg/getinfo`, `RawQuery` `lbl=<label>&res=shp`.                         |

Each request sends `Accept: application/json`. A label matches `^[a-z_]+:[A-Za-z0-9+/]+$`.
Any other label gives `ErrInvalidLabel` and no call, since the label goes into the query
with no encoding. `Search`, `ParcelsAt`, `Parcel`, and the shapes read the array under
`result` or `data`. A body that does not decode is an error. The body limit is the
default 1 MiB. The largest answer seen is 165 KB.

**The repair of `ParcelsAt`.** The answer puts each value on its own line. Before the
decode, each line that matches `^(\s*"[a-z_]+"\s*:\s*")(.*)("\s*,?\s*)$` gets a backslash
before each quotation mark of the middle group that has none. Any other line stays. A
line whose value is an object, as `shape_wkt`, does not match.

## 2. Recorded answers

`pkg/napr/testdata/` holds the answers of 2026-11-15. Each private owner is `სახელი გვარი`.

| File                  | Request                                                         | Content |
| --------------------- | --------------------------------------------------------------- | ------- |
| `search-code.json`    | `Search("01.15.05.066.020")`                                    | 1 result. |
| `search-address.json` | `Search("ჭავჭავაძის გამზირი 44")`                              | 10 addresses: Gori 2, Telavi 2, Tbilisi 5, Kutaisi 1. |
| `search-empty.json`   | `Search("99.99.99.999.999")`                                    | `{"status":true,"result":[]}`. |
| `near-freedom.json`   | `ParcelsAt(41.6934, 44.8015)`                                    | 15 parcels, not valid JSON. |
| `near-outside.json`   | `ParcelsAt(48.85, 2.35)`                                         | `{"status":true,"result":[]}`. |
| `info-freedom.json`   | `Parcel("lr_parcels:AAArWyAATAAIs9BAAB")`                        | 01.15.05.066.020. 1 owner, 2 documents. |
| `info-coowned.json`   | `Parcel("lr_parcels:AAArWyAAaAAOMFUAAF")`                        | 01.14.11.005.006. `თანასაკუთრება`, 1 owner, 13 documents, 3 with no date. |
| `info-no-owner.json`  | `Parcel("lr_parcels:AAArWyAANAAIO6TAAA")`                        | 01.15.05.066.015. No owner, no document. |
| `info-empty.json`     | `Parcel` with an encoded label                                   | `{"provider":"immovable","data":[]}`. |
| `shape-freedom.json`  | `ParcelShape("lr_parcels:AAArWyAATAAIs9BAAB")`                   | One `POLYGON`. |
| `shape-address.json`  | `AddressShape("ract_g_borders:AAAlnvAAyAAEGARAAD")`              | One `MULTIPOLYGON`. |

Real answers, cut:

```
POST https://maps.gov.ge/map/portal/search   keyword=01.15.05.066.020&keyword_description=
200 application/json; charset=UTF-8 (854 bytes)
{"status":true,"result":[{"id": 1285613,"name": "01.15.05.066.020",
  "descript": "ქალაქი თბილისი , მოედანი თავისუფლება ,   (ნაკვეთი 66\/020)",
  "resultlink" :"/map/portal/getbylbl?lbl=lr_parcels:AAArWyAATAAIs9BAAB",
  "details": {"type": "address","mode": "absolute",
    "info_link": "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB",
    "geometry_link": "/lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB&res=shp"}, ...}]}

POST .../map/portal/search   keyword=ჭავჭავაძის გამზირი 44
{"status":true,"result":[{"id": 936616,"name": " ქალაქი გორი, ილია ჭავჭავაძის გამზირი, N 44",
  "descript": "გორის მუნიციპალიტეტი",
  "details": {"type": "address","info_link": "/ar/mg/getinfo?lbl=ract_g_borders:AAAlnvAAvAADyJ1AAF", ...}}, ...]}

GET .../lr/bo/mg/getinfo2?LON=44.8015&LAT=41.6934&C=0&FRAME_NAME=BY_COORDS_AND_COMPASS.GETINFO.MAPGOV
200 application/json; charset=UTF-8 (68 KB)
{"status":true,"result":[{"id": "1285613","name": "01.15.05.066.020",
  "resultlink": "#search/result/getinfo?lbl=lr_parcels:AAArWyAATAAIs9BAAB",
  "distance": "0","shape_wkt": { "type": "Polygon", "coordinates": [...] }}, ...
 {"id": "4392","name": "01.15.04.028.003",
  "descript": "ქალაქი თბილისი , მოედანი თავისუფლება ,   "ცეკავშირის" მიმდებარედ , N 1", ...}]}

GET .../lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB&fmt=json&lang=ka
200 application/json; charset=UTF-8 (1293 bytes)
{"provider":"immovable","data":[{"_obj_type_id_":"1002","_obj_type_":"ნაკვეთი",
  "_obj_name_":"01.15.05.066.020",
  "_obj_detail_":{"_cadcode_":"01.15.05.066.020","_area_measured_":"618","_unit_":"კვ.მ.",
    "_address_":"...","_land_type_":"არასასოფლო სამეურნეო","_owntype_name_":"საკუთრება"},
  "_owners_":[{"_person_":" ქალაქ თბილისის თვითმმართველი ერთეული"}],
  "_docs_":[{"_doc_name_":"საკადასტრო გეგმა","_list_":[{"_blob_uri_":"https:\/\/bs.napr.gov.ge\/GetBlob?pid=101&bid=...",
    "_regnumber_":"882009244097","_doc_date_":"13.08.2009"}]}, ...]}]}

GET .../lr/bo/mg/getinfo.alpha?lbl=lr_parcels:AAArWyAATAAIs9BAAB&res=shp
{"provider":"immovable","data":[{"id":1285613,"name":"01.15.05.066.020","proj":"EPSG:4326",
  "shape":"POLYGON ((44.8016627156479 41.693392129304, ...))","shape_format":"WKT"}]}

GET .../ar/mg/getinfo?lbl=ract_g_borders:AAAlnvAAyAAEGARAAD&res=shp
{"provider":"address","data":[{"id":66315,"name":"N 44","proj":"EPSG:4326",
  "shape":"MULTIPOLYGON (((44.7655334770166 41.7093179449302, ...)))","shape_format":"WKT"}]}
```

Other answers seen:

- `lang=en` gives the same Georgian answer as `lang=ka`.
- `lbl=lr_parcels%3A...` or `...%2F11AAI` gives `"data":[]` with the status 200.
- An empty keyword gives an "Access Denied" page of 840 bytes with the status 200.
- `robots.txt` gives the same page.
- A document link gives `application/pdf`. The links of `01.15.05.066.020` were the same
  at two reads, 2 hours apart.

## Retired identifiers

This file has no retired identifier.
