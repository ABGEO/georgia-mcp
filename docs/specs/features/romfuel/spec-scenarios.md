---
id: ROMFUEL
title: The fuel prices of Rompetrol, the scenarios
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [TST]
requirements: features/romfuel/requirements.md
---

# Specification: The fuel prices of Rompetrol, the scenarios

`pkg/rompetrol/testdata/` holds the pages that the tests serve:

| File                 | Content                                                                            |
| -------------------- | ---------------------------------------------------------------------------------- |
| `home.html`          | The real answer of `GET /`, recorded on 2026-10-19. 10 rows: 5 products, each with a self-service row. |
| `home-edge.html`     | A copy of `home.html`. efix Super has the name `efix სუპერი პლიუსი`. The self-service row of Euro Diesel is absent. The self-service price of efix Euro Premium is `0`. |
| `home-no-table.html` | A copy of `home.html` with the anchor `pricelist` and the table removed.            |

A test of the adapter for a broken table serves a short page: the anchor, and a table
with the rows that the scenario gives. An integration test serves the pages and the
answer of Wissol from `httptest`, and calls the tool through the in-memory MCP transport.

## The client

### `ROMFUEL-SC-001`

**Verifies:** `ROMFUEL-FR-001`
**Layer:** unit

**Given** an `httptest` server that records the request and serves `home.html`.
**When** `Prices` runs.
**Then** the path is `/`, `Accept` is `text/html`, the User-Agent starts with
`georgia-mcp/`, and the list holds 10 rows. The first row is
`{"efix სუპერი", "4.65"}`. The sixth row is
`{"- ევრო რეგულარი (თვითმომსახურების სადგურებზე)", "3.92"}`.

### `ROMFUEL-SC-002`

**Verifies:** `WISFUEL-FR-014`
**Layer:** unit

**Given** the vendor answers 503 with the body `error`.
**When** `Prices` runs.
**Then** the error is an `*APIError` with 503. `errors.Is(err, ErrUnexpectedStatus)` holds.

### `ROMFUEL-SC-003`

**Verifies:** `ROMFUEL-FR-007`
**Layer:** unit

| Page                                         | Result                    |
| -------------------------------------------- | ------------------------- |
| `home-no-table.html`                         | `ErrNoPriceTable`         |
| The anchor, and no table after it            | `ErrNoPriceTable`         |
| The anchor, and a row with 3 `td` cells      | `ErrMalformedTable`       |
| The anchor, and a table with a head only     | An empty list, no error   |

A fuzz test of the parse finds no page that panics.

### `ROMFUEL-SC-004`

**Verifies:** `ROMFUEL-FR-001`
**Layer:** live

**Given** the real Rompetrol.
**When** `Prices` runs.
**Then** the list holds one row or more, and each row has a product and a price.

## The domain

### `ROMFUEL-SC-005`

**Verifies:** `ROMFUEL-FR-004`
**Layer:** unit

| Date         | Result                                    |
| ------------ | ----------------------------------------- |
| `2026-10-08` | The price, with the date `2026-10-08`     |
| zero         | The price. `EffectiveDate.IsZero` holds.  |

## The adapter

### `ROMFUEL-SC-006`

**Verifies:** `ROMFUEL-FR-001`, `ROMFUEL-FR-002`, `ROMFUEL-FR-003`, `ROMFUEL-FR-004`, `ROMFUEL-FR-006`
**Layer:** unit

**Given** the vendor serves `home.html`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 5 prices, and `ReadAt` is the instant of the clock. Every price
has the zero date.

| Code                | Grade     | Standard | Self-service | English name      |
| ------------------- | --------- | -------- | ------------ | ----------------- |
| `efix_super`        | `super`   | `4.65`   | `4.55`       | efix Super        |
| `efix_euro_premium` | `premium` | `4.32`   | `4.22`       | efix Euro Premium |
| `efix_euro_regular` | `regular` | `4.02`   | `3.92`       | efix Euro Regular |
| `efix_euro_diesel`  | `diesel`  | `4.99`   | `4.89`       | efix Euro Diesel  |
| `euro_diesel`       | `diesel`  | `4.92`   | `4.82`       | Euro Diesel       |

### `ROMFUEL-SC-007`

**Verifies:** `ROMFUEL-FR-002`, `ROMFUEL-FR-003`, `ROMFUEL-FR-006`
**Layer:** unit

**Given** the vendor serves `home-edge.html`.
**When** the adapter reads the prices two times.
**Then** `efix სუპერი პლიუსი` has the grade `other`, the code `efix სუპერი პლიუსი`, no
English name, and the self-service price `4.55`. The log holds one warning with its
name. Euro Diesel has no self-service price. efix Euro Premium has no self-service price.

### `ROMFUEL-SC-008`

**Verifies:** `ROMFUEL-FR-001`, `ROMFUEL-FR-002`, `ROMFUEL-FR-007`
**Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Rows of the table, or answer                                   | Reason                              |
| -------------------------------------------------------------- | ----------------------------------- |
| `- efix სუპერი (...)` `4.55` first                             | A self-service row with no product. |
| `efix სუპერი` `4.65`, then two self-service rows               | A second self-service row.          |
| `efix სუპერი` `4,65`                                           | A price that is not a number.       |
| `efix სუპერი` `0`                                              | A standard price of zero.           |
| `efix სუპერი` `4.65`, then a self-service row with `-1`        | A negative self-service price.      |
| A table with a head only                                       | An empty list.                      |
| `home-no-table.html`                                           | No price table.                     |
| The status 500                                                 | An error of the client.             |

### `ROMFUEL-SC-009`

**Verifies:** `ROMFUEL-FR-005`
**Layer:** unit

**Given** the adapter of Rompetrol and the adapter of Wissol.
**When** `Company` runs on each.
**Then** Rompetrol is `{rompetrol, Rompetrol, false}`. Wissol is `{wissol, Wissol, true}`.

## The tool

### `ROMFUEL-SC-010`

**Verifies:** `ROMFUEL-FR-004`, `WISFUEL-INV-003`
**Layer:** unit

**Given** a price with the zero date, and a price with the date `2026-10-08`.
**When** the tool converts each to its output and encodes it as JSON.
**Then** the first JSON has no key `effective_date`. The second has
`"effective_date":"2026-10-08"`.

### `ROMFUEL-SC-011`

**Verifies:** `ROMFUEL-FR-001`, `ROMFUEL-FR-004`, `ROMFUEL-FR-006`, `WISFUEL-INV-001`
**Layer:** integration

**Given** Rompetrol serves `home.html`, Wissol serves `fuelprice.json`, and a fixed clock.
**When** a client calls `fuel_get_prices` with no input, then with `language` `ka`.
**Then** `companies` holds `rompetrol`, then `wissol`. `rompetrol` has `source`
`Rompetrol`, `as_of` at the instant of the clock, and 5 prices with no `effective_date`.
efix Super has `price` `4.65` and `self_service_price` `4.55`. Each price of `wissol` has
an `effective_date`. In the first answer the product of `efix_euro_regular` is
`efix Euro Regular`. In the second it is `efix ევრო რეგულარი`.

### `ROMFUEL-SC-012`

**Verifies:** `ROMFUEL-FR-005`
**Layer:** integration

**Given** Rompetrol and Wissol are on.
**When** a client lists the tools.
**Then** the description of `fuel_get_prices` holds
`rompetrol (Rompetrol, publishes no effective date), wissol (Wissol)`. The schema of the
output does not list `effective_date` as required.

### `ROMFUEL-SC-013`

**Verifies:** `ROMFUEL-FR-007`, `WISFUEL-FR-014`
**Layer:** integration

**Given** Rompetrol serves `home-no-table.html`, and Wissol serves `fuelprice.json`.
**When** a client calls `fuel_get_prices` with no input.
**Then** the result is not an error. `unavailable` holds `rompetrol`, and `companies`
holds `wissol` only.

## The cache and the configuration

### `ROMFUEL-SC-014`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002`
**Layer:** unit

**Given** a cache over the adapter of Rompetrol, a vendor that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the vendor counts 1 call after the first 10, and 2 calls after the last.

### `ROMFUEL-SC-015`

**Verifies:** `ROMFUEL-FR-008`
**Layer:** unit

**Given** no variable of fuel.
**When** the configuration loads.
**Then** Rompetrol is on, with the base address `https://www.rompetrol.ge`, a timeout of
10 seconds, the rate 1, and the burst 1.

### `ROMFUEL-SC-016`

**Verifies:** `ROMFUEL-FR-008`
**Layer:** integration

**Given** `GEOMCP_FUEL_ROMPETROL_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company
`rompetrol`.
**Then** the log holds `Rompetrol is off` with `variable=GEOMCP_FUEL_ROMPETROL_ENABLED`.
The tool result is an error: `rompetrol` is not a fuel company of this server.
