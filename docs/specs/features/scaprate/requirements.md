---
id: SCAPRATE
title: The commercial exchange rates of Scapp
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Scapp

## 1. Problem

A person compares the commercial rates before they change money. On 2026-10-25 Scapp
sold the dollar for 2.599, and Halyk Bank for 2.612. A model that cannot read Scapp
misses that price.

Scapp was Swiss Capital. The address `swisscapital.ge` now leads to `scapp.ge`. Scapp
is a microfinance organization, not a bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                     |
| ---------------------- | ------------------------------------------------------------------------ |
| A person with a client | The buy rate and the sell rate of Scapp, next to the rates of the banks. |
| The operator           | Turns Scapp off with no change to the code.                              |

## 3. Source

| Vendor | Source                                                   | Official                   | Publication period | Terms of use |
| ------ | -------------------------------------------------------- | -------------------------- | ------------------ | ------------ |
| Scapp  | The page `scapp.ge/ka/currency`, the table "მიმდინარე კურსები" | No. A web page for people. | Not published.     | No key. No published limit. `robots.txt` allows the page. |

Scapp publishes no API. A search on 2026-10-25 found no developer portal.

Observed behavior of the page on 2026-10-25:

- The page renders the table on the server. A plain GET returns it.
- The table has three columns: the currency with its unit, the buy price, and the sell price.
- The table lists 6 currencies, among them `USD`, `EUR`, `GBP`, `ILS`, and `TRY`.
- Each row states its unit: `1 USD`. One row had a unit of 10. The prices are for that unit.
- Each price has three decimal places, with the trailing zeros: `2.900`.
- The page also holds the same prices for its converter, with the trailing zeros cut: `2.9`.
- The page states no date and no time.
- A cache in front of the page keeps it for up to 5 minutes (`max-age=300`).
- The footer names the company: "სს მიკროსაფინანსო ორგანიზაცია სკაპ".

## 4. Out of scope

- The converter of the page.
- The other products of Scapp: loans and deposits.

## 5. Definitions

| Term  | Meaning                                                                        |
| ----- | ------------------------------------------------------------------------------ |
| Scapp | Scapp, a microfinance organization of Georgia, formerly Swiss Capital. Bank code `scapp`. |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE`
defines bank, bank kind, and microfinance organization.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Scapp, as `BOGRATE` 6 lists it.
The statements below add what is particular to Scapp.

### `SCAPRATE-FR-001`

The server must return the commercial rate of Scapp under the bank code `scapp`.

**Examples:**

- Normal case: `USD` at `scapp` on 2026-10-25. The buy rate is `2.595`, the sell rate `2.599`.
- Limit case: `EUR` with a buy price of `2.900`. The answer gives `2.9`.
- Unwanted case: a request names `swisscapital`. The server refuses it as a bank that
  it cannot read.

### `SCAPRATE-FR-002`

The server must give the price of one unit when Scapp prices a currency per 10, 100,
or 1000 units.

**Examples:**

- Normal case: a row `10 JPY` with a buy price of `0.170`. The answer gives `0.017`.
- Limit case: `USD` per 1 unit. The answer gives the price as Scapp publishes it.
- Unwanted case: the answer gives `0.17` for that row. A comparison with another bank is
  then wrong by a factor of 10.

### `SCAPRATE-FR-003`

The server must treat a currency that the table does not list, or lists with a price
of zero, as a currency that Scapp does not quote.

**Examples:**

- Normal case: `CHF` at `scapp`. The answer names it as a currency that Scapp does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `SCAPRATE-FR-004`

The server must name Scapp as a microfinance organization in each place that lists the
banks.

**Why:** A person who expects a bank account at Scapp is wrong. `RICORATE-FR-004` gives
the rule.

### `SCAPRATE-FR-005`

The server must read Scapp with no key. The operator must be able to turn Scapp off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `scapp`.
- Unwanted case: a request names `scapp` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `SCAPRATE-NFR-001`

The server must send at most 1 call to Scapp in each second, and at most 60 calls in a
minute. Measure the calls at the source, for any count of requests.

**Why:** Scapp publishes no limit. One call returns every currency. The limit protects
a page for people.

## 8. Invariants

### `SCAPRATE-INV-001`

A commercial rate of Scapp in an answer has the exact value of the published price,
divided by the published unit. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Scapp joins `rates_get_commercial_rates`. No new tool.                   |
| `MCP-005`            | MCP tools     | The answer names Scapp and the instant at which the server read it.     |
| `CLT-001`            | Clients       | The client is the package `pkg/scapp`.                                   |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Which bank code and name? | Temuri | `scapp` and Scapp (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
