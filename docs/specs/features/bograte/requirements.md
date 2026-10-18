---
id: BOGRATE
title: The commercial exchange rates of Bank of Georgia
type: requirements
status: approved
created: 2026-10-19
updated: 2026-10-19
context: rates
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The commercial exchange rates of Bank of Georgia

## 1. Problem

A person compares the commercial rates of the banks before they change money. The
server reads one bank, TBC. Bank of Georgia is the other large bank of Georgia. A
model cannot tell whether TBC or Bank of Georgia pays more for 100 dollars.

Bank of Georgia publishes its commercial rate through a documented API with no key.

## 2. Users

| User                   | Need                                                                         |
| ---------------------- | ---------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of Bank of Georgia, next to the rates of TBC. |
| The operator           | Turns Bank of Georgia off with no change to the code.                        |

## 3. Source

| Vendor          | Source                                                      | Official | Publication period | Terms of use |
| --------------- | ----------------------------------------------------------- | -------- | ------------------ | ------------ |
| Bank of Georgia | `api.businessonline.ge/api/rates/commercial/{currency}`     | Yes. Documented at `api.bog.ge/docs/en/bonline/rates/commercial-currency.md`. | Not published. The bank changes its rates during the day. | No key. At most 60 calls in a minute. Past the limit, the source answers 429 with a `Retry-After` header in seconds. |

Observed behavior of the source on 2026-10-19:

- One call returns one currency: `{"Buy":2.56500000,"Sell":2.62900000}` for `USD`.
- Each price is for one unit. `JPY` has a buy price of 0.01577, `IDR` of 0.000126.
- Each price has eight decimal places, with the trailing zeros.
- The source accepts a code in lower case.
- `GEL` and an unknown code answer 204 with no body.
- A path with no currency answers 404.
- The answer holds no instant.
- `Buy` and `Sell` are equal to `buyRate` and `sellRate` of the page
  `bankofgeorgia.ge/ka/valutis-kursi`, the rate of the bank. The rate of the digital
  channels is a different method.

## 4. Out of scope

- The rate of the digital channels, and the cross rates.
- The official rate through Bank of Georgia. `NBGRATE` reads it from the NBG.
- The other methods of the API that need a business account.
- A conversion of an amount, and a history of commercial rates.

## 5. Definitions

| Term            | Meaning                                        |
| --------------- | ---------------------------------------------- |
| Bank of Georgia | A commercial bank of Georgia. Bank code `bog`. |

`TBCRATE` defines bank, commercial rate, buy rate, sell rate, and bank code.

## 6. Functional requirements

Every statement of `TBCRATE` about a bank holds for Bank of Georgia: the answer for
every bank, the unknown bank, the missing currency, the price of one unit, the meaning
of buy and sell, the read instant, the failed bank, the stale rates, the limit of
10 currencies, and one call for each currency in each 5 minutes. The statements below
add what is particular to Bank of Georgia.

### `BOGRATE-FR-001`

The server must return the commercial rate of Bank of Georgia under the bank code `bog`.

**Examples:**

- Normal case: `USD` at `bog` on 2026-10-19. The buy rate is `2.565`, the sell rate `2.629`.
- Limit case: no bank named. The answer holds `bog` and `tbc`.
- Unwanted case: the answer gives `2.567` for the buy rate of `USD`. That is the rate of
  the digital channels, not the rate of the bank.

### `BOGRATE-FR-002`

The server must treat a currency that Bank of Georgia answers with no rate, or with a
price of zero, as a currency that the bank does not quote.

**Examples:**

- Normal case: `GEL` at `bog`. The answer names it as a currency that the bank does not quote.
- Limit case: `XYZ` at `bog`. The answer names it in the same way.
- Unwanted case: a price of zero. The server must not return it as a rate.

### `BOGRATE-FR-003`

The server must read Bank of Georgia with no key. The operator must be able to turn
Bank of Georgia off. It is on by default.

**Why:** The source needs no key, so the absence of a key cannot turn the bank off.

**Examples:**

- Normal case: no configuration. The server offers `bog`.
- Limit case: Bank of Georgia off, and no key of TBC. The server offers no commercial rate.
- Unwanted case: a request names `bog` while it is off. The server refuses it as a bank
  that it cannot read.

## 7. Non-functional requirements

### `BOGRATE-NFR-001`

The server must send at most 60 calls to Bank of Georgia in a minute. Measure the calls
at the source, for any count of requests.

**Why:** The source allows 60 calls in a minute.

## 8. Invariants

### `BOGRATE-INV-001`

A commercial rate of Bank of Georgia in an answer has the exact value of the published
price. The server rounds no digit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                        |
| -------------------- | ------------- | ----------------------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `rates`. Bank of Georgia is its third vendor.                  |
| `DOM-006`, `MCP-009` | Domain        | A rate is an exact decimal, a string in the answer.                           |
| `MCP-002`            | MCP tools     | Bank of Georgia joins `rates_get_commercial_rates`. No new tool.              |
| `MCP-005`            | MCP tools     | The answer names Bank of Georgia and the instant at which the server read it. |
| `CLT-001`            | Clients       | The client is a module in `pkg/bog`.                                          |
| `CLT-005`            | Clients       | The User-Agent names the repository.                                          |
| `CFG-004`            | Configuration | The switch is a field of the configuration of `rates`.                        |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency.       |

## 10. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
