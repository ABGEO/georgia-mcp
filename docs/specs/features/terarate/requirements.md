---
id: TERARATE
title: The commercial exchange rates of Terabank
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Terabank

## 1. Problem

A person compares the commercial rates of the banks before they change money.
On 2026-10-21 Terabank sold the dollar for 2.601, and Bank of Georgia for 2.629. A
model that cannot read Terabank misses that price.

## 2. Users

| User                   | Need                                                                         |
| ---------------------- | ---------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Terabank, next to the rates of the other banks. |
| The operator           | Turns Terabank off with no change to the code.                               |

## 3. Source

| Vendor   | Source                                                       | Official | Publication period | Terms of use |
| -------- | ------------------------------------------------------------ | -------- | ------------------ | ------------ |
| Terabank | `terabank.ge/_mvcapi/CurrencyRatesApi/GetTeraCrossRates`     | No. An unofficial API: the page `terabank.ge/ka/retail/exchangerates` reads it. | Not published. On 2026-10-21 the rates differed from those of the day before. | No key. No published limit. |

Terabank publishes no API. A search on 2026-10-21 found no developer portal. The page
renders the rates in the browser, so the HTML of the page holds no rate.

Observed behavior of the source on 2026-10-21:

- The method takes a POST with an empty body. A request with no `Content-Length`
  header answers 411. A GET answers 404.
- One call returns every currency: `USD`, `EUR`, and `GBP`.
- Each currency has a buy price and a sell price, for one unit, against the lari. The
  vendor calls them cross rates. They are the rates that the page shows under
  "ტერაბანკის კურსი".
- Each price has four decimal places, with the trailing zeros: `2.9300`.
- Each currency also holds the prices of the day before, and a flag that tells the
  page to show them. The answer holds no instant.
- The answer has a field `succeeded`, true on each call.

## 4. Out of scope

- The prices of the day before.
- The cross rates between two foreign currencies, from the method of the converter.
- The official rate through Terabank. `NBGRATE` reads it from the NBG.

## 5. Definitions

| Term     | Meaning                                                  |
| -------- | -------------------------------------------------------- |
| Terabank | Terabank, a commercial bank of Georgia. Bank code `terabank`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Terabank, as `BOGRATE` 6 lists it.
The statements below add what is particular to Terabank.

### `TERARATE-FR-001`

The server must return the commercial rate of Terabank under the bank code `terabank`.

**Examples:**

- Normal case: `USD` at `terabank` on 2026-10-21. The buy rate is `2.596`, the sell rate `2.601`.
- Limit case: `EUR` with a sell price of `2.93`. The answer gives `2.93`.
- Unwanted case: the answer gives `2.547` for the buy rate of `USD`. That is the price
  of the day before.

### `TERARATE-FR-002`

The server must treat a currency that Terabank does not list, or lists with a price of
zero, as a currency that Terabank does not quote.

**Examples:**

- Normal case: `CHF` at `terabank`. The answer names it as a currency that Terabank does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `TERARATE-FR-003`

The server must report Terabank as a bank that did not answer when the source states
that the call did not succeed.

**Why:** The source can answer 200 with no data and a false `succeeded`. That is a
failure, not an absent quote.

### `TERARATE-FR-004`

The server must read Terabank with no key. The operator must be able to turn Terabank
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `terabank`.
- Unwanted case: a request names `terabank` while it is off. The server refuses it as
  a bank that it cannot read.

## 7. Non-functional requirements

### `TERARATE-NFR-001`

The server must send at most 1 call to Terabank in each second, and at most 60 calls
in a minute. Measure the calls at the source, for any count of requests.

**Why:** Terabank publishes no limit. One call returns every currency.

## 8. Invariants

### `TERARATE-INV-001`

A commercial rate of Terabank in an answer has the exact value of the published price.
The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Terabank joins `rates_get_commercial_rates`. No new tool.                |
| `MCP-005`            | MCP tools     | The answer names Terabank and the instant at which the server read it.  |
| `CLT-001`            | Clients       | The client is a module in `pkg/terabank`.                                |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial, and stays at `v0`.          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
