---
id: CRYSRATE
title: The commercial exchange rates of Crystal
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Crystal

## 1. Problem

A person outside Tbilisi often changes cash at a microfinance branch. Crystal has
branches in 40 cities. On 2026-10-25 it sold the dollar for 2.599, and Halyk Bank for
2.612. A model that cannot read Crystal misses that price.

Crystal is a microfinance organization, not a bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Crystal, next to the rates of the banks. |
| The operator           | Turns Crystal off with no change to the code.                              |

## 3. Source

| Vendor  | Source | Official | Publication period | Terms of use |
| ------- | ------ | -------- | ------------------ | ------------ |
| Crystal | `currency.crystal.ge/api/Conversion/GetWebRateByProductID?IsCash=true` | No. An unofficial method of the rate app. The page `crystal.ge/valutis-kursebi/` shows the app in a frame. | Not published. | No key needed. The script of the app sends a token, but the method answers without it. No published limit. `robots.txt` of both hosts allows every page. |

Crystal publishes no documented API. The page renders no rate on the server.

Observed behavior of the method on 2026-10-25:

- The answer is JSON with two envelopes: `errorCode`, then `data` with `success`, then
  `data` with the list of rates.
- Each item has `SourceISO`, `TargetISO`, `Multiplier`, `BuyRate`, `SellRate`, and
  `NBGRate`, the official rate of the NBG.
- The list holds 11 items: 10 currencies against the lari, among them `USD`, `EUR`,
  `GBP`, `CHF`, `TRY`, `AZN`, `AMD`, `ILS`, and `PLN`, and one cross rate, `EUR`
  against `USD`.
- `Multiplier` is 1, 10, 100, or 1000: `ILS` is per 10, `AMD` per 1000. The prices are
  for that unit.
- Each price is a JSON number with the trailing zeros cut: `6`, `2.595`.
- The answer states no date and no time.
- The app offers two products of cash rates: the default "NBG & Comercial (Cash)", and the
  rates of the branches of one of 40 cities. A note says that a branch can give a
  different rate.
- The app sends the header `authorization` with a token from its script. A call with
  no token answers 200 with the same list.

## 4. Out of scope

- The rates of a branch or a city.
- The rates without cash (`IsCash=false`), and the online exchange behind a login.
- The cross rate `EUR` against `USD`, and the official column.

## 5. Definitions

| Term    | Meaning                                                                   |
| ------- | ------------------------------------------------------------------------- |
| Crystal | Crystal (კრისტალი), a microfinance organization of Georgia. Bank code `crystal`. |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE`
defines bank, bank kind, and microfinance organization.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Crystal, as `BOGRATE` 6 lists it.
The statements below add what is particular to Crystal.

### `CRYSRATE-FR-001`

The server must return the default cash rate of Crystal under the bank code `crystal`.

**Examples:**

- Normal case: `USD` at `crystal` on 2026-10-25. The buy rate is `2.595`, the sell rate `2.599`.
- Unwanted case: the answer holds a rate of `EUR` against `USD`. A commercial rate is
  against the lari.
- Unwanted case: the answer gives the official rate `2.5971` as a price of Crystal.

### `CRYSRATE-FR-002`

The server must give the price of one unit when Crystal prices a currency per 10, 100,
or 1000 units.

**Examples:**

- Normal case: Crystal buys `10 ILS` for `3.5`. The answer gives `0.35`.
- Limit case: Crystal buys `1000 AMD` for `6.7`. The answer gives `0.0067`.
- Unwanted case: the answer gives `3.5` for `ILS`.

### `CRYSRATE-FR-003`

The server must treat a currency that the list does not hold, or holds with a price of
zero, as a currency that Crystal does not quote.

**Examples:**

- Normal case: `JPY` at `crystal`. The answer names it as a currency that Crystal does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `CRYSRATE-FR-004`

The server must treat Crystal as a source that it cannot read when an envelope of the
answer reports a failure.

**Why:** The method answers 200 with an error code in the body.

**Examples:**

- Normal case: `errorCode` is `"0"` and `success` is true. The server reads the list.
- Unwanted case: `success` is false. The server reports Crystal as unavailable.

### `CRYSRATE-FR-005`

The server must name Crystal as a microfinance organization in each place that lists
the banks.

**Why:** `RICORATE-FR-004` gives the rule.

### `CRYSRATE-FR-006`

The server must read Crystal with no key. The operator must be able to turn Crystal
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `crystal`.
- Unwanted case: a request names `crystal` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `CRYSRATE-NFR-001`

The server must send at most 1 call to Crystal in each second, and at most 60 calls in
a minute. Measure the calls at the source, for any count of requests.

**Why:** Crystal publishes no limit. One call returns every currency.

## 8. Invariants

### `CRYSRATE-INV-001`

A commercial rate of Crystal in an answer has the exact value of the published price,
divided by the published multiplier. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Crystal joins `rates_get_commercial_rates`. No new tool.                 |
| `MCP-005`            | MCP tools     | The answer names Crystal and the instant at which the server read it.   |
| `MCP-009`            | MCP tools     | A price in the JSON is a number. The server keeps its exact digits.     |
| `CLT-001`            | Clients       | The client is the package `pkg/crystal`.                                 |
| `CLT-009`            | Clients       | The client says that the method is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server send the token of the app? | Temuri | No. The method answers without it. A demand for it later is a fast-lane change (2026-10-25). |
| 2   | Which bank code and name? | Temuri | `crystal` and Crystal (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
