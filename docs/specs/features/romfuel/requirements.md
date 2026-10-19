---
id: ROMFUEL
title: The fuel prices of Rompetrol
type: requirements
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Rompetrol

## 1. Problem

A driver asks "is diesel cheaper at Rompetrol or at Wissol". The server answers for
Wissol only. Rompetrol is a second fuel company with stations across Georgia.

Rompetrol publishes no API. Its web page shows the current prices in a table, with no
date and no product code. The page in English leaves most product names in Georgian.

## 2. Users

| User                   | Need                                                                              |
| ---------------------- | --------------------------------------------------------------------------------- |
| A person with a client | The current price of each product of Rompetrol, and its self-service price.      |
| The operator           | Turns Rompetrol off with no change to the code.                                  |

## 3. Source

| Vendor    | Source                                      | Official | Publication period | Terms of use |
| --------- | ------------------------------------------- | -------- | ------------------ | ------------ |
| Rompetrol | The price table of the page `www.rompetrol.ge/` | No. A web page, not an API. | Not published. The page shows no date. | No key. No published terms and no published limit. `robots.txt` holds no rule. |

Observed behavior of the source on 2026-10-19:

- The home page holds the table "საწვავის ფასი": a product name and a price in lari
  for one litre. The table has 10 rows.
- Each of the 5 products has a row, and a second row directly after it: the price at
  the self-service stations. The second row starts with `- ` and ends with
  `(თვითმომსახურების სადგურებზე)`.
- The name in the second row does not always repeat the name of the product:
  `- ევრო რეგულარი (...)` follows `efix ევრო რეგულარი`.
- The prices are decimal text: `4.65`, `3.92`.
- The table holds no product code, no date, and no instant. The answer of the page has no
  date of change.
- The page `www.rompetrol.ge/en/personal` names 4 of 10 rows in English.
- The list of stations of the site holds no price.

| Product            | Standard price | Self-service price |
| ------------------ | -------------- | ------------------ |
| efix სუპერი         | 4.65           | 4.55               |
| efix ევრო პრემიუმი   | 4.32           | 4.22               |
| efix ევრო რეგულარი   | 4.02           | 3.92               |
| efix ევრო დიზელი     | 4.99           | 4.89               |
| ევრო დიზელი         | 4.92           | 4.82               |

## 4. Out of scope

- A history of prices.
- A price at one station, a price of a card, or a price of a loyalty program.
- The list of the stations of Rompetrol.
- A date that the server guesses for a price.

## 5. Definitions

| Term      | Meaning                                                   |
| --------- | --------------------------------------------------------- |
| Rompetrol | A fuel company of Georgia. Company code `rompetrol`.      |

## 6. Functional requirements

### `ROMFUEL-FR-001`

The server must return the standard price of each product of Rompetrol.

**Examples:**

- Normal case: `rompetrol` on 2026-10-19. efix Euro Regular has the standard price `4.02`.
- Limit case: efix Super has the standard price `4.65`.
- Unwanted case: a product with a price that is not a number, or a price of zero. The
  server treats the answer of Rompetrol as a failure.

### `ROMFUEL-FR-002`

The server must give a product of Rompetrol the self-service price that Rompetrol lists
directly after the product.

**Why:** The name of a self-service row does not always repeat the name of its product.

**Examples:**

- Normal case: efix Euro Regular has the self-service price `3.92`. Its row names
  "ევრო რეგულარი" with no "efix".
- Limit case: a product with no self-service row after it. The answer holds no
  self-service price for it.
- Unwanted case: a self-service row with no product before it. The server treats the
  answer of Rompetrol as a failure.

### `ROMFUEL-FR-003`

The server must give each product of Rompetrol its grade.

**Examples:**

- Normal case: efix Super has the grade `super`. Its research octane number is 98.
- Limit case: efix Euro Diesel and Euro Diesel have the grade `diesel`.
- Unwanted case: a product that the server does not know. It has the grade `other`.

| Product            | Grade     |
| ------------------ | --------- |
| efix სუპერი         | `super`   |
| efix ევრო პრემიუმი   | `premium` |
| efix ევრო რეგულარი   | `regular` |
| efix ევრო დიზელი     | `diesel`  |
| ევრო დიზელი         | `diesel`  |

### `ROMFUEL-FR-004`

The server must give no effective date to a price of a fuel company that publishes none.

**Why:** A date that the server guesses is false. The read instant tells the age of
the price.

**Examples:**

- Normal case: efix Super at `rompetrol`. The price has no effective date.
- Limit case: a request for `rompetrol` and `wissol`. The prices of Wissol keep their
  effective dates.

### `ROMFUEL-FR-005`

The server must state, for each fuel company, whether the company publishes an effective
date. A client reads the statement before a call.

**Why:** A model must not read the absence of a date as an error.

### `ROMFUEL-FR-006`

The server must give the English name of each known product of Rompetrol when the request
asks `en`. It must give the Georgian name of a product that it does not know.

**Why:** The page in English leaves 6 of 10 rows in Georgian.

**Examples:**

- Normal case: `en`. The name is "efix Euro Regular".
- Limit case: `ka`. The name is "efix ევრო რეგულარი".
- Unwanted case: `en`, and a product that the server does not know. The name is the
  Georgian name of the page.

| Georgian name      | English name      |
| ------------------ | ----------------- |
| efix სუპერი         | efix Super        |
| efix ევრო პრემიუმი   | efix Euro Premium |
| efix ევრო რეგულარი   | efix Euro Regular |
| efix ევრო დიზელი     | efix Euro Diesel  |
| ევრო დიზელი         | Euro Diesel       |

### `ROMFUEL-FR-007`

The server must treat a page of Rompetrol with no price table, or with an empty one, as a
failure of Rompetrol.

**Why:** Rompetrol can change its page with no notice. An empty list says that
Rompetrol sells nothing.

**Examples:**

- Normal case: the page holds the table. The server returns the 5 products.
- Unwanted case: the page holds no table. The server reports that Rompetrol did not
  answer, and returns the prices of the other companies.

### `ROMFUEL-FR-008`

The server must read Rompetrol with no key. The operator must be able to turn Rompetrol
off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `rompetrol` and `wissol`.
- Unwanted case: a request names `rompetrol` while it is off. The server refuses it as
  a company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Rompetrol. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Rompetrol. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                       |
| -------------------- | ------------- | ---------------------------------------------------------------------------- |
| `DOM-005`            | Domain        | The pairing of rows, the grade, and the English name belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.                         |
| `MCP-005`            | MCP tools     | Each answer names Rompetrol and the instant at which the server read it.      |
| `CLT-002`, `CLT-004` | Clients       | The source is a web page. The two rules admit a JSON or XML body only. The specification needs an ADR that changes them. |
| `CLT-005`            | Clients       | The client asks for a web page, not for JSON. The same ADR changes the rule. |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`.              |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.                        |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.                      |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the owner accept a web page as the source? Rompetrol publishes no API. | Temuri | Yes, on 2026-10-19. |
| 2   | What is the research octane number of efix Super? | Temuri | 98. The grade is `super` (2026-10-19). |
| 3   | Does the server give a date when Rompetrol publishes none? | Temuri | No. The answer holds no effective date, and the tool says why (2026-10-19). |
| 4   | Where does an English name come from? | Temuri | The server holds it for each known product (2026-10-19). |
| 5   | How does the server give a grade? | Temuri | From the exact name of a known product, as for Wissol (2026-10-19). |

## Retired identifiers

This file has no retired identifier.
