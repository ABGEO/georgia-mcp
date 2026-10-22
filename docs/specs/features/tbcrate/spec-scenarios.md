---
id: TBCRATE
title: The commercial exchange rates of TBC Bank, the scenarios
type: spec
status: approved
created: 2026-10-18
updated: 2026-10-22
context: rates
approved_by: Temuri
approved_on: 2026-10-18
constrained_by: [TST]
requirements: features/tbcrate/requirements.md
---

# Specification: The commercial exchange rates of TBC Bank, the scenarios

`pkg/tbc/testdata/` holds the answers that the tests serve:

| File                     | Content                                                                   |
| ------------------------ | ------------------------------------------------------------------------- |
| `commercial.json`        | The real answer of TBC to `currency=USD,EUR,GBP,JPY,XXX`, recorded on 2026-10-18. It holds four rates and no `XXX`. |
| `problem-401.json`       | The real answer to a missing key, recorded on 2026-10-18, with its trace identifier removed. |

A unit test of `app` uses fake sources. An integration test serves the files from
`httptest` and calls the tool through the in-memory MCP transport.

## The client

### `TBCRATE-SC-001`

**Verifies:** `TBCRATE-FR-001`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `CommercialRates` runs with the key `test-key` and the currencies `USD` and `EUR`.
**Then** the path is `/v1/exchange-rates/commercial`, the query is `currency=USD%2CEUR`,
the header `apikey` is `test-key`, and `Accept` is `application/json`. With no
currencies, the query is empty.

### `TBCRATE-SC-002`

**Verifies:** `TBCRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `commercial.json`.
**When** `CommercialRates` runs.
**Then** `Base` is `GEL`, the list holds `EUR`, `GBP`, `JPY`, and `USD`, and the rate of
`USD` has the buy rate `2.566` and the sell rate `2.628`.

### `TBCRATE-SC-003`

**Verifies:** `TBCRATE-FR-008`
**Layer:** unit

**Given** the vendor serves `problem-401.json` with status 401.
**When** `CommercialRates` runs.
**Then** the error is an `*APIError` with 401, the title `Invalid Api Key`, and the
system code. `errors.Is(err, ErrUnexpectedStatus)` holds.

### `TBCRATE-SC-004`

**Verifies:** `TBCRATE-FR-001`
**Layer:** live

**Given** the real TBC and the key in `GEOMCP_RATES_TBC_API_KEY`.
**When** `CommercialRates` runs with `USD` and `JPY`.
**Then** the base is `GEL`, and the list holds both codes. The test skips when the variable is empty.

## The domain

### `TBCRATE-SC-005`

**Verifies:** `TBCRATE-FR-003`
**Layer:** unit

| Input   | Result                  |
| ------- | ----------------------- |
| `tbc`   | The code `tbc`          |
| `TBC`   | `ErrInvalidBankCode`    |
| `t`     | `ErrInvalidBankCode`    |
| `tb-c`  | `ErrInvalidBankCode`    |

A fuzz test of `ParseBankCode` finds no input that panics.

### `TBCRATE-SC-006`

**Verifies:** `TBCRATE-FR-001`
**Layer:** unit

| Buy      | Sell     | Result                        |
| -------- | -------- | ----------------------------- |
| `2.5800` | `2.6200` | The rate                      |
| `0`      | `2.6200` | `ErrInvalidCommercialRate`    |
| `2.5800` | `-1`     | `ErrInvalidCommercialRate`    |

## The use case

### `TBCRATE-SC-007`

**Verifies:** `TBCRATE-FR-001`
**Layer:** unit

**Given** a fake bank `tbc` that quotes `USD` and `EUR`.
**When** `Get` runs with `USD` and the bank `tbc`.
**Then** the answer holds one bank with the rate of `USD` only, and no missing code.

### `TBCRATE-SC-008`

**Verifies:** `TBCRATE-FR-002`
**Layer:** unit

**Given** two fake banks, `tbc` and `bog`.
**When** `Get` runs with `USD` and no bank.
**Then** the answer holds both banks, sorted by code.

### `TBCRATE-SC-009`

**Verifies:** `TBCRATE-FR-003`
**Layer:** unit

**Given** the fake bank `tbc` only.
**When** `Get` runs with the bank `bog`.
**Then** the error is an `*UnknownBankError` with the code `bog` and the known codes `[tbc]`,
and no bank was asked.

### `TBCRATE-SC-010`

**Verifies:** `TBCRATE-FR-004`
**Layer:** unit

**Given** a fake bank that quotes `USD` and no `XXX`.
**When** `Get` runs with `USD` and `XXX`.
**Then** the bank holds `USD`, and its `Missing` is `[XXX]`.

### `TBCRATE-SC-011`

**Verifies:** `TBCRATE-FR-008`
**Layer:** unit

| Banks                                   | Result                                                      |
| --------------------------------------- | ----------------------------------------------------------- |
| `tbc` fails, `bog` answers              | `bog` in `Banks`, `tbc` in `Unavailable`, no error          |
| `tbc` fails, and it is the only request | An error that wraps `kernel.ErrSourceUnavailable`           |
| `tbc` waits 1 second, `bog` answers     | Both banks. The call takes less than 1.5 seconds.           |

## The adapter

### `TBCRATE-SC-012`

**Verifies:** `TBCRATE-INV-003`
**Layer:** unit

**Given** an answer with `{"currency":"USD","buy":2.5800,"sell":2.6200}`, and one with
`{"currency":"AZN","buy":1.35,"sell":1.715}`.
**When** the adapter maps them.
**Then** the rates format as `2.5800`, `2.6200`, `1.35`, and `1.715`.

### `TBCRATE-SC-013`

**Verifies:** `TBCRATE-FR-008`
**Layer:** unit

| Vendor answer                     | Adapter error                          |
| --------------------------------- | -------------------------------------- |
| 401 with `problem-401.json`       | Wraps `kernel.ErrSourceUnavailable`    |
| 200 with a base other than `GEL`  | Wraps `kernel.ErrSourceUnavailable`    |
| 200 with a rate that is not a number | Wraps `kernel.ErrSourceUnavailable` |

### `TBCRATE-SC-014`

**Verifies:** `TBCRATE-FR-004`, `TBCRATE-FR-005`
**Layer:** unit

**Given** the vendor serves `commercial.json`.
**When** the adapter asks for `USD`, `EUR`, `GBP`, `JPY`, and `XXX`.
**Then** the result holds four rates, `JPY` at the buy rate `0.0157` for one unit
(`TBCRATE-DD-008`), and `Missing` is `[XXX]`.

## The cache

### `TBCRATE-SC-015`

**Verifies:** `TBCRATE-FR-009`
**Layer:** unit

| Read at  | Bank fails from | Asked at   | Codes asked    | Result                                            |
| -------- | --------------- | ---------- | -------------- | ------------------------------------------------- |
| `USD`, 0 | 1 minute        | 30 minutes | `USD`          | The rate of `USD`, stale, read at 0               |
| `USD`, 0 | 1 minute        | 61 minutes | `USD`          | An error that wraps `kernel.ErrSourceUnavailable` |
| `USD`, 0 | 1 minute        | 30 minutes | `USD`, `EUR`   | An error that wraps `kernel.ErrSourceUnavailable` (`TBCRATE-DD-012`) |

The test runs in `synctest`.

### `TBCRATE-SC-016`

**Verifies:** `TBCRATE-NFR-001`
**Layer:** unit

**Given** a counting fake bank.
**When** the cache is asked for `USD` at 0, at 4 minutes, and at 6 minutes.
**Then** the bank counts 2 calls, and no answer is older than 5 minutes. The test runs in `synctest`.

### `TBCRATE-SC-017`

**Verifies:** `TBCRATE-NFR-002`
**Layer:** unit

**Given** a counting fake bank.
**When** 10 asks for `USD` and `EUR` arrive at once.
**Then** the bank counts 1 call.

## The tool and the module

### `TBCRATE-SC-018`

**Verifies:** `TBCRATE-FR-006`, `TBCRATE-FR-007`, `TBCRATE-INV-002`
**Layer:** integration

**Given** the vendor serves `commercial.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD`.
**Then** `meaning` holds the constant of section 4.3, the bank `tbc` has `source`
`TBC Bank`, `as_of` is the instant of the clock, and the rate of `USD` has `buy` and `sell`.

### `TBCRATE-SC-019`

**Verifies:** `TBCRATE-INV-001`
**Layer:** integration

**Given** the key `secret-tbc-key`, a logger at debug level, and a vendor that answers 401.
**When** a client calls the tool.
**Then** the tool error, the log, and every error of the stack hold no `secret-tbc-key`.

### `TBCRATE-SC-021`

**Verifies:** `TBCRATE-FR-003`
**Layer:** unit

**Given** the use case returns `UnknownBankError{Code: "bog", Known: [tbc]}`.
**When** the tool maps it.
**Then** the tool error reads `banks: "bog" is not a bank of this server. Use one of: tbc.`

### `TBCRATE-SC-022`

**Verifies:** `TBCRATE-FR-011`
**Layer:** unit

| Codes                          | Result                                                     |
| ------------------------------ | ---------------------------------------------------------- |
| none                           | `*CurrencyCountError` with 0. The tool error reads `currencies: name at least one code, for example USD.` |
| 11 distinct codes              | `*CurrencyCountError` with 11. The tool error reads `currencies: 11 codes. Ask for 10 or fewer.` |
| 10 distinct codes              | The answer, with one call to the bank                      |
| `USD` and `usd`                | The answer, and the bank was asked for `[USD]`             |

### `TBCRATE-SC-023`

**Verifies:** `TBCRATE-FR-007`, `TBCRATE-NFR-002`
**Layer:** unit

**Given** a counting fake bank that quotes `USD` and `EUR`, and no `KRW`. The cache read
`USD` and `KRW` at 0.
**When** the cache is asked for `USD`, `EUR`, and `KRW` at 2 minutes.
**Then** the bank was asked for `[EUR]` only, `ReadAt` is 0, `Missing` is `[KRW]`, and the
bank counts 2 calls. The test runs in `synctest`.

### `TBCRATE-SC-024`

**Verifies:** `TBCRATE-NFR-002`
**Layer:** unit

**Given** an environment with the key only.
**When** the configuration loads.
**Then** `TBC.HTTP.Rate` is 1 and `TBC.HTTP.Burst` is 1.

### `TBCRATE-SC-025`

**Verifies:** `TBCRATE-FR-012`
**Layer:** integration

| TBC | Tools of the server                                 | Log                                                  |
| --- | --------------------------------------------------- | ---------------------------------------------------- |
| on  | The official tools and `rates_get_commercial_rates` | No record that TBC is off                            |
| off | The official tools only                             | `TBC Bank is off` with `variable=GEOMCP_RATES_TBC_ENABLED` |

### `TBCRATE-SC-026`

**Verifies:** `TBCRATE-FR-013`
**Layer:** unit

| Settings                                | Result of `LoadConfig`                                        |
| --------------------------------------- | ------------------------------------------------------------- |
| No key, no switch                       | An error that names `GEOMCP_RATES_TBC_API_KEY` and `GEOMCP_RATES_TBC_ENABLED`. |
| No key, `GEOMCP_RATES_TBC_ENABLED=false` | No error. TBC is off.                                         |
| A key, no switch                        | No error. TBC is on.                                          |

## Retired identifiers

| ID             | Retired    | Reason                                                   |
| -------------- | ---------- | -------------------------------------------------------- |
| TBCRATE-SC-020 | 2026-10-22 | It verified `TBCRATE-FR-010`. `TBCRATE-SC-025` and `TBCRATE-SC-026` replace it. |
