---
id: MBCRATE
title: The commercial exchange rates of MBC
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/mbcrate/requirements.md
---

# Specification: The commercial exchange rates of MBC

## 1. Summary

This feature adds the vendor client `pkg/mbc`, a client of the unofficial rate method
that the page of MBC reads, and MBC as a source of `rates_get_commercial_rates`. The
tool, the use case, the domain, and the cache of `TBCRATE` do not change. One call
reads every currency. The adapter keeps only the currencies that the page shows.

## 2. Coverage

| Requirement       | Where this specification realizes it                      |
| ----------------- | --------------------------------------------------------- |
| `MBCRATE-FR-001`  | 4.1 `adapter/mbc`. 4.3 vendor endpoint.                    |
| `MBCRATE-FR-002`  | `MBCRATE-DD-001`                                          |
| `MBCRATE-FR-003`  | `MBCRATE-DD-002`. `CREDRATE-DD-002`                        |
| `MBCRATE-FR-004`  | 4.1 `adapter/mbc`. `LIBRATE-DD-004`                       |
| `MBCRATE-FR-005`  | 4.1 `adapter/mbc`. `RICORATE-DD-004`                      |
| `MBCRATE-FR-006`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`     |
| `MBCRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                       |
| `MBCRATE-INV-001` | `MBCRATE-DD-002`                                          |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/mbc` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/mbc`. |
| `CLT-002`, `CLT-004`   | The standard library and `pkg/vendorhttp` only. `pkg/mbc` stays in `clients`. |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags.            |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`.                                 |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the method is unofficial.        |
| `DOM-005`, `DOM-011`   | `adapter/mbc` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Each price is a `json.Number`. The adapter parses its text into a decimal.   |
| `DOM-009`              | MBC is an adapter of `CommercialRateSource`.                                 |
| `MCP-002`              | No new tool. The description lists `mbc` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | MBC sits behind a `CommercialRateCache`, as TBC does.                        |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_MBC_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `MBCRATE` 5.                              |
| Vendor client  | `pkg/mbc/`                            | `Client`, `New`, `Option`, `Rate`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRates`. |
| Driven adapter | `R/adapter/mbc/`                      | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.MBC`, the entry of MBC in `otherEntries`, `mbcSwitchVariable`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |

**`pkg/mbc`.** The shape of `pkg/credo`, with one method and no key.

```go
const DefaultBaseURL = "https://fxrates.mbc.com.ge:8022"

var (
    ErrUnexpectedStatus = errors.New("mbc: unexpected status")
    ErrNoRates          = errors.New("mbc: no rates") // FXRates is absent or null
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) CommercialRates(ctx context.Context) ([]Rate, error)

// Rate is one pair of currencies of an answer.
type Rate struct {
    FromCcy    string      `json:"FromCcy"`
    ToCcy      string      `json:"ToCcy"`
    Multiplier json.Number `json:"Multiplier"`
    Buy        json.Number `json:"Buy"`
    Sell       json.Number `json:"Sell"`
}

type APIError = vendorhttp.StatusError
```

`CommercialRates` sends `GET /api/fxrates/mbc/commercial` with
`Accept: application/json`. It decodes `{"FXRates": [...], "timestamp": n}` with
`UseNumber`, and returns `FXRates`. An absent or null `FXRates` returns `ErrNoRates`.
An empty list is not an error. Any status other than 200 returns an `*APIError` that
wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the answer has 406 bytes. The
client ignores `timestamp` (`MBCRATE-DD-003`).

**`adapter/mbc`.** `Bank` returns
`{Code: "mbc", Name: "MBC", Kind: domain.KindMicrofinance}`. `CommercialRates` sends one
call. It keeps each item whose `ToCcy` is `GEL` in any case, and whose `FromCcy` is in
the set of `MBCRATE-DD-001`. It converts the item of each requested code:

| Field of the item | Domain value                                                     |
| ----------------- | ---------------------------------------------------------------- |
| `FromCcy`         | Matched against the requested code in upper case.                |
| `Multiplier`      | The count of units. `MBCRATE-DD-002`.                            |
| `Buy`, `Sell`     | `decimal.NewFromString`, divided by the count, trailing zeros dropped. |

A requested code outside the set, with no item, or with a price that is absent, empty,
or zero, goes to `Missing`. A price that is not a number or is below zero fails the
bank. Two items of one code against the lari fail the bank. `ReadAt` is the instant of
the answer.

**`module.go` (change).** `otherEntries` gains the entry of MBC after Inteli Express,
with the name `MBC` and the variable `GEOMCP_RATES_MBC_ENABLED`.

### 4.2 Domain model

No change. `RICORATE` 4.2 holds `KindMicrofinance`.

### 4.3 Declarations

**Vendor endpoint.**

| Client     | Method and path                      | Key  | Official                   | Realizes         |
| ---------- | ------------------------------------ | ---- | -------------------------- | ---------------- |
| `pkg/mbc`  | `GET /api/fxrates/mbc/commercial`    | None | No, an unofficial method   | `MBCRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to two of the 5 items:

```
GET https://fxrates.mbc.com.ge:8022/api/fxrates/mbc/commercial
200 application/json; charset=utf-8 (406 bytes)
Cache-Control: no-cache
Access-Control-Allow-Origin: *
{"FXRates":[
 {"FromCcy":"EUR","ToCcy":"GEL","Multiplier":1,"Buy":2.9070,"Sell":2.9140},
 {"FromCcy":"USD","ToCcy":"GEL","Multiplier":1,"Buy":2.5950,"Sell":2.5990}],
 "timestamp":1792945527}
```

**Configuration.**

| Variable                     | Type     | Default                            | Secret | Realizes          |
| ---------------------------- | -------- | ---------------------------------- | ------ | ----------------- |
| `GEOMCP_RATES_MBC_ENABLED`   | bool     | `true`                             | No     | `MBCRATE-FR-006`  |
| `GEOMCP_RATES_MBC_BASE_URL`  | URL      | `https://fxrates.mbc.com.ge:8022`  | No     | `MBCRATE-FR-001`  |
| `GEOMCP_RATES_MBC_TIMEOUT`   | duration | `10s`                              | No     | `MBCRATE-FR-001`  |
| `GEOMCP_RATES_MBC_RATE`      | float    | `1`                                | No     | `MBCRATE-NFR-001` |
| `GEOMCP_RATES_MBC_BURST`     | int      | `1`                                | No     | `MBCRATE-NFR-001` |

`rates.Config` gains `MBC vendors.Config` with `envPrefix:"MBC_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `MBCRATE-DD-001`

**Realizes:** `MBCRATE-FR-002`
**Decision:** The adapter holds a fixed set of the codes that the page shows: `EUR`,
`RUB`, and `USD`. An item of another code is dropped before the adapter reads it.
**Rationale:** The owner chose a fixed set on 2026-10-25. One call reads the rates, and
the set changes only when MBC changes its page. This is the one place where a document
names the rouble.
**Alternatives:** A second call that reads the codes from the page. It follows a change
of the page, but it doubles the calls and parses a page for people. `USD` and `EUR`
only. That drops a currency that the page shows.

### `MBCRATE-DD-002`

**Realizes:** `MBCRATE-FR-003`, `MBCRATE-INV-001`
**Decision:** The adapter drops each item whose `ToCcy` is not `GEL`, and reads
`Multiplier` as `CREDRATE-DD-002` gives: a power of 10 from 1 to 1 000 000, each price
divided with `Shift(-k)`. Any other multiplier fails the bank.
**Rationale:** The answer names the unit of each pair, as the answer of Credo does. The
shift is exact.
**Alternatives:** Ignore `Multiplier`, since it is 1 today. A change would then give a
wrong rate with no error.

### `MBCRATE-DD-003`

**Realizes:** `MBCRATE-FR-001`
**Decision:** `ReadAt` is the instant of the clock of the adapter. The adapter ignores
`timestamp`.
**Rationale:** `timestamp` is the instant of the answer, not of a change of a price.
`MCP-005` asks for the instant at which the server read the rate, and the clock gives it.
**Alternatives:** `timestamp` as `ReadAt`. It differs from the clock only by the delay
of the network and the skew of the clock of MBC.

## 6. Scenarios

`pkg/mbc/testdata/` holds `commercial.json`, the real answer of 2026-10-25, and
`no-rates.json`, `{"timestamp":1792945527}`.

### `MBCRATE-SC-001`

**Verifies:** `MBCRATE-FR-001`, `MBCRATE-FR-006`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `CommercialRates` runs.
**Then** the method is `GET`, the path is `/api/fxrates/mbc/commercial`, the query is
empty, `Accept` is `application/json`, the User-Agent starts with `georgia-mcp/`, and
the request carries no key header.

### `MBCRATE-SC-002`

**Verifies:** `MBCRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `commercial.json`.
**When** `CommercialRates` runs.
**Then** the result holds 5 items. The item of `USD` has `ToCcy` `GEL`, `Multiplier`
`1`, `Buy` `2.5950`, and `Sell` `2.5990`.

### `MBCRATE-SC-003`

**Verifies:** `MBCRATE-FR-001`
**Layer:** unit

| Vendor answer                    | Client result                             |
| -------------------------------- | ----------------------------------------- |
| `no-rates.json`                  | `errors.Is(err, ErrNoRates)`              |
| `{"FXRates":[],"timestamp":1}`   | An empty list, no error                   |
| 503 with a body of 2 KiB         | `*APIError` with 503 and the first 1 KiB  |

### `MBCRATE-SC-004`

**Verifies:** `MBCRATE-FR-001`
**Layer:** live

**Given** the real method of MBC.
**When** `CommercialRates` runs.
**Then** the item of `USD` against `GEL` has a buy price and a sell price above zero.

### `MBCRATE-SC-005`

**Verifies:** `MBCRATE-FR-001`, `MBCRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `commercial.json`.
**When** the adapter asks for `EUR` and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.907`        | `2.914`         |
| `USD`    | `2.595`        | `2.599`         |

### `MBCRATE-SC-006`

**Verifies:** `MBCRATE-FR-002`
**Layer:** unit

**Given** the vendor serves `commercial.json`.
**When** the adapter asks for `GBP`, `TRY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[GBP TRY]`.

### `MBCRATE-SC-007`

**Verifies:** `MBCRATE-FR-003`, `MBCRATE-INV-001`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `commercial.json`, and
`{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":10,"Buy":29.0700,"Sell":29.1400}`.
**When** the adapter asks for `EUR`.
**Then** the buy rate formats as `2.907`, and the sell rate as `2.914`.

### `MBCRATE-SC-008`

**Verifies:** `MBCRATE-FR-004`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `commercial.json`, and the item of
the table.
**When** the adapter asks for `EUR` and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[EUR]`.

| Second item                                                              |
| ------------------------------------------------------------------------ |
| `{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":1,"Buy":0,"Sell":2.914}`    |
| `{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":1,"Buy":2.907}`             |
| `{"FromCcy":"EUR","ToCcy":"USD","Multiplier":1,"Buy":1.11,"Sell":1.13}`  |

### `MBCRATE-SC-009`

**Verifies:** `MBCRATE-FR-001`, `MBCRATE-FR-003`
**Layer:** unit

| Vendor answer                                 | Adapter error                        |
| --------------------------------------------- | ------------------------------------ |
| `no-rates.json`                               | Wraps `kernel.ErrSourceUnavailable`  |
| 503                                           | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"Buy":"n/a"`          | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"Buy":-1`             | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"Multiplier":3`       | Wraps `kernel.ErrSourceUnavailable`  |
| Two items of `USD` against `GEL`              | Wraps `kernel.ErrSourceUnavailable`  |

### `MBCRATE-SC-010`

**Verifies:** `MBCRATE-FR-006`, `MBCRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of MBC.
**When** the configuration loads.
**Then** `MBC.Enabled` is true, `MBC.BaseURL` is `https://fxrates.mbc.com.ge:8022`,
`MBC.HTTP.Rate` is 1, and `MBC.HTTP.Burst` is 1.

### `MBCRATE-SC-011`

**Verifies:** `MBCRATE-FR-001`, `MBCRATE-FR-005`
**Layer:** integration

**Given** MBC on, every other bank off, the vendor serves `commercial.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `GBP`.
**Then** the bank `mbc` has `source` `MBC`, `as_of` is the instant of the clock, `buy`
is `2.595`, `sell` is `2.599`, and `missing` is `["GBP"]`. The description of the tool
lists `mbc (MBC, a microfinance organization)`.

### `MBCRATE-SC-012`

**Verifies:** `MBCRATE-FR-006`
**Layer:** integration

**Given** MBC off and `bog` on.
**When** a client calls the tool with the bank `mbc`.
**Then** the tool result reads `banks: "mbc" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_MBC_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                 | Done |
| --- | -------------------------------------------------------------------------- | ---------------------------------------- | ---- |
| 1   | `pkg/mbc`, `testdata/`, README, example, live test.                        | `MBCRATE-FR-001`                         | [x]  |
| 2   | `adapter/mbc`.                                                             | `MBCRATE-FR-001` to `MBCRATE-FR-005`, `MBCRATE-INV-001` | [x]  |
| 3   | `config.go`: `Config.MBC`, `Defaults`. `configs/.env.example`.             | `MBCRATE-FR-006`, `MBCRATE-NFR-001`      | [x]  |
| 4   | `module.go`: the entry of MBC. The glossary.                               | `MBCRATE-FR-001`, `MBCRATE-FR-006`       | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
