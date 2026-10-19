---
id: ROMFUEL
title: The fuel prices of Rompetrol
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/romfuel/requirements.md
---

# Specification: The fuel prices of Rompetrol

## 1. Summary

This feature adds the vendor client module `pkg/rompetrol`, which reads the price table
of the home page of Rompetrol, and a second source of `fuel`. The tool `fuel_get_prices`
stays. Its effective date becomes optional, and its description says which company
publishes none. `ADR-0003` lets a vendor client read a web page.
`spec-scenarios.md` holds the scenarios.

## 2. Coverage

| Requirement       | Where this specification realizes it                       |
| ----------------- | ---------------------------------------------------------- |
| `ROMFUEL-FR-001`  | 4.1 `adapter/rompetrol`. `ROMFUEL-DD-004`                  |
| `ROMFUEL-FR-002`  | 4.1 `adapter/rompetrol`. `ROMFUEL-DD-003`                  |
| `ROMFUEL-FR-003`  | `ROMFUEL-DD-005`                                           |
| `ROMFUEL-FR-004`  | 4.2 `ProductPrice.EffectiveDate`. 4.3 tool. `ROMFUEL-DD-006` |
| `ROMFUEL-FR-005`  | 4.2 `Company.PublishesEffectiveDate`. 4.3 tool             |
| `ROMFUEL-FR-006`  | `ROMFUEL-DD-005`                                           |
| `ROMFUEL-FR-007`  | 4.1 `pkg/rompetrol`, `adapter/rompetrol`. `ROMFUEL-DD-002` |
| `ROMFUEL-FR-008`  | 4.3 configuration                                          |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `ROMFUEL-SC-014` checks them for Rompetrol.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/rompetrol` is a module: `github.com/abgeo/georgia-mcp/pkg/rompetrol`, `go 1.26.0`. |
| `CLT-002` to `CLT-005` | Changed by `ADR-0003`. The client imports `golang.org/x/net/html` `v0.61.0`, and sets `Accept: text/html`. |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page, not an API. The module stays at `v0`. |
| `DOM-005`, `DOM-011`   | `adapter/rompetrol` pairs the rows, names the grade, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | Every price is a `decimal.Decimal`, and a string in the output.                 |
| `MCP-003`              | The struct `priceOutput` changes. 4.3 gives it.                                 |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_ROMPETROL_` and a default in its tag. |
| `LOG-002`              | The adapter logs through the `slog` logger of the module only.                  |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Guidelines     | `docs/specs/guidelines/clients.md`, `glossary.md` (change) | The rules and terms of `ADR-0003`.    |
| Vendor client  | `pkg/rompetrol/`                      | `Client`, `New`, `Option`, `PriceRow`, `APIError`, `ErrUnexpectedStatus`, `ErrNoPriceTable`, `ErrMalformedTable`. |
| Domain         | `F/domain/` (change)                  | `Company.PublishesEffectiveDate`. `NewProductPrice` accepts a zero date. 4.2. |
| Driven adapter | `F/adapter/rompetrol/`                | `Adapter`, `NewAdapter`.                                   |
| Driven adapter | `F/adapter/wissol/` (change)          | `Company` sets `PublishesEffectiveDate: true`.             |
| Tool           | `F/adapter/tool/` (change)            | `priceOutput.EffectiveDate` with `omitempty`. `companyList`. 4.3. |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Rompetrol`, `RompetrolConfig`, `RompetrolHTTPConfig`, `rompetrolSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/rompetrol`.                                     |

**`pkg/rompetrol`.** The shape of `pkg/wissol`, with one method.

```go
const DefaultBaseURL = "https://www.rompetrol.ge"

var (
    ErrUnexpectedStatus = errors.New("rompetrol: unexpected status")
    ErrNoPriceTable     = errors.New("rompetrol: no price table")
    ErrMalformedTable   = errors.New("rompetrol: malformed price table")
)

func New(opts ...Option) *Client // WithBaseURL, WithHTTPClient, WithUserAgent
func (c *Client) Prices(ctx context.Context) ([]PriceRow, error)

// PriceRow is one row of the body of the price table, as the page shows it.
type PriceRow struct {
    Product string // "efix ევრო რეგულარი", or "- ევრო რეგულარი (თვითმომსახურების სადგურებზე)"
    Price   string // "4.02"
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`Prices` sends `GET /` with `Accept: text/html`. Any status other than 200 returns an
`*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB. The default
User-Agent is `georgia-mcp-rompetrol/<version> (+https://github.com/abgeo/georgia-mcp)`.
`ROMFUEL-DD-001` gives the parse. A page with no table returns `ErrNoPriceTable`. A row
that has `td` cells, but not exactly 2, returns `ErrMalformedTable`. An empty table
returns an empty list and no error: the client does not interpret (`CLT-003`).

**`adapter/rompetrol`.** `Company` returns
`{Code: "rompetrol", Name: "Rompetrol", PublishesEffectiveDate: false}`. `Prices` sends
one call and converts the rows:

| Row of the page                         | Domain value                                                  |
| --------------------------------------- | ------------------------------------------------------------- |
| `Product` with no `- ` at the start     | A new product. Its code, name, and grade come from `ROMFUEL-DD-005`. |
| `Product` with `- ` at the start        | The self-service price of the product directly before it. `ROMFUEL-DD-003`. |
| `Price`                                 | `decimal.NewFromString`. `ROMFUEL-DD-004`.                    |
| No date                                 | `EffectiveDate` is the zero `kernel.Date`.                    |

`ReadAt` is the instant of the answer. Each failure returns an error that wraps
`kernel.ErrSourceUnavailable`: an error of the client, an empty list, a price that is
not a number, a standard price that is not more than zero, a negative self-service
price, and a self-service row with no product before it.

**`module.go`.** `newModule` builds the source of Rompetrol after the source of Wissol,
when `cfg.Rompetrol.Enabled` is true. Else it logs at info level `Rompetrol is off` with
the attribute `variable=GEOMCP_FUEL_ROMPETROL_ENABLED`.

### 4.2 Domain model

Only the changes. `WISFUEL` 4.2 holds the rest.

```go
// domain
type Company struct {
    Code                   CompanyCode
    Name                   string // "Rompetrol"
    PublishesEffectiveDate bool   // false: every price of the company has the zero date
}

type ProductPrice struct {
    Code          string
    Name          ProductName
    Grade         Grade
    Standard      decimal.Decimal
    SelfService   decimal.NullDecimal
    EffectiveDate kernel.Date // the zero date when the company publishes none
}
```

`NewProductPrice` keeps its signature. It accepts the zero date. Its other checks stay.
The use case and the cache do not change.

### 4.3 Declarations

**MCP tool.** `fuel_get_prices` keeps its name, its input, and its errors (`WISFUEL`
4.3, 4.5). One field of the output changes:

| Tool              | Input                             | Output         | Errors             | Realizes |
| ----------------- | --------------------------------- | -------------- | ------------------ | -------- |
| `fuel_get_prices` | `grades`, `companies`, `language` | `pricesOutput` | `WISFUEL` 4.5      | `ROMFUEL-FR-004`, `ROMFUEL-FR-005`, `ROMFUEL-FR-006` |

```go
type priceOutput struct {
    Product          string `json:"product"                      jsonschema:"The name of the product at the company."`
    Grade            string `json:"grade"                        jsonschema:"The grade of the product, the same for every company."`
    Unit             string `json:"unit"                         jsonschema:"The unit of each price: litre, or cubic_metre for cng."`
    Price            string `json:"price"                        jsonschema:"The standard price in GEL of one unit, as a decimal string."`
    SelfServicePrice string `json:"self_service_price,omitempty" jsonschema:"The price in GEL of one unit at a self-service station of the company, as a decimal string. Absent when the company publishes none."`
    EffectiveDate    string `json:"effective_date,omitempty"     jsonschema:"The date from which the company applies the prices, YYYY-MM-DD. Absent when the company publishes no date: as_of then gives the age of the price."`
}
```

`companyList` writes a company that publishes no date with a note:
`rompetrol (Rompetrol, publishes no effective date), wissol (Wissol)`. The description
gives the list, so a client reads it before a call.

**Vendor endpoint.**

| Client          | Method and path | Key  | Official                    | Realizes         |
| --------------- | --------------- | ---- | --------------------------- | ---------------- |
| `pkg/rompetrol` | `GET /`         | None | No, a web page for people   | `ROMFUEL-FR-001` |

A real answer, recorded on 2026-10-19, cut to the table and two products:

```
GET https://www.rompetrol.ge/
200 text/html; charset=UTF-8 (123774 bytes, no Last-Modified, no ETag)
<a id="pricelist"></a>
<section class="page-simple-content ...">
  ...<h2>საწვავის ფასი</h2>
  <table class="table table-orange table-striped">
    <thead><tr><th>პროდუქტი</th><th>ლარი / ლ</th></tr></thead>
    <tbody>
      <tr><td>efix სუპერი</td><td>4.65</td></tr>
      <tr><td>- efix სუპერი (თვითმომსახურების სადგურებზე)</td><td>4.55</td></tr>
      <tr><td>efix ევრო რეგულარი</td><td>4.02</td></tr>
      <tr><td>- ევრო რეგულარი (თვითმომსახურების სადგურებზე)</td><td>3.92</td></tr>
    </tbody>
  </table>
```

Each cell of the real page holds the text between new lines and tabs.

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method            | Key      | Time to live | Maximum stale | Entries | Realizes |
| ---------------------- | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Rompetrol  | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                            | Type     | Default                    | Secret | Realizes          |
| ----------------------------------- | -------- | -------------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_ROMPETROL_ENABLED`     | bool     | `true`                     | No     | `ROMFUEL-FR-008`  |
| `GEOMCP_FUEL_ROMPETROL_BASE_URL`    | URL      | `https://www.rompetrol.ge` | No     | `ROMFUEL-FR-001`  |
| `GEOMCP_FUEL_ROMPETROL_TIMEOUT`     | duration | `10s`                      | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_ROMPETROL_RATE`        | float    | `1`                        | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_ROMPETROL_BURST`       | int      | `1`                        | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Rompetrol RompetrolConfig` with `envPrefix:"ROMPETROL_"`.
`RompetrolConfig` and `RompetrolHTTPConfig` have the shape of `WissolConfig` and
`WissolHTTPConfig`, with the defaults of this table.

## 5. Design decisions

### `ROMFUEL-DD-001`

**Realizes:** `ROMFUEL-FR-001`, `ROMFUEL-FR-007`, `ADR-0003`
**Decision:** The client parses the page with `html.Parse`. It finds the first element
with `id="pricelist"`, then the first `table` after it in the order of the document. It
reads each `tr` of the table with `td` cells. The text of a cell is the text of its
descendants, with each run of spaces, tabs, and new lines as one space, and no space at
the ends.
**Rationale:** The anchor `#pricelist` is the target of two links of the page. It is
the most stable mark of the table. The class `table-orange` is a style, and can change
with a new design. The rows of the head have `th` cells, not `td`.
**Alternatives:** The first table with the class `table-orange`. The page has one table
on 2026-10-19, but a style is less stable than an anchor that a link names.

### `ROMFUEL-DD-002`

**Realizes:** `ROMFUEL-FR-007`
**Decision:** The client returns `ErrNoPriceTable` when it finds no anchor or no table
after it. The adapter treats an empty list as a failure.
**Rationale:** The client mirrors the page: an empty table is a fact of the page. The
meaning of an empty table belongs to the server (`CLT-003`, `DOM-005`).
**Alternatives:** The client fails on an empty table. It would interpret the data.

### `ROMFUEL-DD-003`

**Realizes:** `ROMFUEL-FR-002`
**Decision:** A row whose product starts with `- ` gives the self-service price of the
product row directly before it. The adapter ignores the rest of its name. A self-service
row with no product row before it, or after a second self-service row, fails the answer.
A self-service price of zero gives no price. A negative one fails the answer.
**Rationale:** On 2026-10-19 the name of 1 self-service row of 5 differs from the name of
its product. The order of the rows is the only fact that pairs them. Zero and negative
follow `WISFUEL-DD-005`.
**Alternatives:** A match on the name with the words "efix" and the suffix removed. It
fails on the next small change of a name, with no warning.

### `ROMFUEL-DD-004`

**Realizes:** `ROMFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** The adapter parses `Price` with `decimal.NewFromString`. A text that it
refuses fails the answer.
**Rationale:** The digits stay as Rompetrol writes them. A comma or a currency sign tells
that the page changed.
**Alternatives:** A parse that accepts a comma. No price of 2026-10-19 has one.

### `ROMFUEL-DD-005`

**Realizes:** `ROMFUEL-FR-003`, `ROMFUEL-FR-006`
**Decision:** The adapter maps the Georgian name of a product row to a code, a grade, and
an English name with a fixed table. A name that is not in the table gets its Georgian name
as the code, the grade `other`, and no English name. The adapter logs a warning with the
name, one time for each name.

| Georgian name      | Code                | Grade     | English name      |
| ------------------ | ------------------- | --------- | ----------------- |
| efix სუპერი         | `efix_super`        | `super`   | efix Super        |
| efix ევრო პრემიუმი   | `efix_euro_premium` | `premium` | efix Euro Premium |
| efix ევრო რეგულარი   | `efix_euro_regular` | `regular` | efix Euro Regular |
| efix ევრო დიზელი     | `efix_euro_diesel`  | `diesel`  | efix Euro Diesel  |
| ევრო დიზელი         | `euro_diesel`       | `diesel`  | Euro Diesel       |

**Rationale:** The page has no product code. The owner chose the exact name
(requirements, question 5). `ProductName.In` gives the Georgian name when the English one
is empty. The code is not in the output. It orders two products of one grade.
**Alternatives:** A grade from the words of the name. It can give a wrong grade to a new
product.

### `ROMFUEL-DD-006`

**Realizes:** `ROMFUEL-FR-004`, `ROMFUEL-FR-005`
**Decision:** The zero `kernel.Date` means "no effective date". `Company` gains
`PublishesEffectiveDate`. The tool leaves out `effective_date` for a zero date.
**Rationale:** The owner chose it. `kernel.Date` has `IsZero`. A new type in the kernel
changes `rates` too, for one field.
**Alternatives:** `kernel.NullDate`, a date with a flag. More explicit, more change.

### `ROMFUEL-DD-007`

**Realizes:** `ROMFUEL-FR-001`
**Decision:** The client reads `GET /`, the Georgian home page.
**Rationale:** The page of the personal customer has the same 123774 bytes. The page of
the fuels holds no price. The Georgian page names every product in Georgian, the key of
`ROMFUEL-DD-005`.
**Alternatives:** `/en/personal`. Its names mix two languages.

## 6. Scenarios

`spec-scenarios.md` holds `ROMFUEL-SC-001` to `ROMFUEL-SC-016`.

## 7. Build plan

| #   | Step                                                                          | Realizes                            | Done |
| --- | ----------------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | Accept `ADR-0003`. Change `CLT-002` to `CLT-005` and the glossary.            | `ROMFUEL-FR-001`                    | [x]  |
| 2   | `pkg/rompetrol` module, recorded pages, README, example. `go.work`.           | `ROMFUEL-FR-001`, `ROMFUEL-FR-007`  | [x]  |
| 3   | `domain`: `Company.PublishesEffectiveDate`, the zero date in `NewProductPrice`. The sentence of `WISFUEL` 4.2 on the zero date. | `ROMFUEL-FR-004` | [x]  |
| 4   | `adapter/rompetrol`. `adapter/wissol`: `PublishesEffectiveDate`.              | `ROMFUEL-FR-001` to `ROMFUEL-FR-003`, `ROMFUEL-FR-006`, `ROMFUEL-FR-007` | [x]  |
| 5   | `adapter/tool`: `effective_date` with `omitempty`, `companyList`.             | `ROMFUEL-FR-004`, `ROMFUEL-FR-005`  | [x]  |
| 6   | `module.go`, `config.go`, `configs/.env.example`.                             | `ROMFUEL-FR-008`                    | [x]  |
| 7   | Move the term "Rompetrol" to the glossary.                                    | `DOM-002`                           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
