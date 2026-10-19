---
id: SOCFUEL
title: The fuel prices of SOCAR
type: requirements
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of SOCAR

## 1. Problem

A driver asks "how much is LPG at SOCAR" or "is diesel cheaper at SOCAR or at Wissol".
The server answers for Wissol and Rompetrol only. SOCAR is a third fuel company with
stations across Georgia.

SOCAR sells liquefied petroleum gas. No grade of the server names it, so a model cannot
ask for it or compare it.

## 2. Users

| User                   | Need                                                                  |
| ---------------------- | --------------------------------------------------------------------- |
| A person with a client | The current price of each product of SOCAR, LPG included.             |
| The operator           | Turns SOCAR off with no change to the code.                           |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| SOCAR Georgia Petroleum | `sgp.ge/sgp-backend/api/integration/info/current-prices` | No. An unofficial API. The home page `sgp.ge` reads it. | Not published. | No key. No published terms and no published limit. `robots.txt` allows every path but `/health` and `/my-cabinet`. |

Observed behavior of the source on 2026-10-19:

- One call returns every product: 7 products, each with a code, a Georgian name, an
  English name, a price, and a date `ActionDate`.
- The answer wraps the list in a status: `"Status":"1","Message":"Success"`.
- The prices are JSON numbers in lari for one unit: `4.62`, `4.9`.
- `ActionDate` is not the date of the last change. Nano Super has `2023-09-23`, and the
  price archive of the site shows its last change on `2026-10-09`.
- No product has a self-service price.
- The English name of LPG ends with a space: `"LPG "`.
- An error of the source is a JSON body with the text of an exception.
- The price archive of the site gives LPG `1.75` on each day from 2026-10-13. The
  current prices give `2.09` from the same day. The home page shows `2.09`. This is a
  known limit of the source. The server returns the current price.
- The archive lists `A92 ბენზინი` with the price `0`: a product that SOCAR does not sell.

| Code      | Georgian name         | English name        | Price | Octane (archive) |
| --------- | --------------------- | ------------------- | ----- | ---------------- |
| `SUPER`   | ნანო სუპერი           | Nano Super          | 4.62  | A98              |
| `PREMIUM` | ნანო პრემიუმი         | Nano Premium        | 4.31  | A95              |
| `EURREG`  | ნანო ევრო რეგულარი     | Nano Euro Regular   | 4.02  | A93              |
| `DIESEL`  | ევრო 5 დიზელი         | Euro 5 Diesel       | 4.9   |                  |
| `EURODSL` | ნანო ევრო 5 დიზელი     | Nano Euro 5 Diesel  | 4.96  |                  |
| `LPG`     | თხევადი აირი           | LPG                 | 2.09  |                  |
| `CNG`     | ბუნებრივი აირი         | CNG                 | 1.6   |                  |

## 4. Out of scope

- A history of prices. The site has an archive. A later feature can read it.
- A price of a card or of a loyalty program. The site has an API for it.
- A self-service price. SOCAR publishes none.
- A date that the server derives from the archive.

## 5. Definitions

| Term  | Meaning                                                         |
| ----- | --------------------------------------------------------------- |
| SOCAR | SOCAR Georgia Petroleum, a fuel company of Georgia. Company code `socar`. |
| `lpg` | The grade of liquefied petroleum gas. Its unit is the litre.    |

The grade `lpg` joins the table of `WISFUEL` 5.1, after `cng` and before `other`.

## 6. Functional requirements

### `SOCFUEL-FR-001`

The server must return the standard price of each product of SOCAR.

**Examples:**

- Normal case: `socar` on 2026-10-19. Nano Euro Regular has the standard price `4.02`.
- Limit case: Euro 5 Diesel has the standard price `4.9`. The answer keeps the one digit.
- Unwanted case: a product with a price that is not a number. The server treats the
  answer of SOCAR as a failure.

### `SOCFUEL-FR-002`

The server must leave out a product of SOCAR with a price of zero, and return the other
products.

**Why:** SOCAR lists a product that it does not sell with the price zero. One such
product must not hide the prices of the others.

**Examples:**

- Normal case: 7 products above zero. The answer holds 7.
- Limit case: one product at zero. The answer holds the other 6.
- Unwanted case: every product at zero, or a negative price. The server treats the
  answer of SOCAR as a failure.

### `SOCFUEL-FR-003`

The server must give each product of SOCAR its grade.

**Examples:**

- Normal case: Nano Super has the grade `super`.
- Limit case: Euro 5 Diesel and Nano Euro 5 Diesel have the grade `diesel`.
- Unwanted case: a product with a code that the server does not know. It has the grade `other`.

| Code      | Grade     |
| --------- | --------- |
| `SUPER`   | `super`   |
| `PREMIUM` | `premium` |
| `EURREG`  | `regular` |
| `DIESEL`  | `diesel`  |
| `EURODSL` | `diesel`  |
| `LPG`     | `lpg`     |
| `CNG`     | `cng`     |

### `SOCFUEL-FR-004`

The server must accept the grade `lpg` in a request, and give it to each product of
liquefied petroleum gas.

**Why:** A model must find LPG, and compare it across the companies.

**Examples:**

- Normal case: `lpg` at `socar`. The answer holds LPG at `2.09` for one litre.
- Limit case: `lpg` at `wissol`. The answer names `lpg` as a grade that Wissol does not sell.
- Unwanted case: the grade `LPG`. The server refuses it, and the message lists the grades.

### `SOCFUEL-FR-005`

The server must treat SOCAR as a fuel company that publishes no effective date.

**Why:** The date of the source is not the date of the last change. A false date is
worse than no date. `ROMFUEL-FR-004` and `ROMFUEL-FR-005` then apply.

**Examples:**

- Normal case: Nano Super at `socar`. The price has no effective date, though the source
  gives `2023-09-23`.
- Limit case: the description of the tool says that SOCAR publishes no effective date.

### `SOCFUEL-FR-006`

The server must give the name of a product of SOCAR in the requested language, with no
space at its start or its end.

**Examples:**

- Normal case: `en`. The name is "Nano Euro Regular".
- Limit case: `ka`. The name is "ნანო ევრო რეგულარი".
- Unwanted case: the source gives `"LPG "`. The name is "LPG".

### `SOCFUEL-FR-007`

The server must treat an answer of SOCAR that does not report a success as a failure of
SOCAR.

**Why:** The source can send an error in a body, as a text of an exception.

**Examples:**

- Normal case: the answer reports a success. The server returns the prices.
- Unwanted case: the answer holds an exception, or a status other than a success. The
  server reports that SOCAR did not answer, and returns the prices of the other companies.

### `SOCFUEL-FR-008`

The server must read SOCAR with no key. The operator must be able to turn SOCAR off. It
is on by default.

**Examples:**

- Normal case: no configuration. The server offers `rompetrol`, `socar`, and `wissol`.
- Unwanted case: a request names `socar` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to SOCAR. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to SOCAR. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                               |
| -------------------- | ------------- | -------------------------------------------------------------------- |
| `DOM-005`            | Domain        | The grade, the trim of a name, and the zero price belong to the server, not the client. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.                 |
| `MCP-003`            | MCP tools     | The grade `lpg` changes the schema of the input and the description. |
| `MCP-005`            | MCP tools     | Each answer names SOCAR and the instant at which the server read it.  |
| `CLT-001`, `CLT-009` | Clients       | The client says that the API is unofficial.                          |
| `CLT-011`            | Clients       | The client stays at `v0`.                                            |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.                |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.              |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server give the date of the source? | Temuri | No. It is not the date of the last change (2026-10-19). |
| 2   | Does the server add the grade `lpg`? | Temuri | Yes, with the litre as its unit (2026-10-19). |
| 3   | How does the server report the two prices of LPG? | Temuri | The documents record it. The tool returns the current price with no note (2026-10-19). |
| 4   | What does the server do with a product at zero? | Temuri | It leaves the product out (2026-10-19). |

## Retired identifiers

This file has no retired identifier.
