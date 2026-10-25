---
id: CRYSRATE
title: The commercial exchange rates of Crystal
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/crysrate/requirements.md
---

# Specification: The commercial exchange rates of Crystal

## 1. Summary

This feature adds the vendor client `pkg/crystal`, a client of the unofficial method of
the rate app of Crystal, and Crystal as a source of `rates_get_commercial_rates`. The
tool, the use case, the domain, and the cache of `TBCRATE` do not change. One call
reads the default cash rate of every currency. The client checks both envelopes of the
answer, and the adapter divides each price by its multiplier.

## 2. Coverage

| Requirement        | Where this specification realizes it                        |
| ------------------ | ----------------------------------------------------------- |
| `CRYSRATE-FR-001`  | 4.1 `adapter/crystal`. 4.3 vendor endpoint. `CRYSRATE-DD-001` |
| `CRYSRATE-FR-002`  | `CRYSRATE-DD-002`. `CREDRATE-DD-002`                        |
| `CRYSRATE-FR-003`  | 4.1 `adapter/crystal`. `LIBRATE-DD-004`                     |
| `CRYSRATE-FR-004`  | `CRYSRATE-DD-003`                                           |
| `CRYSRATE-FR-005`  | 4.1 `adapter/crystal`. `RICORATE-DD-004`                    |
| `CRYSRATE-FR-006`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`       |
| `CRYSRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                         |
| `CRYSRATE-INV-001` | `CRYSRATE-DD-002`                                           |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/crystal` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/crystal`. |
| `CLT-002`, `CLT-004`   | The standard library and `pkg/vendorhttp` only. `pkg/crystal` stays in `clients`. |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags.            |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`, and no token of the app.        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the method is unofficial.        |
| `DOM-005`, `DOM-011`   | `adapter/crystal` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Each price is a `json.Number`. The adapter parses its text into a decimal.   |
| `DOM-009`              | Crystal is an adapter of `CommercialRateSource`.                             |
| `MCP-002`              | No new tool. The description lists `crystal` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Crystal sits behind a `CommercialRateCache`, as TBC does.                    |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_CRYSTAL_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `CRYSRATE` 5.                             |
| Vendor client  | `pkg/crystal/`                        | `Client`, `New`, `Option`, `Rate`, `APIError`, `ErrUnexpectedStatus`, `ErrNotSucceeded`. |
| Driven adapter | `R/adapter/crystal/`                  | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Crystal`, the entry of Crystal in `otherEntries`, `crystalSwitchVariable`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |

**`pkg/crystal`.** The shape of `pkg/credo`, with one method and no key.

```go
const DefaultBaseURL = "https://currency.crystal.ge"

var (
    ErrUnexpectedStatus = errors.New("crystal: unexpected status")
    ErrNotSucceeded     = errors.New("crystal: not succeeded") // an envelope reports a failure
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) CashRates(ctx context.Context) ([]Rate, error)

// Rate is one pair of currencies of an answer.
type Rate struct {
    SourceISO  string      `json:"SourceISO"`
    TargetISO  string      `json:"TargetISO"`
    Multiplier json.Number `json:"Multiplier"`
    BuyRate    json.Number `json:"BuyRate"`
    SellRate   json.Number `json:"SellRate"`
}

type APIError = vendorhttp.StatusError
```

`CashRates` sends `GET /api/Conversion/GetWebRateByProductID?IsCash=true` with
`Accept: application/json`, and no `authorization` header. It decodes the two envelopes
with `UseNumber`, as `CRYSRATE-DD-003` gives, and returns the inner `data`. Any status
other than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit
is 1 MiB: the answer has 2532 bytes. The type omits `NBGRate`, the symbols, and the
other fields of an item.

**`adapter/crystal`.** `Bank` returns
`{Code: "crystal", Name: "Crystal", Kind: domain.KindMicrofinance}`. `CommercialRates`
sends one call. It keeps each item whose `TargetISO` is `GEL` in any case, and converts
the item of each requested code:

| Field of the item | Domain value                                                     |
| ----------------- | ---------------------------------------------------------------- |
| `SourceISO`       | Matched against the requested code in upper case.                |
| `Multiplier`      | The count of units. `CRYSRATE-DD-002`.                           |
| `BuyRate`, `SellRate` | `decimal.NewFromString`, divided by the count, trailing zeros dropped. |

A requested code with no item, or with a price that is absent, empty, or zero, goes to
`Missing`. A price that is not a number or is below zero fails the bank. Two items of
one code against the lari fail the bank. `ReadAt` is the instant of the answer.

**`module.go` (change).** `otherEntries` gains the entry of Crystal after MBC, with the
name `Crystal` and the variable `GEOMCP_RATES_CRYSTAL_ENABLED`.

### 4.2 Domain model

No change. `RICORATE` 4.2 holds `KindMicrofinance`.

### 4.3 Declarations

**Vendor endpoint.**

| Client         | Method and path                                           | Key  | Official                 | Realizes          |
| -------------- | --------------------------------------------------------- | ---- | ------------------------ | ----------------- |
| `pkg/crystal`  | `GET /api/Conversion/GetWebRateByProductID?IsCash=true`   | None | No, an unofficial method | `CRYSRATE-FR-001` |

A real answer, recorded on 2026-10-25 with no token, cut to three of the 11 items and
to the fields that the client reads:

```
GET https://currency.crystal.ge/api/Conversion/GetWebRateByProductID?IsCash=true
200 application/json; charset=utf-8 (2532 bytes)
{"errorCode":"0","errorText":"","data":{"errorCode":"0","errorMessage":"",
 "requestId":null,"success":true,"statusCode":200,"data":[
 {"SourceISO":"USD","TargetISO":"GEL","Multiplier":1,"NBGRate":2.5971,"BuyRate":2.595,"SellRate":2.599,"IsCash":true,"IsDefault":1},
 {"SourceISO":"ILS","TargetISO":"GEL","Multiplier":10,"NBGRate":8.3629,"BuyRate":3.5,"SellRate":8,"IsCash":true,"IsDefault":1},
 {"SourceISO":"EUR","TargetISO":"USD","Multiplier":1,"NBGRate":2.909,"BuyRate":1.1177,"SellRate":1.1229,"IsCash":true,"IsDefault":1}],
 "jData":null}}
```

**Configuration.**

| Variable                         | Type     | Default                        | Secret | Realizes           |
| -------------------------------- | -------- | ------------------------------ | ------ | ------------------ |
| `GEOMCP_RATES_CRYSTAL_ENABLED`   | bool     | `true`                         | No     | `CRYSRATE-FR-006`  |
| `GEOMCP_RATES_CRYSTAL_BASE_URL`  | URL      | `https://currency.crystal.ge`  | No     | `CRYSRATE-FR-001`  |
| `GEOMCP_RATES_CRYSTAL_TIMEOUT`   | duration | `10s`                          | No     | `CRYSRATE-FR-001`  |
| `GEOMCP_RATES_CRYSTAL_RATE`      | float    | `1`                            | No     | `CRYSRATE-NFR-001` |
| `GEOMCP_RATES_CRYSTAL_BURST`     | int      | `1`                            | No     | `CRYSRATE-NFR-001` |

`rates.Config` gains `Crystal vendors.Config` with `envPrefix:"CRYSTAL_"`.
`rates.Defaults` gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `CRYSRATE-DD-001`

**Realizes:** `CRYSRATE-FR-001`
**Decision:** The client sends `IsCash=true` and no `RateProductID` and no `RateCityID`.
**Rationale:** With no product, the method answers the default cash rate: each item has
`IsDefault` 1. A city gives the rate of its branches, which is out of scope.
**Alternatives:** `RateProductID=5`, the default product of the list of products. It
adds a number that Crystal can change, for the same answer.

### `CRYSRATE-DD-002`

**Realizes:** `CRYSRATE-FR-001`, `CRYSRATE-FR-002`, `CRYSRATE-INV-001`
**Decision:** The adapter drops each item whose `TargetISO` is not `GEL`, before it
reads a code. It reads `Multiplier` as `CREDRATE-DD-002` gives: a power of 10 from 1 to
1 000 000, each price divided with `Shift(-k)`. Any other multiplier fails the bank.
**Rationale:** The item `EUR` against `USD` is a cross rate. The answer names the unit
of each pair. The shift is exact. The type has no field for `NBGRate`, so the official
rate cannot reach a price.
**Alternatives:** A fixed unit for each code. Crystal can change a unit, and the rate
would then be wrong by a factor of 10.

### `CRYSRATE-DD-003`

**Realizes:** `CRYSRATE-FR-004`
**Decision:** The client decodes
`{"errorCode": string, "errorText": string, "data": {"errorCode": string, "success": bool, "data": [...]}}`.
It returns an error that wraps `ErrNotSucceeded` when the outer `errorCode` is not `"0"`,
the inner `data` is absent, the inner `success` is false, the inner `errorCode` is not
`"0"`, or the list is null. The error holds the codes, and `errorText` cut to 200 bytes.
An empty list is not an error.
**Rationale:** The method answers 200 with the failure in the body. Each envelope can
carry it. The script of the app reads the inner `data` only, so a check of both is the
cheapest guard.
**Alternatives:** Check the inner `success` only. A failure of the outer envelope would
then read as an empty list, and every code would go to `Missing`.

## 6. Scenarios

`pkg/crystal/testdata/` holds `web-rate-cash.json`, the real answer of 2026-10-25, and
`not-succeeded.json`, the real answer with the inner `success` set to false.

### `CRYSRATE-SC-001`

**Verifies:** `CRYSRATE-FR-001`, `CRYSRATE-FR-006`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `CashRates` runs.
**Then** the method is `GET`, the path is `/api/Conversion/GetWebRateByProductID`, the
query is `IsCash=true`, `Accept` is `application/json`, the User-Agent starts with
`georgia-mcp/`, and the request carries no `authorization` header.

### `CRYSRATE-SC-002`

**Verifies:** `CRYSRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `web-rate-cash.json`.
**When** `CashRates` runs.
**Then** the result holds 11 items. Two items are:

| SourceISO | TargetISO | Multiplier | BuyRate | SellRate |
| --------- | --------- | ---------- | ------- | -------- |
| `USD`     | `GEL`     | `1`        | `2.595` | `2.599`  |
| `ILS`     | `GEL`     | `10`       | `3.5`   | `8`      |

### `CRYSRATE-SC-003`

**Verifies:** `CRYSRATE-FR-004`
**Layer:** unit

| Vendor answer                                          | Client result                             |
| ------------------------------------------------------ | ----------------------------------------- |
| `not-succeeded.json`                                   | `errors.Is(err, ErrNotSucceeded)`         |
| `web-rate-cash.json` with the outer `errorCode` `"500"` | `errors.Is(err, ErrNotSucceeded)`        |
| `web-rate-cash.json` with the inner `errorCode` `"1"`  | `errors.Is(err, ErrNotSucceeded)`         |
| `{"errorCode":"0","errorText":""}`                     | `errors.Is(err, ErrNotSucceeded)`         |
| `web-rate-cash.json` with the list `[]`                | An empty list, no error                   |
| 503 with a body of 2 KiB                               | `*APIError` with 503 and the first 1 KiB  |

### `CRYSRATE-SC-004`

**Verifies:** `CRYSRATE-FR-001`
**Layer:** live

**Given** the real method of Crystal.
**When** `CashRates` runs.
**Then** the item of `USD` against `GEL` has a buy price and a sell price above zero.

### `CRYSRATE-SC-005`

**Verifies:** `CRYSRATE-FR-001`, `CRYSRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `web-rate-cash.json`.
**When** the adapter asks for `EUR` and `USD`.
**Then** the result holds 2 rates, and no rate of `EUR` against `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.905`        | `2.914`         |
| `USD`    | `2.595`        | `2.599`         |

### `CRYSRATE-SC-006`

**Verifies:** `CRYSRATE-FR-002`, `CRYSRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `web-rate-cash.json`.
**When** the adapter asks for `AMD` and `ILS`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `AMD`    | `0.0067`       | `0.007`         |
| `ILS`    | `0.35`         | `0.8`           |

### `CRYSRATE-SC-007`

**Verifies:** `CRYSRATE-FR-003`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `web-rate-cash.json`, and the item
of the table.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| Second item                                                                       |
| --------------------------------------------------------------------------------- |
| `{"SourceISO":"CHF","TargetISO":"GEL","Multiplier":1,"BuyRate":0,"SellRate":3.14}` |
| `{"SourceISO":"CHF","TargetISO":"GEL","Multiplier":1,"BuyRate":3.08}`              |

### `CRYSRATE-SC-008`

**Verifies:** `CRYSRATE-FR-001`, `CRYSRATE-FR-002`, `CRYSRATE-FR-004`
**Layer:** unit

| Vendor answer                                   | Adapter error                        |
| ----------------------------------------------- | ------------------------------------ |
| `not-succeeded.json`                            | Wraps `kernel.ErrSourceUnavailable`  |
| 503                                             | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"BuyRate":"n/a"`        | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"BuyRate":-1`           | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"Multiplier":3`         | Wraps `kernel.ErrSourceUnavailable`  |
| Two items of `USD` against `GEL`                | Wraps `kernel.ErrSourceUnavailable`  |

### `CRYSRATE-SC-009`

**Verifies:** `CRYSRATE-FR-006`, `CRYSRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Crystal.
**When** the configuration loads.
**Then** `Crystal.Enabled` is true, `Crystal.BaseURL` is `https://currency.crystal.ge`,
`Crystal.HTTP.Rate` is 1, and `Crystal.HTTP.Burst` is 1.

### `CRYSRATE-SC-010`

**Verifies:** `CRYSRATE-FR-001`, `CRYSRATE-FR-005`
**Layer:** integration

**Given** Crystal on, every other bank off, the vendor serves `web-rate-cash.json`, and
a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `JPY`.
**Then** the bank `crystal` has `source` `Crystal`, `as_of` is the instant of the clock,
`buy` is `2.595`, `sell` is `2.599`, and `missing` is `["JPY"]`. The description of the
tool lists `crystal (Crystal, a microfinance organization)`.

### `CRYSRATE-SC-011`

**Verifies:** `CRYSRATE-FR-006`
**Layer:** integration

**Given** Crystal off and `bog` on.
**When** a client calls the tool with the bank `crystal`.
**Then** the tool result reads `banks: "crystal" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_CRYSTAL_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                   | Done |
| --- | -------------------------------------------------------------------------- | ------------------------------------------ | ---- |
| 1   | `pkg/crystal`, `testdata/`, README, example, live test.                    | `CRYSRATE-FR-001`, `CRYSRATE-FR-004`       | [x]  |
| 2   | `adapter/crystal`.                                                         | `CRYSRATE-FR-001` to `CRYSRATE-FR-005`, `CRYSRATE-INV-001` | [x]  |
| 3   | `config.go`: `Config.Crystal`, `Defaults`. `configs/.env.example`.         | `CRYSRATE-FR-006`, `CRYSRATE-NFR-001`      | [x]  |
| 4   | `module.go`: the entry of Crystal. The glossary.                           | `CRYSRATE-FR-001`, `CRYSRATE-FR-006`       | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
