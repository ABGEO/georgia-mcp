---
id: LIBRATE
title: The commercial exchange rates of Liberty Bank
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-21
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/librate/requirements.md
---

# Specification: The commercial exchange rates of Liberty Bank

## 1. Summary

This feature adds the vendor client module `pkg/libertyrates`, a client of the home page
of Liberty Bank, and Liberty as a source of `rates_get_commercial_rates`. The tool, the
use case, the domain, and the cache of `TBCRATE` do not change. One call reads every
currency, and the adapter divides each price by its published unit.

## 2. Coverage

| Requirement       | Where this specification realizes it                        |
| ----------------- | ----------------------------------------------------------- |
| `LIBRATE-FR-001`  | 4.1 `adapter/liberty`. 4.3 vendor endpoint. `LIBRATE-DD-002` |
| `LIBRATE-FR-002`  | 4.1 `adapter/liberty`. `LIBRATE-DD-003`                     |
| `LIBRATE-FR-003`  | `LIBRATE-DD-004`                                            |
| `LIBRATE-FR-004`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`       |
| `LIBRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                         |
| `LIBRATE-INV-001` | `LIBRATE-DD-003`                                            |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                |
| ---------------------- | ------------------------------------------------------------------------------ |
| `ARC-004`, `CLT-001`   | `pkg/libertyrates` is a module: `github.com/abgeo/georgia-mcp/pkg/libertyrates`, `go 1.26.0`. `ADR-0004` names it. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `golang.org/x/net/html` only. `ADR-0003`.            |
| `CLT-003`              | The client returns the text of each cell. The adapter parses the numbers.       |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                           |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                      |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page for people. `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/liberty` converts the rows and wraps `kernel.ErrSourceUnavailable`.    |
| `DOM-009`              | Liberty is an adapter of `CommercialRateSource`.                                |
| `MCP-002`              | No new tool. The description lists `liberty` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Liberty sits behind a `CommercialRateCache`, as TBC does.                       |
| `RSL-008`              | The configuration gives the limit of the calls. `LIBRATE-DD-005`.               |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_LIBERTY_` and a default in its tag.  |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Vendor client  | `pkg/libertyrates/`                   | `Client`, `New`, `Option`, `RateRow`, `Quote`, `APIError`, `ErrUnexpectedStatus`, `ErrNoRateBlock`, `ErrMalformedBlock`. |
| Driven adapter | `R/adapter/liberty/`                  | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Liberty`, `LibertyConfig`, `LibertyHTTPConfig`, `libertySource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |
| Workspace      | `go.work` (change)                    | `use ./pkg/libertyrates`.                                    |

**`pkg/libertyrates`.** The shape of `pkg/rompetrol`, with one method.

```go
const DefaultBaseURL = "https://www.libertybank.ge"

var (
    ErrUnexpectedStatus = errors.New("libertyrates: unexpected status")
    ErrNoRateBlock      = errors.New("libertyrates: no rate block")
    ErrMalformedBlock   = errors.New("libertyrates: malformed rate block")
)

func New(opts ...Option) *Client // WithBaseURL, WithHTTPClient, WithUserAgent
func (c *Client) Rates(ctx context.Context) ([]RateRow, error)

// RateRow is one currency of the rate block of the home page, as the page shows it.
type RateRow struct {
    Currency     string // "AMD"
    Unit         string // "1000 AMD"
    Official     string // "7.1737"
    Commercial   Quote
    InternetBank Quote  // empty when the page shows none
    Transfer     Quote  // empty when the page shows none
}

// Quote is a buy price and a sell price, as the page shows them.
type Quote struct {
    Buy  string // "6.2220"
    Sell string // "7.5730"
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`Rates` sends `GET /ka/` with `Accept: text/html`. Any status other than 200 returns an
`*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 2 MiB: the page has
124 KB. The default User-Agent is
`georgia-mcp-libertyrates/<version> (+https://github.com/abgeo/georgia-mcp)`.
`LIBRATE-DD-001` gives the parse.

**`adapter/liberty`.** `Bank` returns `{Code: "liberty", Name: "Liberty Bank"}`, with
`Kind: domain.KindBank` once `RICORATE` adds the field.
`CommercialRates` sends one call, and converts the row of each requested code:

| Row of the page           | Domain value                                                    |
| ------------------------- | --------------------------------------------------------------- |
| `Currency`                | `domain.ParseCurrencyCode`. A code that fails fails the bank.   |
| `Unit`                    | The count of units. `LIBRATE-DD-003`.                           |
| `Commercial.Buy`, `.Sell` | `decimal.NewFromString`, divided by the unit, trailing zeros dropped. `LIBRATE-DD-003`. |
| `Official`, `InternetBank`, `Transfer` | Not read. `LIBRATE-DD-002`.                        |

A requested code with no row, or with a commercial price that is empty or zero, goes
to `Missing` (`LIBRATE-DD-004`). `ReadAt` is the instant of the answer. Each failure
returns an error that wraps `kernel.ErrSourceUnavailable`: an error of the client, an
empty list, a price that is not a number or is below zero, and a unit that
`LIBRATE-DD-003` refuses.

**`module.go` (change).** `commercialSources` builds the source of Liberty when
`cfg.Liberty.Enabled` is true. Else it logs at info level `Liberty Bank is off` with the
attribute `variable=GEOMCP_RATES_LIBERTY_ENABLED`, as `BOGRATE-DD-007` gives.

### 4.2 Domain model

No change. `TBCRATE` 4.2 holds the model.

### 4.3 Declarations

**Vendor endpoint.**

| Client              | Method and path | Key  | Official                  | Realizes         |
| ------------------- | --------------- | ---- | ------------------------- | ---------------- |
| `pkg/libertyrates`  | `GET /ka/`      | None | No, a web page for people | `LIBRATE-FR-001` |

A real answer, recorded on 2026-10-21, cut to the header and two rows. The real page
holds a second copy of the block, `id="currencyrates2"`, for a narrow screen.

```
GET https://www.libertybank.ge/ka/
200 text/html; charset=utf-8 (124360 bytes)
<div class="currency-rates" id="currencyrates1">
  <div class="currency-rates__head">...<p class="currency-rates__date">21 ოქტომბერი, 2026</p></div>
  <div class="currency-rates__body">
    <div class="currency-rates__row-head"><div class="currency-rates__row">
      <div class="currency-rates__item"><span class="currency-rates__item-title caps bold">ოფიციალური</span></div>
      <div class="currency-rates__item"><span class="currency-rates__item-title caps bold">კომერციული</span></div>
      <div class="currency-rates__item"><span class="currency-rates__item-title caps bold">ინტერნეტ ბანკი</span>...</div>
      <div class="currency-rates__item currency-rates__item--spec"><span class="currency-rates__item-title caps bold">გზავნილების სპეც კურსი</span></div>
      <div class="currency-rates__item currency-rates__item--showonmobile">...</div>
    </div></div>
    <div class="currency-rates__row-body">
      <div class="currency-rates__preview">
        <div class="currency-rates__row js-homepage__currency-item">
          <div class="currency-rates__item">
            <div class="currency-rates__currency-name"><span class="caps medium">USD</span></div>
            <div class="currency-rates__currency-wrap"><span class="currency-rates__currency caps bold">2.5978</span>...</div>
            <span class="currency-rates__currency-description">1 USD</span>
          </div>
          <div class="currency-rates__item"><span class="currency-rates__currency caps bold">2.5470</span><span class="currency-rates__currency caps bold">2.6470</span></div>
          <div class="currency-rates__item"><span ...>2.5870</span><span ...>2.6070</span></div>
          <div class="currency-rates__item"><span ...>2.5960</span><span ...>2.6000</span></div>
          <div class="currency-rates__item currency-rates__item--showonmobile visible-xs">...</div>
        </div>
      </div>
      <div class="currency-rates__full">
        <div class="currency-rates__bloomberg">...</div>
        <div class="currency-rates__row js-homepage__currency-item">
          <div class="currency-rates__item">...<span class="caps medium">AMD</span>...<span class="currency-rates__currency-description">1000 AMD</span></div>
          <div class="currency-rates__item"><span ...>6.2220</span><span ...>7.5730</span></div>
          <div class="currency-rates__item"></div>
          <div class="currency-rates__item currency-rates__item--showonmobile visible-xs">...</div>
        </div>
      </div>
```

**Configuration.**

| Variable                          | Type     | Default                      | Secret | Realizes          |
| --------------------------------- | -------- | ---------------------------- | ------ | ----------------- |
| `GEOMCP_RATES_LIBERTY_ENABLED`    | bool     | `true`                       | No     | `LIBRATE-FR-004`  |
| `GEOMCP_RATES_LIBERTY_BASE_URL`   | URL      | `https://www.libertybank.ge` | No     | `LIBRATE-FR-001`  |
| `GEOMCP_RATES_LIBERTY_TIMEOUT`    | duration | `10s`                        | No     | `LIBRATE-FR-001`  |
| `GEOMCP_RATES_LIBERTY_RATE`       | float    | `1`                          | No     | `LIBRATE-NFR-001` |
| `GEOMCP_RATES_LIBERTY_BURST`      | int      | `1`                          | No     | `LIBRATE-NFR-001` |

`rates.Config` gains `Liberty LibertyConfig` with `envPrefix:"LIBERTY_"`. `LibertyConfig`
and `LibertyHTTPConfig` have the shape of `BOGConfig` and `BOGHTTPConfig`, with the
defaults of this table.

The tool, its structs, its errors, and the cache are those of `TBCRATE` 4.3 and 4.5.
The cache of Liberty is a `CommercialRateCache`: 5 minutes to live, 55 minutes of
maximum stale, 256 entries.

## 5. Design decisions

### `LIBRATE-DD-001`

**Realizes:** `LIBRATE-FR-001`, `CLT-003`
**Decision:** A class matches as a whole token of the `class` attribute, never as a part
of one: `currency-rates__currency` does not match `currency-rates__currency-change`.
The client reads the first `div` with the class `currency-rates`. In
it, the first `currency-rates__row` of `currency-rates__row-head` gives the titles of
the columns: the text of each `currency-rates__item-title` in an item that is not
`--showonmobile`. Each `js-homepage__currency-item` gives one row. Its items that are
not `--showonmobile` match the titles by position. The first item holds the code in
`currency-rates__currency-name`, the official rate in the first
`currency-rates__currency`, and the unit in `currency-rates__currency-description`.
Each other item holds 0 or 2 `currency-rates__currency`: the buy price, then the sell
price. The title `კომერციული` gives `Commercial`, `ინტერნეტ ბანკი` gives
`InternetBank`, and `გზავნილების სპეც კურსი` gives `Transfer`.
**Rationale:** The page holds the block twice, so the first copy is the one source of
truth. A match by title, not by a fixed position, finds a new or a moved column.
**Alternatives:** The second column by position. A new column before it would give the
rate of the internet bank with no error.

The client returns `ErrNoRateBlock` when the page has no block or no row, and
`ErrMalformedBlock` when the titles hold no `კომერციული`, when a row has no code, or
when an item holds a count of prices other than 0 and 2.

### `LIBRATE-DD-002`

**Realizes:** `LIBRATE-FR-001`
**Decision:** The adapter reads the commercial column only.
**Rationale:** `TBCRATE` 4 and `BOGRATE-FR-001` give the rate of the bank, not the rate of
an online channel. The internet bank rate has a limit of 100 000 lari for a person.
**Alternatives:** The best of the two. Two rates under one bank break the comparison.

### `LIBRATE-DD-003`

**Realizes:** `LIBRATE-FR-002`, `LIBRATE-INV-001`
**Decision:** The adapter reads the count of units from `Unit`: the integer before the
space. The text after the space must equal `Currency`. The count must be a power of 10
from 1 to 1 000 000. The adapter divides each price with `Shift(-k)`, where the count is
`10^k`, and drops the trailing zeros: `6.2220` per 1000 gives `0.006222`. Any other unit
fails the bank.
**Rationale:** A shift of the exponent is exact. A division by another count can give an
infinite decimal, so the adapter would round, against `LIBRATE-INV-001`. The page has
used only 1, 100, and 1000. The trailing zeros follow `BOGRATE-DD-005`.
**Alternatives:** `Div` with a fixed precision. It rounds with no error.

### `LIBRATE-DD-004`

**Realizes:** `LIBRATE-FR-003`
**Decision:** A requested code with no row, or with a commercial `Buy` or `Sell` that is
empty or zero, goes to `Missing`. A price that is not a number, or is below zero, fails
the bank.
**Rationale:** The same rule as `BOGRATE-DD-008`.
**Alternatives:** None.

### `LIBRATE-DD-005`

**Realizes:** `LIBRATE-NFR-001`
**Decision:** The vendor limit defaults to 1 call each second with a burst of 1.
**Rationale:** One call serves every code of a request. A burst of 1 gives at most 60
calls in a minute. It is the limit of `ROMFUEL`, the other web page for people.
**Alternatives:** 0.1 calls each second. A second request with a code that the cache
lacks would wait 10 seconds, the timeout of the call.

## 6. Scenarios

`pkg/libertyrates/testdata/` holds `home-ka.html`, the real page, recorded on 2026-10-21.

### `LIBRATE-SC-001`

**Verifies:** `LIBRATE-FR-001`, `LIBRATE-FR-004`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the path is `/ka/`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp-libertyrates/`, and the request carries no key header.

### `LIBRATE-SC-002`

**Verifies:** `LIBRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `home-ka.html`.
**When** `Rates` runs.
**Then** the result holds 8 rows, one for each code. Two rows are:

| Currency | Unit       | Official | Commercial          | InternetBank        | Transfer            |
| -------- | ---------- | -------- | ------------------- | ------------------- | ------------------- |
| `USD`    | `1 USD`    | `2.5978` | `2.5470`, `2.6470`  | `2.5870`, `2.6070`  | `2.5960`, `2.6000`  |
| `AMD`    | `1000 AMD` | `7.1737` | `6.2220`, `7.5730`  | empty               | empty               |

### `LIBRATE-SC-003`

**Verifies:** `LIBRATE-FR-001`
**Layer:** unit

| Vendor answer                                          | Client result                                   |
| ------------------------------------------------------ | ----------------------------------------------- |
| A page with no `currency-rates` block                  | `ErrNoRateBlock`                                |
| `home-ka.html` with the title `კომერციული` renamed      | `ErrMalformedBlock`                             |
| `home-ka.html` with one commercial item of 1 price      | `ErrMalformedBlock`                             |
| 503 with a body of 2 KiB                               | `*APIError` with 503 and the first 1 KiB        |

### `LIBRATE-SC-004`

**Verifies:** `LIBRATE-FR-001`
**Layer:** live

**Given** the real home page of Liberty.
**When** `Rates` runs.
**Then** a row of `USD` has a commercial buy price and sell price above zero.

### `LIBRATE-SC-005`

**Verifies:** `LIBRATE-FR-002`, `LIBRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `home-ka.html`.
**When** the adapter asks for `AMD`, `EUR`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `AMD`    | `0.006222`     | `0.007573`      |
| `EUR`    | `2.855`        | `2.971`         |
| `USD`    | `2.547`        | `2.647`         |

The format is `StringFixed(-Exponent())`, as the tool writes it.

### `LIBRATE-SC-006`

**Verifies:** `LIBRATE-FR-003`
**Layer:** unit

**Given** a page with a row of `USD` and the row of `CHF` of the table.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| Commercial prices of `CHF` |
| -------------------------- |
| `0.0000`, `3.2280`         |
| none: the item is empty    |

### `LIBRATE-SC-007`

**Verifies:** `LIBRATE-FR-002`, `LIBRATE-INV-001`
**Layer:** unit

| Row of `USD`                                    | Adapter error                        |
| ----------------------------------------------- | ------------------------------------ |
| `Unit` is `3 USD`                               | Wraps `kernel.ErrSourceUnavailable`  |
| `Unit` is `1 EUR`                               | Wraps `kernel.ErrSourceUnavailable`  |
| `Commercial.Buy` is `n/a`                       | Wraps `kernel.ErrSourceUnavailable`  |
| `Commercial.Buy` is `-2.5470`                   | Wraps `kernel.ErrSourceUnavailable`  |
| The vendor answers 503                          | Wraps `kernel.ErrSourceUnavailable`  |

### `LIBRATE-SC-008`

**Verifies:** `LIBRATE-FR-004`, `LIBRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Liberty.
**When** the configuration loads.
**Then** `Liberty.Enabled` is true, `Liberty.BaseURL` is `https://www.libertybank.ge`,
`Liberty.HTTP.Rate` is 1, and `Liberty.HTTP.Burst` is 1.

### `LIBRATE-SC-009`

**Verifies:** `LIBRATE-FR-001`, `LIBRATE-INV-001`
**Layer:** integration

**Given** Liberty on, every other bank off, the vendor serves `home-ka.html`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `AMD`.
**Then** the bank `liberty` has `source` `Liberty Bank`, `as_of` is the instant of the
clock, and the rates of `AMD` and `USD` are those of `LIBRATE-SC-005`. The vendor
counts 1 call.

### `LIBRATE-SC-010`

**Verifies:** `LIBRATE-FR-004`
**Layer:** integration

| Liberty | Other banks | Tools of the server                                        | Log                                                        |
| ------- | ----------- | ---------------------------------------------------------- | ---------------------------------------------------------- |
| on      | off         | The official tools and `rates_get_commercial_rates`. Its description lists `liberty`. | No record about Liberty |
| off     | `bog` on    | A call with the bank `liberty` reads `banks: "liberty" is not a bank of this server. Use one of: bog.` | One info record that names `GEOMCP_RATES_LIBERTY_ENABLED` |

## 7. Build plan

| #   | Step                                                                          | Realizes                              | Done |
| --- | ----------------------------------------------------------------------------- | ------------------------------------- | ---- |
| 1   | `pkg/libertyrates` module, `testdata/home-ka.html`, README, example. `go.work`. | `LIBRATE-FR-001`                      | [ ]  |
| 2   | `adapter/liberty`.                                                            | `LIBRATE-FR-001` to `LIBRATE-FR-003`, `LIBRATE-INV-001` | [ ]  |
| 3   | `config.go`: `LibertyConfig`, `LibertyHTTPConfig`. `configs/.env.example`.    | `LIBRATE-FR-004`, `LIBRATE-NFR-001`   | [ ]  |
| 4   | `module.go`: the source of Liberty, and the info record.                      | `LIBRATE-FR-001`, `LIBRATE-FR-004`    | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
