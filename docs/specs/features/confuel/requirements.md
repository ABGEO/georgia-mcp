---
id: CONFUEL
title: The fuel prices of Connect
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Connect

## 1. Problem

A driver asks "is diesel cheaper at Connect or at Rompetrol". The server answers for
Wissol, Rompetrol, and SOCAR. Connect is a fuel company with stations across Georgia,
and a self-service price for each product.

## 2. Users

| User                   | Need                                                                    |
| ---------------------- | ----------------------------------------------------------------------- |
| A person with a client | The current price of each product of Connect, and its self-service price. |
| The operator           | Turns Connect off with no change to the code.                           |

## 3. Source

| Vendor  | Source | Official | Publication period | Terms of use |
| ------- | ------ | -------- | ------------------ | ------------ |
| Connect | `connect.com.ge/wp-json/fuel-prices/v1/current` | No. An unofficial API of a plugin of the site. The page `connect.com.ge/საწვავი/` shows the same prices. | A snapshot each day at 04:00. Connect enters a change by hand at any hour. | No key. No published terms and no published limit. `robots.txt` disallows `/wp-admin/` only. |

Observed behavior of the source on 2026-10-21:

- One call returns every product: 5 products, each under a key (`super_ron98`), with a
  Georgian label, a standard price, and a self-service price. 994 bytes.
- The prices are JSON numbers with the full digits of a binary float:
  `4.3499999999999996447286321199499070644378662109375`. The page shows `4.35 ₾`.
- The answer has `date`, `updated`, and `is_manual`. `date` is the day of the snapshot,
  not the day of the last change: 2026-10-14 to 2026-10-21 carry the prices of the
  change of 2026-10-13. The page writes `(გადატანილი)`, "carried over".
- The site has no English name. A plugin of the site translates the page by machine.
- `/fuel-prices/v1/history` and `/fuel-prices/v1/date/{date}` give the past days. A day
  with no snapshot answers 404.

| Key                 | Label              | Standard | Self-service |
| ------------------- | ------------------ | -------- | ------------ |
| `super_ron98`       | სუპერი RON 98       | 4.35     | 4.25         |
| `ron96`             | პრემიუმი RON 96     | 4.15     | 3.95         |
| `regular_ron93`     | რეგულარი RON 93     | 3.95     | 3.75         |
| `euro_diesel_10ppm` | ევრო დიზელი 10PPM   | 4.90     | 4.75         |
| `diesel_10ppm`      | დიზელი 10PPM        | 4.80     | 4.65         |

## 4. Out of scope

- A history of prices.
- A price of a card or of a loyalty program.
- A date that the server derives from the history of the source.

## 5. Definitions

| Term    | Meaning                                              |
| ------- | ---------------------------------------------------- |
| Connect | A fuel company of Georgia. Company code `connect`.   |

## 6. Functional requirements

### `CONFUEL-FR-001`

The server must return the standard price of each product of Connect as the decimal
that Connect entered, not as the binary float that the source sends.

**Why:** The source sends the full digits of a binary float. `4.3499999999999996...` is
not the price of a station.

**Examples:**

- Normal case: `connect` on 2026-10-21. Super RON 98 has the standard price `4.35`.
- Limit case: Euro Diesel 10PPM has the standard price `4.9`. The page shows `4.90`.
- Unwanted case: a product with a price that is not a number, or not above zero. The
  server treats the answer of Connect as a failure.

### `CONFUEL-FR-002`

The server must give each product of Connect its self-service price.

**Examples:**

- Normal case: Premium RON 96 has the self-service price `3.95`.
- Limit case: a product with no self-service price, or a self-service price of zero.
  The answer holds no self-service price for it.
- Unwanted case: a negative self-service price. The server treats the answer of Connect
  as a failure.

### `CONFUEL-FR-003`

The server must give each product of Connect its grade.

**Examples:**

- Normal case: Super RON 98 has the grade `super`.
- Limit case: Euro Diesel 10PPM and Diesel 10PPM have the grade `diesel`.
- Unwanted case: a key that the server does not know. The product has the grade `other`.

| Key                 | Grade     |
| ------------------- | --------- |
| `super_ron98`       | `super`   |
| `ron96`             | `premium` |
| `regular_ron93`     | `regular` |
| `euro_diesel_10ppm` | `diesel`  |
| `diesel_10ppm`      | `diesel`  |

### `CONFUEL-FR-004`

The server must give the English name of each known product of Connect when the request
asks `en`. It must give the Georgian label of a product that it does not know.

**Examples:**

- Normal case: `en`. The name is "Premium RON 96".
- Limit case: `ka`. The name is "პრემიუმი RON 96".
- Unwanted case: `en`, and a key that the server does not know. The name is the
  Georgian label.

| Key                 | English name      |
| ------------------- | ----------------- |
| `super_ron98`       | Super RON 98      |
| `ron96`             | Premium RON 96    |
| `regular_ron93`     | Regular RON 93    |
| `euro_diesel_10ppm` | Euro Diesel 10PPM |
| `diesel_10ppm`      | Diesel 10PPM      |

### `CONFUEL-FR-005`

The server must treat Connect as a fuel company that publishes no effective date.

**Why:** The date of the source is the day of the snapshot. A false date is worse than
no date. `ROMFUEL-FR-004` and `ROMFUEL-FR-005` then apply.

**Examples:**

- Normal case: Super RON 98 at `connect`. The price has no effective date, though the
  source gives `2026-10-21`.

### `CONFUEL-FR-006`

The server must treat an answer of Connect with no product as a failure of Connect.

**Examples:**

- Normal case: 5 products. The server returns them.
- Unwanted case: an empty list, or an answer with no list. The server reports that
  Connect did not answer, and returns the prices of the other companies.

### `CONFUEL-FR-007`

The server must read Connect with no key. The operator must be able to turn Connect off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `connect`.
- Unwanted case: a request names `connect` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Connect. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Connect. For Connect, the digits that
it publishes are the shortest decimal of each float (`CONFUEL-FR-001`). This feature
adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                               |
| -------------------- | ------------- | -------------------------------------------------------------------- |
| `DOM-005`            | Domain        | The grade, the English name, and the digits belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer. `DOM-006` gives the conversion of a binary float. |
| `MCP-005`            | MCP tools     | Each answer names Connect and the instant at which the server read it. |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial. It stays at `v0`.        |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.                |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.              |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | The REST route or the HTML page? | Temuri | The REST route, with the shortest decimal of each float (2026-10-21). |
| 2   | Does the server give the date of the source? | Temuri | No. It is the day of the snapshot (2026-10-21). |

## Retired identifiers

This file has no retired identifier.
