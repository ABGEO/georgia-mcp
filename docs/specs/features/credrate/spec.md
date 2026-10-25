---
id: CREDRATE
title: The commercial exchange rates of Credo Bank
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/credrate/requirements.md
---

# Specification: The commercial exchange rates of Credo Bank

## 1. Summary

This feature adds the vendor client `pkg/credo`, a client of the rate method that the site
of Credo Bank reads, and Credo as a source of `rates_get_commercial_rates`. Credo needs a
key that the operator sets, as TBC does. The tool, the use case, the domain, and the cache
of `TBCRATE` do not change. One call reads every currency.

## 2. Coverage

| Requirement        | Where this specification realizes it                        |
| ------------------ | ----------------------------------------------------------- |
| `CREDRATE-FR-001`  | 4.1 `adapter/credo`. 4.3 vendor endpoint. `CREDRATE-DD-001` |
| `CREDRATE-FR-002`  | `CREDRATE-DD-002`                                           |
| `CREDRATE-FR-003`  | 4.1 `adapter/credo`. `CREDRATE-DD-002`, `BOGRATE-DD-008`    |
| `CREDRATE-FR-004`  | 4.1 `pkg/credo`. `TERARATE-DD-002`                          |
| `CREDRATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`       |
| `CREDRATE-FR-006`  | `CREDRATE-DD-003`                                           |
| `CREDRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                         |
| `CREDRATE-INV-001` | `CREDRATE-DD-002`                                           |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/credo` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/credo`. |
| `CLT-002`, `CLT-004`   | The standard library and `pkg/vendorhttp` only.                              |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags.            |
| `CLT-005`              | `Accept: application/json`. `WithAPIKey` sets the header `x-api-key`.        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | No key in the source. The package comment and the README say that the API is unofficial, and that the key comes from the caller. |
| `DOM-005`, `DOM-011`   | `adapter/credo` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-009`              | Credo is an adapter of `CommercialRateSource`.                               |
| `MCP-002`              | No new tool. The description lists `credo` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Credo sits behind a `CommercialRateCache`, as TBC does.                      |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_CREDO_` and a default in `rates.Defaults`. |
| `CFG-007`, `CFG-008`   | On by default. The key is required while Credo is on, and is not in the repository. |
| `LOG-008`              | No record holds the key.                                                     |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `CREDRATE` 5.                             |
| Vendor client  | `pkg/credo/`                          | `Client`, `New`, `Option`, `WithAPIKey`, `MethodCommercial`, `Rate`, `APIError`, `ErrUnexpectedStatus`, `ErrNotSucceeded`. |
| Driven adapter | `R/adapter/credo/`                    | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Credo`, `CredoConfig`, `credoSource`.                |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |

**`pkg/credo`.** The shape of `pkg/tbc`, with one method.

```go
const DefaultBaseURL = "https://n1.noxtton.com"

// MethodCommercial is the kind of rate of the branches of Credo.
const MethodCommercial = "FxRatesCommercial"

var (
    ErrUnexpectedStatus = errors.New("credo: unexpected status")
    ErrNotSucceeded     = errors.New("credo: not succeeded") // success is false
)

func New(opts ...Option) *Client // the options of vendorhttp
func WithAPIKey(key string) Option
func (c *Client) CurrentRates(ctx context.Context, method string, at time.Time) ([]Rate, error)

// Rate is one pair of currencies of an answer.
type Rate struct {
    FromCcy    string      `json:"fromCcy"`
    ToCcy      string      `json:"toCcy"`
    Multiplier json.Number `json:"multiplier"`
    Buy        json.Number `json:"buy"`
    Sell       json.Number `json:"sell"`
}

type APIError = vendorhttp.StatusError
```

`CurrentRates` sends `GET /v2/credo/current-rates?method=<method>&date=<at>`, with
`at` in UTC in the format `2006-01-02T15:04:05.000Z`, and the header `x-api-key`. It
decodes `{"success": bool, "rates": [...]}` and returns `rates`. A false `success`
returns an error that wraps `ErrNotSucceeded`. Any status other than 200 returns an
`*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the answer has
243 bytes.

**`adapter/credo`.** `Bank` returns
`{Code: "credo", Name: "Credo Bank", Kind: domain.KindBank}`. `CommercialRates` sends one
call with `MethodCommercial` and the instant of the clock of the adapter. It keeps each
item whose `ToCcy` is `GEL`, and converts the item of each requested code:

| Field of the item | Domain value                                                     |
| ----------------- | ---------------------------------------------------------------- |
| `FromCcy`         | Matched against the requested code in upper case.                |
| `Multiplier`      | The count of units. `CREDRATE-DD-002`.                           |
| `Buy`, `Sell`     | `decimal.NewFromString`, divided by the count, trailing zeros dropped. |

A requested code with no item, or with a price that is absent, empty, or zero, goes to
`Missing`. A price that is not a number or is below zero fails the bank. Two items of
one code against the lari fail the bank. `ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Credo when
`cfg.Credo.Enabled` is true, with `credo.WithAPIKey(cfg.Credo.APIKey)`. Else it logs at
info level `Credo Bank is off` with the attribute `variable=GEOMCP_RATES_CREDO_ENABLED`.

### 4.2 Domain model

No change. `TBCRATE` 4.2 holds the model.

### 4.3 Declarations

**Vendor endpoint.**

| Client       | Method and path                 | Key          | Official              | Realizes          |
| ------------ | ------------------------------- | ------------ | --------------------- | ----------------- |
| `pkg/credo`  | `GET /v2/credo/current-rates`   | `x-api-key`  | No, an unofficial API | `CREDRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to two of the 3 items:

```
GET https://n1.noxtton.com/v2/credo/current-rates?method=FxRatesCommercial&date=2026-10-25T08:42:36.000Z
x-api-key: <the key>
200 application/json; charset=utf-8 (243 bytes)
{"success":true,"rates":[
 {"fromCcy":"EUR","toCcy":"GEL","multiplier":1,"buy":2.875,"sell":2.95},
 {"fromCcy":"USD","toCcy":"GEL","multiplier":1,"buy":2.593,"sell":2.607}]}

No key, or a wrong key: 401 Unauthorized
```

**Configuration.**

| Variable                       | Type     | Default                   | Secret | Realizes           |
| ------------------------------ | -------- | ------------------------- | ------ | ------------------ |
| `GEOMCP_RATES_CREDO_ENABLED`   | bool     | `true`                    | No     | `CREDRATE-FR-005`  |
| `GEOMCP_RATES_CREDO_API_KEY`   | string   | none, required while on   | Yes    | `CREDRATE-FR-005`  |
| `GEOMCP_RATES_CREDO_BASE_URL`  | URL      | `https://n1.noxtton.com`  | No     | `CREDRATE-FR-001`  |
| `GEOMCP_RATES_CREDO_TIMEOUT`   | duration | `10s`                     | No     | `CREDRATE-FR-001`  |
| `GEOMCP_RATES_CREDO_RATE`      | float    | `1`                       | No     | `CREDRATE-NFR-001` |
| `GEOMCP_RATES_CREDO_BURST`     | int      | `1`                       | No     | `CREDRATE-NFR-001` |

```go
// CredoConfig is the configuration of the calls to Credo Bank. Enabled turns Credo off.
// The key does not: while Credo is on, the key is required.
type CredoConfig struct {
    vendors.Config

    APIKey string `env:"API_KEY" validate:"required_if=Enabled true"`
}
```

`rates.Config` gains `Credo CredoConfig` with `envPrefix:"CREDO_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). `config.Load` refuses an empty key
while `Enabled` is true, and names `GEOMCP_RATES_CREDO_ENABLED` (`PLATFORM-DD-012`). The
cache is a `CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256
entries.

## 5. Design decisions

### `CREDRATE-DD-001`

**Realizes:** `CREDRATE-FR-001`
**Decision:** The adapter asks for `FxRatesCommercial` only.
**Rationale:** The page labels it the exchange rate of Credo Bank. `LIBRATE-DD-002` gives
the rate of the bank, not of an online channel or a card.
**Alternatives:** `FxRatesForTransfer`. It is the rate of the internet bank.

### `CREDRATE-DD-002`

**Realizes:** `CREDRATE-FR-002`, `CREDRATE-FR-003`, `CREDRATE-INV-001`
**Decision:** The adapter drops each item whose `ToCcy` is not `GEL`, before it reads a
code. It reads `Multiplier` as an integer. The count must be a power of 10 from 1 to
1 000 000, and each price is divided with `Shift(-k)`, as `LIBRATE-DD-003` gives. Any
other multiplier fails the bank.
**Rationale:** The answer names both currencies and the unit of each pair, so the adapter
reads them, not a fixed lari and a fixed unit of 1. The shift is exact.
**Alternatives:** Ignore `ToCcy` and `Multiplier`, since each is `GEL` and 1 today. A change
of the source would then give a wrong rate with no error.

### `CREDRATE-DD-003`

**Realizes:** `CREDRATE-FR-006`
**Decision:** On a 401, the adapter logs at error level `Credo Bank refused the key` with
`variable=GEOMCP_RATES_CREDO_API_KEY`, then returns the error with
`kernel.ErrSourceUnavailable`.
**Rationale:** The same rule as `TTCTRANS-DD-012`. The model reports an outage. The
operator reads the cause. The rate limit bounds the count of records.
**Alternatives:** `TTCTRANS-DD-012` lists them.

## 6. Scenarios

`pkg/credo/testdata/` holds `current-rates.json`, the real answer of 2026-10-25, and
`not-succeeded.json`, `{"success":false,"rates":null}`.

### `CREDRATE-SC-001`

**Verifies:** `CREDRATE-FR-001`, `CREDRATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request, and a client with the key `k1`.
**When** `CurrentRates` runs with `FxRatesCommercial` and `2026-10-25T08:42:36Z`.
**Then** the method is `GET`, the path is `/v2/credo/current-rates`, `method` is
`FxRatesCommercial`, `date` is `2026-10-25T08:42:36.000Z`, `x-api-key` is `k1`, `Accept`
is `application/json`, and the User-Agent starts with `georgia-mcp/`.

### `CREDRATE-SC-002`

**Verifies:** `CREDRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `current-rates.json`.
**When** `CurrentRates` runs.
**Then** the result holds 3 items. The item of `USD` has `ToCcy` `GEL`, `Multiplier` `1`,
`Buy` `2.593`, and `Sell` `2.607`.

### `CREDRATE-SC-003`

**Verifies:** `CREDRATE-FR-004`, `CREDRATE-FR-006`
**Layer:** unit

| Vendor answer                    | Client result                                  |
| -------------------------------- | ---------------------------------------------- |
| `not-succeeded.json`             | `errors.Is(err, ErrNotSucceeded)`              |
| 401 with the body `Unauthorized` | `*APIError` with 401 and `Unauthorized`        |
| 500 with a body of 2 KiB         | `*APIError` with 500 and the first 1 KiB       |

### `CREDRATE-SC-004`

**Verifies:** `CREDRATE-FR-001`
**Layer:** live

**Given** the real method, and a key in `GEOMCP_RATES_CREDO_API_KEY`. The test skips
with no key.
**When** `CurrentRates` runs with `FxRatesCommercial` and the current instant.
**Then** the item of `USD` against `GEL` has a buy price and a sell price above zero.

### `CREDRATE-SC-005`

**Verifies:** `CREDRATE-FR-001`, `CREDRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `current-rates.json`.
**When** the adapter asks for `EUR` and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.875`        | `2.95`          |
| `USD`    | `2.593`        | `2.607`         |

### `CREDRATE-SC-006`

**Verifies:** `CREDRATE-FR-002`, `CREDRATE-INV-001`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `current-rates.json`, and
`{"fromCcy":"JPY","toCcy":"GEL","multiplier":100,"buy":1.7,"sell":2.1}`.
**When** the adapter asks for `JPY`.
**Then** the buy rate formats as `0.017`, and the sell rate as `0.021`.

### `CREDRATE-SC-007`

**Verifies:** `CREDRATE-FR-003`
**Layer:** unit

**Given** a vendor that serves the item of `USD` of `current-rates.json`, and the item of
the table.
**When** the adapter asks for `CHF`, `GBP`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF GBP]`.

| Second item                                                           |
| --------------------------------------------------------------------- |
| `{"fromCcy":"CHF","toCcy":"GEL","multiplier":1,"buy":0,"sell":3.2}`   |
| `{"fromCcy":"CHF","toCcy":"GEL","multiplier":1,"buy":3.1}`            |
| `{"fromCcy":"CHF","toCcy":"USD","multiplier":1,"buy":0.8,"sell":0.9}` |

### `CREDRATE-SC-008`

**Verifies:** `CREDRATE-FR-001`, `CREDRATE-FR-002`, `CREDRATE-FR-004`
**Layer:** unit

| Vendor answer                                 | Adapter error                        |
| --------------------------------------------- | ------------------------------------ |
| `not-succeeded.json`                          | Wraps `kernel.ErrSourceUnavailable`  |
| 500                                           | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"buy":"n/a"`          | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"buy":-1`             | Wraps `kernel.ErrSourceUnavailable`  |
| The item of `USD` with `"multiplier":3`       | Wraps `kernel.ErrSourceUnavailable`  |
| Two items of `USD` against `GEL`              | Wraps `kernel.ErrSourceUnavailable`  |

### `CREDRATE-SC-009`

**Verifies:** `CREDRATE-FR-006`
**Layer:** unit

**Given** a vendor that answers 401, and an adapter with a captured logger.
**When** the adapter asks for `USD`.
**Then** the error wraps `kernel.ErrSourceUnavailable`. The log holds one error record
`Credo Bank refused the key` with `variable=GEOMCP_RATES_CREDO_API_KEY`. No record
holds the key.

### `CREDRATE-SC-010`

**Verifies:** `CREDRATE-FR-005`, `CREDRATE-NFR-001`
**Layer:** unit

| Environment                                | Result of `config.Load`                                |
| ------------------------------------------ | ------------------------------------------------------ |
| `GEOMCP_RATES_CREDO_API_KEY=k1`            | `Credo.Enabled` true, `Credo.BaseURL` `https://n1.noxtton.com`, `Credo.HTTP.Rate` 1, `Credo.HTTP.Burst` 1 |
| No variable of Credo                       | An error that names `GEOMCP_RATES_CREDO_API_KEY` and `GEOMCP_RATES_CREDO_ENABLED` |
| `GEOMCP_RATES_CREDO_ENABLED=false`, no key | `Credo.Enabled` false, no error                        |

### `CREDRATE-SC-011`

**Verifies:** `CREDRATE-FR-001`
**Layer:** integration

**Given** Credo on with a key, every other bank off, the vendor serves
`current-rates.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the vendor receives `date` as the instant of the clock, the bank `credo` has
`source` `Credo Bank`, `as_of` is the instant of the clock, `buy` is `2.593`, `sell` is
`2.607`, and `missing` is `["CHF"]`.

### `CREDRATE-SC-012`

**Verifies:** `CREDRATE-FR-005`
**Layer:** integration

**Given** Credo off and `bog` on.
**When** a client calls the tool with the bank `credo`.
**Then** the tool result reads `banks: "credo" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_CREDO_ENABLED`.

## 7. Build plan

| #   | Step                                                                     | Realizes                                     | Done |
| --- | ------------------------------------------------------------------------ | -------------------------------------------- | ---- |
| 1   | `pkg/credo`, `testdata/`, README, example, live test.                    | `CREDRATE-FR-001`, `CREDRATE-FR-004`         | [ ]  |
| 2   | `adapter/credo`.                                                         | `CREDRATE-FR-001` to `CREDRATE-FR-004`, `CREDRATE-FR-006`, `CREDRATE-INV-001` | [ ]  |
| 3   | `config.go`: `CredoConfig`, `Defaults`. `configs/.env.example`.          | `CREDRATE-FR-005`, `CREDRATE-NFR-001`        | [ ]  |
| 4   | `module.go`: the source of Credo with its key, and the info record. The glossary. | `CREDRATE-FR-001`, `CREDRATE-FR-005` | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
