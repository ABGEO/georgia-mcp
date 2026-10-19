---
id: SOCFUEL
title: The fuel prices of SOCAR, the scenarios
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [TST]
requirements: features/socfuel/requirements.md
---

# Specification: The fuel prices of SOCAR, the scenarios

`pkg/socar/testdata/` holds the answers that the tests serve:

| File                       | Content                                                                    |
| -------------------------- | -------------------------------------------------------------------------- |
| `current-prices.json`      | The real answer of `GET /integration/info/current-prices`, recorded on 2026-10-19. 7 products. |
| `current-prices-edge.json` | A copy of `current-prices.json` with a product `A92` at `0`, and CNG under the unknown code `HYDRO`. |
| `error.json`               | The real error body of the backend, recorded on 2026-10-19, served with 400. |

An integration test serves the answers of the three companies from `httptest`, and calls
the tool through the in-memory MCP transport.

## The client

### `SOCFUEL-SC-001`

**Verifies:** `SOCFUEL-FR-001`
**Layer:** unit

**Given** an `httptest` server that records the request and serves `current-prices.json`.
**When** `CurrentPrices` runs.
**Then** the path is `/integration/info/current-prices`, `Accept` is `application/json`,
the User-Agent starts with `georgia-mcp-socar/`, `Status` is `1`, and the list holds 7
products. `EURREG` has the price `4.02`. `LPG` has the English name `"LPG "` and the price
`2.09`. `DIESEL` has the price `4.9`.

### `SOCFUEL-SC-002`

**Verifies:** `SOCFUEL-FR-007`
**Layer:** unit

**Given** the vendor answers 400 with `error.json`.
**When** `CurrentPrices` runs.
**Then** the error is an `*APIError` with 400 and the first 1 KiB of the body.
`errors.Is(err, ErrUnexpectedStatus)` holds.

### `SOCFUEL-SC-003`

**Verifies:** `SOCFUEL-FR-001`
**Layer:** live

**Given** the real SOCAR.
**When** `CurrentPrices` runs.
**Then** `Status` is `1`, the list holds one product or more, and each product has a code
and a price.

## The domain

### `SOCFUEL-SC-004`

**Verifies:** `SOCFUEL-FR-004`
**Layer:** unit

| Input      | Result                        |
| ---------- | ----------------------------- |
| `lpg`      | `GradeLPG`, unit `litre`      |
| `LPG`      | `ErrInvalidGrade`             |
| `kerosene` | `ErrInvalidGrade`             |

`Grades` returns `regular, premium, super, super_100, diesel, cng, lpg, other`.

## The adapter

### `SOCFUEL-SC-005`

**Verifies:** `SOCFUEL-FR-001`, `SOCFUEL-FR-003`, `SOCFUEL-FR-005`, `SOCFUEL-FR-006`
**Layer:** unit

**Given** the vendor serves `current-prices.json`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 7 prices, and `ReadAt` is the instant of the clock. No price has a
self-service price. Every price has the zero date.

| Code      | Grade     | Standard | English name       |
| --------- | --------- | -------- | ------------------ |
| `SUPER`   | `super`   | `4.62`   | Nano Super         |
| `PREMIUM` | `premium` | `4.31`   | Nano Premium       |
| `EURREG`  | `regular` | `4.02`   | Nano Euro Regular  |
| `DIESEL`  | `diesel`  | `4.9`    | Euro 5 Diesel      |
| `EURODSL` | `diesel`  | `4.96`   | Nano Euro 5 Diesel |
| `LPG`     | `lpg`     | `2.09`   | LPG                |
| `CNG`     | `cng`     | `1.6`    | CNG                |

### `SOCFUEL-SC-006`

**Verifies:** `SOCFUEL-FR-002`, `SOCFUEL-FR-003`
**Layer:** unit

**Given** the vendor serves `current-prices-edge.json`.
**When** the adapter reads the prices two times.
**Then** the list holds 7 prices, and no `A92`. `HYDRO` has the grade `other`. The log holds
one warning with `HYDRO`.

### `SOCFUEL-SC-007`

**Verifies:** `SOCFUEL-FR-001`, `SOCFUEL-FR-002`, `SOCFUEL-FR-007`
**Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Answer                                                  | Reason                          |
| ------------------------------------------------------- | ------------------------------- |
| 200, `Status` `"0"`                                     | Not a success.                  |
| 200, `{}`                                               | No status.                      |
| 200, a product with `"FuelUnitPrice":-1`                | A negative price.               |
| 200, a product with `"FuelUnitPrice":"abc"`             | A price that is not a number.   |
| 200, every product at `0`                               | No product left.                |
| 200, `Status` `"1"` and no product                      | No product left.                |
| 400, `error.json`                                       | An error of the client.         |

### `SOCFUEL-SC-008`

**Verifies:** `SOCFUEL-FR-005`
**Layer:** unit

**Given** the adapter of SOCAR.
**When** `Company` runs.
**Then** it is `{socar, SOCAR, false}`.

## The tool

### `SOCFUEL-SC-009`

**Verifies:** `SOCFUEL-FR-004`, `WISFUEL-FR-009`
**Layer:** integration

**Given** SOCAR serves `current-prices.json`, Rompetrol serves `home.html`, and Wissol
serves `fuelprice.json`.
**When** a client calls `fuel_get_prices` with the grade `lpg`.
**Then** `socar` holds one price: LPG, grade `lpg`, unit `litre`, price `2.09`, and no
`effective_date`. `rompetrol` and `wissol` hold no price, and name `lpg` in `missing`.

### `SOCFUEL-SC-010`

**Verifies:** `SOCFUEL-FR-004`, `SOCFUEL-FR-005`
**Layer:** integration

**Given** the three companies are on.
**When** a client lists the tools.
**Then** the description of `fuel_get_prices` holds `lpg (litre)` and
`rompetrol (Rompetrol, publishes no effective date), socar (SOCAR, publishes no effective date), wissol (Wissol)`.
The schema of `grades` names `lpg`.

### `SOCFUEL-SC-011`

**Verifies:** `SOCFUEL-FR-004`
**Layer:** unit

**Given** the tool with a fake use case.
**When** a client calls `fuel_get_prices` with the grade `LPG`.
**Then** the result is a tool error:
`grades: "LPG" is not a grade. Use one of: regular, premium, super, super_100, diesel, cng, lpg, other.`

### `SOCFUEL-SC-012`

**Verifies:** `SOCFUEL-FR-007`, `WISFUEL-FR-014`
**Layer:** integration

**Given** SOCAR answers 400 with `error.json`, and Wissol serves `fuelprice.json`.
**When** a client calls `fuel_get_prices` with no input.
**Then** the result is not an error. `unavailable` holds `socar`, and `companies` holds
`wissol`.

## The cache and the configuration

### `SOCFUEL-SC-013`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002`
**Layer:** unit

**Given** a cache over the adapter of SOCAR, a transport that counts its calls, and a
fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `SOCFUEL-SC-014`

**Verifies:** `SOCFUEL-FR-008`
**Layer:** unit

**Given** no variable of fuel.
**When** the configuration loads.
**Then** SOCAR is on, with the base address `https://sgp.ge/sgp-backend/api`, a timeout of
10 seconds, the rate 1, and the burst 1.

### `SOCFUEL-SC-015`

**Verifies:** `SOCFUEL-FR-008`
**Layer:** integration

**Given** `GEOMCP_FUEL_SOCAR_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `socar`.
**Then** the log holds `SOCAR is off` with `variable=GEOMCP_FUEL_SOCAR_ENABLED`. The tool
result is an error: `socar` is not a fuel company of this server.
