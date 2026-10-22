---
id: PORFUEL
title: The fuel prices of Portal
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/porfuel/requirements.md
---

# Specification: The fuel prices of Portal

## 1. Summary

This feature adds the vendor client module `pkg/portal`, which reads the price list
that the page of Portal sends as JSON in its data stream, and a source of `fuel`. The
tool, the use case, the domain, and the cache do not change. `CLT-003` to `CLT-005`
admit a client of a data stream from 2026-10-21.

## 2. Coverage

| Requirement       | Where this specification realizes it                 |
| ----------------- | ---------------------------------------------------- |
| `PORFUEL-FR-001`  | 4.1 `adapter/portal`. `PORFUEL-DD-003`               |
| `PORFUEL-FR-002`  | `PORFUEL-DD-004`                                     |
| `PORFUEL-FR-003`  | `PORFUEL-DD-004`                                     |
| `PORFUEL-FR-004`  | 4.1 `pkg/portal`, `adapter/portal`. `PORFUEL-DD-001`, `PORFUEL-DD-002` |
| `PORFUEL-FR-005`  | 4.1 `adapter/portal`, `Company`                      |
| `PORFUEL-FR-006`  | 4.3 configuration                                    |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `PORFUEL-SC-008` checks them for Portal.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/portal` is a module: `github.com/abgeo/georgia-mcp/pkg/portal`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, `strings`, and `encoding/json` only. No HTML parser.     |
| `CLT-003`              | The types carry the field names of the page in their JSON tags.                 |
| `CLT-005`              | `Accept: text/x-component` and `RSC: 1`. No `WithAPIKey`.                       |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is the data of a web page. `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/portal` names the grade and the Georgian name, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Every price is a `decimal.Decimal`, and a string in the output.                 |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_PORTAL_` and a default in its tag.    |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Guidelines     | `docs/specs/guidelines/clients.md` (change, done) | `CLT-003` to `CLT-005`: the client of a data stream. |
| Vendor client  | `pkg/portal/`                         | `Client`, `New`, `Option`, `FuelPrice`, `FuelType`, `APIError`, `ErrUnexpectedStatus`, `ErrNoPriceList`, `ErrMalformedPriceList`. |
| Driven adapter | `F/adapter/portal/`                   | `Adapter`, `NewAdapter`.                                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Portal`, `PortalConfig`, `PortalHTTPConfig`, `portalSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/portal`.                                        |
| Website        | `website/src/content/contexts.yaml` (change) | Portal, პორტალი, in the list of `fuel`.           |

**`pkg/portal`.** The shape of `pkg/socar`, with one method.

```go
const DefaultBaseURL = "https://portal.com.ge"

var (
    ErrUnexpectedStatus   = errors.New("portal: unexpected status")
    ErrNoPriceList        = errors.New("portal: no price list")
    ErrMalformedPriceList = errors.New("portal: malformed price list")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) FuelPrices(ctx context.Context) ([]FuelPrice, error)

// FuelPrice is one entry of the list "fuelPricesData" of the page.
type FuelPrice struct {
    ID                string      `json:"id"`
    Name              string      `json:"name"`              // "EURO REGULAR"
    Type              string      `json:"type"`              // "PETROL" or "DIESEL"
    Octane            json.Number `json:"octane"`            // "93.7"
    LatestPrice       string      `json:"latestPrice"`       // "3.94"
    LatestOnlinePrice string      `json:"latestOnlinePrice"` // "3.74"
    FuelType          FuelType    `json:"fuelType"`
}

type FuelType struct {
    Slug string `json:"slug"` // "euro-regular"
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`FuelPrices` sends `GET /ka/fuel-prices` with `Accept: text/x-component` and `RSC: 1`.
Any status other than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The
body limit is 2 MiB: the stream has 114 KB. The default User-Agent is
`georgia-mcp-portal/<version> (+https://github.com/abgeo/georgia-mcp)`.
`PORFUEL-DD-001` gives the parse.

**`adapter/portal`.** `Company` returns
`{Code: "portal", Name: "Portal", PublishesEffectiveDate: false}`. `Prices` sends one
call and maps each entry:

| Field of the vendor  | Domain value                                                   |
| -------------------- | -------------------------------------------------------------- |
| `FuelType.Slug`      | `ProductPrice.Code`, the grade and the Georgian name of `PORFUEL-DD-004`. |
| `Name`               | `ProductName.English`, trimmed.                                |
| `LatestPrice`        | `Standard`. `PORFUEL-DD-003`.                                  |
| `LatestOnlinePrice`, `ID`, `Type`, `Octane` | Not read.                               |

`SelfService` is none. `EffectiveDate` is the zero date. `ReadAt` is the instant of the
answer. An empty list fails the answer. Each failure returns an error that wraps
`kernel.ErrSourceUnavailable`.

**`module.go` (change).** `newModule` builds the source of Portal after the last source
that it builds today, when `cfg.Portal.Enabled` is true. Else it logs at info level
`Portal is off` with the attribute `variable=GEOMCP_FUEL_PORTAL_ENABLED`.

### 4.2 Domain model

No change. `WISFUEL` 4.2 and `ROMFUEL` 4.2 hold the model.

### 4.3 Declarations

**MCP tool.** No change. The description lists `portal (Portal, publishes no effective
date)` through `companyList`.

**Vendor endpoint.**

| Client       | Method and path               | Key  | Official                          | Realizes         |
| ------------ | ----------------------------- | ---- | --------------------------------- | ---------------- |
| `pkg/portal` | `GET /ka/fuel-prices`, `RSC: 1` | None | No, the data of a web page for people | `PORFUEL-FR-001` |

A real answer, recorded on 2026-10-21, cut to the start of the row and one product:

```
GET https://portal.com.ge/ka/fuel-prices
RSC: 1
200 text/x-component (114412 bytes, Cache-Control: private, no-cache, no-store)
...["$","$L1c",null,{"fuelPricesData":[{"id":"05103e96-4008-4b0d-bbad-6263c5eab4ac",
"name":"SUPER","type":"PETROL","octane":100,"latestPrice":"4.39",
"latestOnlinePrice":"4.19","fuelType":{"id":"05103e96-...","name":"SUPER",
"slug":"super","type":"PETROL","octane":100,"country":"საბერძნეთი",...}}, ...]}]...
```

The same request with no `RSC` header returns the HTML page, 197 KB. It holds the list
inside the strings of its scripts, with each quote escaped.

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method        | Key      | Time to live | Maximum stale | Entries | Realizes |
| ------------------ | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Portal | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                      | Type     | Default                 | Secret | Realizes          |
| ----------------------------- | -------- | ----------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_PORTAL_ENABLED`  | bool     | `true`                  | No     | `PORFUEL-FR-006`  |
| `GEOMCP_FUEL_PORTAL_BASE_URL` | URL      | `https://portal.com.ge` | No     | `PORFUEL-FR-001`  |
| `GEOMCP_FUEL_PORTAL_TIMEOUT`  | duration | `10s`                   | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_PORTAL_RATE`     | float    | `1`                     | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_PORTAL_BURST`    | int      | `1`                     | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Portal PortalConfig` with `envPrefix:"PORTAL_"`, with the shape of
`WissolConfig` and the defaults of this table.

## 5. Design decisions

### `PORFUEL-DD-001`

**Realizes:** `PORFUEL-FR-004`, `CLT-004`
**Decision:** The client finds the first `"fuelPricesData":` in the body, and decodes
the JSON value that follows it with `json.Decoder.Decode` into `[]FuelPrice`. No key
returns `ErrNoPriceList`. A value that does not decode returns `ErrMalformedPriceList`.
An empty array returns an empty list and no error.
**Rationale:** The stream holds the props of the component of the page as plain JSON.
The decoder stops at the end of the array, so the rest of the row needs no parse. The
key is the name of a prop that the code of Portal chose, not a style.
**Alternatives:** The HTML page with `golang.org/x/net/html`: the cards carry Tailwind
classes only (`font-digital-numbers`), and no slug. The HTML page with the list from a
script: the client must undo the escape of a JavaScript string.

### `PORFUEL-DD-002`

**Realizes:** `PORFUEL-FR-004`
**Decision:** The adapter treats an empty list as a failure.
**Rationale:** `ROMFUEL-DD-002`. The meaning of an empty list belongs to the server.
**Alternatives:** The client fails on an empty list. It would interpret the data.

### `PORFUEL-DD-003`

**Realizes:** `PORFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** The adapter parses `LatestPrice` with `decimal.NewFromString`. A text that
it refuses, or a value not above zero, fails the answer.
**Rationale:** `ROMFUEL-DD-004`. The page sends the price as text: `"3.94"` stays `3.94`.
**Alternatives:** None. The text is exact.

### `PORFUEL-DD-004`

**Realizes:** `PORFUEL-FR-002`, `PORFUEL-FR-003`
**Decision:** The adapter maps the slug to a grade and a Georgian name with the tables of
`PORFUEL-FR-002` and `PORFUEL-FR-003`. A slug that is not in the table gets the grade
`other` and no Georgian name. The adapter logs a warning with the slug and the name, one
time for each slug.
**Rationale:** `SOCFUEL-DD-003`. The slug is stable. `ProductName.In` gives the English
name when the Georgian one is empty.
**Alternatives:** A grade from `Octane` and `Type`. It reads `93.7` and `0`, and maps a
new range with a rule that no statement gives.

## 6. Scenarios

`pkg/portal/testdata/` holds `fuel-prices.rsc`, the real stream of 2026-10-21, and
`fuel-prices-edge.rsc`, a copy with a product of the slug `hydro`. An integration test
serves the streams from `httptest` and calls the tool through the in-memory MCP transport.

### `PORFUEL-SC-001`

**Verifies:** `PORFUEL-FR-001` **Layer:** unit

**Given** an `httptest` server that records the request and serves `fuel-prices.rsc`.
**When** `FuelPrices` runs.
**Then** the path is `/ka/fuel-prices`, `Accept` is `text/x-component`, `RSC` is `1`, the
User-Agent starts with `georgia-mcp-portal/`, and the list holds 5 entries. The first
has the slug `super`, `LatestPrice` `4.39`, and `LatestOnlinePrice` `4.19`.

### `PORFUEL-SC-002`

**Verifies:** `PORFUEL-FR-004` **Layer:** unit

| Answer of the vendor                          | Result                                      |
| --------------------------------------------- | ------------------------------------------- |
| 200, a stream with no `"fuelPricesData":`     | `ErrNoPriceList`.                           |
| 200, `"fuelPricesData":[{"name":`             | `ErrMalformedPriceList`.                    |
| 200, `"fuelPricesData":[]`                    | An empty list, no error.                    |
| 500                                           | An `*APIError` with 500. `ErrUnexpectedStatus`. |

### `PORFUEL-SC-003`

**Verifies:** `PORFUEL-FR-001` **Layer:** live

**Given** the real Portal. **When** `FuelPrices` runs.
**Then** the list holds one entry or more, and each has a slug and a price.

### `PORFUEL-SC-004`

**Verifies:** `PORFUEL-FR-001` to `PORFUEL-FR-003`, `PORFUEL-FR-005` **Layer:** unit

**Given** the vendor serves `fuel-prices.rsc`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 5 prices with no self-service price and the zero date. `ReadAt`
is the instant of the clock. `Company` is `{portal, Portal, false}`.

| Code            | Grade       | Standard | English name  | Georgian name |
| --------------- | ----------- | -------- | ------------- | ------------- |
| `super`         | `super`     | `4.39`   | SUPER         | სუპერი         |
| `premium`       | `premium`   | `4.19`   | PREMIUM       | პრემიუმი       |
| `euro-regular`  | `regular`   | `3.94`   | EURO REGULAR  | ევრო რეგულარი   |
| `euro-diesel`   | `diesel`    | `4.94`   | EURO DIESEL   | ევრო დიზელი     |
| `effect-diesel` | `diesel`    | `4.84`   | EFFECT DIESEL | ეფექტ დიზელი    |

### `PORFUEL-SC-005`

**Verifies:** `PORFUEL-FR-002`, `PORFUEL-FR-003` **Layer:** unit

**Given** the vendor serves `fuel-prices-edge.rsc`.
**When** the adapter reads the prices two times.
**Then** `hydro` has the grade `other`, and its name in `ka` is the name of Portal. The
log holds one warning with `hydro`.

### `PORFUEL-SC-006`

**Verifies:** `PORFUEL-FR-001`, `PORFUEL-FR-004` **Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Answer                                          | Reason                        |
| ----------------------------------------------- | ----------------------------- |
| 200, `"fuelPricesData":[]`                      | No product.                   |
| 200, no `"fuelPricesData":`                     | An error of the client.       |
| 200, a product with `"latestPrice":"4,39"`      | A price that is not a number. |
| 200, a product with `"latestPrice":"0"`         | A price not above zero.       |
| 500                                             | An error of the client.       |

### `PORFUEL-SC-007`

**Verifies:** `PORFUEL-FR-001`, `PORFUEL-FR-002`, `PORFUEL-FR-005` **Layer:** integration

**Given** Portal serves `fuel-prices.rsc`, and every other company answers 500.
**When** a client lists the tools, then calls `fuel_get_prices` with the grade `super`.
**Then** the description holds `portal (Portal, publishes no effective date)`. `portal`
holds SUPER at `4.39`, with no `self_service_price` and no `effective_date`.

### `PORFUEL-SC-008`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002` **Layer:** unit

**Given** a cache over the adapter of Portal, a transport that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `PORFUEL-SC-009`

**Verifies:** `PORFUEL-FR-006` **Layer:** unit

**Given** no variable of fuel. **When** the configuration loads.
**Then** Portal is on, with the base address `https://portal.com.ge`, a timeout of 10
seconds, the rate 1, and the burst 1.

### `PORFUEL-SC-010`

**Verifies:** `PORFUEL-FR-006` **Layer:** integration

**Given** `GEOMCP_FUEL_PORTAL_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `portal`.
**Then** the log holds `Portal is off` with `variable=GEOMCP_FUEL_PORTAL_ENABLED`. The
tool result is an error: `portal` is not a fuel company of this server.

## 7. Build plan

| #   | Step                                                                   | Realizes                            | Done |
| --- | ---------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `CLT-003` to `CLT-005`: the client of a data stream.                   | `PORFUEL-FR-001`                    | [x]  |
| 2   | `pkg/portal` module, recorded streams, README, example. `go.work`.     | `PORFUEL-FR-001`, `PORFUEL-FR-004`  | [x]  |
| 3   | `adapter/portal`.                                                      | `PORFUEL-FR-001` to `PORFUEL-FR-005` | [x]  |
| 4   | `module.go`, `config.go`, `configs/.env.example`.                      | `PORFUEL-FR-006`                    | [x]  |
| 5   | Move the term "Portal" to the glossary. Add Portal to the website.     | `DOM-002`                           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
