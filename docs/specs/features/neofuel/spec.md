---
id: NEOFUEL
title: The price of compressed natural gas of Neogas
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-21
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/neofuel/requirements.md
---

# Specification: The price of compressed natural gas of Neogas

## 1. Summary

This feature adds the vendor client module `pkg/neogas`, which reads the gas price of
the calculator of the home page of Neogas, and a source of `fuel` with one product. The
tool, the use case, the domain, and the cache do not change.

## 2. Coverage

| Requirement       | Where this specification realizes it              |
| ----------------- | ------------------------------------------------- |
| `NEOFUEL-FR-001`  | 4.1 `adapter/neogas`. `NEOFUEL-DD-002`            |
| `NEOFUEL-FR-002`  | 4.1 `adapter/neogas`                              |
| `NEOFUEL-FR-003`  | 4.1 `pkg/neogas`, `adapter/neogas`. `NEOFUEL-DD-001`, `NEOFUEL-DD-003` |
| `NEOFUEL-FR-004`  | 4.1 `adapter/neogas`, `Company`                   |
| `NEOFUEL-FR-005`  | 4.3 configuration                                 |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `NEOFUEL-SC-007` checks them for Neogas.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/neogas` is a module: `github.com/abgeo/georgia-mcp/pkg/neogas`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `golang.org/x/net/html` `v0.61.0` only. `ADR-0003`.  |
| `CLT-003`              | The client returns the text of each gas price, as the page shows it.            |
| `CLT-005`              | `Accept: text/html`. No `WithAPIKey`.                                           |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the source is a web page. `v0`.     |
| `DOM-005`, `DOM-011`   | `adapter/neogas` checks the values, names the product, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | The price is a `decimal.Decimal`, and a string in the output.                   |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_NEOGAS_` and a default in its tag.    |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Vendor client  | `pkg/neogas/`                         | `Client`, `New`, `Option`, `APIError`, `ErrUnexpectedStatus`. |
| Driven adapter | `F/adapter/neogas/`                   | `Adapter`, `NewAdapter`.                                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Neogas`, `NeogasConfig`, `NeogasHTTPConfig`, `neogasSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/neogas`.                                        |
| Website        | `website/src/content/contexts.yaml` (change) | Neogas, ნეოგაზი, in the list of `fuel`.           |

**`pkg/neogas`.** The shape of `pkg/rompetrol`, with one method.

```go
const DefaultBaseURL = "https://neogas.ge"

var ErrUnexpectedStatus = errors.New("neogas: unexpected status")

func New(opts ...Option) *Client // WithBaseURL, WithHTTPClient, WithUserAgent

// GasPrices returns the text of each gas price of the calculator of the home page,
// in the order of the page: ["1.60", "1.60"]. A page with none returns an empty list.
func (c *Client) GasPrices(ctx context.Context) ([]string, error)

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`GasPrices` sends `GET /ka/` with `Accept: text/html`. Any status other than 200 returns
an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB: the page has
44 KB. The default User-Agent is
`georgia-mcp-neogas/<version> (+https://github.com/abgeo/georgia-mcp)`.
`NEOFUEL-DD-001` gives the parse.

**`adapter/neogas`.** `Company` returns
`{Code: "neogas", Name: "Neogas", PublishesEffectiveDate: false}`. `Prices` sends one
call, checks the list (`NEOFUEL-DD-003`), and returns one price:

| Field                | Value                                              |
| -------------------- | -------------------------------------------------- |
| `Code`               | `cng`                                              |
| `Name`               | `{Georgian: "ბუნებრივი აირი", English: "CNG"}`     |
| `Grade`              | `cng`                                              |
| `Standard`           | The value of the list. `NEOFUEL-DD-002`.           |
| `SelfService`        | None.                                              |
| `EffectiveDate`      | The zero date.                                     |

`ReadAt` is the instant of the answer. Each failure returns an error that wraps
`kernel.ErrSourceUnavailable`.

**`module.go` (change).** `newModule` builds the source of Neogas after the last source
that it builds today, when `cfg.Neogas.Enabled` is true. Else it logs at info level
`Neogas is off` with the attribute `variable=GEOMCP_FUEL_NEOGAS_ENABLED`.

### 4.2 Domain model

No change. `GradeCNG.Unit` is `UnitCubicMetre`.

### 4.3 Declarations

**MCP tool.** No change. The description lists `neogas (Neogas, publishes no effective
date)` through `companyList`.

**Vendor endpoint.**

| Client       | Method and path | Key  | Official                  | Realizes         |
| ------------ | --------------- | ---- | ------------------------- | ---------------- |
| `pkg/neogas` | `GET /ka/`      | None | No, a web page for people | `NEOFUEL-FR-001` |

A real answer, recorded on 2026-10-21, cut to one pane of the calculator:

```
GET https://neogas.ge/ka/
200 text/html (43919 bytes, Cache-Control: no-store, no-cache, no Last-Modified)
<div class="options gas">
  <span>გაზის ფასი</span>
  <div class="price">
    <span class="gas_price">1.60</span>
    <i class="lari lari-normal"></i>
  </div>
```

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method        | Key      | Time to live | Maximum stale | Entries | Realizes |
| ------------------ | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Neogas | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                      | Type     | Default             | Secret | Realizes          |
| ----------------------------- | -------- | ------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_NEOGAS_ENABLED`  | bool     | `true`              | No     | `NEOFUEL-FR-005`  |
| `GEOMCP_FUEL_NEOGAS_BASE_URL` | URL      | `https://neogas.ge` | No     | `NEOFUEL-FR-001`  |
| `GEOMCP_FUEL_NEOGAS_TIMEOUT`  | duration | `10s`               | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_NEOGAS_RATE`     | float    | `1`                 | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_NEOGAS_BURST`    | int      | `1`                 | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Neogas NeogasConfig` with `envPrefix:"NEOGAS_"`, with the shape of
`WissolConfig` and the defaults of this table.

## 5. Design decisions

### `NEOFUEL-DD-001`

**Realizes:** `NEOFUEL-FR-001`, `NEOFUEL-FR-003`, `ADR-0003`
**Decision:** The client parses the page with `html.Parse`. It returns the text of each
element whose `class` holds the token `gas_price`, in the order of the document, with
the spaces at the ends removed.
**Rationale:** The class names the value, and the script of the calculator reads it.
The page has no `id` near the price. The text of a PHP notice sits outside the element.
**Alternatives:** The text after "გაზის ფასი". The text changes with the language.

### `NEOFUEL-DD-002`

**Realizes:** `NEOFUEL-FR-001`, `WISFUEL-INV-001`
**Decision:** The adapter parses the value with `decimal.NewFromString`. A text that it
refuses, or a value not above zero, fails the answer.
**Rationale:** `ROMFUEL-DD-004`. `1.60` stays `1.60`.
**Alternatives:** A parse that accepts a comma. No value of 2026-10-21 has one.

### `NEOFUEL-DD-003`

**Realizes:** `NEOFUEL-FR-003`
**Decision:** The adapter fails on an empty list. It fails when two values of the list
differ as decimals. Else it takes the first value.
**Rationale:** The client mirrors the page. The meaning of zero, one, or two panes
belongs to the server (`CLT-003`, `DOM-005`). `1.6` and `1.60` are one price.
**Alternatives:** The first value with no check. A change of one pane gives a wrong
price with no warning.

## 6. Scenarios

`pkg/neogas/testdata/` holds `home.html`, the real page of 2026-10-21, and
`home-split.html`, a copy with `1.65` in the second pane. An integration test serves
the pages from `httptest` and calls the tool through the in-memory MCP transport.

### `NEOFUEL-SC-001`

**Verifies:** `NEOFUEL-FR-001` **Layer:** unit

**Given** an `httptest` server that records the request and serves `home.html`.
**When** `GasPrices` runs.
**Then** the path is `/ka/`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp-neogas/`, and the list is `["1.60", "1.60"]`.

### `NEOFUEL-SC-002`

**Verifies:** `NEOFUEL-FR-003` **Layer:** unit

| Page of the vendor                     | Result                                        |
| -------------------------------------- | --------------------------------------------- |
| 200, `<html><body></body></html>`      | An empty list, no error.                      |
| 503                                    | An `*APIError` with 503. `ErrUnexpectedStatus`. |

### `NEOFUEL-SC-003`

**Verifies:** `NEOFUEL-FR-001` **Layer:** live

**Given** the real Neogas. **When** `GasPrices` runs.
**Then** the list holds one value or more, and the first parses as a decimal.

### `NEOFUEL-SC-004`

**Verifies:** `NEOFUEL-FR-001`, `NEOFUEL-FR-002`, `NEOFUEL-FR-004` **Layer:** unit

**Given** the vendor serves `home.html`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds one price: code `cng`, grade `cng`, standard `1.60`, no
self-service price, the zero date, the names "ბუნებრივი აირი" and "CNG". `ReadAt` is
the instant of the clock. `Company` is `{neogas, Neogas, false}`.

### `NEOFUEL-SC-005`

**Verifies:** `NEOFUEL-FR-001`, `NEOFUEL-FR-003` **Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Page of the vendor                 | Reason                        |
| ---------------------------------- | ----------------------------- |
| 200, no gas price                  | No price.                     |
| 200, `home-split.html`             | Two values that differ.       |
| 200, one gas price `1,60`          | A price that is not a number. |
| 200, one gas price `0.00`          | A price not above zero.       |
| 503                                | An error of the client.       |

### `NEOFUEL-SC-006`

**Verifies:** `NEOFUEL-FR-001`, `NEOFUEL-FR-004`, `WISFUEL-FR-009` **Layer:** integration

**Given** Neogas serves `home.html`, and every other company answers 500.
**When** a client lists the tools, then calls `fuel_get_prices` with the grades `cng`
and `diesel`.
**Then** the description holds `neogas (Neogas, publishes no effective date)`. `neogas`
holds CNG, unit `cubic_metre`, price `1.60`, no `effective_date`, and names `diesel` in
`missing`.

### `NEOFUEL-SC-007`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002` **Layer:** unit

**Given** a cache over the adapter of Neogas, a transport that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `NEOFUEL-SC-008`

**Verifies:** `NEOFUEL-FR-005` **Layer:** unit

**Given** no variable of fuel. **When** the configuration loads.
**Then** Neogas is on, with the base address `https://neogas.ge`, a timeout of 10
seconds, the rate 1, and the burst 1.

### `NEOFUEL-SC-009`

**Verifies:** `NEOFUEL-FR-005` **Layer:** integration

**Given** `GEOMCP_FUEL_NEOGAS_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `neogas`.
**Then** the log holds `Neogas is off` with `variable=GEOMCP_FUEL_NEOGAS_ENABLED`. The
tool result is an error: `neogas` is not a fuel company of this server.

## 7. Build plan

| #   | Step                                                                   | Realizes                            | Done |
| --- | ---------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `pkg/neogas` module, recorded pages, README, example. `go.work`.       | `NEOFUEL-FR-001`, `NEOFUEL-FR-003`  | [ ]  |
| 2   | `adapter/neogas`.                                                      | `NEOFUEL-FR-001` to `NEOFUEL-FR-004` | [ ]  |
| 3   | `module.go`, `config.go`, `configs/.env.example`.                      | `NEOFUEL-FR-005`                    | [ ]  |
| 4   | Move the term "Neogas" to the glossary. Add Neogas to the website.     | `DOM-002`                           | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
