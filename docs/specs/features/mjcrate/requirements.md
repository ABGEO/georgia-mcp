---
id: MJCRATE
title: The commercial exchange rates of MJC
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of MJC

## 1. Problem

A person compares the commercial rates before they change money. On 2026-10-25 MJC
sold the dollar for 2.598, and Halyk Bank for 2.612. A model that cannot read MJC misses
that price.

MJC is a microfinance organization, not a bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                   |
| ---------------------- | ---------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of MJC, next to the rates of the banks. |
| The operator           | Turns MJC off with no change to the code.                              |

## 3. Source

| Vendor | Source                                                | Official                   | Publication period | Terms of use |
| ------ | ----------------------------------------------------- | -------------------------- | ------------------ | ------------ |
| MJC    | The page `mjc.ge/rates`, the table "ვალუტის კურსები" | No. A web page for people. | Not published.     | No key. No published limit. `robots.txt` allows every page. |

MJC publishes no API. The page offers no other source of the rates.

Observed behavior of the page on 2026-10-25:

- The page renders the table on the server. A plain GET returns it.
- The table has four columns with the titles `ვალუტა`, `ყიდვა`, `გაყიდვა`, and
  `ეროვნული`: the unit with the currency, the buy price, the sell price, and the
  official rate of the NBG.
- The table lists 17 currencies, among them `USD`, `EUR`, `GBP`, `CHF`, `TRY`, `AMD`,
  `PLN`, `KWD`, `SAR`, `KZT`, and `CNY`.
- Each row states its unit: `1 USD`, `10 PLN`, `1000 AMD`. The prices are for that unit.
- Each price has five decimal places, with the trailing zeros: `2.59500`.
- The official column has four decimal places. It reads `N/A` for `SAR`.
- The page states no date and no time. The answer forbids a shared cache (`no-cache, private`).
- The title names the company: "მისო „MJC (ემჯეის კრედიტი)”".

## 4. Out of scope

- The official column of the table. `NBGRATE` reads the official rate at the source.
- The calculator of the page.
- The other products of MJC: gold loans and other loans.

## 5. Definitions

| Term | Meaning                                                                   |
| ---- | ------------------------------------------------------------------------- |
| MJC  | MJC Credit, a microfinance organization of Georgia. Bank code `mjc`.      |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE`
defines bank, bank kind, and microfinance organization.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for MJC, as `BOGRATE` 6 lists it.
The statements below add what is particular to MJC.

### `MJCRATE-FR-001`

The server must return the commercial rate of MJC under the bank code `mjc`.

**Examples:**

- Normal case: `USD` at `mjc` on 2026-10-25. The buy rate is `2.595`, the sell rate `2.598`.
- Limit case: `EUR` with a buy price of `2.90700`. The answer gives `2.907`.
- Unwanted case: the answer gives the official rate `2.5971` as a price of MJC.

### `MJCRATE-FR-002`

The server must give the price of one unit when MJC prices a currency per 10, 100, or
1000 units.

**Examples:**

- Normal case: MJC buys `10 PLN` for `6.34000`. The answer gives `0.634`.
- Limit case: MJC buys `1000 AMD` for `7.00000`. The answer gives `0.007`.
- Unwanted case: the answer gives `6.34` for `PLN`. A comparison with another bank is
  then wrong by a factor of 10.

### `MJCRATE-FR-003`

The server must treat a currency that the table does not list, or lists with a price
of zero, as a currency that MJC does not quote.

**Examples:**

- Normal case: `JPY` at `mjc`. The answer names it as a currency that MJC does not quote.
- Limit case: `SAR`, with `N/A` in the official column. The answer gives its buy rate and its sell rate.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `MJCRATE-FR-004`

The server must name MJC as a microfinance organization in each place that lists the
banks.

**Why:** `RICORATE-FR-004` gives the rule.

### `MJCRATE-FR-005`

The server must read MJC with no key. The operator must be able to turn MJC off. It is
on by default.

**Examples:**

- Normal case: no configuration. The server offers `mjc`.
- Unwanted case: a request names `mjc` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `MJCRATE-NFR-001`

The server must send at most 1 call to MJC in each second, and at most 60 calls in a
minute. Measure the calls at the source, for any count of requests.

**Why:** MJC publishes no limit. One call returns every currency. The limit protects a
page for people.

## 8. Invariants

### `MJCRATE-INV-001`

A commercial rate of MJC in an answer has the exact value of the published price,
divided by the published unit. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | MJC joins `rates_get_commercial_rates`. No new tool.                     |
| `MCP-005`            | MCP tools     | The answer names MJC and the instant at which the server read it.       |
| `CLT-001`            | Clients       | The client is the package `pkg/mjc`.                                     |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Which bank code and name? | Temuri | `mjc` and MJC (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
