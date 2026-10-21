---
id: KURSRATE
title: The commercial exchange rates of Kursi.ge
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: rates
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Kursi.ge

## 1. Problem

A person with a Georgian bank account can change money online at Kursi.ge, an exchange
bureau with no branch. The person sends lari or a currency from their bank account,
and Kursi.ge sends the other currency back. On 2026-10-21 Kursi.ge sold the dollar for
2.601, and Bank of Georgia for 2.629. A model cannot tell the person that price.

## 2. Users

| User                   | Need                                                                     |
| ---------------------- | ------------------------------------------------------------------------ |
| A person with a client | The buy rate and the sell rate of Kursi.ge, next to the rates of the banks. |
| The operator           | Turns Kursi.ge off with no change to the code.                           |

## 3. Source

| Vendor   | Source                                          | Official | Publication period | Terms of use |
| -------- | ----------------------------------------------- | -------- | ------------------ | ------------ |
| Kursi.ge | `api-core.kursi.ge/api/public/currencies`        | No. An unofficial API: the page `kursi.ge/ka/` reads it. | Not published. The answer forbids a cache: `Cache-Control: no-cache, no-store`. | No key. No published limit. |

Kursi.ge documents a merchant API at `docs.kursi.ge/merchant/`. Its method
`GET /api/currency` gives the same rates, but needs the token of a merchant account.
The method of the page needs no token, and is not in the documentation.

Observed behavior of the source on 2026-10-21:

- One call returns every pair: 8 pairs. 4 of them are against the lari, among them
  `USD`, `EUR`, and `CNY`. The others are between two foreign currencies.
- A pair has `baseCurrencyCode`, `secondaryCurrencyCode`, a `buyRate`, and a
  `sellRate`. For the base `GEL`, each rate is the price in lari of one unit of the
  secondary currency.
- `buyRate` is the price that Kursi.ge pays. Proof: the same answer holds the rates of
  TBC and Bank of Georgia under `bankRates`, and those of Bank of Georgia equal its own
  API at the same minute: `2.565` to buy and `2.629` to sell.
- A pair also holds the official rate of the NBG, and its change.
- The JSON numbers drop the trailing zeros. The answer holds no instant.
- The page says that Kursi.ge works from 08:00 to 00:00, every day.

## 4. Out of scope

- The pairs between two foreign currencies.
- The rates of TBC and Bank of Georgia that Kursi.ge repeats. `TBCRATE` and `BOGRATE`
  read them from each bank.
- A rate that Kursi.ge agrees with a registered person.
- An order, a conversion, or any method that needs an account.

## 5. Definitions

| Term                   | Meaning                                                                       |
| ---------------------- | ----------------------------------------------------------------------------- |
| online exchange bureau | A company of Georgia that the NBG registers to change money, and that changes it by bank transfer only. |
| Kursi.ge               | Kursi.ge, an online exchange bureau of Georgia. Bank code `kursi`.            |

`TBCRATE` defines commercial rate, buy rate, sell rate, and bank code. `RICORATE` 5
widens bank.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Kursi.ge, as `BOGRATE` 6 lists it.
`RICORATE-FR-004` holds for Kursi.ge. The statements below add what is particular to
Kursi.ge.

### `KURSRATE-FR-001`

The server must return the commercial rate of Kursi.ge under the bank code `kursi`.

**Examples:**

- Normal case: `USD` at `kursi` on 2026-10-21. The buy rate is `2.594`, the sell rate `2.601`.
- Limit case: `CNY`. The buy rate is `0.3858`, the sell rate `0.3925`.
- Unwanted case: the answer gives `2.565` for the buy rate of `USD`. That is the rate of
  Bank of Georgia that Kursi.ge repeats.

### `KURSRATE-FR-002`

The server must read only the pairs whose base is the lari.

**Examples:**

- Normal case: the pair `GEL` and `EUR` gives the rate of `EUR`.
- Unwanted case: the pair `USD` and `EUR` gives a rate of `EUR` in dollars. The server
  must not return it as a rate of `EUR`.

### `KURSRATE-FR-003`

The server must treat a currency with no pair against the lari, or with a price of
zero, as a currency that Kursi.ge does not quote.

**Examples:**

- Normal case: `GBP` at `kursi`. The answer names it as a currency that Kursi.ge does not quote.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `KURSRATE-FR-004`

The server must name Kursi.ge as an online exchange bureau in the list of banks that it
gives to a client.

**Why:** A person cannot walk in with cash. The person needs a bank account and a transfer.

**Examples:**

- Normal case: the list reads `kursi (Kursi.ge, an online exchange bureau)`.

### `KURSRATE-FR-005`

The server must read Kursi.ge with no key. The operator must be able to turn Kursi.ge
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `kursi`.
- Unwanted case: a request names `kursi` while it is off. The server refuses it as a
  bank that it cannot read.

## 7. Non-functional requirements

### `KURSRATE-NFR-001`

The server must send at most 1 call to Kursi.ge in each second, and at most 60 calls in
a minute. Measure the calls at the source, for any count of requests.

**Why:** Kursi.ge publishes no limit. One call returns every currency.

## 8. Invariants

### `KURSRATE-INV-001`

A commercial rate of Kursi.ge in an answer has the exact value of the published price.
The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                  |
| -------------------- | ------------- | ----------------------------------------------------------------------- |
| `MCP-002`            | MCP tools     | Kursi.ge joins `rates_get_commercial_rates`. No new tool.                |
| `MCP-005`            | MCP tools     | The answer names Kursi.ge and the instant at which the server read it.  |
| `CLT-001`            | Clients       | The client is a module in `pkg/kursi`.                                   |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial, and stays at `v0`.          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Kursi.ge works from 08:00 to 00:00. The answer outside those hours is not known. Must the server mark a rate read outside them? The proposal: no. A check at 00:30 decides. | Temuri | No marking, on 2026-10-21. |
| 2   | Kursi.ge repeats the rates of TBC. A deployment with no key of TBC could read them here. The proposal: no. The rate of a bank comes from the bank. | Temuri | No, on 2026-10-21. The rate of a bank comes from the bank. |

## Retired identifiers

This file has no retired identifier.
