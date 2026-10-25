---
id: HALRATE
title: The commercial exchange rates of Halyk Bank
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Halyk Bank

## 1. Problem

A person compares the commercial rates of the banks before they change money. On
2026-10-25 Halyk sold the dollar for 2.612, and Basisbank for 2.630. Halyk also quotes
the tenge. A model that cannot read Halyk misses both prices.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Halyk, next to the rates of the other banks. |
| The operator           | Turns Halyk off with no change to the code.                                |

## 3. Source

| Vendor             | Source                                                     | Official                   | Publication period | Terms of use |
| ------------------ | ---------------------------------------------------------- | -------------------------- | ------------------ | ------------ |
| Halyk Bank Georgia | The page `halykbank.ge/ka/individuals/currency`, the tab "მიმდინარე ვალუტის კურსი" | No. A web page for people. | Not published. | No key. No published limit. `robots.txt` allows every path. |

Halyk publishes no API. A search on 2026-10-25 found no developer portal. The scripts
of the site call no method for the rates.

Observed behavior of the page on 2026-10-25:

- The page renders the table on the server. A plain GET returns it.
- The table has three columns: the currency, the buy price, and the sell price.
- The table lists 7 currencies, among them `USD`, `EUR`, `GBP`, `CNY`, `CZK`, and `KZT`.
- The table states no unit. Each price is for one unit: `KZT` buys at `0.0054`.
- Each price has four decimal places, with the trailing zeros: `2.5870`.
- The page also holds the same prices as JSON for its converter, with the trailing zeros
  cut, and a row `GEL` at `1.0`.
- The line "თარიღი" above the table gives the minute of the request, not of the
  publication. Two requests 8 minutes apart gave two times.
- The home page holds the same JSON, and no table.

## 4. Out of scope

- The history of the rates, on the second tab of the page.
- The converter of the page.

## 5. Definitions

| Term  | Meaning                                                              |
| ----- | -------------------------------------------------------------------- |
| Halyk | Halyk Bank Georgia, a commercial bank of Georgia. Bank code `halyk`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Halyk, as `BOGRATE` 6 lists it.
The statements below add what is particular to Halyk.

### `HALRATE-FR-001`

The server must return the commercial rate of Halyk under the bank code `halyk`.

**Examples:**

- Normal case: `USD` at `halyk` on 2026-10-25. The buy rate is `2.587`, the sell rate `2.612`.
- Limit case: `KZT`. The buy rate is `0.0054`, the sell rate `0.0059`.
- Unwanted case: `GEL` at `halyk`. The tool refuses it, as for every bank.

### `HALRATE-FR-002`

The server must treat a currency that the table does not list, or lists with a price of
zero, as a currency that Halyk does not quote.

**Examples:**

- Normal case: `CHF` at `halyk`. The answer names it as a currency that Halyk does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `HALRATE-FR-003`

The server must read Halyk with no key. The operator must be able to turn Halyk off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `halyk`.
- Unwanted case: a request names `halyk` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `HALRATE-NFR-001`

The server must send at most 1 call to Halyk in each second, and at most 60 calls in a
minute. Measure the calls at the source, for any count of requests.

**Why:** Halyk publishes no limit. One call returns every currency. The limit protects
a page for people.

## 8. Invariants

### `HALRATE-INV-001`

A commercial rate of Halyk in an answer has the exact value of the published price. The
server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Halyk joins `rates_get_commercial_rates`. No new tool.                   |
| `MCP-005`            | MCP tools     | The answer names Halyk and the instant at which the server read it.     |
| `CLT-001`            | Clients       | The client is the package `pkg/halyk`.                                   |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
