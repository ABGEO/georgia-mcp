---
id: LIBRATE
title: The commercial exchange rates of Liberty Bank
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Liberty Bank

## 1. Problem

A person compares the commercial rates of the banks before they change money. The
server reads TBC and Bank of Georgia. Liberty Bank pays pensions and social benefits
in Georgia, so many people have an account there and change money at its branches. A
model cannot tell them the price of Liberty.

## 2. Users

| User                   | Need                                                                      |
| ---------------------- | ------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Liberty, next to the rates of the other banks. |
| The operator           | Turns Liberty off with no change to the code.                             |

## 3. Source

| Vendor       | Source                              | Official                  | Publication period | Terms of use |
| ------------ | ----------------------------------- | ------------------------- | ------------------ | ------------ |
| Liberty Bank | The home page `www.libertybank.ge/ka/`, the block "ვალუტის კურსი" | No. A web page for people. | Not published. The block gives the date of the day. | No key. No published limit. `robots.txt` allows every path. |

Liberty publishes no API. A search on 2026-10-21 found no developer portal.

The page that the owner named, `.../kompaniis-shesakheb/valutis-kursi`, holds no
current rate. It draws a chart from `/api/currencyrates/history`, an unofficial
method. That method answers JSONP, one currency in each call, with one point for each
day and no flag for the current rate. The home page holds the current rates of every
currency in one answer, so this feature reads the home page.

Observed behavior of the home page on 2026-10-21:

- The block has four columns: the official rate of the NBG, the commercial rate, the
  rate of the internet bank for a person up to 100 000 lari, and a special rate for
  money transfers. Each column except the official one has a buy price and a sell price.
- The block lists 8 currencies, among them `USD`, `EUR`, `GBP`, `CHF`, `TRY`, `AZN`,
  and `AMD`. Every currency has a commercial rate. `CHF`, `TRY`, `AZN`, and `AMD` have
  no rate of the internet bank and no special rate for transfers.
- Each row states its unit: `1 USD`, `1000 AMD`. The prices are for that unit.
- Each price has four decimal places, with the trailing zeros: `2.5470`.
- The commercial rate of `USD` is 2.5470 to buy and 2.6470 to sell. The internet bank
  gives 2.5870 and 2.6070.
- The page renders the block on the server. A plain GET returns it.

## 4. Out of scope

- The rate of the internet bank, and the special rate for transfers.
- The official rate through Liberty. `NBGRATE` reads it from the NBG.
- The history of the commercial rates of Liberty.

## 5. Definitions

| Term    | Meaning                                                     |
| ------- | ----------------------------------------------------------- |
| Liberty | Liberty Bank, a commercial bank of Georgia. Bank code `liberty`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code. The
glossary already holds Liberty for the offers.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Liberty, as `BOGRATE` 6 lists it.
The statements below add what is particular to Liberty.

### `LIBRATE-FR-001`

The server must return the commercial rate of Liberty under the bank code `liberty`.

**Why:** A person changes money at a branch at the commercial rate. `TBCRATE` 4 leaves
out the rate of an online channel when it differs.

**Examples:**

- Normal case: `USD` at `liberty` on 2026-10-21. The buy rate is `2.547`, the sell rate `2.647`.
- Limit case: no bank named. The answer holds `bog`, `liberty`, and `tbc`.
- Unwanted case: the answer gives `2.587` for the buy rate of `USD`. That is the rate
  of the internet bank.

### `LIBRATE-FR-002`

The server must give the price of one unit when Liberty prices a currency per 10, 100,
or 1000 units.

**Examples:**

- Normal case: Liberty buys `1000 AMD` for `6.2220`. The answer gives `0.006222`.
- Limit case: `USD` per 1 unit. The answer gives the price as Liberty publishes it.
- Unwanted case: the answer gives `6.222` for `AMD`. A comparison with another bank is
  then wrong by a factor of 1000.

### `LIBRATE-FR-003`

The server must treat a currency that the block does not list, or lists with a price
of zero, as a currency that Liberty does not quote.

**Examples:**

- Normal case: `JPY` at `liberty`. The answer names it as a currency that Liberty does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `LIBRATE-FR-004`

The server must read Liberty with no key. The operator must be able to turn Liberty
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `liberty`.
- Unwanted case: a request names `liberty` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `LIBRATE-NFR-001`

The server must send at most 1 call to Liberty in each second, and at most 60 calls in
a minute. Measure the calls at the source, for any count of requests.

**Why:** Liberty publishes no limit. One call returns every currency, so one call each
second serves any request. The limit protects a page for people.

## 8. Invariants

### `LIBRATE-INV-001`

A commercial rate of Liberty in an answer has the exact value of the published price,
divided by the published unit. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Liberty joins `rates_get_commercial_rates`. No new tool.                 |
| `MCP-005`            | MCP tools     | The answer names Liberty and the instant at which the server read it.   |
| `CLT-001`            | Clients       | `pkg/liberty` holds the offers. The client of the rates is a second module. |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | The home page instead of the page you named: is that correct? The page you named holds only a chart. | Temuri | Yes, on 2026-10-21. The feature reads the home page. |

## Retired identifiers

This file has no retired identifier.
