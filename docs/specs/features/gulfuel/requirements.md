---
id: GULFUEL
title: The fuel prices of Gulf
type: requirements
status: approved
created: 2026-10-22
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-22
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Gulf

## 1. Problem

A driver asks "how much is G-Force Premium at Gulf today". The server answers for Wissol,
Rompetrol, SOCAR, Connect, Lukoil, Neogas, and Portal. Gulf is a fuel company with 154
stations across Georgia, and it publishes the date of each change of its prices.

## 2. Users

| User                   | Need                                                                     |
| ---------------------- | ------------------------------------------------------------------------ |
| A person with a client | The current price of each product of Gulf, and the date of its last change. |
| The operator           | Turns Gulf off with no change to the code.                               |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Gulf | The table of the page `gulf.ge/en/fuel_prices` | No. A web page for people. | At each change. On 2026-10-22 the newest change was 2026-10-19. | No key. No published terms and no published limit. `robots.txt` allows the page. |

Gulf publishes no API. The page "Fuel prices" shows each change, the newest first.

Observed behavior of the page on 2026-10-22:

- A bot filter guards the site. It refuses a reader whose TLS handshake is not the
  handshake of a browser, with the status 403. A reader with the handshake of a browser
  gets the page, with the User-Agent of this server, and with no cookie.
- One table: the date of the change (`2026-10-19`, no time), then 5 columns of the tier
  "Gulf", then the same 5 products in the tier "Gulf+". A tab of the page shows one tier.
- 100 rows on each page, 11 pages, 1,077 rows from 2012-11-30. 111 KB. No two rows share a date.
- The prices are decimal text with two places, in lari for one litre: `4.69`.
- The page `/en/` names the products in English, the page `/ge/` in Georgian.
- The tier "Gulf+" applies at 44 stations whose name starts with "Gulf +". It has the
  price `0.00` in each row before 2026-10-02.
- The page shows no self-service price.

| Product              | Georgian name          | Gulf | Gulf+ | Octane (the site) |
| -------------------- | ---------------------- | ---- | ----- | ----------------- |
| G-Force Super        | G-Force სუპერი          | 4.69 | 4.59  | 98                |
| G-Force Premium      | G-Force პრემიუმი        | 4.39 | 4.29  | 95                |
| G-Force Euro Regular | G-Force ევრო რეგულარი    | 4.09 | 3.99  | 93                |
| G-Force Euro Diesel  | G-Force ევრო დიზელი      | 5.05 | 4.95  |                   |
| Euro Diesel          | ევრო დიზელი              | 4.99 | 4.89  |                   |

## 4. Out of scope

- The prices of the tier "Gulf+". A later feature can add them.
- A history of prices.
- A price of the card of the loyalty program "Gulf Club".
- Compressed natural gas. The page gives it no price.

## 5. Definitions

| Term         | Meaning                                                                    |
| ------------ | -------------------------------------------------------------------------- |
| Gulf         | Gulf Georgia, a fuel company of Georgia. Company code `gulf`.              |
| browser handshake | The TLS handshake that a browser sends. A bot filter can refuse any other. |

## 6. Functional requirements

### `GULFUEL-FR-001`

The server must return the standard price of each product of Gulf, from the tier "Gulf"
of its newest change.

**Examples:**

- Normal case: `gulf` on 2026-10-22. G-Force Premium has the standard price `4.39`.
- Limit case: the answer holds no price `4.29` for G-Force Premium. That is the price of Gulf+.
- Unwanted case: a price that is not a number, or a date that is not a date. The server
  treats the answer of Gulf as a failure.

### `GULFUEL-FR-002`

The server must give each price of Gulf the date of the newest change as its effective date.

**Why:** Gulf publishes the date of each change. A driver can tell how old a price is.

**Examples:**

- Normal case: Euro Diesel at `gulf` on 2026-10-22. The effective date is `2026-10-19`.
- Limit case: the description of the tool lists Gulf with no note on the date.

### `GULFUEL-FR-003`

The server must leave out a product of Gulf with a price of zero, and return the other products.

**Why:** Gulf writes `0.00` for a price that it does not apply.

**Examples:**

- Normal case: G-Force Super at `0.00`. The answer holds the other 4.
- Unwanted case: every product at zero, or a negative price. The server treats the
  answer of Gulf as a failure.

### `GULFUEL-FR-004`

The server must give each product of Gulf its grade.

**Examples:**

- Normal case: G-Force Euro Regular has the grade `regular`. The site gives it the octane number 93.
- Limit case: G-Force Euro Diesel and Euro Diesel have the grade `diesel`.
- Unwanted case: a product that the server does not know. It has the grade `other`.

| Product              | Grade     |
| -------------------- | --------- |
| G-Force Super        | `super`   |
| G-Force Premium      | `premium` |
| G-Force Euro Regular | `regular` |
| G-Force Euro Diesel  | `diesel`  |
| Euro Diesel          | `diesel`  |

### `GULFUEL-FR-005`

The server must give the Georgian name of each known product of Gulf when the request
asks `ka`, and the English name of Gulf when it asks `en`.

**Why:** Gulf names each product in both languages, on two pages.

**Examples:**

- Normal case: `ka`. The name is "G-Force პრემიუმი".
- Limit case: `en`. The name is "G-Force Premium".
- Unwanted case: `ka`, and a product that the server does not know. The name is the
  English name of Gulf.

### `GULFUEL-FR-006`

The server must treat a page of Gulf with no price table, or with no row, as a failure of Gulf.

**Examples:**

- Normal case: the page holds 100 rows. The server returns the newest.
- Unwanted case: the page holds no table. The server reports that Gulf did not answer,
  and returns the prices of the other companies.

### `GULFUEL-FR-007`

The server must give no self-service price for Gulf.

**Why:** The price of Gulf+ is the price of a set of stations. The page does not say
that they serve themselves. A model must not offer it as a self-service price.

### `GULFUEL-FR-008`

The server must read the page of Gulf with no cookie and no step of a person.

**Why:** The bot filter of the site refuses a reader with no browser handshake.

**Examples:**

- Normal case: the filter accepts the browser handshake. The server returns the prices.
- Unwanted case: the filter refuses the server with 403. The server reports that Gulf
  did not answer, and returns the prices of the other companies.

### `GULFUEL-FR-009`

The server must read Gulf with no key. The operator must be able to turn Gulf off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `gulf`.
- Unwanted case: a request names `gulf` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Gulf. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Gulf. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                           |
| -------------------- | ------------- | ---------------------------------------------------------------- |
| `DOM-005`            | Domain        | The tier, the newest row, the grade, and the Georgian name belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.             |
| `MCP-005`            | MCP tools     | Each answer names Gulf and the instant at which the server read it. |
| `CLT-002`, `CLT-004` | Clients       | The client sends a browser handshake. The rules admit it from 2026-10-22. |
| `CLT-005`            | Clients       | The client sends the User-Agent of this server, not of a browser. |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`. |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.            |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.          |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server return the prices of Gulf+? | Temuri | No. The tier "Gulf" only (2026-10-22). |
| 2   | Which library sends the browser handshake? Does it need a decision record? | Temuri | A library of TLS under `net/http`. The rules of `CLT` change with no record (2026-10-22). |
| 3   | Which User-Agent? | Temuri | The User-Agent of this server, while the filter accepts it. Else the one of a browser (2026-10-22). |

## Retired identifiers

This file has no retired identifier.
