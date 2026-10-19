---
id: WISFUEL
title: The fuel prices of Wissol
type: requirements
status: approved
created: 2026-10-19
updated: 2026-10-19
context: fuel
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The fuel prices of Wissol

## 1. Problem

A driver in Georgia asks "how much is diesel at Wissol today" or "where is premium
petrol cheapest". A model has no current fuel price. Each fuel company sets its own
prices, and changes them a few times in a month.

Some fuel companies sell the same product at a lower price at their self-service
stations. A model that gives one price, or the wrong one of the two, gives a wrong answer.

Each company names its products in its own way: Wissol sells "Eko Premium", another
company sells "Nano Premium". A model must know that the two are the same grade.

This feature adds the first fuel company, Wissol. Other companies join the same answer
in later features.

## 2. Users

| User                   | Need                                                                                |
| ---------------------- | ----------------------------------------------------------------------------------- |
| A person with a client | The current price of each product of a fuel company, and its self-service price.   |
| The operator           | Turns Wissol off with no change to the code.                                       |

## 3. Source

| Vendor | Source                             | Official | Publication period | Terms of use |
| ------ | ---------------------------------- | -------- | ------------------ | ------------ |
| Wissol | `api.wissol.ge/FuelPrice`          | No. An unofficial API. The page `wissol.ge/ka/fuel-prices` reads it. | Not published. Each product carries the date of its last change. On 2026-10-19 the newest was 2026-10-17. | No key. No published terms and no published limit. |

Observed behavior of the source on 2026-10-19:

- One call returns every product: 8 products, each with a code, a Georgian name, an
  English name, a Russian name, a flag of activity, a price, and a date.
- A product sold at the self-service stations carries a second price. 6 products of 8
  carry it. `A1 SUPER 100` and `Wissol Gas` carry none: the field is absent.
- The prices are JSON numbers in lari for one unit: `4.08`, `3.93`.
- The date has no zone: `2026-10-08T00:00:00`. The page shows no date.
- The answer holds no instant and no unit. `Wissol Gas` is compressed natural gas.
- The prices are equal to the prices of the page `wissol.ge/en/fuel-prices`.
- The address is case insensitive: `/FuelPrice` and `/fuelprice/` give the same answer.
- `/fuelpricehistory/?days=N` gives the history of the standard price.

## 4. Out of scope

- A history of prices.
- A price at one station, a price of a card, of a loyalty program, or of an application.
- The description of a product, and its Russian name.
- The cost of an amount of fuel.
- A fuel company other than Wissol. A later feature adds each one.

## 5. Definitions

| Term               | Meaning                                                                                    |
| ------------------ | ------------------------------------------------------------------------------------------ |
| fuel company       | A company that sells fuel at its own network of stations in Georgia.                       |
| company code       | The short lowercase name of a fuel company in a request: `wissol`.                         |
| product            | One fuel that a fuel company sells under its own name: "Eko Premium".                       |
| grade              | The class of a product that every fuel company shares. 5.1 lists the grades.                |
| standard price     | The price in lari of one unit of a product at a station with an attendant.                 |
| self-service price | The price in lari of one unit of a product at a self-service station of the company.       |
| effective date     | The date from which the fuel company applies a price.                                      |
| Wissol             | A fuel company of Georgia. Company code `wissol`.                                          |

### 5.1 The grades

| Grade       | Meaning                                           | Unit        |
| ----------- | ------------------------------------------------- | ----------- |
| `regular`   | Petrol with a research octane number of 92 or 93. | litre       |
| `premium`   | Petrol with a research octane number of 95 or 96. | litre       |
| `super`     | Petrol with a research octane number of 98.       | litre       |
| `super_100` | Petrol with a research octane number of 100.      | litre       |
| `diesel`    | Diesel fuel.                                      | litre       |
| `cng`       | Compressed natural gas.                           | cubic metre |
| `other`     | A product that the server cannot classify.        | litre       |

## 6. Functional requirements

### `WISFUEL-FR-001`

The server must return the standard price of each product of a fuel company.

**Examples:**

- Normal case: `wissol` on 2026-10-19. Euro Regular has the standard price `4.08`.
- Limit case: `Wissol Gas` has the standard price `1.64`.
- Unwanted case: a product with no standard price, or a price of zero. The server
  treats the answer of the company as a failure.

### `WISFUEL-FR-002`

The server must give the self-service price of a product only when the fuel company
publishes one.

**Why:** A model must not offer a self-service price that no station applies.

**Examples:**

- Normal case: Euro Regular has the self-service price `3.93`.
- Limit case: `A1 SUPER 100` has no self-service price. The answer holds none for it.
- Unwanted case: a self-service price of zero. The server must not return it.

### `WISFUEL-FR-003`

The server must give each product its grade.

**Why:** Each company names its products in its own way. The grade lets a model
compare the companies.

**Examples:**

- Normal case: Eko Premium has the grade `premium`.
- Limit case: Eko Diesel, Euro Diesel, and Diesel Energy have the grade `diesel`.
- Unwanted case: a new product that the server does not know. It has the grade `other`.

### `WISFUEL-FR-004`

The server must give each product its unit: the cubic metre for `cng`, the litre for
every other grade.

**Why:** The price of compressed natural gas is for one cubic metre, not one litre.

### `WISFUEL-FR-005`

The server must give each price its effective date.

**Why:** A model must tell a price of yesterday from a price of last year.

**Examples:**

- Normal case: Euro Regular has the effective date `2026-10-08`.
- Limit case: `Wissol Gas` has the effective date `2023-11-07`.

### `WISFUEL-FR-006`

The server must answer for every fuel company that it can read when the request names
no company.

**Why:** A person who asks "where is diesel cheapest" names no company.

### `WISFUEL-FR-007`

The server must refuse a company code that it does not know. The message must name the
codes that it accepts.

### `WISFUEL-FR-008`

The server must return the products of the requested grades only, when the request
names one grade or more. It must answer every grade when the request names none.

**Examples:**

- Normal case: `diesel` at `wissol`. The answer holds the three diesel products.
- Limit case: no grade. The answer holds the 8 products.
- Unwanted case: the grade `lpg`. The server refuses it, and the message lists the grades.

### `WISFUEL-FR-009`

The server must name each requested grade that a fuel company does not sell, and return
the products of the other grades.

**Examples:**

- Normal case: `super_100` and `cng` at `wissol`. The answer holds both.
- Limit case: `other` at `wissol`. The answer names `other` as a grade that Wissol does not sell.

### `WISFUEL-FR-010`

The server must leave out a product that the fuel company marks as not active.

### `WISFUEL-FR-011`

The server must give the name of each product in the requested language: `ka` or `en`.
The default is `en`.

**Examples:**

- Normal case: `en`. The name is "Euro Regular".
- Limit case: `ka`. The name is "ევრო რეგულარი".

### `WISFUEL-FR-012`

The server must state in each answer what the standard price and the self-service price are.

**Why:** A model that reads "self-service" as a discount at any station gives a wrong answer.

### `WISFUEL-FR-013`

The server must give, for each fuel company, the instant at which it read the prices.

**Why:** The source publishes no instant.

### `WISFUEL-FR-014`

The server must report a fuel company that does not answer, and return the prices of
the other companies.

**Examples:**

- Normal case: one company fails, a second answers. The answer holds the second, and names the first.
- Limit case: the only requested company fails, and the server holds no prices of it.
  The server reports that the company did not answer.

### `WISFUEL-FR-015`

The server must return prices that it read less than 12 hours before, marked as stale,
when a fuel company does not answer.

**Why:** A company changes its prices a few times in a month, at most one time in a
day. A price of the same morning helps more than no price.

### `WISFUEL-FR-016`

The server must read Wissol with no key. The operator must be able to turn Wissol off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `wissol`.
- Limit case: Wissol off, and no other fuel company. The server offers no fuel price.
- Unwanted case: a request names `wissol` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

### `WISFUEL-NFR-001`

A price in an answer must be less than 30 minutes old, unless the answer marks it as
stale. Measure from the read instant to the receipt of the request.

### `WISFUEL-NFR-002`

The server must send at most one call to a fuel company in each 30 minutes while the
company answers, for any count of requests. Measure the calls at the source.

**Why:** Wissol publishes no limit. One call returns every product.

## 8. Invariants

### `WISFUEL-INV-001`

A price in an answer has the exact digits that the fuel company publishes. The server
rounds no digit.

### `WISFUEL-INV-003`

Every price in an answer carries its fuel company, its grade, and its unit. It carries
its effective date when the fuel company publishes one.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                     |
| -------------------- | ------------- | -------------------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `fuel`. Wissol is its first vendor.                         |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.                       |
| `DOM-007`            | Domain        | The effective date is a civil date in Tbilisi. `ADR-0002`.                 |
| `MCP-002`            | MCP tools     | The tool is `fuel_get_prices`.                                             |
| `MCP-005`            | MCP tools     | Each answer names the fuel company and the instant at which the server read it. |
| `MCP-007`            | MCP tools     | The tool takes the optional `language`.                                    |
| `CLT-001`, `CLT-009` | Clients       | The client is a module in `pkg/wissol`. It says that the API is unofficial. |
| `CLT-011`            | Clients       | The client stays at `v0`.                                                  |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.                      |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours, for each company.  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does Wissol sell `Wissol Gas` for one cubic metre? The API and the page show no unit. | Temuri | Yes. `Wissol Gas` is for one cubic metre. Every other product is for one litre (2026-10-19). |
| 2   | Does the owner accept an unofficial API with no published terms? The page of Wissol reads the same API with no key. | Temuri | Yes. The owner chose the API over the page on 2026-10-19. |

## Retired identifiers

| ID              | Retired    | Reason                                                              |
| --------------- | ---------- | ------------------------------------------------------------------- |
| WISFUEL-INV-002 | 2026-10-19 | Rompetrol publishes no effective date. `WISFUEL-INV-003` replaces it. |
