---
id: WISFUEL
title: The fuel prices of Wissol, the scenarios
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [TST]
requirements: features/wisfuel/requirements.md
---

# Specification: The fuel prices of Wissol, the scenarios

`pkg/wissol/testdata/` holds the answers that the tests serve:

| File                 | Content                                                                          |
| -------------------- | -------------------------------------------------------------------------------- |
| `fuelprice.json`     | The real answer of `GET /FuelPrice`, recorded on 2026-10-19. 8 active products, two with no `priceSelfService`. |
| `fuelprice-edge.json` | A copy of `fuelprice.json` with one product not active, one with an unknown code `99`, and one with `"priceSelfService":0`. |

A unit test of `app` uses fake sources. An integration test serves the files from
`httptest` and calls the tool through the in-memory MCP transport.

## The client

### `WISFUEL-SC-001`

**Verifies:** `WISFUEL-FR-001`
**Layer:** unit

**Given** an `httptest` server that records the request and serves `fuelprice.json`.
**When** `FuelPrices` runs.
**Then** the path is `/FuelPrice`, `Accept` is `application/json`, the User-Agent starts
with `georgia-mcp-wissol/`, and the list holds 8 products. Euro Regular has the price
`4.08` and the self-service price `3.93`. `A1 SUPER 100` has an empty self-service price.

### `WISFUEL-SC-002`

**Verifies:** `WISFUEL-FR-014`
**Layer:** unit

**Given** the vendor answers 500 with the body `error`.
**When** `FuelPrices` runs.
**Then** the error is an `*APIError` with 500. `errors.Is(err, ErrUnexpectedStatus)` holds.

### `WISFUEL-SC-003`

**Verifies:** `WISFUEL-FR-001`
**Layer:** live

**Given** the real Wissol.
**When** `FuelPrices` runs.
**Then** the list holds one product or more, and each active product has a price.

## The domain

### `WISFUEL-SC-004`

**Verifies:** `WISFUEL-FR-007`
**Layer:** unit

| Input     | Result                     |
| --------- | -------------------------- |
| `wissol`  | The code `wissol`          |
| `Wissol`  | `ErrInvalidCompanyCode`    |
| `w`       | `ErrInvalidCompanyCode`    |

A fuzz test of `ParseCompanyCode` finds no input that panics.

### `WISFUEL-SC-005`

**Verifies:** `WISFUEL-FR-004`, `WISFUEL-FR-008`
**Layer:** unit

| Input        | Result                         |
| ------------ | ------------------------------ |
| `diesel`     | `GradeDiesel`, unit `litre`    |
| `cng`        | `GradeCNG`, unit `cubic_metre` |
| `super_100`  | `GradeSuper100`, unit `litre`  |
| `Diesel`     | `ErrInvalidGrade`              |
| `lpg`        | `ErrInvalidGrade`              |

A fuzz test of `ParseGrade` finds no input that panics.

### `WISFUEL-SC-006`

**Verifies:** `WISFUEL-FR-001`, `WISFUEL-FR-002`
**Layer:** unit

| Standard | Self-service      | Result                     |
| -------- | ----------------- | -------------------------- |
| `4.08`   | `3.93`            | The price                  |
| `4.79`   | not valid         | The price, no self-service |
| `0`      | not valid         | `ErrInvalidProductPrice`   |
| `4.08`   | valid, `0`        | `ErrInvalidProductPrice`   |

### `WISFUEL-SC-007`

**Verifies:** `WISFUEL-FR-011`
**Layer:** unit

| Georgian         | English         | Language | Result          |
| ---------------- | --------------- | -------- | --------------- |
| `ევრო რეგულარი`  | `Euro Regular`  | `ka`     | `ევრო რეგულარი` |
| `ევრო რეგულარი`  | `Euro Regular`  | `en`     | `Euro Regular`  |
| `ევრო რეგულარი`  | empty           | `en`     | `ევრო რეგულარი` |

## The use case

### `WISFUEL-SC-008`

**Verifies:** `WISFUEL-FR-006`, `WISFUEL-FR-008`
**Layer:** unit

**Given** two fake companies, `wissol` and `beta`, that sell `regular` and `diesel`.
**When** `Get` runs with no grade and no company, then with the grade `diesel` only.
**Then** the first answer holds both companies, sorted by code, with both grades. The
second holds the `diesel` products only.

### `WISFUEL-SC-009`

**Verifies:** `WISFUEL-FR-007`
**Layer:** unit

**Given** the fake company `wissol` only.
**When** `Get` runs with the company `gulf`.
**Then** the error is an `*UnknownCompanyError` with the code `gulf` and the known codes
`[wissol]`, and no company was asked.

### `WISFUEL-SC-010`

**Verifies:** `WISFUEL-FR-009`
**Layer:** unit

**Given** a fake company that sells `regular` and no `cng`.
**When** `Get` runs with `cng`, `regular`, and `regular`.
**Then** the company holds `regular` once, and its `Missing` is `[cng]`.

### `WISFUEL-SC-011`

**Verifies:** `WISFUEL-FR-014`
**Layer:** unit

| Companies                                  | Result                                                    |
| ------------------------------------------ | --------------------------------------------------------- |
| `wissol` fails, `beta` answers             | `beta` in `Companies`, `wissol` in `Unavailable`, no error |
| `wissol` fails, and it is the only request | An error that wraps `kernel.ErrSourceUnavailable`         |
| `wissol` waits 1 second, `beta` answers    | Both companies. The call takes less than 1.5 seconds.     |

## The adapter

### `WISFUEL-SC-012`

**Verifies:** `WISFUEL-FR-002`, `WISFUEL-FR-003`, `WISFUEL-FR-005`, `WISFUEL-FR-010`, `WISFUEL-INV-001`
**Layer:** unit

**Given** the vendor serves `fuelprice-edge.json`.
**When** the adapter reads the prices.
**Then** the product that is not active is absent. The product with the code `99` has
the grade `other`, and the log holds one warning with `99`. The product with the
self-service price `0` has no self-service price. Euro Regular has the grade `regular`,
the prices `4.08` and `3.93`, and the effective date `2026-10-08`. The three diesel
products have the grade `diesel`.

### `WISFUEL-SC-013`

**Verifies:** `WISFUEL-FR-014`
**Layer:** unit

| Vendor answer                                    | Adapter error                        |
| ------------------------------------------------ | ------------------------------------ |
| 500                                              | Wraps `kernel.ErrSourceUnavailable`  |
| 200 with a body that is not JSON                 | Wraps `kernel.ErrSourceUnavailable`  |
| 200 with a product with no `price`               | Wraps `kernel.ErrSourceUnavailable`  |
| 200 with `"priceSelfService":-1`                 | Wraps `kernel.ErrSourceUnavailable`  |
| 200 with `"date":"08.10.2026"`                   | Wraps `kernel.ErrSourceUnavailable`  |

## The cache

### `WISFUEL-SC-014`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-FR-015`
**Layer:** unit

| Read at | Company fails from | Asked at   | Result                                            |
| ------- | ------------------ | ---------- | ------------------------------------------------- |
| 0       | never              | 29 minutes | The list read at 0, not stale, and no new call    |
| 0       | never              | 31 minutes | A new list read at 31 minutes                     |
| 0       | 1 minute           | 11 hours   | The list read at 0, stale                         |
| 0       | 1 minute           | 13 hours   | An error that wraps `kernel.ErrSourceUnavailable` |

The test runs in `synctest`.

### `WISFUEL-SC-015`

**Verifies:** `WISFUEL-NFR-002`
**Layer:** unit

**Given** a counting fake company.
**When** 10 asks arrive at once, with different grades.
**Then** the company counts 1 call.

## The tool and the module

### `WISFUEL-SC-016`

**Verifies:** `WISFUEL-FR-004`, `WISFUEL-FR-012`, `WISFUEL-FR-013`, `WISFUEL-INV-002`
**Layer:** integration

**Given** the vendor serves `fuelprice.json`, and a fixed clock.
**When** a client calls `fuel_get_prices` with no input.
**Then** `meaning` holds the constant of section 4.3. The company `wissol` has `source`
`Wissol` and `as_of` at the instant of the clock. Each of the 8 prices has `grade`,
`unit`, `price`, and `effective_date`. `Wissol Gas` has the unit `cubic_metre`, Euro
Regular the unit `litre`. Euro Regular has `self_service_price` `3.93`.
`A1 SUPER 100` has no `self_service_price`.

### `WISFUEL-SC-017`

**Verifies:** `WISFUEL-FR-011`
**Layer:** integration

| Language | Name of the product `05` | Result                                   |
| -------- | ------------------------ | ---------------------------------------- |
| none     | `Euro Regular`           | The answer                               |
| `ka`     | `ევრო რეგულარი`          | The answer                               |
| `ru`     |                          | The tool error of section 4.5            |

### `WISFUEL-SC-018`

**Verifies:** `WISFUEL-FR-007`, `WISFUEL-FR-008`
**Layer:** unit

| Input                    | Tool error                                                                       |
| ------------------------ | -------------------------------------------------------------------------------- |
| grades `lpg`             | `grades: "lpg" is not a grade. Use one of: regular, premium, super, super_100, diesel, cng, other.` |
| companies `gulf`         | `companies: "gulf" is not a fuel company of this server. Use one of: wissol.`    |

### `WISFUEL-SC-019`

**Verifies:** `WISFUEL-FR-016`
**Layer:** integration

| `GEOMCP_FUEL_WISSOL_ENABLED` | Tools of the server             | Log                                                  |
| ---------------------------- | ------------------------------- | ---------------------------------------------------- |
| absent                       | `fuel_get_prices` is present    | No record about Wissol                               |
| `false`                      | `fuel_get_prices` is absent     | One info record that names `GEOMCP_FUEL_WISSOL_ENABLED` |

### `WISFUEL-SC-020`

**Verifies:** `WISFUEL-NFR-002`
**Layer:** unit

**Given** an environment with no variable of `fuel`.
**When** the configuration loads.
**Then** `Wissol.Enabled` is true, `Wissol.HTTP.Rate` is 1, and `Wissol.HTTP.Burst` is 1.

## Retired identifiers

This file has no retired identifier.
