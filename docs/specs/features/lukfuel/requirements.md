---
id: LUKFUEL
title: The fuel prices of Lukoil
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Lukoil

## 1. Problem

A driver asks "did Lukoil change the price of diesel this week". The server answers for
Wissol, Rompetrol, and SOCAR. Lukoil is a fuel company with stations across Georgia,
and it publishes the date of each change of its prices.

## 2. Users

| User                   | Need                                                              |
| ---------------------- | ----------------------------------------------------------------- |
| A person with a client | The current price of each product of Lukoil, and the date of its last change. |
| The operator           | Turns Lukoil off with no change to the code.                      |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Lukoil | The table of the page `www.lukoil.ge/prices-history` | No. A web page for people. | At each change. On 2026-10-21 the newest change was 2026-10-05 10:01:46. | No key. No published terms and no published limit. `robots.txt` holds no rule. |

Lukoil publishes no API. The home page shows the current prices with no date. The page
"ფასების ისტორია" shows each change, with the same current prices in its newest row.

Observed behavior of the history page on 2026-10-21:

- One table: a number, one column for each product, and the column "შეცვლის თარიღი",
  the instant of the change: `2026-10-05 10:01:46`, with no zone.
- 106 rows from 2022-07-15, the oldest first, all on one page. 99 KB. A change adds a row.
- Two rows can share a day: `2022-07-15 19:47:43` and `2022-07-15 23:02:42`.
- The prices are decimal text with two places, in lari for one litre: `4.60`, `3.95`.
- The names are in English on each page of the site: "Super Ecto", "Euro Diesel".
- Super Ecto 100 has the price `0.00` in each row: a product that Lukoil does not sell.
- The newest row equals the 5 cards of the home page.
- No product has a self-service price.

| Product          | Price | Changed on |
| ---------------- | ----- | ---------- |
| Super Ecto 100   | 0.00  | 2026-10-05 |
| Super Ecto       | 4.60  | 2026-10-05 |
| Premium Avangard | 4.25  | 2026-10-05 |
| Euro Regular     | 3.95  | 2026-10-05 |
| Euro Diesel      | 4.97  | 2026-10-05 |

## 4. Out of scope

- A history of prices. The page holds it. A later feature can read it.
- A price of a card or of a loyalty program.
- A self-service price. Lukoil publishes none.

## 5. Definitions

| Term   | Meaning                                           |
| ------ | ------------------------------------------------- |
| Lukoil | Lukoil Georgia, a fuel company of Georgia. Company code `lukoil`. |

## 6. Functional requirements

### `LUKFUEL-FR-001`

The server must return the standard price of each product of Lukoil from its newest
change.

**Examples:**

- Normal case: `lukoil` on 2026-10-21. Euro Regular has the standard price `3.95`.
- Limit case: two changes on one day. The server takes the later instant.
- Unwanted case: a price that is not a number, or an instant that is not a date and a
  time. The server treats the answer of Lukoil as a failure.

### `LUKFUEL-FR-002`

The server must give each price of Lukoil the date of the newest change as its effective
date.

**Why:** Lukoil publishes the instant of each change. A driver can tell how old a price is.

**Examples:**

- Normal case: Euro Diesel at `lukoil` on 2026-10-21. The effective date is `2026-10-05`.
- Limit case: the description of the tool lists Lukoil with no note on the date.

### `LUKFUEL-FR-003`

The server must leave out a product of Lukoil with a price of zero, and return the other
products.

**Why:** Lukoil lists Super Ecto 100 at `0.00`. One such product must not hide the others.

**Examples:**

- Normal case: Super Ecto 100 at `0.00`. The answer holds the other 4.
- Unwanted case: every product at zero, or a negative price. The server treats the
  answer of Lukoil as a failure.

### `LUKFUEL-FR-004`

The server must give each product of Lukoil its grade.

**Examples:**

- Normal case: Premium Avangard has the grade `premium`.
- Limit case: Super Ecto 100 has the grade `super_100`, once Lukoil gives it a price.
- Unwanted case: a product that the server does not know. It has the grade `other`.

| Product          | Grade       |
| ---------------- | ----------- |
| Super Ecto 100   | `super_100` |
| Super Ecto       | `super`     |
| Premium Avangard | `premium`   |
| Euro Regular     | `regular`   |
| Euro Diesel      | `diesel`    |

### `LUKFUEL-FR-005`

The server must give the name of a product of Lukoil as Lukoil writes it, in each language.

**Why:** Lukoil writes the names in English on its Georgian page.

### `LUKFUEL-FR-006`

The server must treat a page of Lukoil with no price table, or with no row, as a failure
of Lukoil.

**Examples:**

- Normal case: the page holds 106 rows. The server returns the newest.
- Unwanted case: the page holds no table. The server reports that Lukoil did not answer,
  and returns the prices of the other companies.

### `LUKFUEL-FR-007`

The server must read Lukoil with no key. The operator must be able to turn Lukoil off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `lukoil`.
- Unwanted case: a request names `lukoil` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Lukoil. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Lukoil. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                           |
| -------------------- | ------------- | ---------------------------------------------------------------- |
| `DOM-005`            | Domain        | The newest row, the grade, and the zero price belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.             |
| `DOM-007`            | Domain        | The instant of a change is a civil time in `Asia/Tbilisi`.       |
| `MCP-005`            | MCP tools     | Each answer names Lukoil and the instant at which the server read it. |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page, as `ADR-0003` allows.               |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`. |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.            |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.          |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | The home page or the history page? | Temuri | The history page, for the date (2026-10-21). |
| 2   | Is Super Ecto petrol of 98, and Premium Avangard of 95 or 96? The site states no octane number. | Temuri | Yes, as the table of `LUKFUEL-FR-004` (2026-10-21). |

## Retired identifiers

This file has no retired identifier.
