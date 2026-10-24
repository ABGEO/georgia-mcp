---
id: SOCFUEL
title: The fuel prices of SOCAR
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/socfuel/requirements.md
---

# Specification: The fuel prices of SOCAR

## 1. Summary

This feature adds the vendor client module `pkg/socar`, a third source of `fuel`, and the
grade `lpg`. The tool `fuel_get_prices` stays. Its input, its description, and its error
of a grade list `lpg`. `spec-scenarios.md` holds the scenarios.

## 2. Coverage

| Requirement       | Where this specification realizes it                    |
| ----------------- | ------------------------------------------------------- |
| `SOCFUEL-FR-001`  | 4.1 `adapter/socar`. `SOCFUEL-DD-002`                   |
| `SOCFUEL-FR-002`  | 4.1 `adapter/socar`. `SOCFUEL-DD-004`                   |
| `SOCFUEL-FR-003`  | `SOCFUEL-DD-003`                                        |
| `SOCFUEL-FR-004`  | 4.2 `GradeLPG`. 4.3 tool. `SOCFUEL-DD-007`              |
| `SOCFUEL-FR-005`  | 4.1 `adapter/socar`, `Company`. `SOCFUEL-DD-006`        |
| `SOCFUEL-FR-006`  | 4.1 `adapter/socar`. `SOCFUEL-DD-008`                   |
| `SOCFUEL-FR-007`  | 4.1 `pkg/socar`, `adapter/socar`. `SOCFUEL-DD-005`      |
| `SOCFUEL-FR-008`  | 4.3 configuration                                       |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `SOCFUEL-SC-013` checks them for SOCAR.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/socar` is a module: `github.com/abgeo/georgia-mcp/pkg/socar`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `pkg/socar` uses `net/http`, `net/url`, and `encoding/json` only.               |
| `CLT-003`              | The types carry the field names of SOCAR in their JSON tags. The client does not check `Status`. |
| `CLT-005`              | No `WithAPIKey`: the vendor needs no key.                                       |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the API is unofficial. The module stays at `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/socar` checks the status, names the grade, trims the names, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Every price is a `decimal.Decimal`, and a string in the output.                 |
| `MCP-003`              | The tag of `getPricesInput.Grades` changes. 4.3 gives it.                       |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_SOCAR_` and a default in its tag.     |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Vendor client  | `pkg/socar/`                          | `Client`, `New`, `Option`, `CurrentPrices`, `PriceList`, `FuelPrice`, `APIError`, `ErrUnexpectedStatus`. |
| Domain         | `F/domain/grade.go` (change)          | `GradeLPG`. `Grades` lists it after `GradeCNG`.            |
| Driven adapter | `F/adapter/socar/`                    | `Adapter`, `NewAdapter`.                                   |
| Tool           | `F/adapter/tool/schema.go` (change)   | The tag of `getPricesInput.Grades`. 4.3.                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.SOCAR`, `socarSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/socar`.                                         |
| Glossary       | `docs/specs/guidelines/glossary.md` (change) | The terms of requirements 5.                        |
| `WISFUEL`      | `features/wisfuel/spec.md`, `spec-scenarios.md` (change) | The example `lpg` of a refused grade becomes `kerosene`. |

**`pkg/socar`.** The shape of `pkg/wissol`.

```go
const DefaultBaseURL = "https://sgp.ge/sgp-backend/api"

var ErrUnexpectedStatus = errors.New("socar: unexpected status")

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) CurrentPrices(ctx context.Context) (PriceList, error)

type currentPricesResponse struct {
    GetCurrentPrices PriceList `json:"GetCurrentPrices"`
}

type PriceList struct {
    Status  string      `json:"Status"`  // "1" on a success
    Message string      `json:"Message"` // "Success"
    Results []FuelPrice `json:"Results"`
}

type FuelPrice struct {
    ActionDate    string      `json:"ActionDate"`  // "2023-09-23T10:06:54.137", with no zone
    FuelNameGeo   string      `json:"FuelNameGeo"`
    FuelNameEng   string      `json:"FuelNameEng"` // "LPG ", with the space of the answer
    FuelUnitPrice json.Number `json:"FuelUnitPrice"`
    FuelCode      string      `json:"FuelCode"`
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`CurrentPrices` sends `GET /integration/info/current-prices` with
`Accept: application/json`, and returns the object under `GetCurrentPrices`. Any status
other than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is
1 MiB. The default User-Agent is
`georgia-mcp/<version> (+https://github.com/abgeo/georgia-mcp)`.

**`adapter/socar`.** `Company` returns
`{Code: "socar", Name: "SOCAR", PublishesEffectiveDate: false}`. `Prices` sends one call
and maps the list:

| Field of the vendor          | Domain value                                                   |
| ---------------------------- | -------------------------------------------------------------- |
| `Status`                     | `"1"`: a success. Any other value fails the answer. `SOCFUEL-DD-005`. |
| `FuelCode`                   | `ProductPrice.Code`, and the grade of `SOCFUEL-DD-003`.        |
| `FuelNameGeo`, `FuelNameEng` | `ProductName{Georgian, English}`, each trimmed. `SOCFUEL-DD-008`. |
| `FuelUnitPrice`              | `Standard`. Zero: the product is left out. Not a number, or negative: the answer fails. `SOCFUEL-DD-004`. |
| `ActionDate`                 | Ignored. `EffectiveDate` is the zero date. `SOCFUEL-DD-006`.   |

`ReadAt` is the instant of the answer. A list with no product left fails the answer. Each
failure returns an error that wraps `kernel.ErrSourceUnavailable`.

**`module.go`.** `newModule` builds the source of SOCAR after the source of Rompetrol,
when `cfg.SOCAR.Enabled` is true. Else it logs at info level `SOCAR is off` with the
attribute `variable=GEOMCP_FUEL_SOCAR_ENABLED`.

### 4.2 Domain model

Only the change. `WISFUEL` 4.2 holds the rest.

```go
const (
    GradeRegular  Grade = "regular"
    GradePremium  Grade = "premium"
    GradeSuper    Grade = "super"
    GradeSuper100 Grade = "super_100"
    GradeDiesel   Grade = "diesel"
    GradeCNG      Grade = "cng"
    GradeLPG      Grade = "lpg" // liquefied petroleum gas, for one litre
    GradeOther    Grade = "other"
)
```

`Grades` returns the order of the table. `GradeLPG.Unit` is `UnitLitre`: `Unit` changes
nothing. The use case sorts the prices of a company in the order of `Grades`, so `lpg`
comes after `cng`.

### 4.3 Declarations

**MCP tool.** `fuel_get_prices` keeps its name, its output, and its errors (`WISFUEL`
4.3, 4.5). The tag of one input field changes:

| Tool              | Input                             | Output         | Errors        | Realizes         |
| ----------------- | --------------------------------- | -------------- | ------------- | ---------------- |
| `fuel_get_prices` | `grades`, `companies`, `language` | `pricesOutput` | `WISFUEL` 4.5 | `SOCFUEL-FR-004` |

```go
type getPricesInput struct {
    Grades    []string `json:"grades,omitempty"    jsonschema:"Grades: regular, premium, super, super_100, diesel, cng, lpg, other. Empty asks every grade."`
    Companies []string `json:"companies,omitempty" jsonschema:"Fuel company codes. Empty asks every company. The description of the tool lists the codes."`
    Language  string   `json:"language,omitempty"  jsonschema:"The language of the product names: ka or en. The default is en."`
}
```

The description and the error of a grade list the grades from `domain.Grades`, so they
gain `lpg (litre)` and `lpg` with no change of code. The error of a refused grade reads
`grades: "kerosene" is not a grade. Use one of: regular, premium, super, super_100,
diesel, cng, lpg, other.`

**Vendor endpoint.**

| Client      | Method and path                          | Key  | Official              | Realizes          |
| ----------- | ---------------------------------------- | ---- | --------------------- | ----------------- |
| `pkg/socar` | `GET /integration/info/current-prices`   | None | No, an unofficial API | `SOCFUEL-FR-001`  |

A real answer, recorded on 2026-10-19, cut to two products:

```
GET https://sgp.ge/sgp-backend/api/integration/info/current-prices
200 application/json (1212 bytes, Cache-Control: no-cache, no-store)
{"GetCurrentPrices":{"Status":"1","Message":"Success","Results":[
 {"ActionDate":"2023-10-17T09:45:07","FuelNameGeo":"ნანო ევრო რეგულარი",
  "FuelNameEng":"Nano Euro Regular","FuelUnitPrice":4.02,"FuelCode":"EURREG"},
 {"ActionDate":"2026-10-13T06:21:01.54","FuelNameGeo":"თხევადი აირი",
  "FuelNameEng":"LPG ","FuelUnitPrice":2.09,"FuelCode":"LPG"}]}}
```

A real error of the same backend, recorded on 2026-10-19:

```
GET https://sgp.ge/sgp-backend/api/integration/info/get-archive-price-list
400 application/json
{"time":"19/10/2026 21:06:13","exception":"com.optimo.socar.sgp.sgpbackend.exception.impl.GenericException: ...","message":"..."}
```

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method        | Key      | Time to live | Maximum stale | Entries | Realizes |
| ------------------ | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of SOCAR  | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                        | Type     | Default                          | Secret | Realizes          |
| ------------------------------- | -------- | -------------------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_SOCAR_ENABLED`     | bool     | `true`                           | No     | `SOCFUEL-FR-008`  |
| `GEOMCP_FUEL_SOCAR_BASE_URL`    | URL      | `https://sgp.ge/sgp-backend/api` | No     | `SOCFUEL-FR-001`  |
| `GEOMCP_FUEL_SOCAR_TIMEOUT`     | duration | `10s`                            | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_SOCAR_RATE`        | float    | `1`                              | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_SOCAR_BURST`       | int      | `1`                              | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `SOCAR vendors.Config` with `envPrefix:"SOCAR_"`. `fuel.Defaults` gives the defaults of this table (`PLATFORM-DD-014`).

## 5. Design decisions

### `SOCFUEL-DD-001`

**Realizes:** `SOCFUEL-FR-001`, `CLT-001`
**Decision:** `pkg/socar` covers `GET /integration/info/current-prices` only.
**Rationale:** The requirements leave out the archive and the prices of a card.
**Alternatives:** The archive and the card prices now. Two methods with no user.

### `SOCFUEL-DD-002`

**Realizes:** `SOCFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** `pkg/socar` decodes `FuelUnitPrice` as `json.Number`. The adapter parses it
with `decimal.NewFromString`.
**Rationale:** `WISFUEL-DD-004`. `4.9` stays `4.9`.
**Alternatives:** `float64`. `DOM-006` forbids it.

### `SOCFUEL-DD-003`

**Realizes:** `SOCFUEL-FR-003`
**Decision:** The adapter maps `FuelCode` to a grade with the table of `SOCFUEL-FR-003`.
A code that is not in the table gets `other`, and the adapter logs a warning with the
code and the English name, one time for each code.
**Rationale:** `WISFUEL-DD-002`. The code is stable. The names carry a brand: "Nano".
**Alternatives:** A match on the name. A new campaign name breaks it.

### `SOCFUEL-DD-004`

**Realizes:** `SOCFUEL-FR-002`
**Decision:** The adapter leaves out a product with the price zero. A price that is
negative, or that `decimal.NewFromString` refuses, fails the answer. A list with no product
left fails the answer.
**Rationale:** The archive of 2026-10-19 lists `A92 ბენზინი` at `0`: SOCAR uses zero for a
product that it does not sell. A negative price tells that the answer changed its shape.
**Alternatives:** A zero fails the answer, as for Wissol. One dead product hides SOCAR.

### `SOCFUEL-DD-005`

**Realizes:** `SOCFUEL-FR-007`
**Decision:** The adapter accepts a list with `Status` `"1"` only. It ignores `Message`.
**Rationale:** The client mirrors the answer (`CLT-003`). The meaning of the status belongs
to the server (`DOM-005`). On 2026-10-19 an error of the backend came with the status 400,
which the client already returns as an `*APIError`. A body with 200 and no
`GetCurrentPrices` decodes to an empty `Status`, and fails here.
**Alternatives:** The client checks `Status`. It would interpret the data.

### `SOCFUEL-DD-006`

**Realizes:** `SOCFUEL-FR-005`, `ROMFUEL-DD-006`
**Decision:** The adapter ignores `ActionDate`. Every price has the zero date, and
`Company.PublishesEffectiveDate` is false.
**Rationale:** The owner chose it. `ActionDate` of Nano Super is `2023-09-23`. Its last
change was on `2026-10-09`.
**Alternatives:** A date from the archive. A second call, out of scope.

### `SOCFUEL-DD-007`

**Realizes:** `SOCFUEL-FR-004`
**Decision:** `GradeLPG` sits after `GradeCNG` and before `GradeOther` in `Grades`. Its unit
is the litre.
**Rationale:** `other` stays last. The gases sit together.
**Alternatives:** `lpg` after `diesel`. It splits the gases.

### `SOCFUEL-DD-008`

**Realizes:** `SOCFUEL-FR-006`
**Decision:** The adapter trims the white space at both ends of each name with
`strings.TrimSpace`.
**Rationale:** The client mirrors `"LPG "`. A name with a space reads as a different name
in a comparison of a model.
**Alternatives:** The client trims. It would interpret the data.

## 6. Scenarios

`spec-scenarios.md` holds `SOCFUEL-SC-001` to `SOCFUEL-SC-015`.

## 7. Build plan

| #   | Step                                                                          | Realizes                            | Done |
| --- | ----------------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `pkg/socar` module, recorded answers, README, example. `go.work`.             | `SOCFUEL-FR-001`, `SOCFUEL-FR-007`  | [x]  |
| 2   | `domain`: `GradeLPG`. The tests of `WISFUEL-SC-005` and `WISFUEL-SC-018` with `kerosene`. | `SOCFUEL-FR-004`                    | [x]  |
| 3   | `adapter/socar`.                                                              | `SOCFUEL-FR-001` to `SOCFUEL-FR-003`, `SOCFUEL-FR-005` to `SOCFUEL-FR-007` | [x]  |
| 4   | `adapter/tool`: the tag of `grades`.                                          | `SOCFUEL-FR-004`                    | [x]  |
| 5   | `module.go`, `config.go`, `configs/.env.example`.                             | `SOCFUEL-FR-008`                    | [x]  |
| 6   | Move the terms of requirements 5 to the glossary.                             | `DOM-002`                           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
