---
id: INTLRATE
title: The commercial exchange rates of Inteli Express
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Inteli Express

## 1. Problem

A person who sends or receives a money transfer often changes the cash at the same
desk. Inteli Express changes money in its branches. On 2026-10-25 it sold the dollar for
2.598, and Halyk Bank for 2.612. A model that cannot read Inteli Express misses that price.

Inteli Express is a microfinance organization, not a bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                               |
| ---------------------- | ---------------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Inteli Express, next to the rates of the banks.  |
| The operator           | Turns Inteli Express off with no change to the code.                               |

## 3. Source

| Vendor         | Source | Official | Publication period | Terms of use |
| -------------- | ------ | -------- | ------------------ | ------------ |
| Inteli Express | The home page `inteliexpress.com/ka/main-page-geo/`, the table "ვალუტის სტანდარტული კურსები" | No. A web page for people. | Not published. | No key. No published limit. `robots.txt` allows the page. |

Inteli Express publishes no API. The page offers no other source of the rates.

Observed behavior of the home page on 2026-10-25:

- The page renders the table on the server. A plain GET returns it.
- The table has one title row and no column titles.
- Each row has four cells: the currency of the price, the currency that the row prices,
  and two prices. The first price is the lower: `GEL`, `USD`, `2.595`, `2.598`.
- The table lists 5 currencies against the lari, among them `USD`, `EUR`, `GBP`, and
  `TRY`. It also lists one cross rate, `EUR` against `USD`.
- No row states a unit. Each price is for one unit: `TRY` buys at `0.050`.
- Each price has three decimal places, with the trailing zeros: `0.050`.
- The page states no date and no time.
- The word "სტანდარტული" says that a branch can give a different rate.

## 4. Out of scope

- The cross rate `EUR` against `USD`.
- The other products of Inteli Express: money transfers, SEPA transfers, payments, and loans.

## 5. Definitions

| Term           | Meaning                                                                                |
| -------------- | -------------------------------------------------------------------------------------- |
| Inteli Express | Inteli Express (ინტელიექსპრესი), a microfinance organization of Georgia. Bank code `inteli`. |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE`
defines bank, bank kind, and microfinance organization.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Inteli Express, as `BOGRATE` 6 lists
it. The statements below add what is particular to Inteli Express.

### `INTLRATE-FR-001`

The server must return the commercial rate of Inteli Express under the bank code `inteli`.

**Examples:**

- Normal case: `USD` at `inteli` on 2026-10-25. The buy rate is `2.595`, the sell rate `2.598`.
- Limit case: `TRY` with a buy price of `0.050`. The answer gives `0.05`.
- Unwanted case: the answer holds a rate of `EUR` against `USD`. A commercial rate is
  against the lari.

### `INTLRATE-FR-002`

The server must treat a currency that the table does not list, or lists with a price
of zero, as a currency that Inteli Express does not quote.

**Examples:**

- Normal case: `CHF` at `inteli`. The answer names it as a currency that Inteli Express
  does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `INTLRATE-FR-003`

The server must treat Inteli Express as a source that it cannot read when a row gives
a buy price above its sell price.

**Why:** The table names no column. A swap of the two prices would give a person the
wrong side of the trade, with no sign of the error.

**Examples:**

- Normal case: `2.595`, then `2.598`. The server reads the row.
- Limit case: two equal prices. The server reads the row.
- Unwanted case: `2.598`, then `2.595`. The server reports Inteli Express as unavailable.

### `INTLRATE-FR-004`

The server must name Inteli Express as a microfinance organization in each place that
lists the banks.

**Why:** `RICORATE-FR-004` gives the rule.

### `INTLRATE-FR-005`

The server must read Inteli Express with no key. The operator must be able to turn
Inteli Express off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `inteli`.
- Unwanted case: a request names `inteli` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `INTLRATE-NFR-001`

The server must send at most 1 call to Inteli Express in each second, and at most 60
calls in a minute. Measure the calls at the source, for any count of requests.

**Why:** Inteli Express publishes no limit. One call returns every currency.

## 8. Invariants

### `INTLRATE-INV-001`

A commercial rate of Inteli Express in an answer has the exact value of the published
price. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Inteli Express joins `rates_get_commercial_rates`. No new tool.          |
| `MCP-005`            | MCP tools     | The answer names Inteli Express and the instant at which the server read it. |
| `CLT-001`            | Clients       | The client is the package `pkg/inteli`.                                  |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | The page calls Inteli Express a money transfer company. Which kind? | Temuri | A microfinance organization (2026-10-25). |
| 2   | Which bank code and name? | Temuri | `inteli` and Inteli Express (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
