---
id: GULFUEL
title: The fuel prices of Gulf
type: spec
status: approved
created: 2026-10-22
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-22
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/gulfuel/requirements.md
---

# Specification: The fuel prices of Gulf

## 1. Summary

This feature adds the vendor client module `pkg/gulf`, which reads the table of the
fuel prices page of Gulf with a browser handshake, and a source of `fuel`. The adapter
takes the newest row, and the columns of the tier "Gulf" only. `internal/pkg/httpclient`
gains a builder over a given transport. The tool, the use case, the domain, and the
cache do not change.

## 2. Coverage

| Requirement       | Where this specification realizes it                    |
| ----------------- | ------------------------------------------------------- |
| `GULFUEL-FR-001`  | 4.1 `adapter/gulf`. `GULFUEL-DD-003`, `GULFUEL-DD-004`, `GULFUEL-DD-005` |
| `GULFUEL-FR-002`  | 4.1 `adapter/gulf`, `Company`. `GULFUEL-DD-004`         |
| `GULFUEL-FR-003`  | `GULFUEL-DD-005`                                        |
| `GULFUEL-FR-004`  | `GULFUEL-DD-006`                                        |
| `GULFUEL-FR-005`  | `GULFUEL-DD-006`                                        |
| `GULFUEL-FR-006`  | 4.1 `pkg/gulf`, `adapter/gulf`. `GULFUEL-DD-001`        |
| `GULFUEL-FR-007`  | 4.1 `adapter/gulf`. `GULFUEL-DD-005`                    |
| `GULFUEL-FR-008`  | 4.1 `pkg/gulf`, `httpclient`. `GULFUEL-DD-002`. `GULFUEL-SC-003`, `GULFUEL-SC-004`, `GULFUEL-SC-012` |
| `GULFUEL-FR-009`  | 4.3 configuration                                       |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `GULFUEL-SC-009` checks them for Gulf.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/gulf` is a module: `github.com/abgeo/georgia-mcp/pkg/gulf`, `go 1.26.0`.   |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, `golang.org/x/net/html` `v0.61.0`, and `github.com/refraction-networking/utls` `v1.8.2` only. |
| `CLT-003`              | The client returns the text of each cell, as the page shows it, and the class of each column. |
| `CLT-005`              | `Accept: text/html`. The User-Agent of this server. No `WithAPIKey`.            |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page behind a bot filter. `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/gulf` picks the tier and the row, names the grade, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Every price is a `decimal.Decimal`, and a string in the output.                 |
| `DOM-008`              | The adapter reads the date with `kernel.ParseDate`.                             |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_GULF_` and a default in its tag.      |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Vendor client  | `pkg/gulf/`                           | `Client`, `New`, `Option`, `NewTransport`, `PriceTable`, `Column`, `APIError`, `ErrUnexpectedStatus`, `ErrNoPriceTable`, `ErrMalformedTable`. |
| Shared         | `internal/pkg/httpclient/` (change)   | `NewWithTransport`. `New` calls it with `http.DefaultTransport`. |
| Driven adapter | `F/adapter/gulf/`                     | `Adapter`, `NewAdapter`.                                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Gulf`, `gulfSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/gulf`.                                          |
| Linter         | `.golangci.yaml` (change)             | The depguard list `browser-handshake-clients`: `pkg/gulf` imports `x/net/html` and `utls`. `CLT-002`. |
| Website        | `website/src/content/contexts.yaml` (change) | Gulf, გალფი, in the list of `fuel`.                 |

**`pkg/gulf`.** The shape of `pkg/lukoil`, with one method and one transport.

```go
const DefaultBaseURL = "https://gulf.ge"

var (
    ErrUnexpectedStatus = errors.New("gulf: unexpected status")
    ErrNoPriceTable     = errors.New("gulf: no price table")
    ErrMalformedTable   = errors.New("gulf: malformed price table")
)

func New(opts ...Option) *Client // the options of vendorhttp
func NewTransport() *http.Transport
func (c *Client) PriceArchive(ctx context.Context) (PriceTable, error)

// PriceTable is the table of the fuel prices page, as the page shows it.
type PriceTable struct {
    Columns []Column   // [{"Date", "first"}, {"G-Force Super", "g_col"}, ..., {"Euro Diesel", "gp_col"}]
    Rows    [][]string // ["2026-10-19", "4.69", "4.39", "4.09", "5.05", "4.99", "4.59", ...]
}

// Column is one cell of the head of the table.
type Column struct {
    Name  string // the label of the cell
    Class string // the class attribute of the cell, as the page writes it
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`PriceArchive` sends `GET /en/fuel_prices` with `Accept: text/html`. Any status other
than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is
1 MiB: the page has 111 KB, and holds 100 rows at most. The default User-Agent is
`georgia-mcp/<version> (+https://github.com/abgeo/georgia-mcp)`. The default HTTP
client has a timeout of 10 seconds and the transport of `NewTransport`.
`GULFUEL-DD-001` gives the parse. `GULFUEL-DD-002` gives the transport.

**`httpclient` (change).** `NewWithTransport(cfg Config, base http.RoundTripper) *http.Client`
returns the client of `New` over `base`. `New(cfg)` returns `NewWithTransport(cfg, http.DefaultTransport)`.

**`adapter/gulf`.** `Company` returns
`{Code: "gulf", Name: "Gulf", PublishesEffectiveDate: true}`. `Prices` sends one call,
picks the newest row (`GULFUEL-DD-004`), and converts each column of the tier "Gulf":

| Cell of the page                         | Domain value                                      |
| ---------------------------------------- | ------------------------------------------------- |
| `Columns[i].Name`, class `g_col`         | The English name. The code, the grade, and the Georgian name of `GULFUEL-DD-006`. |
| The price of the newest row              | `Standard`. `GULFUEL-DD-003`, `GULFUEL-DD-005`.   |
| The cell of the column of class `first`  | `EffectiveDate`. `GULFUEL-DD-004`.                |

`SelfService` is none. `ReadAt` is the instant of the answer. Each failure returns an
error that wraps `kernel.ErrSourceUnavailable`: an error of the client, no column of
class `first`, no column of class `g_col`, no row, a date that does not parse, a price
that is not a number or is negative, and no product left.

**`module.go` (change).** `newModule` builds the source of Gulf after the last source
that it builds today, when `cfg.Gulf.Enabled` is true. Else it logs at info level
`Gulf is off` with the attribute `variable=GEOMCP_FUEL_GULF_ENABLED`. `gulfSource`
passes `httpclient.NewWithTransport(cfg.HTTP, gulfclient.NewTransport())` to
`vendorhttp.WithHTTPClient`.

### 4.2 Domain model

No change. `WISFUEL` 4.2 holds the model.

### 4.3 Declarations

**MCP tool.** No change. The description lists `gulf (Gulf)` through `companyList`,
with no note: Gulf publishes an effective date.

**Vendor endpoint.**

| Client     | Method and path       | Key  | Official                  | Realizes         |
| ---------- | --------------------- | ---- | ------------------------- | ---------------- |
| `pkg/gulf` | `GET /en/fuel_prices` | None | No, a web page for people | `GULFUEL-FR-001` |

A real answer, recorded on 2026-10-22, cut to the head and the newest row:

```
GET https://gulf.ge/en/fuel_prices
200 text/html; charset=UTF-8 (111148 bytes)
<table id="price_archive_tbl" cellpadding="0" cellspacing="0">
<thead><tr class="prices_header">
  <td class="first"><div class="th-inner"><a href="...?sort=date&dir=asc" class="sort-desc">
    <span class="normal">Date</span><span class="small"><span class="icon_calendar"></span></span></a></div></td>
  <td class="g_col"><div class="th-inner"><a href="...?sort=gf_super&dir=asc" class="sort-none">
    <span class="normal">G-Force Super</span><span class="small">GS</span></a></div></td>
  ... G-Force Premium (GP), G-Force Euro Regular (GER), G-Force Euro Diesel (GED), Euro Diesel (ED)
  <td class="gp_col"><div class="th-inner"> G-Force Super<div class="dummy"></div> </div></td>
  ... the same 4 others, class gp_col
</tr></thead>
<tbody>
  <tr class="prices_cnt odd">
    <td><span>2026-10-19</span></td>
    <td class="g_col"><span>4.69</span></td> ... <td class="g_col"><span>4.99</span></td>
    <td class="gp_col"><span>4.59</span></td> ... <td class="gp_col last"><span>4.89</span></td>
  </tr>
```

The same request with the handshake of Go returns 403 with `cf-mitigated: challenge`.

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method      | Key      | Time to live | Maximum stale | Entries | Realizes |
| ---------------- | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Gulf | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                    | Type     | Default           | Secret | Realizes          |
| --------------------------- | -------- | ----------------- | ------ | ----------------- |
| `GEOMCP_FUEL_GULF_ENABLED`  | bool     | `true`            | No     | `GULFUEL-FR-009`  |
| `GEOMCP_FUEL_GULF_BASE_URL` | URL      | `https://gulf.ge` | No     | `GULFUEL-FR-001`  |
| `GEOMCP_FUEL_GULF_TIMEOUT`  | duration | `10s`             | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_GULF_RATE`     | float    | `1`               | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_GULF_BURST`    | int      | `1`               | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Gulf vendors.Config` with `envPrefix:"GULF_"`. `fuel.Defaults` gives the defaults of this table (`PLATFORM-DD-014`).

## 5. Design decisions

### `GULFUEL-DD-001`

**Realizes:** `GULFUEL-FR-006`
**Decision:** The client parses the page with `html.Parse`, and reads the `table` with
the `id` `price_archive_tbl`. Each `td` or `th` of the first `tr` of its `thead` gives
a `Column`. `Name` is the text of the first descendant `span` with the class `normal`,
or the text of the cell when it has none. `Class` is the `class` attribute. Each `tr`
of its `tbody` gives a row: the text of each `td`. The text of a cell follows
`LUKFUEL-DD-001`. No table, or no head cell, returns `ErrNoPriceTable`. A row with a
count of cells other than the count of `Columns` returns `ErrMalformedTable`. A table
with no row returns no row and no error.
**Rationale:** The head uses `td`, not `th`. A cell of the tier "Gulf" holds a label and
an abbreviation: its whole text is "G-Force Super GS". The class tells the tiers apart,
as the tabs of the page do.
**Alternatives:** The file `/en/fuel_prices/download`. It is a spreadsheet, which needs
a parser of the format.

### `GULFUEL-DD-002`

**Realizes:** `GULFUEL-FR-008`
**Decision:** `NewTransport` returns an `http.Transport` with no proxy, whose
`DialTLSContext` dials TCP, wraps the connection in
`utls.UClient(conn, &utls.Config{ServerName: host}, utls.HelloChrome_131)`, sets the
ALPN extension to `http/1.1` only, and runs the handshake. An unexported
`newTransport(roots *x509.CertPool)` holds the code. `NewTransport` passes nil, the
roots of the system.
**Rationale:** The filter of Gulf reads the handshake only. `HelloChrome_131` passed on
2026-10-22. A fixed hello does not move with an update of `utls`, as `HelloChrome_Auto`
does. `net/http` speaks HTTP/2 only over a `*tls.Conn`. With `h2` in the ALPN, the site
selects HTTP/2 and the transport cannot read the answer.
**Alternatives:** `github.com/bogdanfinn/tls-client`. It also sends the HTTP/2 fingerprint,
needs 33 modules against 9, and replaces `net/http`, so `httptest` and `httpclient` no
longer apply. The filter does not demand it today.

### `GULFUEL-DD-003`

**Realizes:** `GULFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** The adapter parses each price with `decimal.NewFromString`. A text that it
refuses fails the answer.
**Rationale:** `ROMFUEL-DD-004`. `4.69` stays `4.69`.
**Alternatives:** None.

### `GULFUEL-DD-004`

**Realizes:** `GULFUEL-FR-001`, `GULFUEL-FR-002`, `DOM-008`
**Decision:** The adapter parses the cell of the column of class `first` with
`kernel.ParseDate`. It takes the row with the latest date. Of two rows with one date,
it takes the earlier in the document. The effective date is that date.
**Rationale:** The page lists the newest row first, and no two of its 1,077 rows share a
date. A date orders the rows with no trust in the order of the page. `LUKFUEL-DD-003`.
**Alternatives:** The first row of the document. A change of the sort order of the page
gives the oldest prices with no warning.

### `GULFUEL-DD-005`

**Realizes:** `GULFUEL-FR-001`, `GULFUEL-FR-003`, `GULFUEL-FR-007`
**Decision:** The adapter reads only a column whose class holds the word `g_col`. It
ignores each column of class `gp_col`. It leaves out a product with the price zero. A
negative price fails the answer. A row with no product left fails the answer.
**Rationale:** `g_col` is the tier "Gulf", `gp_col` the tier "Gulf+" (requirements,
question 1). `LUKFUEL-DD-004` for the zero.
**Alternatives:** The order of the columns: 5 then 5. A new product moves a price of
Gulf+ to the tier of Gulf with no error.

### `GULFUEL-DD-006`

**Realizes:** `GULFUEL-FR-004`, `GULFUEL-FR-005`
**Decision:** The adapter maps the English name of the head to a code, a grade, and a
Georgian name with a fixed table. `ProductName.English` is the name of the page. A name
that is not in the table gets the grade `other`, an empty Georgian name, and as its
code the name in lower case with `_` for each run of characters that are not a letter
or a digit. The adapter logs a warning with the name, one time for each name.

| Name                 | Code                   | Grade     | Georgian name         |
| -------------------- | ---------------------- | --------- | --------------------- |
| G-Force Super        | `g_force_super`        | `super`   | G-Force სუპერი         |
| G-Force Premium      | `g_force_premium`      | `premium` | G-Force პრემიუმი       |
| G-Force Euro Regular | `g_force_euro_regular` | `regular` | G-Force ევრო რეგულარი   |
| G-Force Euro Diesel  | `g_force_euro_diesel`  | `diesel`  | G-Force ევრო დიზელი     |
| Euro Diesel          | `euro_diesel`          | `diesel`  | ევრო დიზელი             |

**Rationale:** `PORFUEL-DD-004`. The page has no product code. The
Georgian names are the names of the page `/ge/fuel_prices`. `ProductName.In` gives the
English name in `ka` when the Georgian name is empty.
**Alternatives:** A second call to `/ge/fuel_prices`. It doubles the calls to a site
behind a filter for 5 fixed names.

## 6. Scenarios

`pkg/gulf/testdata/` holds `fuel-prices.html`, the real page `/en/` of 2026-10-22 with
100 rows, and `fuel-prices-edge.html`: the oldest row first, a column `G-Force Hydro` of
class `g_col`, G-Force Super at `0.00` in the newest row, and after it a second row of
the same date with Euro Diesel at `4.95`. An integration test serves
the pages from `httptest` and calls the tool through the in-memory MCP transport.

### `GULFUEL-SC-001`

**Verifies:** `GULFUEL-FR-001` **Layer:** unit

**Given** an `httptest` server that records the request and serves `fuel-prices.html`.
**When** `PriceArchive` runs.
**Then** the path is `/en/fuel_prices`, `Accept` is `text/html`, the User-Agent starts
with `georgia-mcp/`, `Columns` holds 11 columns, and `Rows` holds 100 rows.
`Columns[1]` is `{"G-Force Super", "g_col"}`, and `Columns[6]` is `{"G-Force Super", "gp_col"}`.
The first row is `["2026-10-19", "4.69", "4.39", "4.09", "5.05", "4.99", "4.59", "4.29", "3.99", "4.95", "4.89"]`.

### `GULFUEL-SC-002`

**Verifies:** `GULFUEL-FR-006` **Layer:** unit

| Page of the vendor                          | Result                                      |
| ------------------------------------------- | ------------------------------------------- |
| 200, no `table` with the `id`               | `ErrNoPriceTable`.                          |
| 200, a row of 10 cells under a head of 11   | `ErrMalformedTable`.                        |
| 200, a head and no row                      | No row, no error.                           |
| 403                                         | An `*APIError` with 403. `ErrUnexpectedStatus`. |

### `GULFUEL-SC-003`

**Verifies:** `GULFUEL-FR-008` **Layer:** unit

**Given** an `httptest` TLS server that records the hello of the client.
**When** a client over `newTransport` with the roots of the server sends a GET.
**Then** the answer is 200. The ALPN of the hello is `["http/1.1"]`, and its cipher
suites hold a GREASE value, which the handshake of Go never sends.

### `GULFUEL-SC-004`

**Verifies:** `GULFUEL-FR-008` **Layer:** live

**Given** the real Gulf, and the default client of `New`. **When** `PriceArchive` runs.
**Then** no error. `Columns` holds a column of class `first` and one of class `g_col`
or more. The first cell of each row parses as a date.

### `GULFUEL-SC-005`

**Verifies:** `GULFUEL-FR-001` to `GULFUEL-FR-005`, `GULFUEL-FR-007` **Layer:** unit

**Given** the vendor serves `fuel-prices.html`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 5 prices with the effective date `2026-10-19` and no self-service
price. `ReadAt` is the instant of the clock. `Company` is `{gulf, Gulf, true}`.

| Code                   | Grade     | Standard | Name in `ka`          |
| ---------------------- | --------- | -------- | --------------------- |
| `g_force_super`        | `super`   | `4.69`   | G-Force სუპერი         |
| `g_force_premium`      | `premium` | `4.39`   | G-Force პრემიუმი       |
| `g_force_euro_regular` | `regular` | `4.09`   | G-Force ევრო რეგულარი   |
| `g_force_euro_diesel`  | `diesel`  | `5.05`   | G-Force ევრო დიზელი     |
| `euro_diesel`          | `diesel`  | `4.99`   | ევრო დიზელი             |

### `GULFUEL-SC-006`

**Verifies:** `GULFUEL-FR-001` to `GULFUEL-FR-004` **Layer:** unit

**Given** the vendor serves `fuel-prices-edge.html`.
**When** the adapter reads the prices two times.
**Then** the effective date is `2026-10-19`, the latest date of the document. Euro Diesel
has `4.99`, from the earlier of the two rows of that date. The answer
holds no `g_force_super`. `g_force_hydro` has the grade `other` and the name
"G-Force Hydro" in `ka`. The log holds one warning with `G-Force Hydro`.

### `GULFUEL-SC-007`

**Verifies:** `GULFUEL-FR-001`, `GULFUEL-FR-003`, `GULFUEL-FR-006` **Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Newest row or head of the page                 | Reason                          |
| ---------------------------------------------- | ------------------------------- |
| Every price of `g_col` `0.00`                  | No product left.                |
| Euro Diesel `-1.00`                            | A negative price.               |
| Euro Diesel `4,99`                             | A price that is not a number.   |
| The date `19.10.2026`                          | A date that does not parse.     |
| A head with no column of class `first`         | No date column.                 |
| A head with no column of class `g_col`         | No column of the tier.          |
| A head and no row                              | No row.                         |
| 403                                            | An error of the client.         |

### `GULFUEL-SC-008`

**Verifies:** `GULFUEL-FR-002`, `WISFUEL-INV-003` **Layer:** integration

**Given** Gulf serves `fuel-prices.html`, and every other company answers 500.
**When** a client lists the tools, then calls `fuel_get_prices` with the grade `diesel`.
**Then** the description holds `gulf (Gulf)` with no note on the date. `gulf` holds
G-Force Euro Diesel at `5.05` and Euro Diesel at `4.99`, each with `effective_date` `2026-10-19`.

### `GULFUEL-SC-009`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002` **Layer:** unit

**Given** a cache over the adapter of Gulf, a transport that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `GULFUEL-SC-010`

**Verifies:** `GULFUEL-FR-009` **Layer:** unit

**Given** no variable of fuel. **When** the configuration loads.
**Then** Gulf is on, with the base address `https://gulf.ge`, a timeout of 10 seconds,
the rate 1, and the burst 1.

### `GULFUEL-SC-011`

**Verifies:** `GULFUEL-FR-009` **Layer:** integration

**Given** `GEOMCP_FUEL_GULF_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `gulf`.
**Then** the log holds `Gulf is off` with `variable=GEOMCP_FUEL_GULF_ENABLED`. The tool
result is an error: `gulf` is not a fuel company of this server.

### `GULFUEL-SC-012`

**Verifies:** `GULFUEL-FR-008`, `WISFUEL-NFR-002` **Layer:** unit

**Given** the client of `httpclient.NewWithTransport` over a transport that counts its
calls, with the rate 1 and the burst 1.
**When** a request runs, then a second request with a deadline of 100 ms.
**Then** the second request fails with `kernel.ErrSourceUnavailable`, and the transport
counts 1 call: the limit applies over the transport of Gulf.

## 7. Build plan

| #   | Step                                                                   | Realizes                            | Done |
| --- | ---------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `pkg/gulf` module, transport, recorded pages, README, example. `go.work`. | `GULFUEL-FR-001`, `GULFUEL-FR-006`, `GULFUEL-FR-008` | [x]  |
| 2   | `httpclient.NewWithTransport`.                                         | `GULFUEL-FR-008`                    | [x]  |
| 3   | `adapter/gulf`.                                                        | `GULFUEL-FR-001` to `GULFUEL-FR-007` | [x]  |
| 4   | `module.go`, `config.go`, `configs/.env.example`.                      | `GULFUEL-FR-009`                    | [x]  |
| 5   | Move the terms "Gulf" and "browser handshake" to the glossary. Add Gulf to the website. | `DOM-002`           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

Step 5 adds a fuel company to the website. `WEBSITE-SC-040` and `WEBSITE-SC-046`
change with it: 16 vendors, and "+5 more" on the card of `fuel`.

## Retired identifiers

This file has no retired identifier.
