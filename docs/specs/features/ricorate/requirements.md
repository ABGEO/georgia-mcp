---
id: RICORATE
title: The commercial exchange rates of Rico Credit
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Rico Credit

## 1. Problem

A person who changes cash in Georgia often goes to an exchange point, not to a bank.
Rico Credit publishes the rates at which it changes money. On 2026-10-21 it sold the
dollar for 2.600, and TBC for 2.627. A model that reads only the banks misses that price.

Rico Credit is a microfinance organization, not a bank. A model must not call it a bank.

## 2. Users

| User                   | Need                                                                           |
| ---------------------- | ------------------------------------------------------------------------------ |
| A person with a client | The buy rate and the sell rate of Rico Credit, next to the rates of the banks. |
| The operator           | Turns Rico Credit off with no change to the code.                              |

## 3. Source

| Vendor      | Source                                  | Official                  | Publication period | Terms of use |
| ----------- | --------------------------------------- | ------------------------- | ------------------ | ------------ |
| Rico Credit | The home page `www.rico.ge/ka/`, the table "ვალუტის კურსები" | No. A web page for people. | Not published. | No key. No published limit. `robots.txt` is empty. |

Rico Credit publishes no API. A search on 2026-10-21 found no developer portal.

Observed behavior of the home page on 2026-10-21:

- The page renders the table on the server. A plain GET returns it.
- The table has four columns: the currency, the unit, the buy price, and the sell price.
- The table lists 12 currencies, among them `USD`, `EUR`, `GBP`, `CHF`, `TRY`, `CAD`,
  `AED`, `AMD`, `AZN`, `PLN`, and `CNY`. It also lists one cross rate, `USD-EUR`.
- The unit is 1, 10, or 1000: `PLN` is per 10, `AMD` per 1000. The prices are for that unit.
- Each price has four decimal places, with the trailing zeros: `2.6000`.
- The page also holds the same rates for its calculator, divided by the unit and
  rounded by the page: `AMD` buys at `0.0068`.

## 4. Out of scope

- The cross rate `USD-EUR`.
- The other products of Rico Credit: loans, deposits, and money transfers.

## 5. Definitions

| Term                       | Meaning                                                                       |
| -------------------------- | ----------------------------------------------------------------------------- |
| microfinance organization  | A company of Georgia that the NBG registers to lend money. It is not a bank.  |
| Rico Credit                | Rico Credit, a microfinance organization of Georgia. Bank code `rico`.        |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code. This
feature widens bank: in the context `rates`, a bank is any company that publishes
commercial rates. `RICORATE-FR-004` keeps the kind of the company visible.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Rico Credit, as `BOGRATE` 6 lists
it. The statements below add what is particular to Rico Credit.

### `RICORATE-FR-001`

The server must return the commercial rate of Rico Credit under the bank code `rico`.

**Examples:**

- Normal case: `USD` at `rico` on 2026-10-21. The buy rate is `2.596`, the sell rate `2.6`.
- Limit case: no bank named. The answer holds `rico` next to the banks.
- Unwanted case: the answer holds a rate of `USD-EUR`. A commercial rate is against the lari.

### `RICORATE-FR-002`

The server must give the price of one unit when Rico Credit prices a currency per 10,
100, or 1000 units.

**Examples:**

- Normal case: Rico Credit buys `10 PLN` for `6.1000`. The answer gives `0.61`.
- Limit case: Rico Credit buys `1000 AMD` for `6.8000`. The answer gives `0.0068`.
- Unwanted case: the answer gives `6.1` for `PLN`.

### `RICORATE-FR-003`

The server must treat a currency that the table does not list, or lists with a price
of zero, as a currency that Rico Credit does not quote.

**Examples:**

- Normal case: `JPY` at `rico`. The answer names it as a currency that Rico Credit does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `RICORATE-FR-004`

The server must name the kind of each company that is not a bank in the list of banks
that it gives to a client.

**Why:** A person who reads "the banks" expects a bank. The kind tells the model what
the person deals with.

**Examples:**

- Normal case: the list reads `rico (Rico Credit, a microfinance organization)`.
- Limit case: a bank. The list reads `bog (Bank of Georgia)`, as before.

### `RICORATE-FR-005`

The server must read Rico Credit with no key. The operator must be able to turn Rico
Credit off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `rico`.
- Unwanted case: a request names `rico` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `RICORATE-NFR-001`

The server must send at most 1 call to Rico Credit in each second, and at most 60
calls in a minute. Measure the calls at the source, for any count of requests.

**Why:** Rico Credit publishes no limit. One call returns every currency.

## 8. Invariants

### `RICORATE-INV-001`

A commercial rate of Rico Credit in an answer has the exact value of the published
price, divided by the published unit. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Rico Credit joins `rates_get_commercial_rates`. No new tool. The description of the tool changes. |
| `MCP-005`            | MCP tools     | The answer names Rico Credit and the instant at which the server read it. |
| `CLT-001`            | Clients       | The client is a module in `pkg/rico`.                                   |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page.                                            |
| `CLT-009`            | Clients       | The client says that the source is unofficial.                          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | A company that is not a bank under the word bank, with its kind in the list (`RICORATE-FR-004`). The alternative renames bank in the tool: a change of the input and the output. Is the wider term correct? | Temuri | Yes, on 2026-10-21. The wider term, with the kind in the list. |

## Retired identifiers

This file has no retired identifier.
