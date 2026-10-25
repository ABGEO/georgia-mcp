---
id: CREDRATE
title: The commercial exchange rates of Credo Bank
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Credo Bank

## 1. Problem

A person compares the commercial rates of the banks before they change money. Credo Bank
serves many people outside Tbilisi. On 2026-10-25 it sold the euro for 2.95, and Scapp
for 2.914. A model that cannot read Credo misses that price.

## 2. Users

| User                   | Need                                                                      |
| ---------------------- | ------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Credo, next to the rates of the other banks. |
| The operator           | Sets the key of Credo. Turns Credo off with no change to the code.        |

## 3. Source

| Vendor     | Source                                         | Official | Publication period | Terms of use |
| ---------- | ---------------------------------------------- | -------- | ------------------ | ------------ |
| Credo Bank | `n1.noxtton.com/v2/credo/current-rates`        | No. An unofficial API of Noxtton, the company that runs the site of Credo. The page `credobank.ge/currency` reads it. | Not published. | A key in the header `x-api-key`. The page publishes the key, and Credo can change it. No published terms. No published limit. |

Credo publishes no API. A search on 2026-10-25 found no developer portal. The page
renders no rate on the server: the HTML holds none.

Observed behavior of the method on 2026-10-25:

- The page sends `GET /v2/credo/current-rates` with two parameters: `method`, the kind of
  rate, and `date`, the current instant in UTC: `2026-10-25T08:25:16.899Z`.
- The kind of the commercial rate is `FxRatesCommercial`. The page offers four more: the
  card rates, the card operation rates, the internet bank and transfer rates, and the
  official rate. The page labels the commercial rate "კრედო ბანკის გაცვლითი კურსი".
- The answer holds `success` and a list `rates`. Each item has `fromCcy`, `toCcy`,
  `multiplier`, `buy`, and `sell`.
- The list holds 3 currencies, among them `USD` (2.593 to buy, 2.607 to sell) and `EUR`
  (2.875 to buy, 2.95 to sell). `toCcy` is `GEL` and `multiplier` is 1 for each.
- Each price is a JSON number with the trailing zeros cut: `2.95`.
- A call with no key, or with a wrong key, answers 401 with the text `Unauthorized`.

## 4. Out of scope

- The card rates, the internet bank and transfer rates, and the official rate through Credo.
- The history of the rates (`rates-by-dates`).

## 5. Definitions

| Term  | Meaning                                                      |
| ----- | ------------------------------------------------------------ |
| Credo | Credo Bank, a commercial bank of Georgia. Bank code `credo`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Credo, as `BOGRATE` 6 lists it.
The statements below add what is particular to Credo.

### `CREDRATE-FR-001`

The server must return the commercial rate of Credo under the bank code `credo`.

**Examples:**

- Normal case: `USD` at `credo` on 2026-10-25. The buy rate is `2.593`, the sell rate `2.607`.
- Unwanted case: the answer gives the rate of the internet bank or of a card for `USD`.

### `CREDRATE-FR-002`

The server must give the price of one unit when Credo prices a currency per 10, 100, or
1000 units.

**Examples:**

- Normal case: an item of `JPY` with the multiplier 100 and a buy price of `1.7`. The
  answer gives `0.017`.
- Limit case: `USD` with the multiplier 1. The answer gives the price as Credo publishes it.
- Unwanted case: the answer gives `1.7` for that item.

### `CREDRATE-FR-003`

The server must treat a currency that Credo does not list against the lari, or lists
with a price of zero, as a currency that Credo does not quote.

**Examples:**

- Normal case: `GBP` at `credo`. The answer names it as a currency that Credo does not quote.
- Limit case: an item of `USD` against `EUR`. The server does not read it as a rate in lari.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `CREDRATE-FR-004`

The server must report Credo as a bank that did not answer when the source states that
the call did not succeed.

**Why:** The script of the page treats an answer with a false `success` as no answer.

### `CREDRATE-FR-005`

The server must read the key of Credo from its configuration. The operator must be able
to turn Credo off. It is on by default. If Credo is on and the key is empty, the server
must not start, and must name the missing setting.

**Examples:**

- Normal case: a key set. The server offers `credo`.
- Limit case: Credo off and no key. The server starts, and does not offer `credo`.
- Unwanted case: no configuration. The server does not start, and names the setting of
  the key.

### `CREDRATE-FR-006`

The server must log an error that names the setting of the key when Credo rejects the key.

**Why:** Credo can change the key that its page publishes. The model reports an outage,
and the operator needs the cause.

## 7. Non-functional requirements

### `CREDRATE-NFR-001`

The server must send at most 1 call to Credo in each second, and at most 60 calls in a
minute. Measure the calls at the source, for any count of requests.

**Why:** Credo publishes no limit. One call returns every currency.

## 8. Invariants

### `CREDRATE-INV-001`

A commercial rate of Credo in an answer has the exact value of the published price,
divided by the published multiplier. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Credo joins `rates_get_commercial_rates`. No new tool.                   |
| `MCP-005`            | MCP tools     | The answer names Credo and the instant at which the server read it.     |
| `CLT-001`            | Clients       | The client is the package `pkg/credo`.                                   |
| `CLT-005`, `CLT-009` | Clients       | The client takes the key with `WithAPIKey`, holds no key, and says that the API is unofficial. |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-007`, `CFG-008` | Configuration | The key is a secret, not in the repository. The switch is on by default. |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | How does the server use the key that the page of Credo publishes? | Temuri | As for the TTC and TBC: the operator sets it, Credo is on by default, and the start fails with no key (2026-10-25). |
| 2   | Who records an answer of the method? | Temuri | The analysis, with one call (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
