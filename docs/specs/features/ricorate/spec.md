---
id: RICORATE
title: The commercial exchange rates of Rico Credit
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/ricorate/requirements.md
---

# Specification: The commercial exchange rates of Rico Credit

## 1. Summary

This feature adds the vendor client module `pkg/rico`, a client of the home page of Rico
Credit, and Rico Credit as a source of `rates_get_commercial_rates`. It also adds the
kind of a bank to the domain, so the description of the tool names a company that is
not a bank. The use case and the cache of `TBCRATE` do not change.

## 2. Coverage

| Requirement        | Where this specification realizes it                         |
| ------------------ | ------------------------------------------------------------ |
| `RICORATE-FR-001`  | 4.1 `adapter/rico`. 4.3 vendor endpoint. `RICORATE-DD-001`, `RICORATE-DD-002` |
| `RICORATE-FR-002`  | 4.1 `adapter/rico`. `RICORATE-DD-003`                        |
| `RICORATE-FR-003`  | `RICORATE-DD-002`, `LIBRATE-DD-004`                          |
| `RICORATE-FR-004`  | 4.2 `BankKind`. 4.3 MCP tool. `RICORATE-DD-004`              |
| `RICORATE-FR-005`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`        |
| `RICORATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                          |
| `RICORATE-INV-001` | `RICORATE-DD-003`                                            |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                |
| ---------------------- | ------------------------------------------------------------------------------ |
| `ARC-004`, `CLT-001`   | `pkg/rico` is a module: `github.com/abgeo/georgia-mcp/pkg/rico`, `go 1.26.0`.   |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `golang.org/x/net/html` only. `ADR-0003`.            |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.       |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                           |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                      |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page for people. `v0`. |
| `DOM-003`              | `BankKind` is a value object with a closed set of values.                       |
| `DOM-005`, `DOM-011`   | `adapter/rico` converts the rows and wraps `kernel.ErrSourceUnavailable`.       |
| `MCP-002`              | No new tool. The description of `rates_get_commercial_rates` changes. 4.3.      |
| `MCP-003`              | The input and the output keep their structs and their tags.                     |
| `RSL-003` to `RSL-006` | Rico Credit sits behind a `CommercialRateCache`, as TBC does.                   |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_RICO_` and a default in its tag.     |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The terms of `RICORATE` 5 and the wider bank.         |
| Vendor client  | `pkg/rico/`                           | `Client`, `New`, `Option`, `RateRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateTable`, `ErrMalformedTable`. |
| Domain         | `R/domain/bank.go` (change)           | `BankKind`, `KindBank`, `KindMicrofinance`, `Bank.Kind`. 4.2. |
| Driven adapter | `R/adapter/rico/`                     | `Adapter`, `NewAdapter`.                                     |
| Driven adapter | `R/adapter/tbc/`, `R/adapter/bog/` (change) | `Bank` sets `Kind: domain.KindBank`.                   |
| Tool           | `R/adapter/tool/commercial.go` (change) | `getCommercialDescription`, `bankList`. 4.3.               |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Rico`, `ricoSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Workspace      | `go.work` (change)                    | `use ./pkg/rico`.                                            |

**`pkg/rico`.** The shape of `pkg/rompetrol`, with one method.

```go
const DefaultBaseURL = "https://www.rico.ge"

var (
    ErrUnexpectedStatus = errors.New("rico: unexpected status")
    ErrNoRateTable      = errors.New("rico: no rate table")
    ErrMalformedTable   = errors.New("rico: malformed rate table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one row of the rate table of the home page, as the page shows it.
type RateRow struct {
    Currency string // "PLN", or "USD-EUR" for the cross rate
    Unit     string // "10"
    Buy      string // "6.1000"
    Sell     string // "6.7000"
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`Rates` sends `GET /ka/` with `Accept: text/html`. Any status other than 200 returns an
`*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the page has
84 KB. The default User-Agent is
`georgia-mcp/<version> (+https://github.com/abgeo/georgia-mcp)`.
`RICORATE-DD-001` gives the parse.

**`adapter/rico`.** `Bank` returns
`{Code: "rico", Name: "Rico Credit", Kind: domain.KindMicrofinance}`. `CommercialRates`
sends one call. It skips each row whose `Currency` is not a code (`RICORATE-DD-002`),
and converts the row of each requested code:

| Row of the page | Domain value                                                       |
| --------------- | ------------------------------------------------------------------ |
| `Currency`      | `domain.ParseCurrencyCode`.                                        |
| `Unit`          | The count of units. `RICORATE-DD-003`.                             |
| `Buy`, `Sell`   | `decimal.NewFromString`, divided by the unit, trailing zeros dropped. |

A requested code with no row, or with a price that is empty or zero, goes to `Missing`.
A price that is not a number or is below zero fails the bank, as `LIBRATE-DD-004`
gives. `ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Rico Credit when
`cfg.Rico.Enabled` is true. Else it logs at info level `Rico Credit is off` with the
attribute `variable=GEOMCP_RATES_RICO_ENABLED`.

### 4.2 Domain model

```go
// BankKind is the kind of a company that publishes commercial rates.
type BankKind string

const (
    KindBank         BankKind = "bank"
    KindMicrofinance BankKind = "microfinance organization"
)

// Bank is a company that publishes commercial rates. Kind tells a bank from a
// company that is not one.
type Bank struct {
    Code BankCode
    Name string
    Kind BankKind
}
```

`KURSRATE` adds a third kind. The use case and the cache pass `Bank` through with no
change.

### 4.3 Declarations

**MCP tool.** `rates_get_commercial_rates` keeps its name, its input, its output, and
its errors (`TBCRATE` 4.3, 4.5). Its description changes:

| Tool                          | Input                    | Output                  | Errors        | Realizes          |
| ----------------------------- | ------------------------ | ----------------------- | ------------- | ----------------- |
| `rates_get_commercial_rates`  | `currencies`, `banks`    | `commercialRatesOutput` | `TBCRATE` 4.5 | `RICORATE-FR-004` |

```go
getCommercialDescription = "Returns the current commercial exchange rates of Georgian banks: " +
    "the prices in GEL at which a bank buys and sells one unit of a currency. Use it for the " +
    "price of a currency at a bank today, or to compare banks. Ask for 1 to 10 currencies. %s " +
    "The banks of this server: %s. The list gives the kind of each one that is not a bank. " +
    "The rates are of today. The tool has no history."
```

`bankList` writes `code (Name)` for `KindBank`, and `code (Name, a <kind>)` for each
other kind: `bog (Bank of Georgia), rico (Rico Credit, a microfinance organization)`.
The article is `an` when the kind starts with a vowel.

**Vendor endpoint.**

| Client      | Method and path | Key  | Official                  | Realizes          |
| ----------- | --------------- | ---- | ------------------------- | ----------------- |
| `pkg/rico`  | `GET /ka/`      | None | No, a web page for people | `RICORATE-FR-001` |

A real answer, recorded on 2026-10-21, cut to the table and two rows. The real page
writes each Georgian letter as an entity: `&#x10E7;&#x10D8;&#x10D3;&#x10D5;&#x10D0;`.

```
GET https://www.rico.ge/ka/
200 text/html; charset=utf-8 (83754 bytes)
<section class="currencies-section"><div class="currencies">
  <div class="currencies-table-card">
    <div class="currencies-table-head"><span></span><span>ერთეული</span><span>ყიდვა</span><span>გაყიდვა</span></div>
    <div class="currencies-table-body">
      <div class="currency-row">
        <div class="currency-info"><img class="currency-flag" src="/media/fpqljjpr/usd.png" alt="USD" /><strong>USD</strong></div>
        <div class="currency-unit"><span>ერთეული</span><strong>1</strong></div>
        <div class="currency-rate"><span>ყიდვა</span><strong>2.5960</strong></div>
        <div class="currency-rate"><span>გაყიდვა</span><strong>2.6000</strong></div>
      </div>
      <div class="currency-row">
        <div class="currency-info"><img ... alt="USD-EUR" /><strong>USD-EUR</strong></div>
        <div class="currency-unit"><span>ერთეული</span><strong>1</strong></div>
        <div class="currency-rate"><span>ყიდვა</span><strong>1.1204</strong></div>
        <div class="currency-rate"><span>გაყიდვა</span><strong>1.1260</strong></div>
      </div>
```

**Configuration.**

| Variable                       | Type     | Default               | Secret | Realizes           |
| ------------------------------ | -------- | --------------------- | ------ | ------------------ |
| `GEOMCP_RATES_RICO_ENABLED`    | bool     | `true`                | No     | `RICORATE-FR-005`  |
| `GEOMCP_RATES_RICO_BASE_URL`   | URL      | `https://www.rico.ge` | No     | `RICORATE-FR-001`  |
| `GEOMCP_RATES_RICO_TIMEOUT`    | duration | `10s`                 | No     | `RICORATE-FR-001`  |
| `GEOMCP_RATES_RICO_RATE`       | float    | `1`                   | No     | `RICORATE-NFR-001` |
| `GEOMCP_RATES_RICO_BURST`      | int      | `1`                   | No     | `RICORATE-NFR-001` |

`rates.Config` gains `Rico vendors.Config` with `envPrefix:"RICO_"`. `rates.Defaults` gives the defaults of this table (`PLATFORM-DD-014`). The cache is a `CommercialRateCache`, as for Bank of Georgia.

## 5. Design decisions

### `RICORATE-DD-001`

**Realizes:** `RICORATE-FR-001`, `CLT-003`
**Decision:** A class matches as a whole token of the `class` attribute, as
`LIBRATE-DD-001` gives. The client reads each `div.currency-row` inside the first
`div.currencies-table-body`. `Currency` is the text of the `strong` in `currency-info`.
`Unit` is the text of the `strong` in `currency-unit`. In each `currency-rate`, the text
of the `span` names the price: `ყიდვა` gives `Buy`, `გაყიდვა` gives `Sell`. The text of
the `strong` is the price.
**Rationale:** The label of a cell, not its position, tells buy from sell. A swap of the
two columns would otherwise give wrong rates with no error.
**Alternatives:** The embedded JSON of the calculator. The page divides and rounds it:
`AMD` reads `0.0068`, so a published digit can be lost. `RICORATE-INV-001` forbids it.

The client returns `ErrNoRateTable` when the page has no table or no row, and
`ErrMalformedTable` when a row lacks the code, the unit, the buy price, or the sell price.

### `RICORATE-DD-002`

**Realizes:** `RICORATE-FR-001`, `RICORATE-FR-003`
**Decision:** The adapter skips a row whose `Currency` fails `domain.ParseCurrencyCode`.
**Rationale:** The table holds the cross rate `USD-EUR`. The requirements leave it out.
A row that is not a code cannot match a requested code, so to skip it loses nothing.
**Alternatives:** Fail the bank on such a row. The cross rate is on the page every day.

### `RICORATE-DD-003`

**Realizes:** `RICORATE-FR-002`, `RICORATE-INV-001`
**Decision:** The adapter reads `Unit` as an integer. It must be a power of 10 from 1 to
1 000 000. Each price is divided with `Shift(-k)`, where the unit is `10^k`, as
`LIBRATE-DD-003` gives. Any other unit fails the bank.
**Rationale:** The shift is exact.
**Alternatives:** `LIBRATE-DD-003` lists them.

### `RICORATE-DD-004`

**Realizes:** `RICORATE-FR-004`
**Decision:** `domain.Bank` gains `Kind`. The tool writes the kind in the list of banks of
its description. The output keeps `source` as the name of the company. `bankList` treats
an empty `Kind` as `KindBank`, so a `Bank` literal of a test stays a bank.
**Rationale:** A client reads the description before a call, so the model knows the kind
when it names the company. A new output field is a change of the output that a client
can see.
**Alternatives:** A field `kind` in `bankRatesOutput`. It repeats the description in each
answer. The name `Rico Credit (microfinance organization)` in `source` mixes two
facts in one field.

## 6. Scenarios

`pkg/rico/testdata/` holds `home-ka.html`, the real page, recorded on 2026-10-21.

### `RICORATE-SC-001`

**Verifies:** `RICORATE-FR-001`, `RICORATE-FR-005`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/ka/`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp/`, and the request carries no key header.

### `RICORATE-SC-002`

**Verifies:** `RICORATE-FR-001`
**Layer:** unit

**Given** the vendor serves `home-ka.html`.
**When** `Rates` runs.
**Then** the result holds 13 rows. Three rows are:

| Currency  | Unit   | Buy      | Sell     |
| --------- | ------ | -------- | -------- |
| `USD`     | `1`    | `2.5960` | `2.6000` |
| `USD-EUR` | `1`    | `1.1204` | `1.1260` |
| `PLN`     | `10`   | `6.1000` | `6.7000` |

### `RICORATE-SC-003`

**Verifies:** `RICORATE-FR-001`
**Layer:** unit

| Vendor answer                                        | Client result                             |
| ---------------------------------------------------- | ----------------------------------------- |
| A page with no `currencies-table-body`               | `ErrNoRateTable`                          |
| `home-ka.html` with the `span` `გაყიდვა` of `USD` removed | `ErrMalformedTable`                  |
| `home-ka.html` with the two `currency-rate` of `USD` in the other order | The row of `USD` has `Buy` `2.5960` and `Sell` `2.6000` |
| 503 with a body of 2 KiB                             | `*APIError` with 503 and the first 1 KiB  |

### `RICORATE-SC-004`

**Verifies:** `RICORATE-FR-001`
**Layer:** live

**Given** the real home page of Rico Credit.
**When** `Rates` runs.
**Then** a row of `USD` has a buy price and a sell price above zero.

### `RICORATE-SC-005`

**Verifies:** `RICORATE-FR-001`, `RICORATE-FR-002`, `RICORATE-INV-001`
**Layer:** unit

**Given** the vendor serves `home-ka.html`.
**When** the adapter asks for `AMD`, `PLN`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `AMD`    | `0.0068`       | `0.0071`        |
| `PLN`    | `0.61`         | `0.67`          |
| `USD`    | `2.596`        | `2.6`           |

### `RICORATE-SC-006`

**Verifies:** `RICORATE-FR-003`
**Layer:** unit

**Given** a page with the row of `USD`, the row of `USD-EUR`, and a row of `CHF` with the buy price `0.0000`.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

### `RICORATE-SC-007`

**Verifies:** `RICORATE-FR-002`, `RICORATE-INV-001`
**Layer:** unit

| Row of `USD`              | Adapter error                        |
| ------------------------- | ------------------------------------ |
| `Unit` is `3`             | Wraps `kernel.ErrSourceUnavailable`  |
| `Unit` is `ten`           | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `n/a`            | Wraps `kernel.ErrSourceUnavailable`  |
| `Buy` is `-2.5960`        | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503    | Wraps `kernel.ErrSourceUnavailable`  |

### `RICORATE-SC-008`

**Verifies:** `RICORATE-FR-004`
**Layer:** unit

| Banks of the getter                                          | The list in the description                                            |
| ------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `{bog, Bank of Georgia, bank}`                               | `bog (Bank of Georgia)`                                                |
| `{bog, Bank of Georgia}`, with no kind                       | `bog (Bank of Georgia)`                                                |
| `{bog, Bank of Georgia, bank}`, `{rico, Rico Credit, microfinance organization}` | `bog (Bank of Georgia), rico (Rico Credit, a microfinance organization)` |

### `RICORATE-SC-009`

**Verifies:** `RICORATE-FR-005`, `RICORATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Rico Credit.
**When** the configuration loads.
**Then** `Rico.Enabled` is true, `Rico.BaseURL` is `https://www.rico.ge`,
`Rico.HTTP.Rate` is 1, and `Rico.HTTP.Burst` is 1.

### `RICORATE-SC-010`

**Verifies:** `RICORATE-FR-001`, `RICORATE-FR-004`
**Layer:** integration

**Given** Rico Credit on, every other bank off, the vendor serves `home-ka.html`, and a fixed clock.
**When** a client lists the tools, then calls `rates_get_commercial_rates` with `USD`.
**Then** the description holds `rico (Rico Credit, a microfinance organization)`. The
bank `rico` has `source` `Rico Credit`, `as_of` is the instant of the clock, `buy` is
`2.596`, and `sell` is `2.6`.

### `RICORATE-SC-011`

**Verifies:** `RICORATE-FR-005`
**Layer:** integration

**Given** Rico Credit off and `bog` on.
**When** a client calls the tool with the bank `rico`.
**Then** the tool result reads `banks: "rico" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_RICO_ENABLED`.

## 7. Build plan

| #   | Step                                                                        | Realizes                                 | Done |
| --- | --------------------------------------------------------------------------- | ---------------------------------------- | ---- |
| 1   | `domain.BankKind`, `Bank.Kind`. The adapters of TBC and Bank of Georgia set `KindBank`. | `RICORATE-FR-004`             | [x]  |
| 2   | The description of the tool and `bankList`.                                 | `RICORATE-FR-004`                        | [x]  |
| 3   | `pkg/rico` module, `testdata/home-ka.html`, README, example. `go.work`.     | `RICORATE-FR-001`                        | [x]  |
| 4   | `adapter/rico`.                                                             | `RICORATE-FR-001` to `RICORATE-FR-003`, `RICORATE-INV-001` | [x]  |
| 5   | `config.go`: `RicoConfig`, `RicoHTTPConfig`. `configs/.env.example`.        | `RICORATE-FR-005`, `RICORATE-NFR-001`    | [x]  |
| 6   | `module.go`: the source of Rico Credit, and the info record.                | `RICORATE-FR-001`, `RICORATE-FR-005`     | [x]  |

Step 1 sets `KindBank` in every adapter of a bank that exists at that time, the adapters
of `LIBRATE` and `TERARATE` included.

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
