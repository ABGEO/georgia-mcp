---
id: SCAPRATE
title: The commercial exchange rates of Scapp
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/scaprate/requirements.md
---

# Specification: The commercial exchange rates of Scapp

## 1. Summary

This feature adds the vendor client `pkg/scapp`, a client of the rate page of Scapp, and
Scapp as a source of `rates_get_commercial_rates`. The tool, the use case, the domain,
and the cache of `TBCRATE` do not change. One call reads every currency, and the adapter
divides each price by its published unit.

## 2. Coverage

| Requirement        | Where this specification realizes it                       |
| ------------------ | ---------------------------------------------------------- |
| `SCAPRATE-FR-001`  | 4.1 `adapter/scapp`. 4.3 vendor endpoint. `SCAPRATE-DD-001` |
| `SCAPRATE-FR-002`  | `SCAPRATE-DD-002`                                          |
| `SCAPRATE-FR-003`  | 4.1 `adapter/scapp`. `LIBRATE-DD-004`                      |
| `SCAPRATE-FR-004`  | 4.1 `adapter/scapp`. `RICORATE-DD-004`                     |
| `SCAPRATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`      |
| `SCAPRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                        |
| `SCAPRATE-INV-001` | `SCAPRATE-DD-002`                                          |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/scapp` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/scapp`. |
| `CLT-002`, `CLT-004`   | The standard library, `pkg/vendorhttp`, and `golang.org/x/net/html`. `ADR-0003`. |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.    |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                        |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the source is a web page for people. |
| `DOM-005`, `DOM-011`   | `adapter/scapp` converts the rows and wraps `kernel.ErrSourceUnavailable`.   |
| `DOM-009`              | Scapp is an adapter of `CommercialRateSource`.                               |
| `MCP-002`              | No new tool. The description lists `scapp` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Scapp sits behind a `CommercialRateCache`, as TBC does.                      |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_SCAPP_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `SCAPRATE` 5.                             |
| Vendor client  | `pkg/scapp/`                          | `Client`, `New`, `Option`, `RateRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateTable`, `ErrMalformedTable`. |
| Driven adapter | `R/adapter/scapp/`                    | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Scapp`, `scappSource`.                               |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Linter         | `.golangci.yaml` (change)             | `pkg/scapp` joins `web-page-clients`, and leaves `clients`.  |

**`pkg/scapp`.** The shape of `pkg/rico`, with one method.

```go
const DefaultBaseURL = "https://scapp.ge"

var (
    ErrUnexpectedStatus = errors.New("scapp: unexpected status")
    ErrNoRateTable      = errors.New("scapp: no rate table")
    ErrMalformedTable   = errors.New("scapp: malformed rate table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one row of the rate table of the page, as the page shows it.
type RateRow struct {
    Unit     string // "1 USD"
    Buy      string // "2.595"
    Sell     string // "2.599"
}

type APIError = vendorhttp.StatusError
```

`Rates` sends `GET /ka/currency` with `Accept: text/html`. Any status other than 200
returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the
page has 329 KB. `SCAPRATE-DD-001` gives the parse.

**`adapter/scapp`.** `Bank` returns
`{Code: "scapp", Name: "Scapp", Kind: domain.KindMicrofinance}`. `CommercialRates` sends
one call, and converts the row of each requested code:

| Row of the page | Domain value                                                       |
| --------------- | ------------------------------------------------------------------ |
| `Unit`          | The count and the code. `SCAPRATE-DD-002`.                         |
| `Buy`, `Sell`   | `decimal.NewFromString`, divided by the count, trailing zeros dropped. |

A requested code with no row, or with a price that is empty or zero, goes to `Missing`.
A price that is not a number or is below zero fails the bank, as `LIBRATE-DD-004`
gives. Two rows of one code fail the bank. `ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Scapp when
`cfg.Scapp.Enabled` is true. Else it logs at info level `Scapp is off` with the
attribute `variable=GEOMCP_RATES_SCAPP_ENABLED`.

### 4.2 Domain model

No change. `RICORATE` 4.2 holds `KindMicrofinance`.

### 4.3 Declarations

**Vendor endpoint.**

| Client       | Method and path    | Key  | Official                  | Realizes          |
| ------------ | ------------------ | ---- | ------------------------- | ----------------- |
| `pkg/scapp`  | `GET /ka/currency` | None | No, a web page for people | `SCAPRATE-FR-001` |

A real answer, recorded on 2026-10-25, cut to the table and one row:

```
GET https://scapp.ge/ka/currency
200 text/html; charset=UTF-8 (329277 bytes)
Cache-Control: max-age=300, public
<h2 class="..."><div>მიმდინარე კურსები</div></h2>
<div class="bg-white ...">
  <table class="table-auto w-full">
    <thead><tr class="...">
      <th class="pb-3"><span class="hidden">ვალუტის კურსი</span></th>
      <th class="pb-3"><span>ყიდვა</span></th>
      <th class="pb-3"><span>გაყიდვა</span></th>
    </tr></thead>
    <tr class="border-b ...">
      <td class="flex ..."><img ... alt="Currency icon"><span class="text-lg ...">
        1
        USD
      </span></td>
      <td class="text-left pt-3 pb-4"><span class="...">2.595</span></td>
      <td class="text-left pt-3 pb-4"><span class="...">2.599</span></td>
    </tr>
```

The converter below the table holds `<li data-sell="2.599" data-buy="2.595" data-currency="USD" ...>`
for each currency. The client does not read it (`SCAPRATE-DD-001`).

**Configuration.**

| Variable                       | Type     | Default            | Secret | Realizes           |
| ------------------------------ | -------- | ------------------ | ------ | ------------------ |
| `GEOMCP_RATES_SCAPP_ENABLED`   | bool     | `true`             | No     | `SCAPRATE-FR-005`  |
| `GEOMCP_RATES_SCAPP_BASE_URL`  | URL      | `https://scapp.ge` | No     | `SCAPRATE-FR-001`  |
| `GEOMCP_RATES_SCAPP_TIMEOUT`   | duration | `10s`              | No     | `SCAPRATE-FR-001`  |
| `GEOMCP_RATES_SCAPP_RATE`      | float    | `1`                | No     | `SCAPRATE-NFR-001` |
| `GEOMCP_RATES_SCAPP_BURST`     | int      | `1`                | No     | `SCAPRATE-NFR-001` |

`rates.Config` gains `Scapp vendors.Config` with `envPrefix:"SCAPP_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `SCAPRATE-DD-001`

**Realizes:** `SCAPRATE-FR-001`, `CLT-003`
**Decision:** The client reads the first `table` of the page. Its `thead` gives the
titles of the columns: the text of each `th`, with the spaces trimmed. The title `ყიდვა`
names the buy column, `გაყიდვა` the sell column. Each `tr` outside `thead` gives one row.
`Unit` is the text of the first `td`, with each run of white space made one space and
the ends trimmed: `1 USD`. `Buy` and `Sell` are the trimmed text of the cell under their
titles.
**Rationale:** A match by title, not by a fixed position, finds a swap of the two
columns. The page holds one table.
**Alternatives:** The `data-buy` and `data-sell` of the converter. They cut the
trailing zeros, carry no unit, and a change of the converter would go unseen.

The client returns `ErrNoRateTable` when the page has no `table` or no row, and
`ErrMalformedTable` when the titles lack `ყიდვა` or `გაყიდვა`, or a row lacks a cell.

### `SCAPRATE-DD-002`

**Realizes:** `SCAPRATE-FR-002`, `SCAPRATE-INV-001`
**Decision:** The adapter splits `Unit` at the space. The first part is the count, the
second the code. The code goes through `domain.ParseCurrencyCode`. The count follows
`LIBRATE-DD-003`: a power of 10 from 1 to 1 000 000, and each price is divided with
`Shift(-k)`. A unit with no space, a code that fails, or another count fails the bank.
**Rationale:** The page writes the unit and the code in one cell. The shift is exact.
**Alternatives:** The code from the flag image. The image has the code in its file name
only, and `alt` is `Currency icon` for each row.

## 6. Scenarios

`pkg/scapp/testdata/` holds `currency-ka.html`, the real page, recorded on 2026-10-25.

### `SCAPRATE-SC-001`

**Verifies:** `SCAPRATE-FR-001`, `SCAPRATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/ka/currency`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp/`, and the request carries no key header.

### `SCAPRATE-SC-002`

**Verifies:** `SCAPRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `currency-ka.html`.
**When** `Rates` runs.
**Then** the result holds 6 rows. Two rows are:

| Unit    | Buy     | Sell    |
| ------- | ------- | ------- |
| `1 USD` | `2.595` | `2.599` |
| `1 EUR` | `2.900` | `2.914` |

### `SCAPRATE-SC-003`

**Verifies:** `SCAPRATE-FR-001`
**Layer:** unit

| Vendor answer                                     | Client result                             |
| ------------------------------------------------- | ----------------------------------------- |
| A page with no `table`                            | `ErrNoRateTable`                          |
| `currency-ka.html` with the title `ყიდვა` renamed  | `ErrMalformedTable`                       |
| `currency-ka.html` with a row of two cells        | `ErrMalformedTable`                       |
| 503 with a body of 2 KiB                          | `*APIError` with 503 and the first 1 KiB  |

### `SCAPRATE-SC-004`

**Verifies:** `SCAPRATE-FR-001`
**Layer:** live

**Given** the real page of Scapp.
**When** `Rates` runs.
**Then** a row with the unit `1 USD` has a buy price and a sell price above zero.

### `SCAPRATE-SC-005`

**Verifies:** `SCAPRATE-FR-001`, `SCAPRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `currency-ka.html`.
**When** the adapter asks for `EUR`, `TRY`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `EUR`    | `2.9`          | `2.914`         |
| `TRY`    | `0.049`        | `0.064`         |
| `USD`    | `2.595`        | `2.599`         |

### `SCAPRATE-SC-006`

**Verifies:** `SCAPRATE-FR-002`, `SCAPRATE-INV-001`
**Layer:** unit

**Given** a page with the row of `USD` and a row `10 JPY`, `0.170`, `0.210`.
**When** the adapter asks for `JPY`.
**Then** the buy rate formats as `0.017`, and the sell rate as `0.021`.

### `SCAPRATE-SC-007`

**Verifies:** `SCAPRATE-FR-003`
**Layer:** unit

**Given** a page with the row of `USD` and the row of `CHF` of the table.
**When** the adapter asks for `CHF`, `GBP`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF GBP]`.

| Prices of `CHF`    |
| ------------------ |
| `0.000`, `3.250`   |
| empty, `3.250`     |

### `SCAPRATE-SC-008`

**Verifies:** `SCAPRATE-FR-001`, `SCAPRATE-FR-002`
**Layer:** unit

| Row of `USD`                    | Adapter error                        |
| ------------------------------- | ------------------------------------ |
| `Unit` is `3 USD`               | Wraps `kernel.ErrSourceUnavailable`  |
| `Unit` is `USD`                 | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `n/a`                  | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `-2.595`               | Wraps `kernel.ErrSourceUnavailable`  |
| A second row `1 USD`            | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503          | Wraps `kernel.ErrSourceUnavailable`  |

### `SCAPRATE-SC-009`

**Verifies:** `SCAPRATE-FR-005`, `SCAPRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Scapp.
**When** the configuration loads.
**Then** `Scapp.Enabled` is true, `Scapp.BaseURL` is `https://scapp.ge`,
`Scapp.HTTP.Rate` is 1, and `Scapp.HTTP.Burst` is 1.

### `SCAPRATE-SC-010`

**Verifies:** `SCAPRATE-FR-001`, `SCAPRATE-FR-004`
**Layer:** integration

**Given** Scapp on, every other bank off, the vendor serves `currency-ka.html`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the bank `scapp` has `source` `Scapp`, `as_of` is the instant of the clock,
`buy` is `2.595`, `sell` is `2.599`, and `missing` is `["CHF"]`. The description of the
tool lists `scapp (Scapp, a microfinance organization)`.

### `SCAPRATE-SC-011`

**Verifies:** `SCAPRATE-FR-005`
**Layer:** integration

**Given** Scapp off and `bog` on.
**When** a client calls the tool with the bank `scapp`.
**Then** the tool result reads `banks: "scapp" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_SCAPP_ENABLED`.

## 7. Build plan

| #   | Step                                                                       | Realizes                                   | Done |
| --- | -------------------------------------------------------------------------- | ------------------------------------------ | ---- |
| 1   | `pkg/scapp`, `testdata/currency-ka.html`, README, example, live test. `.golangci.yaml`. | `SCAPRATE-FR-001`             | [ ]  |
| 2   | `adapter/scapp`.                                                           | `SCAPRATE-FR-001` to `SCAPRATE-FR-004`, `SCAPRATE-INV-001` | [ ]  |
| 3   | `config.go`: `Config.Scapp`, `Defaults`. `configs/.env.example`.           | `SCAPRATE-FR-005`, `SCAPRATE-NFR-001`      | [ ]  |
| 4   | `module.go`: the source of Scapp, and the info record. The glossary.       | `SCAPRATE-FR-001`, `SCAPRATE-FR-005`       | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
