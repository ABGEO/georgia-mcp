---
id: KURSRATE
title: The commercial exchange rates of Kursi.ge
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/kursrate/requirements.md
---

# Specification: The commercial exchange rates of Kursi.ge

## 1. Summary

This feature adds the vendor client module `pkg/kursi`, a client of the unofficial
rate API of Kursi.ge, and Kursi.ge as a source of `rates_get_commercial_rates`. It adds
the kind of an online exchange bureau to the `BankKind` of `RICORATE`. One call reads
every currency.

## 2. Coverage

| Requirement        | Where this specification realizes it                          |
| ------------------ | ------------------------------------------------------------- |
| `KURSRATE-FR-001`  | 4.1 `adapter/kursi`. 4.3 vendor endpoint. `KURSRATE-DD-001`    |
| `KURSRATE-FR-002`  | 4.1 `adapter/kursi`. `KURSRATE-DD-002`                         |
| `KURSRATE-FR-003`  | `KURSRATE-DD-003`                                              |
| `KURSRATE-FR-004`  | 4.2 `KindExchangeBureau`. `RICORATE-DD-004`                    |
| `KURSRATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`          |
| `KURSRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                            |
| `KURSRATE-INV-001` | `TERARATE-DD-004`                                              |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/kursi` is a module: `github.com/abgeo/georgia-mcp/pkg/kursi`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `encoding/json` only.                             |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags.            |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`.                                 |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the API is unofficial, and name the documented merchant API. `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/kursi` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `MCP-002`              | No new tool. The description lists `kursi` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Kursi.ge sits behind a `CommercialRateCache`, as TBC does. `KURSRATE-DD-004`. |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_KURSI_` and a default in its tag. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Vendor client  | `pkg/kursi/`                          | `Client`, `New`, `Option`, `Currency`, `BankRate`, `APIError`, `ErrUnexpectedStatus`. |
| Domain         | `R/domain/bank.go` (change)           | `KindExchangeBureau`. 4.2.                                   |
| Driven adapter | `R/adapter/kursi/`                    | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Kursi`, `kursiSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Workspace      | `go.work` (change)                    | `use ./pkg/kursi`.                                           |

**`pkg/kursi`.** The shape of `pkg/bog`, with one method.

```go
const DefaultBaseURL = "https://api-core.kursi.ge"

var ErrUnexpectedStatus = errors.New("kursi: unexpected status")

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Currencies(ctx context.Context) ([]Currency, error)

// Currency is one pair of currencies, with the rates of Kursi.ge for it.
type Currency struct {
    BaseCurrencyCode      string              `json:"baseCurrencyCode"`
    SecondaryCurrencyCode string              `json:"secondaryCurrencyCode"`
    Name                  string              `json:"name"`
    NBGRate               json.Number         `json:"nbgRate"`
    Diff                  json.Number         `json:"diff"`
    BuyRate               json.Number         `json:"buyRate"`
    SellRate              json.Number         `json:"sellRate"`
    BankRates             map[string]BankRate `json:"bankRates"`
}

// BankRate is the rate of a bank that Kursi.ge repeats.
type BankRate struct {
    BuyRate  json.Number `json:"buyRate"`
    SellRate json.Number `json:"sellRate"`
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`Currencies` sends `GET /api/public/currencies`. Any status other than 200 returns an
`*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB. The default
User-Agent is `georgia-mcp-kursi/<version> (+https://github.com/abgeo/georgia-mcp)`.

**`adapter/kursi`.** `Bank` returns
`{Code: "kursi", Name: "Kursi.ge", Kind: domain.KindExchangeBureau}`. `CommercialRates`
sends one call. It keeps each pair whose `BaseCurrencyCode` is `GEL`, and converts the
pair of each requested code:

| Field of the pair         | Domain value                                                   |
| ------------------------- | -------------------------------------------------------------- |
| `SecondaryCurrencyCode`   | `domain.ParseCurrencyCode`. A code that fails fails the bank.  |
| `BuyRate`                 | `Buy`. `KURSRATE-DD-001`.                                      |
| `SellRate`                | `Sell`.                                                        |
| `NBGRate`, `Diff`, `BankRates` | Not read.                                                 |

The prices are parsed as `TERARATE-DD-004` gives. `ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Kursi.ge when
`cfg.Kursi.Enabled` is true. Else it logs at info level `Kursi.ge is off` with the
attribute `variable=GEOMCP_RATES_KURSI_ENABLED`.

### 4.2 Domain model

`RICORATE` 4.2 holds `BankKind`. This feature adds one value:

```go
const KindExchangeBureau BankKind = "online exchange bureau"
```

`bankList` then writes `kursi (Kursi.ge, an online exchange bureau)`.

### 4.3 Declarations

**Vendor endpoint.**

| Client       | Method and path               | Key  | Official              | Realizes          |
| ------------ | ----------------------------- | ---- | --------------------- | ----------------- |
| `pkg/kursi`  | `GET /api/public/currencies`  | None | No, an unofficial API | `KURSRATE-FR-001` |

A real answer, recorded on 2026-10-21, cut to three of the 8 pairs:

```
GET https://api-core.kursi.ge/api/public/currencies
200 application/json
Cache-Control: no-cache, no-store, max-age=0, must-revalidate
[
 {"baseCurrencyCode":"GEL","secondaryCurrencyCode":"USD","name":"აშშ დოლარი","nbgRate":2.5978,"diff":-0.0001,
  "buyRate":2.5940,"sellRate":2.6010,
  "bankRates":{"TBC":{"buyRate":2.5650,"sellRate":2.6270},"BOG":{"buyRate":2.5650,"sellRate":2.6290}}},
 {"baseCurrencyCode":"USD","secondaryCurrencyCode":"EUR","name":"EUR","nbgRate":0,"diff":0,
  "buyRate":1.1200,"sellRate":1.1290,
  "bankRates":{"TBC":{"buyRate":1.0944,"sellRate":1.1509},"BOG":{"buyRate":1.0928,"sellRate":1.1521}}},
 {"baseCurrencyCode":"GEL","secondaryCurrencyCode":"CNY","name":"ჩინური იუანი","nbgRate":0.3891,"diff":0.0023,
  "buyRate":0.3858,"sellRate":0.3925,
  "bankRates":{"TBC":{"buyRate":0.3720,"sellRate":0.4060},"BOG":{"buyRate":0.3708,"sellRate":0.4070}}}
]
```

**Configuration.**

| Variable                        | Type     | Default                      | Secret | Realizes           |
| ------------------------------- | -------- | ---------------------------- | ------ | ------------------ |
| `GEOMCP_RATES_KURSI_ENABLED`    | bool     | `true`                       | No     | `KURSRATE-FR-005`  |
| `GEOMCP_RATES_KURSI_BASE_URL`   | URL      | `https://api-core.kursi.ge`  | No     | `KURSRATE-FR-001`  |
| `GEOMCP_RATES_KURSI_TIMEOUT`    | duration | `10s`                        | No     | `KURSRATE-FR-001`  |
| `GEOMCP_RATES_KURSI_RATE`       | float    | `1`                          | No     | `KURSRATE-NFR-001` |
| `GEOMCP_RATES_KURSI_BURST`      | int      | `1`                          | No     | `KURSRATE-NFR-001` |

`rates.Config` gains `Kursi vendors.Config` with `envPrefix:"KURSI_"`. `rates.Defaults` gives the defaults of this table (`PLATFORM-DD-014`).

## 5. Design decisions

### `KURSRATE-DD-001`

**Realizes:** `KURSRATE-FR-001`
**Decision:** For a pair with the base `GEL`, `BuyRate` is the buy rate and `SellRate` is
the sell rate of the secondary currency.
**Rationale:** In the answer of 2026-10-21, `bankRates.BOG` of `USD` equals the answer of
the API of Bank of Georgia at the same minute: `Buy` 2.565 and `Sell` 2.629. The
calculator of the page also multiplies an amount of the secondary currency by `buyRate`
to give the lari that the person gets.
**Alternatives:** None. The other reading swaps the two prices.

### `KURSRATE-DD-002`

**Realizes:** `KURSRATE-FR-002`
**Decision:** The adapter drops each pair whose `BaseCurrencyCode` is not `GEL`, before it
reads a code. Two pairs with the base `GEL` and the same secondary code fail the bank.
**Rationale:** A pair between two foreign currencies has a price in a currency other
than the lari. Two prices for one code is a broken answer.
**Alternatives:** Keep the first of two pairs. The choice would hide a change of the source.

### `KURSRATE-DD-003`

**Realizes:** `KURSRATE-FR-003`
**Decision:** A requested code with no pair against the lari, or with a `BuyRate` or
`SellRate` that is absent, empty, or zero, goes to `Missing`. A price that is not a
number, or is below zero, fails the bank.
**Rationale:** The same rule as `BOGRATE-DD-008`.
**Alternatives:** None.

### `KURSRATE-DD-004`

**Realizes:** `KURSRATE-FR-001`, `RSL-004`
**Decision:** The cache keeps a rate of Kursi.ge for 5 minutes, as for each bank, against
the `no-store` of the answer.
**Rationale:** `TBCRATE-NFR-001` and `TBCRATE-NFR-002` give 5 minutes for every bank.
The header protects a person in a browser from an old price before an order. The tool
gives `as_of`, so the model knows the age of the rate.
**Alternatives:** No cache for Kursi.ge. Each request then sends one call, against
`TBCRATE-NFR-002`.

## 6. Scenarios

`pkg/kursi/testdata/` holds `currencies.json`, the real answer of 2026-10-21.

### `KURSRATE-SC-001`

**Verifies:** `KURSRATE-FR-001`, `KURSRATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Currencies` runs.
**Then** the method is `GET`, the path is `/api/public/currencies`, `Accept` is
`application/json`, the User-Agent starts with `georgia-mcp-kursi/`, and the request
carries no `Authorization` header.

### `KURSRATE-SC-002`

**Verifies:** `KURSRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `currencies.json`.
**When** `Currencies` runs.
**Then** the result holds 8 pairs. The pair of `GEL` and `USD` has `BuyRate` `2.5940`,
`SellRate` `2.6010`, and `BankRates["BOG"].BuyRate` `2.5650`.

### `KURSRATE-SC-003`

**Verifies:** `KURSRATE-FR-001`
**Layer:** unit

| Vendor answer             | Client result                                 |
| ------------------------- | --------------------------------------------- |
| 500 with a body of 2 KiB  | `*APIError` with 500 and the first 1 KiB      |
| 200 with `[` only         | An error that is not an `*APIError`           |

### `KURSRATE-SC-004`

**Verifies:** `KURSRATE-FR-001`
**Layer:** live

**Given** the real API of Kursi.ge.
**When** `Currencies` runs.
**Then** the pair of `GEL` and `USD` has a buy price and a sell price above zero.

### `KURSRATE-SC-005`

**Verifies:** `KURSRATE-FR-001`, `KURSRATE-FR-002`, `KURSRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `currencies.json`.
**When** the adapter asks for `CNY`, `EUR`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `CNY`    | `0.3858`       | `0.3925`        |
| `EUR`    | `2.911`        | `2.926`         |
| `USD`    | `2.594`        | `2.601`         |

The rate of `EUR` is not `1.12`, the price of the pair of `USD` and `EUR`.

### `KURSRATE-SC-006`

**Verifies:** `KURSRATE-FR-003`
**Layer:** unit

**Given** a vendor that serves the pair of `GEL` and `USD` of `currencies.json`, and the
pair of the table.
**When** the adapter asks for `CHF`, `GBP`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF GBP]`.

| Second pair                                                                    |
| ------------------------------------------------------------------------------ |
| `{"baseCurrencyCode":"GEL","secondaryCurrencyCode":"CHF","buyRate":0,"sellRate":3.2}` |
| `{"baseCurrencyCode":"USD","secondaryCurrencyCode":"CHF","buyRate":0.8,"sellRate":0.9}` |

### `KURSRATE-SC-007`

**Verifies:** `KURSRATE-FR-001`, `KURSRATE-FR-002`
**Layer:** unit

| Vendor answer                                                          | Adapter error                       |
| ---------------------------------------------------------------------- | ----------------------------------- |
| 500                                                                    | Wraps `kernel.ErrSourceUnavailable` |
| The pair of `GEL` and `USD` with `"buyRate":"n/a"`                     | Wraps `kernel.ErrSourceUnavailable` |
| The pair of `GEL` and `USD` with `"buyRate":-1`                        | Wraps `kernel.ErrSourceUnavailable` |
| Two pairs of `GEL` and `USD`                                           | Wraps `kernel.ErrSourceUnavailable` |

### `KURSRATE-SC-008`

**Verifies:** `KURSRATE-FR-004`
**Layer:** unit

**Given** the getter holds `{kursi, Kursi.ge, online exchange bureau}`.
**When** the tool builds its description.
**Then** the list reads `kursi (Kursi.ge, an online exchange bureau)`.

### `KURSRATE-SC-009`

**Verifies:** `KURSRATE-FR-005`, `KURSRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Kursi.ge.
**When** the configuration loads.
**Then** `Kursi.Enabled` is true, `Kursi.BaseURL` is `https://api-core.kursi.ge`,
`Kursi.HTTP.Rate` is 1, and `Kursi.HTTP.Burst` is 1.

### `KURSRATE-SC-010`

**Verifies:** `KURSRATE-FR-001`, `KURSRATE-FR-004`
**Layer:** integration

**Given** Kursi.ge on, every other bank off, the vendor serves `currencies.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `GBP`.
**Then** the bank `kursi` has `source` `Kursi.ge`, `as_of` is the instant of the clock,
`buy` is `2.594`, `sell` is `2.601`, and `missing` is `["GBP"]`.

### `KURSRATE-SC-011`

**Verifies:** `KURSRATE-FR-005`
**Layer:** integration

**Given** Kursi.ge off and `bog` on.
**When** a client calls the tool with the bank `kursi`.
**Then** the tool result reads `banks: "kursi" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_KURSI_ENABLED`.

## 7. Build plan

`RICORATE` steps 1 and 2 come first: they add `BankKind`.

| #   | Step                                                                     | Realizes                                   | Done |
| --- | ------------------------------------------------------------------------ | ------------------------------------------ | ---- |
| 1   | `domain.KindExchangeBureau`.                                             | `KURSRATE-FR-004`                          | [x]  |
| 2   | `pkg/kursi` module, `testdata/currencies.json`, README, example. `go.work`. | `KURSRATE-FR-001`                       | [x]  |
| 3   | `adapter/kursi`.                                                         | `KURSRATE-FR-001` to `KURSRATE-FR-003`, `KURSRATE-INV-001` | [x]  |
| 4   | `config.go`: `KursiConfig`, `KursiHTTPConfig`. `configs/.env.example`.   | `KURSRATE-FR-005`, `KURSRATE-NFR-001`      | [x]  |
| 5   | `module.go`: the source of Kursi.ge, and the info record.                | `KURSRATE-FR-001`, `KURSRATE-FR-005`       | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
