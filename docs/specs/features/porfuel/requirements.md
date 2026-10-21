---
id: PORFUEL
title: The fuel prices of Portal
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Portal

## 1. Problem

A driver asks "how much is Euro Regular at Portal". The server answers for Wissol,
Rompetrol, and SOCAR. Portal is a fuel company with stations across Georgia.

## 2. Users

| User                   | Need                                                |
| ---------------------- | --------------------------------------------------- |
| A person with a client | The current price of each product of Portal.        |
| The operator           | Turns Portal off with no change to the code.        |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Portal | The data of the page `portal.com.ge/ka/fuel-prices` | No. A web page for people. | The page says "ფასები ყოველდღე ახლდება": the prices update each day. It shows no date. | No key. No published limit. `robots.txt` disallows `/api/`, the dashboards, and the login pages. |

Portal publishes no API. The page renders on the server from a private backend. It sends
the list of prices as JSON inside the page, for the script of the page.

Observed behavior of the source on 2026-10-21:

- The list holds 5 products. Each has an identifier, a name, a type (`PETROL` or
  `DIESEL`), an octane number, `latestPrice`, `latestOnlinePrice`, and a slug.
- The prices are decimal text: `"4.39"`.
- `latestOnlinePrice` is the price of a "QR liter" bought online, 20 tetri less. The page
  does not show it.
- The names are in English in each language of the page: `SUPER`, `EURO REGULAR`.
- The SUPER of Portal has the octane number 100. The page shows it.
- The page shows no date and no self-service price.

| Slug            | Name          | Type   | Octane | Price | Online price |
| --------------- | ------------- | ------ | ------ | ----- | ------------ |
| `super`         | SUPER         | PETROL | 100    | 4.39  | 4.19         |
| `premium`       | PREMIUM       | PETROL | 96     | 4.19  | 3.99         |
| `euro-regular`  | EURO REGULAR  | PETROL | 93.7   | 3.94  | 3.74         |
| `euro-diesel`   | EURO DIESEL   | DIESEL | 0      | 4.94  | 4.74         |
| `effect-diesel` | EFFECT DIESEL | DIESEL | 0      | 4.84  | 4.64         |

## 4. Out of scope

- The online price of a QR liter. `WISFUEL` 4 leaves out the price of an application.
- The country of origin, the standard, and the sulphur content of a product.
- A history of prices.

## 5. Definitions

| Term   | Meaning                                            |
| ------ | -------------------------------------------------- |
| Portal | A fuel company of Georgia. Company code `portal`.  |

## 6. Functional requirements

### `PORFUEL-FR-001`

The server must return the standard price of each product of Portal.

**Examples:**

- Normal case: `portal` on 2026-10-21. EURO REGULAR has the standard price `3.94`.
- Limit case: the answer holds no price `4.19` for SUPER. That is the online price.
- Unwanted case: a product with a price that is not a number, or not above zero. The
  server treats the answer of Portal as a failure.

### `PORFUEL-FR-002`

The server must give each product of Portal its grade.

**Why:** The SUPER of Portal has the octane number 100. The SUPER of SOCAR has 98.

**Examples:**

- Normal case: SUPER has the grade `super_100`.
- Limit case: EURO DIESEL and EFFECT DIESEL have the grade `diesel`.
- Unwanted case: a slug that the server does not know. The product has the grade `other`.

| Slug            | Grade       |
| --------------- | ----------- |
| `super`         | `super_100` |
| `premium`       | `premium`   |
| `euro-regular`  | `regular`   |
| `euro-diesel`   | `diesel`    |
| `effect-diesel` | `diesel`    |

### `PORFUEL-FR-003`

The server must give the Georgian name of each known product of Portal when the request
asks `ka`, and the name of Portal when it asks `en`.

**Why:** Portal gives the English name in each language. Its terms name the products
in Georgian.

**Examples:**

- Normal case: `ka`. The name is "ევრო რეგულარი".
- Limit case: `en`. The name is "EURO REGULAR".
- Unwanted case: `ka`, and a slug that the server does not know. The name is the name
  of Portal.

| Slug            | Georgian name  |
| --------------- | -------------- |
| `super`         | სუპერი          |
| `premium`       | პრემიუმი        |
| `euro-regular`  | ევრო რეგულარი    |
| `euro-diesel`   | ევრო დიზელი      |
| `effect-diesel` | ეფექტ დიზელი     |

### `PORFUEL-FR-004`

The server must treat a page of Portal with no price list, or with an empty one, as a
failure of Portal.

**Why:** Portal can change its page with no notice.

**Examples:**

- Normal case: the page holds the list. The server returns the 5 products.
- Unwanted case: the page holds no list. The server reports that Portal did not answer,
  and returns the prices of the other companies.

### `PORFUEL-FR-005`

The server must treat Portal as a fuel company that publishes no effective date, and no
self-service price.

**Why:** The page shows neither. `ROMFUEL-FR-004` and `ROMFUEL-FR-005` then apply.

### `PORFUEL-FR-006`

The server must read Portal with no key. The operator must be able to turn Portal off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `portal`.
- Unwanted case: a request names `portal` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Portal. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Portal. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                          |
| -------------------- | ------------- | --------------------------------------------------------------- |
| `DOM-005`            | Domain        | The grade and the Georgian name belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.            |
| `MCP-005`            | MCP tools     | Each answer names Portal and the instant at which the server read it. |
| `CLT-003` to `CLT-005` | Clients     | The client reads the JSON of a web page. The rules admit it from 2026-10-21. |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`. |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.           |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.         |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server return the online price of a QR liter? | Temuri | No (2026-10-21). |
| 2   | Is the SUPER of Portal `super_100`? | Temuri | Yes. Portal states the octane number 100 (2026-10-21). |

## Retired identifiers

This file has no retired identifier.
