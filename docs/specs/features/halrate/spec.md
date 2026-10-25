---
id: HALRATE
title: The commercial exchange rates of Halyk Bank
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/halrate/requirements.md
---

# Specification: The commercial exchange rates of Halyk Bank

## 1. Summary

This feature adds the vendor client `pkg/halyk`, a client of the rate page of Halyk Bank
Georgia, and Halyk as a source of `rates_get_commercial_rates`. The tool, the use case,
the domain, and the cache of `TBCRATE` do not change. One call reads every currency.

## 2. Coverage

| Requirement       | Where this specification realizes it                       |
| ----------------- | ---------------------------------------------------------- |
| `HALRATE-FR-001`  | 4.1 `adapter/halyk`. 4.3 vendor endpoint. `HALRATE-DD-001` |
| `HALRATE-FR-002`  | 4.1 `adapter/halyk`. `LIBRATE-DD-004`                      |
| `HALRATE-FR-003`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`      |
| `HALRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                        |
| `HALRATE-INV-001` | `TERARATE-DD-004`                                          |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/halyk` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/halyk`. |
| `CLT-002`, `CLT-004`   | The standard library, `pkg/vendorhttp`, and `golang.org/x/net/html`. `ADR-0003`. |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.    |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the source is a web page for people. |
| `DOM-005`, `DOM-011`   | `adapter/halyk` converts the rows and wraps `kernel.ErrSourceUnavailable`.   |
| `DOM-009`              | Halyk is an adapter of `CommercialRateSource`.                               |
| `MCP-002`              | No new tool. The description lists `halyk` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Halyk sits behind a `CommercialRateCache`, as TBC does.                      |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_HALYK_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `HALRATE` 5.                              |
| Vendor client  | `pkg/halyk/`                          | `Client`, `New`, `Option`, `RateRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateTable`, `ErrMalformedTable`. |
| Driven adapter | `R/adapter/halyk/`                    | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Halyk`, `halykSource`.                               |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Linter         | `.golangci.yaml` (change)             | `pkg/halyk` joins `web-page-clients`, and leaves `clients`.  |

**`pkg/halyk`.** The shape of `pkg/rico`, with one method.

```go
const DefaultBaseURL = "https://halykbank.ge"

var (
    ErrUnexpectedStatus = errors.New("halyk: unexpected status")
    ErrNoRateTable      = errors.New("halyk: no rate table")
    ErrMalformedTable   = errors.New("halyk: malformed rate table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one row of the rate table of the page, as the page shows it.
type RateRow struct {
    Currency string // "USD"
    Buy      string // "2.5870"
    Sell     string // "2.6120"
}

type APIError = vendorhttp.StatusError
```

`Rates` sends `GET /ka/individuals/currency` with `Accept: text/html`. Any status other
than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is
1 MiB: the page has 95 KB. `HALRATE-DD-001` gives the parse.

**`adapter/halyk`.** `Bank` returns
`{Code: "halyk", Name: "Halyk Bank", Kind: domain.KindBank}`. `CommercialRates` sends one
call, and converts the row of each requested code:

| Row of the page | Domain value                                                   |
| --------------- | -------------------------------------------------------------- |
| `Currency`      | `domain.ParseCurrencyCode`. A code that fails fails the bank.  |
| `Buy`, `Sell`   | Parsed as `TERARATE-DD-004` gives.                             |

A requested code with no row, or with a price that is empty or zero, goes to `Missing`.
A price that is not a number or is below zero fails the bank, as `LIBRATE-DD-004`
gives. Two rows of one code fail the bank. `ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Halyk when
`cfg.Halyk.Enabled` is true. Else it logs at info level `Halyk Bank is off` with the
attribute `variable=GEOMCP_RATES_HALYK_ENABLED`.

### 4.2 Domain model

No change. `TBCRATE` 4.2 holds the model.

### 4.3 Declarations

**Vendor endpoint.**

| Client       | Method and path               | Key  | Official                  | Realizes         |
| ------------ | ----------------------------- | ---- | ------------------------- | ---------------- |
| `pkg/halyk`  | `GET /ka/individuals/currency` | None | No, a web page for people | `HALRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to the table and two rows:

```
GET https://halykbank.ge/ka/individuals/currency
200 text/html; charset=utf-8 (95555 bytes)
Cache-Control: private, max-age=356
<div class="tab-pane fade currencies__tab-pane show active" id="currentexchangerate" role="tabpanel" ...>
  <h6 class="currencies__date">თარიღი: 12:19, 25.10.2026</h6>
  ...
  <table class="table table-bordered table--greenbg">
    <thead class="thead-dark"><tr><th>ვალუტა</th><th>ყიდვა</th><th>გაყიდვა</th></tr></thead>
    <tbody>
      <tr><td><img src="~/m/u/flags/USD.png" alt="USD"> USD </td><td>2.5870</td><td>2.6120</td></tr>
      <tr><td><img src="~/m/u/flags/KZT.png" alt="KZT"> KZT </td><td>0.0054</td><td>0.0059</td></tr>
    </tbody>
  </table>
...
<script>window.currencies= [{"title":"GEL","buy":1.0,"sell":1.0,"text":"GEL","id":0},{"title":"USD","buy":2.587,"sell":2.612,"text":"USD","id":1},...];</script>
```

**Configuration.**

| Variable                       | Type     | Default                | Secret | Realizes          |
| ------------------------------ | -------- | ---------------------- | ------ | ----------------- |
| `GEOMCP_RATES_HALYK_ENABLED`   | bool     | `true`                 | No     | `HALRATE-FR-003`  |
| `GEOMCP_RATES_HALYK_BASE_URL`  | URL      | `https://halykbank.ge` | No     | `HALRATE-FR-001`  |
| `GEOMCP_RATES_HALYK_TIMEOUT`   | duration | `10s`                  | No     | `HALRATE-FR-001`  |
| `GEOMCP_RATES_HALYK_RATE`      | float    | `1`                    | No     | `HALRATE-NFR-001` |
| `GEOMCP_RATES_HALYK_BURST`     | int      | `1`                    | No     | `HALRATE-NFR-001` |

`rates.Config` gains `Halyk vendors.Config` with `envPrefix:"HALYK_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `HALRATE-DD-001`

**Realizes:** `HALRATE-FR-001`, `CLT-003`
**Decision:** The client reads the first `table` inside the element with the id
`currentexchangerate`. Its `thead` gives the titles of the columns: the trimmed text of
each `th`. The title `ვალუტა` names the currency column, `ყიდვა` the buy column, and
`გაყიდვა` the sell column. Each `tr` of `tbody` gives one row. Each cell is its trimmed
text: the `img` of the currency cell adds no text.
**Rationale:** The id names the tab of the current rates, so a table of the history on
the second tab cannot match. A match by title finds a swap of the columns. The table
keeps every published digit.
**Alternatives:** The JSON `window.currencies` of the converter. It cuts the trailing
zeros, adds `GEL`, and sits in a script, so the client would cut a JavaScript
assignment. The home page holds it too, and is 1 KB smaller.

The client returns `ErrNoRateTable` when the page has no such table or no row, and
`ErrMalformedTable` when a title is absent or a row lacks a cell.

## 6. Scenarios

`pkg/halyk/testdata/` holds `currency-ka.html`, the real page, recorded on 2026-10-25.

### `HALRATE-SC-001`

**Verifies:** `HALRATE-FR-001`, `HALRATE-FR-003`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/ka/individuals/currency`, `Accept` is `text/html`, the User-Agent
starts with `georgia-mcp/`, and the request carries no key header.

### `HALRATE-SC-002`

**Verifies:** `HALRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `currency-ka.html`.
**When** `Rates` runs.
**Then** the result holds 7 rows, and no row of `GEL`. Two rows are:

| Currency | Buy      | Sell     |
| -------- | -------- | -------- |
| `USD`    | `2.5870` | `2.6120` |
| `KZT`    | `0.0054` | `0.0059` |

### `HALRATE-SC-003`

**Verifies:** `HALRATE-FR-001`
**Layer:** unit

| Vendor answer                                         | Client result                             |
| ----------------------------------------------------- | ----------------------------------------- |
| A page with no element `currentexchangerate`          | `ErrNoRateTable`                          |
| `currency-ka.html` with the title `გაყიდვა` renamed    | `ErrMalformedTable`                       |
| `currency-ka.html` with a row of two cells            | `ErrMalformedTable`                       |
| 503 with a body of 2 KiB                              | `*APIError` with 503 and the first 1 KiB  |

### `HALRATE-SC-004`

**Verifies:** `HALRATE-FR-001`
**Layer:** live

**Given** the real page of Halyk.
**When** `Rates` runs.
**Then** the row of `USD` has a buy price and a sell price above zero.

### `HALRATE-SC-005`

**Verifies:** `HALRATE-FR-001`, `HALRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `currency-ka.html`.
**When** the adapter asks for `EUR`, `KZT`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.88`         | `2.94`          |
| `KZT`    | `0.0054`       | `0.0059`        |
| `USD`    | `2.587`        | `2.612`         |

### `HALRATE-SC-006`

**Verifies:** `HALRATE-FR-002`
**Layer:** unit

**Given** a page with the row of `USD` and the row of `CHF` of the table.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| Prices of `CHF`      |
| -------------------- |
| `0.0000`, `3.2500`   |
| empty, `3.2500`      |

### `HALRATE-SC-007`

**Verifies:** `HALRATE-FR-001`
**Layer:** unit

| Row of `USD`                    | Adapter error                        |
| ------------------------------- | ------------------------------------ |
| `Currency` is `US$`             | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `n/a`                  | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `-2.5870`              | Wraps `kernel.ErrSourceUnavailable`  |
| A second row of `USD`           | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503          | Wraps `kernel.ErrSourceUnavailable`  |

### `HALRATE-SC-008`

**Verifies:** `HALRATE-FR-003`, `HALRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Halyk.
**When** the configuration loads.
**Then** `Halyk.Enabled` is true, `Halyk.BaseURL` is `https://halykbank.ge`,
`Halyk.HTTP.Rate` is 1, and `Halyk.HTTP.Burst` is 1.

### `HALRATE-SC-009`

**Verifies:** `HALRATE-FR-001`
**Layer:** integration

**Given** Halyk on, every other bank off, the vendor serves `currency-ka.html`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the bank `halyk` has `source` `Halyk Bank`, `as_of` is the instant of the clock,
`buy` is `2.587`, `sell` is `2.612`, and `missing` is `["CHF"]`.

### `HALRATE-SC-010`

**Verifies:** `HALRATE-FR-003`
**Layer:** integration

**Given** Halyk off and `bog` on.
**When** a client calls the tool with the bank `halyk`.
**Then** the tool result reads `banks: "halyk" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_HALYK_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                  | Done |
| --- | -------------------------------------------------------------------------- | ----------------------------------------- | ---- |
| 1   | `pkg/halyk`, `testdata/currency-ka.html`, README, example, live test. `.golangci.yaml`. | `HALRATE-FR-001`             | [ ]  |
| 2   | `adapter/halyk`.                                                           | `HALRATE-FR-001`, `HALRATE-FR-002`, `HALRATE-INV-001` | [ ]  |
| 3   | `config.go`: `Config.Halyk`, `Defaults`. `configs/.env.example`.           | `HALRATE-FR-003`, `HALRATE-NFR-001`       | [ ]  |
| 4   | `module.go`: the source of Halyk, and the info record. The glossary.       | `HALRATE-FR-001`, `HALRATE-FR-003`        | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
