---
id: MBCRATE
title: The commercial exchange rates of MBC
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of MBC

## 1. Problem

A person compares the commercial rates before they change money. On 2026-10-25 MBC
sold the dollar for 2.599, and Halyk Bank for 2.612. A model that cannot read MBC misses
that price.

MBC is not a commercial bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                   |
| ---------------------- | ---------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of MBC, next to the rates of the banks. |
| The operator           | Turns MBC off with no change to the code.                              |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| MBC    | `fxrates.mbc.com.ge:8022/api/fxrates/mbc/commercial` | No. An unofficial method. The page `fx.mbc.com.ge` reads it. | Not published. | No key. No published limit. The method allows a call from any origin. `robots.txt` of the page allows it. |

MBC publishes no documented API. The page renders no rate on the server. Its script
reads the method above and fills the table "MBC კურსები".

Observed behavior of the method on 2026-10-25:

- `GET` with no parameter returns JSON: a list `FXRates` and a field `timestamp`.
- Each item has `FromCcy`, `ToCcy`, `Multiplier`, `Buy`, and `Sell`. `ToCcy` is `GEL`
  and `Multiplier` is 1 for each.
- The list holds 5 currencies, among them `USD` (2.595 to buy, 2.599 to sell), `EUR`
  (2.907 to buy, 2.914 to sell), `GBP`, and `TRY`.
- Each price is a JSON number with four decimal places: `2.5950`.
- `timestamp` is the instant of the answer in Unix seconds, not the instant of a change.
- The page shows 3 of the 5 currencies, among them `USD` and `EUR`. Its script reads
  each currency at a fixed place in the list. It hides `GBP` and `TRY`.
- The prices of `GBP` and `TRY` look stale: `TRY` buys at 0.0335 and sells at 0.0590.
  MJC quoted 0.052 and 0.054 on the same day.
- The method listens on port 8022, not on the port of the web.
- The footer names the company: "მიკრობანკი ემბისი".

## 4. Out of scope

- The online exchange of MBC, behind a login.
- The panel of the official rates on the page. It showed wrong values on 2026-10-25:
  `USD` at 0.219.

## 5. Definitions

| Term | Meaning                                                           |
| ---- | ----------------------------------------------------------------- |
| MBC  | MBC (ემბისი), a microfinance organization of Georgia. Bank code `mbc`. |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE`
defines bank, bank kind, and microfinance organization.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for MBC, as `BOGRATE` 6 lists it.
The statements below add what is particular to MBC.

### `MBCRATE-FR-001`

The server must return the commercial rate of MBC under the bank code `mbc`.

**Examples:**

- Normal case: `USD` at `mbc` on 2026-10-25. The buy rate is `2.595`, the sell rate `2.599`.
- Limit case: `EUR` with a buy price of `2.9070`. The answer gives `2.907`.

### `MBCRATE-FR-002`

The server must return only the currencies that the page of MBC shows. It must treat
every other currency as a currency that MBC does not quote.

**Why:** MBC hides the other prices from a person, and they look stale. A person who
asks for a hidden price at a branch can get a different one.

**Examples:**

- Normal case: `EUR` at `mbc`. The answer gives its rate.
- Unwanted case: `GBP` at `mbc`. The method lists it, but the answer names it as a
  currency that MBC does not quote.

### `MBCRATE-FR-003`

The server must give the price of one unit when the method prices a currency per 10,
100, or 1000 units.

**Examples:**

- Normal case: every item has the multiplier 1. The answer gives the price as MBC publishes it.
- Limit case: an item `EUR` with the multiplier 10 and a buy price of `29.0700`. The
  answer gives `2.907`.

### `MBCRATE-FR-004`

The server must treat a currency with a price of zero as a currency that MBC does not quote.

### `MBCRATE-FR-005`

The server must name MBC as a microfinance organization in each place that lists the banks.

**Why:** `RICORATE-FR-004` gives the rule.

### `MBCRATE-FR-006`

The server must read MBC with no key. The operator must be able to turn MBC off. It is
on by default.

**Examples:**

- Normal case: no configuration. The server offers `mbc`.
- Unwanted case: a request names `mbc` while it is off. The server refuses it as a bank
  that it cannot read.

## 7. Non-functional requirements

### `MBCRATE-NFR-001`

The server must send at most 1 call to MBC in each second, and at most 60 calls in a
minute. Measure the calls at the source, for any count of requests.

**Why:** MBC publishes no limit. One call returns every currency.

## 8. Invariants

### `MBCRATE-INV-001`

A commercial rate of MBC in an answer has the exact value of the published price,
divided by the published multiplier. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | MBC joins `rates_get_commercial_rates`. No new tool.                     |
| `MCP-005`            | MCP tools     | The answer names MBC and the instant at which the server read it.       |
| `MCP-009`            | MCP tools     | A price in the JSON is a number. The server keeps its exact digits.     |
| `CLT-001`            | Clients       | The client is the package `pkg/mbc`.                                     |
| `CLT-009`            | Clients       | The client says that the method is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | The footer says "მიკრობანკი". Which kind? | Temuri | A microfinance organization (2026-10-25). |
| 2   | All 5 currencies of the method, or only the ones the page shows? | Temuri | Only the ones the page shows (2026-10-25). |
| 3   | Which bank code and name? | Temuri | `mbc` and MBC (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
