---
id: BASRATE
title: The commercial exchange rates of Basisbank
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Basisbank

## 1. Problem

A person compares the commercial rates of the banks before they change money. On
2026-10-25 Basisbank bought the dollar for 2.560, and Scapp for 2.595. A model that
cannot read Basisbank cannot tell a person where not to sell.

## 2. Users

| User                   | Need                                                                         |
| ---------------------- | ---------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Basisbank, next to the rates of the other banks. |
| The operator           | Turns Basisbank off with no change to the code.                              |

## 3. Source

| Vendor    | Source                                         | Official | Publication period | Terms of use |
| --------- | ---------------------------------------------- | -------- | ------------------ | ------------ |
| Basisbank | `static.bb.ge/api/view/bb/websiteXrates`       | No. An unofficial API: the page `bb.ge/ge/currencies` reads it. | One set of rates for each date. | No key. A header gives a limit of 1600, with no period. `robots.txt` allows every path. |

Basisbank publishes no API. A search on 2026-10-25 found no developer portal.

The site names a second source, `rates.basisbank.ge/Rate.ashx`. It answers XML with the
rates of every date since the start, 3.4 MB in each call, with no compression. It
answers over HTTP only: the TLS certificate of the host has expired. This feature does
not read it.

Observed behavior of the method on 2026-10-25:

- A GET and a POST give the same answer. The page sends a token header. The method
  answers without it.
- One call returns two dates, the newest first: `2026-10-25` and `2026-10-24`. A date
  of a weekend repeats the rates of the last working day.
- Each date holds three sets of prices as a JSON text inside the JSON: the official rate
  (`kurs`), the buy price (`kursBuy`), and the sell price (`kursSell`).
- The sets list 5 currencies, among them `USD`, `EUR`, `GBP`, and `CNY`.
- Each buy and sell price is for one unit: `CNY` buys at `0.3700`. The official set
  gives `CNY` per 10 units, as the NBG does.
- Each price is a text with four decimal places: `"2.5600"`.
- The buy and sell prices equal the commercial rate that the page shows.

## 4. Out of scope

- The official rate through Basisbank. `NBGRATE` reads it from the NBG.
- The rates of the date before.
- The history of `Rate.ashx`.

## 5. Definitions

| Term      | Meaning                                                    |
| --------- | ---------------------------------------------------------- |
| Basisbank | Basisbank, a commercial bank of Georgia. Bank code `basisbank`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Basisbank, as `BOGRATE` 6 lists it.
The statements below add what is particular to Basisbank.

### `BASRATE-FR-001`

The server must return the commercial rate of Basisbank under the bank code `basisbank`.

**Examples:**

- Normal case: `USD` at `basisbank` on 2026-10-25. The buy rate is `2.56`, the sell rate `2.63`.
- Limit case: `CNY`. The buy rate is `0.37`, the sell rate `0.41`.
- Unwanted case: the answer gives `2.5971` for `USD`. That is the official rate.

### `BASRATE-FR-002`

The server must read the rates of the newest date in the answer.

**Examples:**

- Normal case: the answer holds `2026-10-25` and `2026-10-24`. The server reads `2026-10-25`.
- Limit case: the answer holds the older date first. The server still reads `2026-10-25`.
- Unwanted case: the answer holds no date. The server reports Basisbank as a bank that
  did not answer.

### `BASRATE-FR-003`

The server must treat a currency that Basisbank does not list, or lists with a price of
zero, as a currency that Basisbank does not quote.

**Examples:**

- Normal case: `CHF` at `basisbank`. The answer names it as a currency that Basisbank does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `BASRATE-FR-004`

The server must read Basisbank with no key. The operator must be able to turn Basisbank
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `basisbank`.
- Unwanted case: a request names `basisbank` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `BASRATE-NFR-001`

The server must send at most 1 call to Basisbank in each second, and at most 60 calls
in a minute. Measure the calls at the source, for any count of requests.

**Why:** The source states a limit of 1600 with no period. One call returns every
currency.

## 8. Invariants

### `BASRATE-INV-001`

A commercial rate of Basisbank in an answer has the exact value of the published price.
The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Basisbank joins `rates_get_commercial_rates`. No new tool.               |
| `MCP-005`            | MCP tools     | The answer names Basisbank and the instant at which the server read it. |
| `CLT-001`            | Clients       | The client is the package `pkg/basisbank`.                                   |
| `CLT-009`            | Clients       | The client says that the API is unofficial.                             |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Which bank code? | Temuri | `basisbank` (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
