---
id: MJCRATE
title: The commercial exchange rates of MJC
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/mjcrate/requirements.md
---

# Specification: The commercial exchange rates of MJC

## 1. Summary

This feature adds the vendor client `pkg/mjc`, a client of the rate page of MJC, and MJC
as a source of `rates_get_commercial_rates`. The tool, the use case, the domain, and the
cache of `TBCRATE` do not change. One call reads every currency, and the adapter divides
each price by its published unit. The design is the design of `SCAPRATE`, with a fourth
column that the client skips.

## 2. Coverage

| Requirement       | Where this specification realizes it                      |
| ----------------- | --------------------------------------------------------- |
| `MJCRATE-FR-001`  | 4.1 `adapter/mjc`. 4.3 vendor endpoint. `MJCRATE-DD-001`  |
| `MJCRATE-FR-002`  | `MJCRATE-DD-002`                                          |
| `MJCRATE-FR-003`  | 4.1 `adapter/mjc`. `LIBRATE-DD-004`                       |
| `MJCRATE-FR-004`  | 4.1 `adapter/mjc`. `RICORATE-DD-004`                      |
| `MJCRATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`     |
| `MJCRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                       |
| `MJCRATE-INV-001` | `MJCRATE-DD-002`                                          |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/mjc` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/mjc`. |
| `CLT-002`, `CLT-004`   | The standard library, `pkg/vendorhttp`, and `golang.org/x/net/html`. `ADR-0003`. |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.    |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the source is a web page for people. |
| `DOM-005`, `DOM-011`   | `adapter/mjc` converts the rows and wraps `kernel.ErrSourceUnavailable`.     |
| `DOM-009`              | MJC is an adapter of `CommercialRateSource`.                                 |
| `MCP-002`              | No new tool. The description lists `mjc` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | MJC sits behind a `CommercialRateCache`, as TBC does.                        |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_MJC_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `MJCRATE` 5.                              |
| Vendor client  | `pkg/mjc/`                            | `Client`, `New`, `Option`, `RateRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateTable`, `ErrMalformedTable`. |
| Driven adapter | `R/adapter/mjc/`                      | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.MJC`, the entry of MJC in `otherEntries`, `mjcSwitchVariable`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Linter         | `.golangci.yaml` (change)             | `pkg/mjc` joins `web-page-clients`, and leaves `clients`.    |

**`pkg/mjc`.** The shape of `pkg/scapp`, with one method.

```go
const DefaultBaseURL = "https://mjc.ge"

var (
    ErrUnexpectedStatus = errors.New("mjc: unexpected status")
    ErrNoRateTable      = errors.New("mjc: no rate table")
    ErrMalformedTable   = errors.New("mjc: malformed rate table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one row of the rate table of the page, as the page shows it.
type RateRow struct {
    Unit string // "1 USD"
    Buy  string // "2.59500"
    Sell string // "2.59800"
}

type APIError = vendorhttp.StatusError
```

`Rates` sends `GET /rates` with `Accept: text/html`. Any status other than 200 returns
an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the page has
27 KB. `MJCRATE-DD-001` gives the parse.

**`adapter/mjc`.** `Bank` returns
`{Code: "mjc", Name: "MJC", Kind: domain.KindMicrofinance}`. `CommercialRates` sends one
call, and converts the row of each requested code:

| Row of the page | Domain value                                                       |
| --------------- | ------------------------------------------------------------------ |
| `Unit`          | The count and the code. `MJCRATE-DD-002`.                          |
| `Buy`, `Sell`   | `decimal.NewFromString`, divided by the count, trailing zeros dropped. |

A requested code with no row, or with a price that is empty or zero, goes to `Missing`.
A price that is not a number or is below zero fails the bank, as `LIBRATE-DD-004`
gives. Two rows of one code fail the bank. `ReadAt` is the instant of the answer.

**`module.go` (change).** `otherEntries` gains the entry of MJC after Scapp, with the
name `MJC` and the variable `GEOMCP_RATES_MJC_ENABLED`. The entry logs `MJC is off` at
info level when the switch is false, as each entry does.

### 4.2 Domain model

No change. `RICORATE` 4.2 holds `KindMicrofinance`.

### 4.3 Declarations

**Vendor endpoint.**

| Client     | Method and path | Key  | Official                  | Realizes         |
| ---------- | --------------- | ---- | ------------------------- | ---------------- |
| `pkg/mjc`  | `GET /rates`    | None | No, a web page for people | `MJCRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to the table and one row:

```
GET https://mjc.ge/rates
200 text/html; charset=UTF-8 (27459 bytes)
Cache-Control: no-cache, private
<div class="section-heading"><h3>ვალუტის კურსები</h3></div>
<div class="table-responsive mb-2" style="background-color: #ffffff">
  <table class="table align-middle">
    <thead><tr>
      <th scope="col">ვალუტა</th>
      <th scope="col">ყიდვა</th>
      <th scope="col">გაყიდვა</th>
      <th scope="col">ეროვნული</th>
    </tr></thead>
    <tbody>
    <tr>
      <td><pre>1 USD</pre></td>
      <td><pre>2.59500</pre></td>
      <td><pre>2.59800</pre></td>
      <td><pre>2.5971</pre></td>
    </tr>
```

**Configuration.**

| Variable                     | Type     | Default          | Secret | Realizes          |
| ---------------------------- | -------- | ---------------- | ------ | ----------------- |
| `GEOMCP_RATES_MJC_ENABLED`   | bool     | `true`           | No     | `MJCRATE-FR-005`  |
| `GEOMCP_RATES_MJC_BASE_URL`  | URL      | `https://mjc.ge` | No     | `MJCRATE-FR-001`  |
| `GEOMCP_RATES_MJC_TIMEOUT`   | duration | `10s`            | No     | `MJCRATE-FR-001`  |
| `GEOMCP_RATES_MJC_RATE`      | float    | `1`              | No     | `MJCRATE-NFR-001` |
| `GEOMCP_RATES_MJC_BURST`     | int      | `1`              | No     | `MJCRATE-NFR-001` |

`rates.Config` gains `MJC vendors.Config` with `envPrefix:"MJC_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `MJCRATE-DD-001`

**Realizes:** `MJCRATE-FR-001`, `CLT-003`
**Decision:** The client reads the first `table` of the page. Its `thead` gives the
titles of the columns: the text of each `th`, with the spaces trimmed. The title `ვალუტა`
names the unit column, `ყიდვა` the buy column, and `გაყიდვა` the sell column. The client
skips every other column, `ეროვნული` among them. Each `tr` outside `thead` gives one
row. The text of a cell is the text of its children, with each run of white space made
one space and the ends trimmed.
**Rationale:** A match by title, not by a fixed position, finds a swap of the columns.
The official column must never reach a rate (`MJCRATE-FR-001`). The page holds one table.
**Alternatives:** A fixed position for each column. A new column before the prices
would then give a wrong price with no error.

The client returns `ErrNoRateTable` when the page has no `table` or no row, and
`ErrMalformedTable` when the titles lack `ვალუტა`, `ყიდვა`, or `გაყიდვა`, or a row has
fewer cells than the titles.

### `MJCRATE-DD-002`

**Realizes:** `MJCRATE-FR-002`, `MJCRATE-INV-001`
**Decision:** The adapter reads `Unit` as `SCAPRATE-DD-002` gives: the count, a space,
and the code. The count is a power of 10 from 1 to 1 000 000, and each price is divided
with `Shift(-k)`. A unit with no space, a code that fails, or another count fails the bank.
**Rationale:** The page writes the unit and the code in one cell, as Scapp does. The
shift is exact.
**Alternatives:** A fixed unit for each code. MJC can change a unit, and the rate would
then be wrong by a factor of 10.

## 6. Scenarios

`pkg/mjc/testdata/` holds `rates.html`, the real page, recorded on 2026-10-25.

### `MJCRATE-SC-001`

**Verifies:** `MJCRATE-FR-001`, `MJCRATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/rates`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp/`, and the request carries no key header.

### `MJCRATE-SC-002`

**Verifies:** `MJCRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `rates.html`.
**When** `Rates` runs.
**Then** the result holds 17 rows. Three rows are:

| Unit       | Buy       | Sell      |
| ---------- | --------- | --------- |
| `1 USD`    | `2.59500` | `2.59800` |
| `1000 AMD` | `7.00000` | `7.20000` |
| `1 SAR`    | `0.66900` | `0.68500` |

### `MJCRATE-SC-003`

**Verifies:** `MJCRATE-FR-001`
**Layer:** unit

| Vendor answer                                     | Client result                             |
| ------------------------------------------------- | ----------------------------------------- |
| A page with no `table`                            | `ErrNoRateTable`                          |
| `rates.html` with the title `გაყიდვა` renamed      | `ErrMalformedTable`                       |
| `rates.html` with a row of two cells              | `ErrMalformedTable`                       |
| `rates.html` with `ეროვნული` moved to the second column | 17 rows, and the row of `USD` as in `MJCRATE-SC-002` |
| 503 with a body of 2 KiB                          | `*APIError` with 503 and the first 1 KiB  |

### `MJCRATE-SC-004`

**Verifies:** `MJCRATE-FR-001`
**Layer:** live

**Given** the real page of MJC.
**When** `Rates` runs.
**Then** a row with the unit `1 USD` has a buy price and a sell price above zero.

### `MJCRATE-SC-005`

**Verifies:** `MJCRATE-FR-001`, `MJCRATE-FR-003`, `MJCRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `rates.html`.
**When** the adapter asks for `EUR`, `SAR`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.907`        | `2.912`         |
| `SAR`    | `0.669`        | `0.685`         |
| `USD`    | `2.595`        | `2.598`         |

### `MJCRATE-SC-006`

**Verifies:** `MJCRATE-FR-002`, `MJCRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `rates.html`.
**When** the adapter asks for `AMD` and `PLN`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `AMD`    | `0.007`        | `0.0072`        |
| `PLN`    | `0.634`        | `0.654`         |

### `MJCRATE-SC-007`

**Verifies:** `MJCRATE-FR-003`
**Layer:** unit

**Given** a page with the row of `USD` and the row of `CHF` of the table.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| Prices of `CHF`         |
| ----------------------- |
| `0.00000`, `3.14300`    |
| empty, `3.14300`        |

### `MJCRATE-SC-008`

**Verifies:** `MJCRATE-FR-001`, `MJCRATE-FR-002`
**Layer:** unit

| Row of `USD`                    | Adapter error                        |
| ------------------------------- | ------------------------------------ |
| `Unit` is `3 USD`               | Wraps `kernel.ErrSourceUnavailable`  |
| `Unit` is `USD`                 | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `N/A`                  | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `-2.59500`             | Wraps `kernel.ErrSourceUnavailable`  |
| A second row `1 USD`            | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503          | Wraps `kernel.ErrSourceUnavailable`  |

### `MJCRATE-SC-009`

**Verifies:** `MJCRATE-FR-005`, `MJCRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of MJC.
**When** the configuration loads.
**Then** `MJC.Enabled` is true, `MJC.BaseURL` is `https://mjc.ge`, `MJC.HTTP.Rate` is
1, and `MJC.HTTP.Burst` is 1.

### `MJCRATE-SC-010`

**Verifies:** `MJCRATE-FR-001`, `MJCRATE-FR-004`
**Layer:** integration

**Given** MJC on, every other bank off, the vendor serves `rates.html`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `JPY`.
**Then** the bank `mjc` has `source` `MJC`, `as_of` is the instant of the clock, `buy`
is `2.595`, `sell` is `2.598`, and `missing` is `["JPY"]`. The description of the tool
lists `mjc (MJC, a microfinance organization)`.

### `MJCRATE-SC-011`

**Verifies:** `MJCRATE-FR-005`
**Layer:** integration

**Given** MJC off and `bog` on.
**When** a client calls the tool with the bank `mjc`.
**Then** the tool result reads `banks: "mjc" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_MJC_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                 | Done |
| --- | -------------------------------------------------------------------------- | ---------------------------------------- | ---- |
| 1   | `pkg/mjc`, `testdata/rates.html`, README, example, live test. `.golangci.yaml`. | `MJCRATE-FR-001`                    | [ ]  |
| 2   | `adapter/mjc`.                                                             | `MJCRATE-FR-001` to `MJCRATE-FR-004`, `MJCRATE-INV-001` | [ ]  |
| 3   | `config.go`: `Config.MJC`, `Defaults`. `configs/.env.example`.             | `MJCRATE-FR-005`, `MJCRATE-NFR-001`      | [ ]  |
| 4   | `module.go`: the entry of MJC. The glossary.                               | `MJCRATE-FR-001`, `MJCRATE-FR-005`       | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
