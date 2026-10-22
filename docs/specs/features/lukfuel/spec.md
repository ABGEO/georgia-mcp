---
id: LUKFUEL
title: The fuel prices of Lukoil
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/lukfuel/requirements.md
---

# Specification: The fuel prices of Lukoil

## 1. Summary

This feature adds the vendor client module `pkg/lukoil`, which reads the table of the
price history page of Lukoil, and a source of `fuel`. The adapter takes the row of the
newest change: its prices, and its day as the effective date. The tool, the use case,
the domain, and the cache do not change.

## 2. Coverage

| Requirement       | Where this specification realizes it                 |
| ----------------- | ---------------------------------------------------- |
| `LUKFUEL-FR-001`  | 4.1 `adapter/lukoil`. `LUKFUEL-DD-002`, `LUKFUEL-DD-003` |
| `LUKFUEL-FR-002`  | 4.1 `adapter/lukoil`, `Company`. `LUKFUEL-DD-003`    |
| `LUKFUEL-FR-003`  | `LUKFUEL-DD-004`                                     |
| `LUKFUEL-FR-004`  | `LUKFUEL-DD-005`                                     |
| `LUKFUEL-FR-005`  | `LUKFUEL-DD-005`                                     |
| `LUKFUEL-FR-006`  | 4.1 `pkg/lukoil`, `adapter/lukoil`. `LUKFUEL-DD-001` |
| `LUKFUEL-FR-007`  | 4.3 configuration                                    |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `LUKFUEL-SC-008` checks them for Lukoil.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/lukoil` is a module: `github.com/abgeo/georgia-mcp/pkg/lukoil`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `golang.org/x/net/html` `v0.61.0` only. `ADR-0003`.  |
| `CLT-003`              | The client returns the text of each cell, as the page shows it.                 |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                           |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page. `v0`.     |
| `DOM-005`, `DOM-011`   | `adapter/lukoil` picks the row, names the grade, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Every price is a `decimal.Decimal`, and a string in the output.                 |
| `DOM-007`              | The adapter reads the instant in `kernel.TbilisiZone()`.                        |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_LUKOIL_` and a default in its tag.    |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Vendor client  | `pkg/lukoil/`                         | `Client`, `New`, `Option`, `PriceTable`, `APIError`, `ErrUnexpectedStatus`, `ErrNoPriceTable`, `ErrMalformedTable`. |
| Driven adapter | `F/adapter/lukoil/`                   | `Adapter`, `NewAdapter`.                                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Lukoil`, `LukoilConfig`, `LukoilHTTPConfig`, `lukoilSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/lukoil`.                                        |
| Website        | `website/src/content/contexts.yaml` (change) | Lukoil, ლუკოილი, in the list of `fuel`.           |

**`pkg/lukoil`.** The shape of `pkg/rompetrol`, with one method.

```go
const DefaultBaseURL = "https://www.lukoil.ge"

var (
    ErrUnexpectedStatus = errors.New("lukoil: unexpected status")
    ErrNoPriceTable     = errors.New("lukoil: no price table")
    ErrMalformedTable   = errors.New("lukoil: malformed price table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) PriceHistory(ctx context.Context) (PriceTable, error)

// PriceTable is the table of the price history page, as the page shows it.
type PriceTable struct {
    Header []string   // ["#", "Super Ecto 100", ..., "Euro Diesel", "შეცვლის თარიღი"]
    Rows   [][]string // ["106", "0.00", "4.60", "4.25", "3.95", "4.97", "2026-10-05 10:01:46"]
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`PriceHistory` sends `GET /prices-history` with `Accept: text/html`. Any status other
than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is
2 MiB: the page has 99 KB, and a change adds about 1 KB. The default User-Agent is
`georgia-mcp-lukoil/<version> (+https://github.com/abgeo/georgia-mcp)`.
`LUKFUEL-DD-001` gives the parse.

**`adapter/lukoil`.** `Company` returns
`{Code: "lukoil", Name: "Lukoil", PublishesEffectiveDate: true}`. `Prices` sends one
call, picks the newest row (`LUKFUEL-DD-003`), and converts each product column:

| Cell of the page                  | Domain value                                             |
| --------------------------------- | -------------------------------------------------------- |
| `Header[i]`, from 1 to the last but one | The name. The code, the grade of `LUKFUEL-DD-005`. |
| The price of the newest row       | `Standard`. `LUKFUEL-DD-002`, `LUKFUEL-DD-004`.          |
| The last cell of the newest row   | `EffectiveDate`. `LUKFUEL-DD-003`.                       |

`SelfService` is none. `ReadAt` is the instant of the answer. Each failure returns an
error that wraps `kernel.ErrSourceUnavailable`: an error of the client, a header of
fewer than 3 cells, no row, an instant that does not parse, a price that is not a
number or is negative, and no product left.

**`module.go` (change).** `newModule` builds the source of Lukoil after the last source
that it builds today, when `cfg.Lukoil.Enabled` is true. Else it logs at info level
`Lukoil is off` with the attribute `variable=GEOMCP_FUEL_LUKOIL_ENABLED`.

### 4.2 Domain model

No change. `WISFUEL` 4.2 holds the model.

### 4.3 Declarations

**MCP tool.** No change. The description lists `lukoil (Lukoil)` through `companyList`,
with no note: Lukoil publishes an effective date.

**Vendor endpoint.**

| Client       | Method and path       | Key  | Official                  | Realizes         |
| ------------ | --------------------- | ---- | ------------------------- | ---------------- |
| `pkg/lukoil` | `GET /prices-history` | None | No, a web page for people | `LUKFUEL-FR-001` |

A real answer, recorded on 2026-10-21, cut to the head and the newest row:

```
GET https://www.lukoil.ge/prices-history
200 text/html; charset=UTF-8 (98940 bytes, Cache-Control: no-cache, private)
<table class="min-w-full divide-y divide-gray-300">
  <thead class="bg-gray-50"><tr>
    <th scope="col" ...>#</th>
    <th scope="col" ...>Super Ecto 100</th>
    <th scope="col" ...>Super Ecto</th>
    <th scope="col" ...>Premium Avangard</th>
    <th scope="col" ...>Euro Regular</th>
    <th scope="col" ...>Euro Diesel</th>
    <th scope="col" ...>შეცვლის თარიღი</th>
  </tr></thead>
  <tbody ...>
    <tr>
      <td ...>106</td><td ...>0.00</td><td ...>4.60</td><td ...>4.25</td>
      <td ...>3.95</td><td ...>4.97</td><td ...>2026-10-05 10:01:46</td>
    </tr>
```

The answer sets the cookies `XSRF-TOKEN` and `laravel_session`. The client sends none.

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method        | Key      | Time to live | Maximum stale | Entries | Realizes |
| ------------------ | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Lukoil | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                      | Type     | Default                 | Secret | Realizes          |
| ----------------------------- | -------- | ----------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_LUKOIL_ENABLED`  | bool     | `true`                  | No     | `LUKFUEL-FR-007`  |
| `GEOMCP_FUEL_LUKOIL_BASE_URL` | URL      | `https://www.lukoil.ge` | No     | `LUKFUEL-FR-001`  |
| `GEOMCP_FUEL_LUKOIL_TIMEOUT`  | duration | `10s`                   | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_LUKOIL_RATE`     | float    | `1`                     | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_LUKOIL_BURST`    | int      | `1`                     | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Lukoil LukoilConfig` with `envPrefix:"LUKOIL_"`, with the shape of
`WissolConfig` and the defaults of this table.

## 5. Design decisions

### `LUKFUEL-DD-001`

**Realizes:** `LUKFUEL-FR-006`, `ADR-0003`
**Decision:** The client parses the page with `html.Parse`, and reads the first `table`
of the document. `Header` is the text of each `th` of its `thead`. Each `tr` of its
`tbody` gives a row: the text of each `td`. The text of a cell is the text of its
descendants, with each run of white space as one space, and no space at the ends. No
table, or no `th`, returns `ErrNoPriceTable`. A row with a count of cells other than the
count of `Header` returns `ErrMalformedTable`. A table with no row returns no row and
no error.
**Rationale:** The page has one table, with no `id` and no class of its own. The head
names each column, so a new product moves no price to a wrong name.
**Alternatives:** The cards of the home page. They carry no date (requirements, question 1).

### `LUKFUEL-DD-002`

**Realizes:** `LUKFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** The adapter parses each price with `decimal.NewFromString`. A text that it
refuses fails the answer.
**Rationale:** `ROMFUEL-DD-004`. `4.60` stays `4.60`.
**Alternatives:** A parse that accepts a comma. No price of the 106 rows has one.

### `LUKFUEL-DD-003`

**Realizes:** `LUKFUEL-FR-001`, `LUKFUEL-FR-002`, `DOM-007`
**Decision:** The adapter parses the last cell of each row with
`time.ParseInLocation("2006-01-02 15:04:05", s, kernel.TbilisiZone())`. It takes the row
with the latest instant. Of two rows with one instant, it takes the later in the
document. The effective date is the civil date of that instant.
**Rationale:** The page lists the oldest row first today. An instant orders the rows
with no trust in the order of the page.
**Alternatives:** The last row of the document. A change of the sort order of the page
gives the oldest prices with no warning.

### `LUKFUEL-DD-004`

**Realizes:** `LUKFUEL-FR-003`
**Decision:** The adapter leaves out a product with the price zero. A negative price
fails the answer. A row with no product left fails the answer.
**Rationale:** `SOCFUEL-DD-004`. Super Ecto 100 has `0.00` in each of the 106 rows.
**Alternatives:** A zero fails the answer. One product that Lukoil does not sell hides
Lukoil.

### `LUKFUEL-DD-005`

**Realizes:** `LUKFUEL-FR-004`, `LUKFUEL-FR-005`
**Decision:** The adapter maps the name of the head to a code and a grade with a fixed
table. The name is `ProductName.English`, and `Georgian` stays empty. A name that is not
in the table gets the grade `other`, and the name in lower case with `_` for each space
as its code. The adapter logs a warning with the name, one time for each name.

| Name             | Code               | Grade       |
| ---------------- | ------------------ | ----------- |
| Super Ecto 100   | `super_ecto_100`   | `super_100` |
| Super Ecto       | `super_ecto`       | `super`     |
| Premium Avangard | `premium_avangard` | `premium`   |
| Euro Regular     | `euro_regular`     | `regular`   |
| Euro Diesel      | `euro_diesel`      | `diesel`    |

**Rationale:** `ROMFUEL-DD-005`. The page has no product code. `ProductName.In` gives the
English name in `ka`.
**Alternatives:** A Georgian name that the server holds. Lukoil uses none on its own site.

## 6. Scenarios

`pkg/lukoil/testdata/` holds `prices-history.html`, the real page of 2026-10-21 with 106
rows, and `prices-history-edge.html`: the rows in the order newest first, a second row
at `2026-10-05 10:01:46` with Euro Diesel at `4.99`, and a column `Hydro`. An integration
test serves the pages from `httptest` and calls the tool through the in-memory MCP
transport.

### `LUKFUEL-SC-001`

**Verifies:** `LUKFUEL-FR-001` **Layer:** unit

**Given** an `httptest` server that records the request and serves `prices-history.html`.
**When** `PriceHistory` runs.
**Then** the path is `/prices-history`, `Accept` is `text/html`, the User-Agent starts
with `georgia-mcp-lukoil/`, `Header` holds 7 cells, and `Rows` holds 106 rows. The last
row is `["106", "0.00", "4.60", "4.25", "3.95", "4.97", "2026-10-05 10:01:46"]`.

### `LUKFUEL-SC-002`

**Verifies:** `LUKFUEL-FR-006` **Layer:** unit

| Page of the vendor                        | Result                                      |
| ----------------------------------------- | ------------------------------------------- |
| 200, no `table`                           | `ErrNoPriceTable`.                          |
| 200, a row of 6 cells under a head of 7   | `ErrMalformedTable`.                        |
| 200, a head and no row                    | No row, no error.                           |
| 500                                       | An `*APIError` with 500. `ErrUnexpectedStatus`. |

### `LUKFUEL-SC-003`

**Verifies:** `LUKFUEL-FR-001` **Layer:** live

**Given** the real Lukoil. **When** `PriceHistory` runs.
**Then** `Header` holds 3 cells or more, and the last cell of each row parses as an instant.

### `LUKFUEL-SC-004`

**Verifies:** `LUKFUEL-FR-001` to `LUKFUEL-FR-005` **Layer:** unit

**Given** the vendor serves `prices-history.html`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 4 prices with the effective date `2026-10-05` and no self-service
price. `ReadAt` is the instant of the clock. `Company` is `{lukoil, Lukoil, true}`.

| Code               | Grade     | Standard | Name             |
| ------------------ | --------- | -------- | ---------------- |
| `super_ecto`       | `super`   | `4.60`   | Super Ecto       |
| `premium_avangard` | `premium` | `4.25`   | Premium Avangard |
| `euro_regular`     | `regular` | `3.95`   | Euro Regular     |
| `euro_diesel`      | `diesel`  | `4.97`   | Euro Diesel      |

### `LUKFUEL-SC-005`

**Verifies:** `LUKFUEL-FR-001`, `LUKFUEL-FR-004` **Layer:** unit

**Given** the vendor serves `prices-history-edge.html`.
**When** the adapter reads the prices two times.
**Then** Euro Diesel has `4.99`, from the later of the two rows of one instant. `hydro`
has the grade `other`. The log holds one warning with `Hydro`.

### `LUKFUEL-SC-006`

**Verifies:** `LUKFUEL-FR-001`, `LUKFUEL-FR-003`, `LUKFUEL-FR-006` **Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Newest row of the page                         | Reason                         |
| ---------------------------------------------- | ------------------------------ |
| Every price `0.00`                             | No product left.               |
| Euro Diesel `-1.00`                            | A negative price.              |
| Euro Diesel `4,97`                             | A price that is not a number.  |
| The instant `05.10.2026`                       | An instant that does not parse. |
| A head and no row                              | No row.                        |
| 500                                            | An error of the client.        |

### `LUKFUEL-SC-007`

**Verifies:** `LUKFUEL-FR-002`, `WISFUEL-INV-003` **Layer:** integration

**Given** Lukoil serves `prices-history.html`, and every other company answers 500.
**When** a client lists the tools, then calls `fuel_get_prices` with the grade `diesel`.
**Then** the description holds `lukoil (Lukoil)` with no note on the date. `lukoil` holds
Euro Diesel at `4.97` with `effective_date` `2026-10-05`.

### `LUKFUEL-SC-008`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002` **Layer:** unit

**Given** a cache over the adapter of Lukoil, a transport that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `LUKFUEL-SC-009`

**Verifies:** `LUKFUEL-FR-007` **Layer:** unit

**Given** no variable of fuel. **When** the configuration loads.
**Then** Lukoil is on, with the base address `https://www.lukoil.ge`, a timeout of 10
seconds, the rate 1, and the burst 1.

### `LUKFUEL-SC-010`

**Verifies:** `LUKFUEL-FR-007` **Layer:** integration

**Given** `GEOMCP_FUEL_LUKOIL_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `lukoil`.
**Then** the log holds `Lukoil is off` with `variable=GEOMCP_FUEL_LUKOIL_ENABLED`. The
tool result is an error: `lukoil` is not a fuel company of this server.

## 7. Build plan

| #   | Step                                                                   | Realizes                            | Done |
| --- | ---------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `pkg/lukoil` module, recorded pages, README, example. `go.work`.       | `LUKFUEL-FR-001`, `LUKFUEL-FR-006`  | [x]  |
| 2   | `adapter/lukoil`.                                                      | `LUKFUEL-FR-001` to `LUKFUEL-FR-006` | [x]  |
| 3   | `module.go`, `config.go`, `configs/.env.example`.                      | `LUKFUEL-FR-007`                    | [x]  |
| 4   | Move the term "Lukoil" to the glossary. Add Lukoil to the website.     | `DOM-002`                           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
