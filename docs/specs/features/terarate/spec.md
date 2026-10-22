---
id: TERARATE
title: The commercial exchange rates of Terabank
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/terarate/requirements.md
---

# Specification: The commercial exchange rates of Terabank

## 1. Summary

This feature adds the vendor client module `pkg/terabank`, a client of the unofficial
rate API of Terabank, and Terabank as a source of `rates_get_commercial_rates`. The tool,
the use case, the domain, and the cache of `TBCRATE` do not change. One call reads
every currency.

## 2. Coverage

| Requirement        | Where this specification realizes it                         |
| ------------------ | ------------------------------------------------------------ |
| `TERARATE-FR-001`  | 4.1 `adapter/terabank`. 4.3 vendor endpoint. `TERARATE-DD-001` |
| `TERARATE-FR-002`  | `TERARATE-DD-003`                                            |
| `TERARATE-FR-003`  | 4.1 `pkg/terabank`. `TERARATE-DD-002`                        |
| `TERARATE-FR-004`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`        |
| `TERARATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                          |
| `TERARATE-INV-001` | `TERARATE-DD-004`                                            |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/terabank` is a module: `github.com/abgeo/georgia-mcp/pkg/terabank`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `encoding/json` only.                             |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags.            |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`.                                 |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the API is unofficial. `v0`.     |
| `DOM-005`, `DOM-011`   | `adapter/terabank` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-009`              | Terabank is an adapter of `CommercialRateSource`.                            |
| `MCP-002`              | No new tool. The description lists `terabank` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Terabank sits behind a `CommercialRateCache`, as TBC does.                   |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_TERABANK_` and a default in its tag. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Vendor client  | `pkg/terabank/`                       | `Client`, `New`, `Option`, `CrossRate`, `APIError`, `ErrUnexpectedStatus`, `ErrNotSucceeded`. |
| Driven adapter | `R/adapter/terabank/`                 | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Terabank`, `TerabankConfig`, `TerabankHTTPConfig`, `terabankSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Workspace      | `go.work` (change)                    | `use ./pkg/terabank`.                                        |

**`pkg/terabank`.** The shape of `pkg/bog`, with one method (`TERARATE-DD-001`).

```go
const DefaultBaseURL = "https://terabank.ge"

var (
    ErrUnexpectedStatus = errors.New("terabank: unexpected status")
    ErrNotSucceeded     = errors.New("terabank: not succeeded") // succeeded is false
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) CrossRates(ctx context.Context) ([]CrossRate, error)

type CrossRate struct {
    ISO                   string      `json:"iso"`
    Buy                   json.Number `json:"teraCrossRateBuy"`
    Sell                  json.Number `json:"teraCrossRateSell"`
    PreviousDayBuy        json.Number `json:"previousDayTeraCrossRateBuy"`
    PreviousDaySell       json.Number `json:"previousDayTeraCrossRateSell"`
    ShowPreviousDayRates  bool        `json:"showPreviosDayRates"` // the vendor spells it so
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`CrossRates` sends `POST /_mvcapi/CurrencyRatesApi/GetTeraCrossRates` with the body
`http.NoBody`, so `net/http` sends `Content-Length: 0`. It decodes the envelope
`{"data": [...], "succeeded": bool, "message": string}`, and returns `data`. A false
`succeeded` returns an error that wraps `ErrNotSucceeded` and holds `message`
(`TERARATE-DD-002`). Any status other than 200 returns an `*APIError` that wraps
`ErrUnexpectedStatus`. The body limit is 1 MiB. The documentation comment says that the
API is unofficial and publishes no limit.

**`adapter/terabank`.** `Bank` returns `{Code: "terabank", Name: "Terabank"}`, with
`Kind: domain.KindBank` once `RICORATE` adds the field.
`CommercialRates` sends one call, and converts the item of each requested code. It
matches `ISO` in upper case. It reads `Buy` and `Sell` only. `ReadAt` is the instant of
the answer.

**`module.go` (change).** `commercialSources` builds the source of Terabank when
`cfg.Terabank.Enabled` is true. Else it logs at info level `Terabank is off` with the
attribute `variable=GEOMCP_RATES_TERABANK_ENABLED`.

### 4.2 Domain model

No change. `TBCRATE` 4.2 holds the model.

### 4.3 Declarations

**Vendor endpoint.**

| Client          | Method and path                                       | Key  | Official            | Realizes          |
| --------------- | ----------------------------------------------------- | ---- | ------------------- | ----------------- |
| `pkg/terabank`  | `POST /_mvcapi/CurrencyRatesApi/GetTeraCrossRates`     | None | No, an unofficial API | `TERARATE-FR-001` |

A real answer, recorded on 2026-10-21:

```
POST https://terabank.ge/_mvcapi/CurrencyRatesApi/GetTeraCrossRates
Content-Length: 0
200 application/json; charset=utf-8
{"data":[
 {"iso":"USD","teraCrossRateBuy":2.5960,"teraCrossRateSell":2.6010,"previousDayTeraCrossRateBuy":2.5470,"previousDayTeraCrossRateSell":2.6470,"showPreviosDayRates":true},
 {"iso":"EUR","teraCrossRateBuy":2.9050,"teraCrossRateSell":2.9300,"previousDayTeraCrossRateBuy":2.8320,"previousDayTeraCrossRateSell":3.0020,"showPreviosDayRates":true},
 {"iso":"GBP","teraCrossRateBuy":3.3760,"teraCrossRateSell":3.5600,"previousDayTeraCrossRateBuy":3.3470,"previousDayTeraCrossRateSell":3.5310,"showPreviosDayRates":false}
],"succeeded":true,"message":""}

POST with no Content-Length: 411 text/html
GET: 404
```

**Configuration.**

| Variable                           | Type     | Default               | Secret | Realizes           |
| ---------------------------------- | -------- | --------------------- | ------ | ------------------ |
| `GEOMCP_RATES_TERABANK_ENABLED`    | bool     | `true`                | No     | `TERARATE-FR-004`  |
| `GEOMCP_RATES_TERABANK_BASE_URL`   | URL      | `https://terabank.ge` | No     | `TERARATE-FR-001`  |
| `GEOMCP_RATES_TERABANK_TIMEOUT`    | duration | `10s`                 | No     | `TERARATE-FR-001`  |
| `GEOMCP_RATES_TERABANK_RATE`       | float    | `1`                   | No     | `TERARATE-NFR-001` |
| `GEOMCP_RATES_TERABANK_BURST`      | int      | `1`                   | No     | `TERARATE-NFR-001` |

`rates.Config` gains `Terabank TerabankConfig` with `envPrefix:"TERABANK_"`.
`TerabankConfig` and `TerabankHTTPConfig` have the shape of `BOGConfig` and
`BOGHTTPConfig`, with the defaults of this table. `LIBRATE-DD-005` gives the reason for
the limit.

The tool, its structs, its errors, and the cache are those of `TBCRATE` 4.3 and 4.5.
The cache of Terabank is a `CommercialRateCache`: 5 minutes to live, 55 minutes of
maximum stale, 256 entries.

## 5. Design decisions

### `TERARATE-DD-001`

**Realizes:** `TERARATE-FR-001`
**Decision:** The client covers `GetTeraCrossRates` only. It leaves out
`GetTeraCrossRatesForConverter` and `GetNbgRates`.
**Rationale:** The converter method gives the same rates against the lari, and the cross
rates between two foreign currencies. The requirements leave both out. `NBGRATE` reads
the official rate from the NBG.
**Alternatives:** The converter method. Its pairs need a filter on `iso2` of `GEL`, and
its sell field has the name `terCrossRateSell`.

### `TERARATE-DD-002`

**Realizes:** `TERARATE-FR-003`, `CLT-007`
**Decision:** A 200 with `succeeded` false returns `ErrNotSucceeded`. The adapter wraps it
in `kernel.ErrSourceUnavailable`.
**Rationale:** The page treats such an answer as a failure. An empty `data` with a false
flag is a failure of the source, not a bank that quotes nothing.
**Alternatives:** Return `data` and ignore the flag. The cache would then store every
code as a code that Terabank does not quote.

### `TERARATE-DD-003`

**Realizes:** `TERARATE-FR-002`
**Decision:** A requested code with no item, or with a `Buy` or `Sell` that is absent,
empty, or zero, goes to `Missing`. A price that is not a number, or is below zero, fails
the bank.
**Rationale:** The same rule as `BOGRATE-DD-008`.
**Alternatives:** None.

### `TERARATE-DD-004`

**Realizes:** `TERARATE-INV-001`
**Decision:** The adapter parses the text of `Buy` and `Sell` with
`decimal.NewFromString`, and drops the trailing zeros.
**Rationale:** `json.Number` keeps the digits of the answer. The source pads each price
to four decimal places: `2.9300`. The drop gives `2.93`, as `BOGRATE-DD-005` does.
**Alternatives:** `float64`. `DOM-006` forbids it.

## 6. Scenarios

`pkg/terabank/testdata/` holds `cross-rates.json`, the real answer of 2026-10-21, and
`not-succeeded.json`, `{"data":null,"succeeded":false,"message":"error"}`.

### `TERARATE-SC-001`

**Verifies:** `TERARATE-FR-001`, `TERARATE-FR-004`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `CrossRates` runs.
**Then** the method is `POST`, the path is `/_mvcapi/CurrencyRatesApi/GetTeraCrossRates`,
`Content-Length` is `0`, `Accept` is `application/json`, the User-Agent starts with
`georgia-mcp-terabank/`, and the request carries no key header.

### `TERARATE-SC-002`

**Verifies:** `TERARATE-FR-001`
**Layer:** unit

**Given** the vendor serves `cross-rates.json`.
**When** `CrossRates` runs.
**Then** the result holds 3 items. The item of `USD` has `Buy` `2.5960`, `Sell` `2.6010`,
and `PreviousDayBuy` `2.5470`.

### `TERARATE-SC-003`

**Verifies:** `TERARATE-FR-003`
**Layer:** unit

| Vendor answer                         | Client result                                         |
| ------------------------------------- | ----------------------------------------------------- |
| `not-succeeded.json`                  | `errors.Is(err, ErrNotSucceeded)`. The text holds `error`. |
| 411 with a body of 2 KiB              | `*APIError` with 411 and the first 1 KiB              |

### `TERARATE-SC-004`

**Verifies:** `TERARATE-FR-001`
**Layer:** live

**Given** the real API of Terabank.
**When** `CrossRates` runs.
**Then** the item of `USD` has a buy price and a sell price above zero.

### `TERARATE-SC-005`

**Verifies:** `TERARATE-FR-001`, `TERARATE-INV-001`
**Layer:** unit

**Given** the vendor serves `cross-rates.json`.
**When** the adapter asks for `EUR`, `GBP`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.905`        | `2.93`          |
| `GBP`    | `3.376`        | `3.56`          |
| `USD`    | `2.596`        | `2.601`         |

### `TERARATE-SC-006`

**Verifies:** `TERARATE-FR-002`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `cross-rates.json`, and the item of
the table for `CHF`.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| Item of `CHF`                                              |
| ---------------------------------------------------------- |
| `{"iso":"CHF","teraCrossRateBuy":0,"teraCrossRateSell":3.2}` |
| `{"iso":"CHF","teraCrossRateBuy":3.1}`                     |

### `TERARATE-SC-007`

**Verifies:** `TERARATE-FR-001`, `TERARATE-FR-003`
**Layer:** unit

| Vendor answer                                               | Adapter error                        |
| ----------------------------------------------------------- | ------------------------------------ |
| `not-succeeded.json`                                        | Wraps `kernel.ErrSourceUnavailable`  |
| 500                                                         | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"teraCrossRateBuy":"n/a"`           | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"teraCrossRateBuy":-1`              | Wraps `kernel.ErrSourceUnavailable`  |

### `TERARATE-SC-008`

**Verifies:** `TERARATE-FR-004`, `TERARATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Terabank.
**When** the configuration loads.
**Then** `Terabank.Enabled` is true, `Terabank.BaseURL` is `https://terabank.ge`,
`Terabank.HTTP.Rate` is 1, and `Terabank.HTTP.Burst` is 1.

### `TERARATE-SC-009`

**Verifies:** `TERARATE-FR-001`
**Layer:** integration

**Given** Terabank on, every other bank off, the vendor serves `cross-rates.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the bank `terabank` has `source` `Terabank`, `as_of` is the instant of the clock,
`buy` is `2.596`, `sell` is `2.601`, and `missing` is `["CHF"]`.

### `TERARATE-SC-010`

**Verifies:** `TERARATE-FR-004`
**Layer:** integration

**Given** Terabank off and `bog` on.
**When** a client calls the tool with the bank `terabank`.
**Then** the tool result reads
`banks: "terabank" is not a bank of this server. Use one of: bog.` The log holds one info
record that names `GEOMCP_RATES_TERABANK_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                   | Done |
| --- | -------------------------------------------------------------------------- | ------------------------------------------ | ---- |
| 1   | `pkg/terabank` module, recorded answers, README, example. `go.work`.       | `TERARATE-FR-001`, `TERARATE-FR-003`       | [x]  |
| 2   | `adapter/terabank`.                                                        | `TERARATE-FR-001` to `TERARATE-FR-003`, `TERARATE-INV-001` | [x]  |
| 3   | `config.go`: `TerabankConfig`, `TerabankHTTPConfig`. `configs/.env.example`. | `TERARATE-FR-004`, `TERARATE-NFR-001`      | [x]  |
| 4   | `module.go`: the source of Terabank, and the info record.                  | `TERARATE-FR-001`, `TERARATE-FR-004`       | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
