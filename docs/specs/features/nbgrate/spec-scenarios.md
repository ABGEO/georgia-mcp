---
id: NBGRATE
title: The official exchange rates of the NBG, the scenarios
type: spec
status: approved
created: 2026-10-18
updated: 2026-10-18
context: rates
approved_by: Temuri
approved_on: 2026-10-18
constrained_by: [TST]
requirements: features/nbgrate/requirements.md
---

# Specification: The official exchange rates of the NBG, the scenarios

The recorded answers in `pkg/nbg/testdata/` are real answers of the NBG:

| File                       | Request                                   |
| -------------------------- | ----------------------------------------- |
| `en-2026-10-17.json`       | `en`, 2026-10-17. 42 currencies.          |
| `ka-2026-10-17.json`       | `ka`, 2026-10-17.                         |
| `en-2026-10-11.json`       | `en`, Sunday 2026-10-11. Valid from 2026-10-10. |
| `en-1990-01-01.json`       | `en`, 1990-01-01. An empty list.          |

An integration test serves these files from `httptest` and calls a tool through the
in-memory MCP transport. A unit test of `app` uses a fake source. `now` is a fixed instant.

## The rates of a date

### `NBGRATE-SC-001`

**Verifies:** `NBGRATE-FR-001`
**Layer:** integration

**Given** the vendor serves `en-2026-10-17.json`.
**When** a client calls `rates_get_official_rates` with `USD`, `EUR`, and the date 2026-10-17.
**Then** the rates hold `USD` at `"2.5982"` and `EUR` at `"2.9274"`, and `missing` is absent.

### `NBGRATE-SC-002`

**Verifies:** `NBGRATE-FR-001`
**Layer:** unit

**Given** the code `usd`.
**When** `ParseCurrencyCode` reads it.
**Then** it returns `USD`. A fuzz test of `ParseCurrencyCode` finds no input that panics,
and every accepted code has three upper case letters.

### `NBGRATE-SC-003`

**Verifies:** `NBGRATE-FR-001`
**Layer:** unit

| Input `currencies`      | Tool result                                                      |
| ----------------------- | ---------------------------------------------------------------- |
| `["US"]`                | Error: `currencies: "US" is not a currency code. …`              |
| `["U$D"]`               | Error: `currencies: "U$D" is not a currency code. …`             |
| 51 valid codes          | Error: `currencies: 51 codes. …`                                 |

### `NBGRATE-SC-004`

**Verifies:** `NBGRATE-FR-002`
**Layer:** integration

**Given** the vendor serves `en-2026-10-17.json`.
**When** a client calls the tool with no `currencies` and the date 2026-10-17.
**Then** the output holds 42 rates, sorted by currency, from `AED` to `ZAR`.

### `NBGRATE-SC-005`

**Verifies:** `NBGRATE-FR-003`
**Layer:** unit

**Given** `now` is 2026-10-17T21:30:00Z, which is 01:30 on 2026-10-18 in Tbilisi.
**When** `GetOfficialRates.Get` runs with no date.
**Then** the fake source receives the date 2026-10-18.

### `NBGRATE-SC-006`

**Verifies:** `NBGRATE-FR-004`
**Layer:** integration

**Given** the vendor serves `en-2026-10-11.json`.
**When** a client asks for `USD` on 2026-10-11.
**Then** `valid_from` of the output and of the rate is 2026-10-10.

### `NBGRATE-SC-007`

**Verifies:** `NBGRATE-FR-005`
**Layer:** unit

| `now` (UTC)            | Requested  | Fake source `ValidFrom` | Result                                   |
| ---------------------- | ---------- | ----------------------- | ---------------------------------------- |
| 2026-10-16T14:00:00Z   | 2026-10-17 | 2026-10-17              | The set                                  |
| 2026-10-16T10:00:00Z   | 2026-10-17 | 2026-10-16              | `NotPublishedError`, newest 2026-10-16   |
| 2026-10-18T10:00:00Z   | 2026-10-25 | 2026-10-17              | `NotPublishedError`, newest 2026-10-17   |

### `NBGRATE-SC-008`

**Verifies:** `NBGRATE-FR-005`
**Layer:** unit

**Given** the use case returns `NotPublishedError{Requested: 2026-10-25, Newest: 2026-10-17}`.
**When** the tool maps it.
**Then** the tool error reads `The NBG has not published rates for 2026-10-25. The newest set is valid from 2026-10-17.`

### `NBGRATE-SC-009`

**Verifies:** `NBGRATE-FR-006`
**Layer:** unit

| Input        | Result                    |
| ------------ | ------------------------- |
| `2026-10-17` | The date 2026-10-17       |
| `2026-02-30` | `ErrInvalidDate`          |
| `2026-13-45` | `ErrInvalidDate`          |
| `17/10/2026` | `ErrInvalidDate`          |

A fuzz test of `ParseDate` finds no input that panics, and every accepted date formats
back to its input.

### `NBGRATE-SC-010`

**Verifies:** `NBGRATE-FR-007`
**Layer:** unit

**Given** a fake set that holds `USD` and no `XXX`.
**When** `GetOfficialRates.Get` runs with `USD` and `XXX`.
**Then** the set holds `USD` only, and `Missing` is `[XXX]`.

### `NBGRATE-SC-011`

**Verifies:** `NBGRATE-FR-007`
**Layer:** integration

**Given** the vendor serves `en-1990-01-01.json`.
**When** a client asks for `USD` and `EUR` on 1990-01-01.
**Then** `rates` is empty, `missing` is `["EUR", "USD"]`, and `valid_from` is absent.

### `NBGRATE-SC-012`

**Verifies:** `NBGRATE-FR-008`
**Layer:** unit

| Quantity | Rate      | `RatePerUnit` |
| -------- | --------- | ------------- |
| 1        | `2.5982`  | `2.5982`      |
| 10       | `7.0738`  | `0.70738`     |
| 100      | `1.6455`  | `0.016455`    |
| 10000    | `0.0146`  | `0.00000146`  |
| 12       | any       | `NewOfficialRate` returns `ErrInvalidQuantity` |

### `NBGRATE-SC-013`

**Verifies:** `NBGRATE-FR-009`
**Layer:** unit

**Given** `en-2026-10-17.json`.
**When** the adapter maps it.
**Then** `EUR` has the change `0.0145`, and `AED` has the change `-0.0013`.

### `NBGRATE-SC-014`

**Verifies:** `NBGRATE-FR-010`
**Layer:** integration

| Input `language` | Vendor path        | Name of `USD`  |
| ---------------- | ------------------ | -------------- |
| `ka`             | `/currencies/ka/…` | `აშშ დოლარი`   |
| absent           | `/currencies/en/…` | `US Dollar`    |
| `de`             | none               | Error: `language: "de" is not a language of the NBG. …` |

### `NBGRATE-SC-015`

**Verifies:** `NBGRATE-FR-011`
**Layer:** unit

**Given** a currency with `date` `2026-10-16T17:01:01.982Z`.
**When** the adapter maps it.
**Then** `PublishedAt` is 2026-10-16 17:01:01.982 at UTC+4, and the tool writes
`2026-10-16T17:01:01.982+04:00` in `as_of`.

## The history

### `NBGRATE-SC-016`

**Verifies:** `NBGRATE-FR-012`
**Layer:** unit

**Given** a fake source with a set valid from each business day and each Saturday from
2026-09-26 to 2026-10-17, and `now` 2026-10-18.
**When** `ListOfficialHistory.List` runs for `USD` from Sunday 2026-10-04 to 2026-10-17.
**Then** the first point is valid from Saturday 2026-10-03, each `ValidFrom` appears once,
and the points ascend.

### `NBGRATE-SC-017`

**Verifies:** `NBGRATE-FR-012`
**Layer:** unit

| From       | To         | Result                                             |
| ---------- | ---------- | -------------------------------------------------- |
| 2026-10-15 | 2026-10-15 | One point                                          |
| 2026-09-17 | 2026-10-17 | 31 days: the points                                |
| 2026-09-16 | 2026-10-17 | 32 days: `ErrPeriodTooLong`                        |
| 2026-10-10 | 2026-10-01 | `ErrPeriodReversed`                                |
| 2026-10-10 | 2026-10-25 | `NotPublishedError`, newest 2026-10-17             |

### `NBGRATE-SC-018`

**Verifies:** `NBGRATE-FR-012`
**Layer:** unit

**Given** the use case returns `ErrPeriodTooLong` for 40 days.
**When** the tool maps it.
**Then** the tool error reads `to: the period has 40 days. The longest period is 31 days.`

## The outage of the vendor

### `NBGRATE-SC-019`

**Verifies:** `NBGRATE-FR-013`
**Layer:** unit

**Given** the cache holds the set of today, read at 10:00, and the vendor fails from 10:20.
**When** the use case asks at 12:00.
**Then** the result holds the set, `Stale` is true, and `ReadAt` is 10:00. The test runs
in `synctest`.

### `NBGRATE-SC-020`

**Verifies:** `NBGRATE-FR-014`
**Layer:** integration

**Given** the vendor answers 503, and the cache is empty.
**When** a client calls `rates_get_official_rates`.
**Then** the result has `isError`, and the text reads `The NBG did not answer. Try again in a few minutes.`

### `NBGRATE-SC-021`

**Verifies:** `NBGRATE-FR-014`
**Layer:** unit

| Vendor answer                 | `RateSets` error                                     |
| ----------------------------- | ---------------------------------------------------- |
| 500 with a body               | `*APIError` with 500 and the body. `errors.Is(err, ErrUnexpectedStatus)`. |
| 200 with a body that is not JSON | A decode error                                    |

The adapter wraps both with `kernel.ErrSourceUnavailable`.

### `NBGRATE-SC-022`

**Verifies:** `NBGRATE-FR-001`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `RateSets` runs for `ka` and 2026-10-17.
**Then** the path is `/currencies/ka/json/`, the query is `date=2026-10-17`, `Accept` is
`application/json`, and `User-Agent` starts with `georgia-mcp-nbg/`.

### `NBGRATE-SC-023`

**Verifies:** `NBGRATE-FR-001`
**Layer:** live

**Given** the real NBG.
**When** `RateSets` runs for `en` and a zero date.
**Then** it returns one set with 40 currencies or more, and `USD` is among them.

## The limits

### `NBGRATE-SC-024`

**Verifies:** `NBGRATE-NFR-001`
**Layer:** unit

**Given** the cache, a counting fake adapter, and the date of today.
**When** the use case asks at 0, at 14 minutes, and at 16 minutes.
**Then** the adapter counts 2 calls. The test runs in `synctest`.

### `NBGRATE-SC-025`

**Verifies:** `NBGRATE-NFR-002`
**Layer:** unit

| Case                                               | Adapter calls |
| -------------------------------------------------- | ------------- |
| 10 concurrent asks for one date and one language   | 1             |
| One past date, asked at 0 and at 29 days           | 1             |
| One past date, asked at 0 and at 31 days           | 2             |
| One date in `en`, then in `ka`                     | 2             |

### `NBGRATE-SC-026`

**Verifies:** `NBGRATE-NFR-003`
**Layer:** integration

**Given** an `httptest` vendor that answers each request after 150 ms, `httpclient.New`
with the defaults, and an empty cache.
**When** a client asks the history of `USD` over 31 past days.
**Then** the answer arrives less than 10 seconds after the request.

## The invariants

### `NBGRATE-SC-027`

**Verifies:** `NBGRATE-INV-001`
**Layer:** unit

**Given** a set valid from 2026-10-17 with three rates.
**When** the tool writes the output.
**Then** each element of `rates` has `valid_from` 2026-10-17.

### `NBGRATE-SC-028`

**Verifies:** `NBGRATE-INV-002`
**Layer:** unit

**Given** a fake source that answers the date 2026-10-10 with a set valid from 2026-10-12.
**When** `GetOfficialRates.Get` runs.
**Then** it returns an error that wraps `kernel.ErrSourceUnavailable`, and no set.

### `NBGRATE-SC-029`

**Verifies:** `NBGRATE-INV-003`
**Layer:** unit

| `rateFormated` | Quantity | Output `rate` | Output `rate_per_unit` |
| -------------- | -------- | ------------- | ---------------------- |
| `2.6000`       | 1        | `2.6000`      | `2.6000`               |
| `7.0738`       | 10       | `7.0738`      | `0.70738`              |

## The packaging

### `NBGRATE-SC-030`

**Verifies:** `NBGRATE-FR-001`
**Layer:** manual

**Given** `task build-image` and `task compose-up`.
**When** the owner calls `rates_get_official_rates` for `USD` over HTTP.
**Then** the answer holds the official rate of today.

### `NBGRATE-SC-031`

**Verifies:** `NBGRATE-FR-001`
**Layer:** manual

**Given** a desktop client that starts `georgia-mcp stdio`.
**When** the owner asks "what is the official dollar rate today".
**Then** the model cites the NBG, the rate, and the valid date.

### `NBGRATE-SC-032`

**Verifies:** `NBGRATE-FR-012`
**Layer:** integration

**Given** the vendor serves a recorded set for each day.
**When** a client calls `rates_list_official_history` for `USD` from 2026-10-11 to 2026-10-17.
**Then** `points` ascend by `valid_from`, the first is valid from 2026-10-10, and `name` is `US Dollar`.

### `NBGRATE-SC-033`

**Verifies:** `NBGRATE-FR-013`
**Layer:** unit

**Given** the cache holds a past set read at 0, and the vendor fails.
**When** the use case asks at 30 days and 1 hour, then at 37 days and 1 hour.
**Then** the first answer is stale with `ReadAt` 0, and the second is an error that wraps
`kernel.ErrSourceUnavailable`.

### `NBGRATE-SC-034`

**Verifies:** `NBGRATE-FR-013`
**Layer:** unit

**Given** a module with two stores.
**When** `Close` runs.
**Then** both stores stop, and a second `Close` returns no error.

## Retired identifiers

This file has no retired identifier.
