---
id: INTLRATE
title: The commercial exchange rates of Inteli Express
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/intlrate/requirements.md
---

# Specification: The commercial exchange rates of Inteli Express

## 1. Summary

This feature adds the vendor client `pkg/inteli`, a client of the home page of Inteli
Express, and Inteli Express as a source of `rates_get_commercial_rates`. The tool, the
use case, the domain, and the cache of `TBCRATE` do not change. One call reads every
currency. The table names no column, so the client reads the cells by position, and the
adapter refuses a row whose buy price is above its sell price.

## 2. Coverage

| Requirement        | Where this specification realizes it                        |
| ------------------ | ----------------------------------------------------------- |
| `INTLRATE-FR-001`  | 4.1 `adapter/inteli`. 4.3 vendor endpoint. `INTLRATE-DD-001`, `INTLRATE-DD-002` |
| `INTLRATE-FR-002`  | 4.1 `adapter/inteli`. `LIBRATE-DD-004`                      |
| `INTLRATE-FR-003`  | `INTLRATE-DD-003`                                           |
| `INTLRATE-FR-004`  | 4.1 `adapter/inteli`. `RICORATE-DD-004`                     |
| `INTLRATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`       |
| `INTLRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                         |
| `INTLRATE-INV-001` | 4.1 `adapter/inteli`                                        |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/inteli` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/inteli`. |
| `CLT-002`, `CLT-004`   | The standard library, `pkg/vendorhttp`, and `golang.org/x/net/html`. `ADR-0003`. |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.    |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the source is a web page for people. |
| `DOM-005`, `DOM-011`   | `adapter/inteli` converts the rows and wraps `kernel.ErrSourceUnavailable`.  |
| `DOM-009`              | Inteli Express is an adapter of `CommercialRateSource`.                      |
| `MCP-002`              | No new tool. The description lists `inteli` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Inteli Express sits behind a `CommercialRateCache`, as TBC does.             |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_INTELI_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `INTLRATE` 5.                             |
| Vendor client  | `pkg/inteli/`                         | `Client`, `New`, `Option`, `RateRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateTable`, `ErrMalformedTable`. |
| Driven adapter | `R/adapter/inteli/`                   | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Inteli`, the entry of Inteli Express in `otherEntries`, `inteliSwitchVariable`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Linter         | `.golangci.yaml` (change)             | `pkg/inteli` joins `web-page-clients`, and leaves `clients`. |

**`pkg/inteli`.** The shape of `pkg/scapp`, with one method.

```go
const DefaultBaseURL = "https://inteliexpress.com"

var (
    ErrUnexpectedStatus = errors.New("inteli: unexpected status")
    ErrNoRateTable      = errors.New("inteli: no rate table")
    ErrMalformedTable   = errors.New("inteli: malformed rate table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one row of the rate table of the home page, as the page shows it.
type RateRow struct {
    Base     string // "GEL", the currency of the prices
    Currency string // "USD", the currency that the row prices
    Buy      string // "2.595", the first price
    Sell     string // "2.598", the second price
}

type APIError = vendorhttp.StatusError
```

`Rates` sends `GET /ka/main-page-geo/` with `Accept: text/html`. Any status other than
200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB:
the page has 115 KB. `INTLRATE-DD-001` gives the parse.

**`adapter/inteli`.** `Bank` returns
`{Code: "inteli", Name: "Inteli Express", Kind: domain.KindMicrofinance}`.
`CommercialRates` sends one call. It keeps each row whose `Base` is `GEL` in any case
(`INTLRATE-DD-002`), and converts the row of each requested code:

| Row of the page | Domain value                                                       |
| --------------- | ------------------------------------------------------------------ |
| `Currency`      | `domain.ParseCurrencyCode`. A code that fails fails the bank.       |
| `Buy`, `Sell`   | `decimal.NewFromString`, trailing zeros dropped. No division.      |

A requested code with no row, or with a price that is empty or zero, goes to `Missing`.
A price that is not a number or is below zero fails the bank, as `LIBRATE-DD-004`
gives. Two rows of one code against the lari fail the bank. `INTLRATE-DD-003` gives the
check of the two prices. `ReadAt` is the instant of the answer.

**`module.go` (change).** `otherEntries` gains the entry of Inteli Express after MJC,
with the name `Inteli Express` and the variable `GEOMCP_RATES_INTELI_ENABLED`.

### 4.2 Domain model

No change. `RICORATE` 4.2 holds `KindMicrofinance`.

### 4.3 Declarations

**Vendor endpoint.**

| Client        | Method and path         | Key  | Official                  | Realizes          |
| ------------- | ----------------------- | ---- | ------------------------- | ----------------- |
| `pkg/inteli`  | `GET /ka/main-page-geo/` | None | No, a web page for people | `INTLRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to the table and two rows:

```
GET https://inteliexpress.com/ka/main-page-geo/
200 text/html; charset=UTF-8 (115271 bytes)
<div class="rates-col"><div class="rates-box">
  <table border="1" cellpadding="5"><thead>
    <tr><th colspan="5"><center>
      <i class="fa fa-exchange"></i> ვალუტის სტანდარტული კურსები</center>
    </th></tr>
  </thead><tbody><tr>
    <td>GEL</td><td>USD</td><td>2.595</td><td>2.598</td>
  </tr><tr>
    <td>EUR</td><td>USD</td><td>1.119</td><td>1.123</td>
  </tr>
```

**Configuration.**

| Variable                        | Type     | Default                     | Secret | Realizes           |
| ------------------------------- | -------- | --------------------------- | ------ | ------------------ |
| `GEOMCP_RATES_INTELI_ENABLED`   | bool     | `true`                      | No     | `INTLRATE-FR-005`  |
| `GEOMCP_RATES_INTELI_BASE_URL`  | URL      | `https://inteliexpress.com` | No     | `INTLRATE-FR-001`  |
| `GEOMCP_RATES_INTELI_TIMEOUT`   | duration | `10s`                       | No     | `INTLRATE-FR-001`  |
| `GEOMCP_RATES_INTELI_RATE`      | float    | `1`                         | No     | `INTLRATE-NFR-001` |
| `GEOMCP_RATES_INTELI_BURST`     | int      | `1`                         | No     | `INTLRATE-NFR-001` |

`rates.Config` gains `Inteli vendors.Config` with `envPrefix:"INTELI_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `INTLRATE-DD-001`

**Realizes:** `INTLRATE-FR-001`, `CLT-003`
**Decision:** The client finds the first `table` whose `thead` text holds
`ვალუტის სტანდარტული კურსები`, with each run of white space made one space. Each `tr`
of its `tbody` gives one row of four `td`: `Base`, `Currency`, `Buy`, and `Sell`, in
that order. The text of each cell is trimmed.
**Rationale:** The table names no column, so a position is the only reading. The title
tells the rate table from any other table that the page can gain.
**Alternatives:** The first table of the page. A new table above it would give wrong rows.

The client returns `ErrNoRateTable` when no table has the title, or the table has no
row, and `ErrMalformedTable` when a row has a count of cells other than four.

### `INTLRATE-DD-002`

**Realizes:** `INTLRATE-FR-001`
**Decision:** The adapter drops each row whose `Base` is not `GEL`, before it reads a code.
**Rationale:** The row `EUR`, `USD` is a cross rate. A commercial rate is against the lari.
**Alternatives:** Drop the row with the code `USD` and the base `EUR` only. A new cross
rate would then reach the answer.

### `INTLRATE-DD-003`

**Realizes:** `INTLRATE-FR-003`
**Decision:** The adapter compares the two prices of each row against the lari, before
it looks for the requested codes. When a buy price is above its sell price, the bank
fails with `kernel.ErrSourceUnavailable`. Equal prices pass. A row with an empty or zero
price is not compared.
**Rationale:** A swap of the columns moves every row at once. A check of every row finds
it, even when the requested code is not the one that shows it.
**Alternatives:** Check the requested rows only. A swap would then go unseen for a code
whose two prices are equal.

## 6. Scenarios

`pkg/inteli/testdata/` holds `main-page-geo.html`, the real page, recorded on 2026-10-25.

### `INTLRATE-SC-001`

**Verifies:** `INTLRATE-FR-001`, `INTLRATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/ka/main-page-geo/`, `Accept` is `text/html`, the User-Agent
starts with `georgia-mcp/`, and the request carries no key header.

### `INTLRATE-SC-002`

**Verifies:** `INTLRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `main-page-geo.html`.
**When** `Rates` runs.
**Then** the result holds 6 rows. Three rows are:

| Base  | Currency | Buy     | Sell    |
| ----- | -------- | ------- | ------- |
| `GEL` | `USD`    | `2.595` | `2.598` |
| `EUR` | `USD`    | `1.119` | `1.123` |
| `GEL` | `TRY`    | `0.050` | `0.056` |

### `INTLRATE-SC-003`

**Verifies:** `INTLRATE-FR-001`
**Layer:** unit

| Vendor answer                                       | Client result                            |
| --------------------------------------------------- | ---------------------------------------- |
| A page with no `table`                              | `ErrNoRateTable`                         |
| `main-page-geo.html` with the title renamed         | `ErrNoRateTable`                         |
| A table with no title above the rate table          | The 6 rows of `INTLRATE-SC-002`          |
| `main-page-geo.html` with a row of three cells      | `ErrMalformedTable`                      |
| 503 with a body of 2 KiB                            | `*APIError` with 503 and the first 1 KiB |

### `INTLRATE-SC-004`

**Verifies:** `INTLRATE-FR-001`
**Layer:** live

**Given** the real home page of Inteli Express.
**When** `Rates` runs.
**Then** a row with the base `GEL` and the currency `USD` has a buy price and a sell
price above zero.

### `INTLRATE-SC-005`

**Verifies:** `INTLRATE-FR-001`, `INTLRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `main-page-geo.html`.
**When** the adapter asks for `EUR`, `GBP`, `TRY`, and `USD`.
**Then** the result holds 4 rates, and no rate of `EUR` against `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.903`        | `2.915`         |
| `GBP`    | `3.457`        | `3.496`         |
| `TRY`    | `0.05`         | `0.056`         |
| `USD`    | `2.595`        | `2.598`         |

### `INTLRATE-SC-006`

**Verifies:** `INTLRATE-FR-002`
**Layer:** unit

**Given** a page with the row of `USD` and the row of `GBP` of the table.
**When** the adapter asks for `CHF`, `GBP`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF GBP]`.

| Prices of `GBP`     |
| ------------------- |
| `0.000`, `3.496`    |
| empty, `3.496`      |

### `INTLRATE-SC-007`

**Verifies:** `INTLRATE-FR-003`
**Layer:** unit

**Given** a page with the row of `USD` and a second row of the table.
**When** the adapter asks for `USD`.

| Second row                         | Result                                   |
| ---------------------------------- | ---------------------------------------- |
| `GEL`, `GBP`, `3.457`, `3.496`     | The rate of `USD`                        |
| `GEL`, `GBP`, `3.457`, `3.457`     | The rate of `USD`                        |
| `GEL`, `GBP`, `3.496`, `3.457`     | Wraps `kernel.ErrSourceUnavailable`      |
| `EUR`, `USD`, `1.123`, `1.119`     | The rate of `USD`                        |

### `INTLRATE-SC-008`

**Verifies:** `INTLRATE-FR-001`
**Layer:** unit

| Row of `USD`                        | Adapter error                        |
| ----------------------------------- | ------------------------------------ |
| `Currency` is `US`                  | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `n/a`                      | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `-2.595`                   | Wraps `kernel.ErrSourceUnavailable`  |
| A second row `GEL`, `USD`           | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503              | Wraps `kernel.ErrSourceUnavailable`  |

### `INTLRATE-SC-009`

**Verifies:** `INTLRATE-FR-005`, `INTLRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Inteli Express.
**When** the configuration loads.
**Then** `Inteli.Enabled` is true, `Inteli.BaseURL` is `https://inteliexpress.com`,
`Inteli.HTTP.Rate` is 1, and `Inteli.HTTP.Burst` is 1.

### `INTLRATE-SC-010`

**Verifies:** `INTLRATE-FR-001`, `INTLRATE-FR-004`
**Layer:** integration

**Given** Inteli Express on, every other bank off, the vendor serves
`main-page-geo.html`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the bank `inteli` has `source` `Inteli Express`, `as_of` is the instant of the
clock, `buy` is `2.595`, `sell` is `2.598`, and `missing` is `["CHF"]`. The description
of the tool lists `inteli (Inteli Express, a microfinance organization)`.

### `INTLRATE-SC-011`

**Verifies:** `INTLRATE-FR-005`
**Layer:** integration

**Given** Inteli Express off and `bog` on.
**When** a client calls the tool with the bank `inteli`.
**Then** the tool result reads `banks: "inteli" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_INTELI_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                   | Done |
| --- | -------------------------------------------------------------------------- | ------------------------------------------ | ---- |
| 1   | `pkg/inteli`, `testdata/main-page-geo.html`, README, example, live test. `.golangci.yaml`. | `INTLRATE-FR-001`          | [ ]  |
| 2   | `adapter/inteli`.                                                          | `INTLRATE-FR-001` to `INTLRATE-FR-004`, `INTLRATE-INV-001` | [ ]  |
| 3   | `config.go`: `Config.Inteli`, `Defaults`. `configs/.env.example`.          | `INTLRATE-FR-005`, `INTLRATE-NFR-001`      | [ ]  |
| 4   | `module.go`: the entry of Inteli Express. The glossary.                    | `INTLRATE-FR-001`, `INTLRATE-FR-005`       | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
