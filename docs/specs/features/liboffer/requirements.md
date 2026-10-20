---
id: LIBOFFER
title: The card offers of Liberty Bank for its Studio customers
type: requirements
status: approved
created: 2026-10-20
updated: 2026-10-20
context: offers
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The card offers of Liberty Bank for its Studio customers

## 1. Problem

A person asks "what does Liberty Studio give at hotels" or "which bank gives the best
cashback for a resort". The server answers for BOG and TBC. Liberty Bank publishes the
offers of its premium package Studio on a site of its own.

## 2. Users

| User                   | Need                                                           |
| ---------------------- | -------------------------------------------------------------- |
| A person with a client | The current Studio offers of Liberty in the answer of every bank. |
| The operator           | Turns Liberty off with no change to the code.                  |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Liberty Bank | `studioliberty.ge/_mvcapi/offersapi/` and the page of each offer | No. An unofficial API and a web page source. The page `studioliberty.ge/ka/studio-lifestyle/offers` reads the API. | Not published. | No key. No published terms and no published limit. `robots.txt` does not exist. The site calls itself a test version. |

Observed behavior of the source on 2026-10-20:

- `POST loadOffers` returns 16 offers for each page, from page 1. A page with less than
  16 is the last. 13 pages hold 206 offers.
- The body gives a category id, a type id, a page, and a language `ka` or `en`.
- An offer has an id, a title, labels, an address, and a text of the days left:
  "72 Days left", "დარჩა 72 დღე". It has no category, no type, and no date.
- The labels are free text: "5% Cashback", "Up to 30% Discount", "Special Hotel Offer".
- `POST loadcategories` gives 12 categories. `POST loadtypes` gives 5 types: Discount,
  Cashback, Visa, MasterCard, and Top|Card Black Edition. A list with one category or one
  type is the only way to know the category, the kind, or the card of an offer.
- A read of every list with each filter needs 67 calls.
- The page of an offer `studioliberty.ge/<ka or en>/studio-lifestyle/offers/details/<id>`
  holds the title, the days left, and the terms in paragraphs. The terms of offer 727 end
  with "The offer is valid until December 31, 2026". 72 days after 2026-10-20 is 2026-12-31.
- The answer wraps the list in `"succeeded":true`.
- A title can end with no-break spaces.
- Liberty publishes no list of offers for a customer who is not a Studio customer.

## 4. Out of scope

- The offers for a customer of Liberty who is not a Studio customer. No list exists.
- A start date. Liberty publishes none.
- A filter by partner or by location of Liberty.

## 5. Definitions

| Term    | Meaning                                                        |
| ------- | -------------------------------------------------------------- |
| Liberty | Liberty Bank, a bank. Bank code `liberty`.                     |
| Studio  | The premium package of Liberty.                                |

## 6. Functional requirements

### `LIBOFFER-FR-001`

The server must return the current Studio offers of Liberty in the answer of every bank.
Each offer has its title, its labels as the benefit text, and its end date.

**Examples:**

- Normal case: `liberty` on 2026-10-20. The answer holds "Hyatt Regency Bangkok
  Sukhumvit" with the benefit text "Special Hotel Offer".
- Limit case: a title with no-break spaces at its end. The answer has no space at the end.

### `LIBOFFER-FR-002`

The server must give each offer of Liberty the tier `premium` only.

**Why:** The site lists the offers of Studio only.

### `LIBOFFER-FR-003`

The server must give each offer of Liberty the end date that is the count of the days
left after the date of the read, in Tbilisi. It must give no start date.

**Examples:**

- Normal case: "72 Days left" on 2026-10-20 gives the end date 2026-12-31.
- Unwanted case: a text with no number. The offer has no end date.

### `LIBOFFER-FR-004`

The server must give each offer of Liberty the offer category of each category of
Liberty that lists it, and its name as a bank category.

**Examples:**

- Normal case: an offer in "Fitness" has `health_beauty`.
- Limit case: an offer in "Gifts" has `shopping`.

### `LIBOFFER-FR-005`

The server must give each offer of Liberty the benefit kind of each type that lists it:
Discount is `discount`, Cashback is `cashback`. It must give `other` when none lists it.

### `LIBOFFER-FR-006`

The server must give each offer of Liberty each card of a type that lists it: Visa,
MasterCard, or Top|Card Black Edition.

### `LIBOFFER-FR-007`

The server must return the terms of an offer of Liberty from the page of the offer.

**Examples:**

- Normal case: `liberty`, `727`, `en`. The terms end with the line of the end date.
- Unwanted case: the page has no block of terms. The server reports that Liberty did not answer.

### `LIBOFFER-FR-008`

The server must treat an answer of Liberty that does not report a success as a failure
of Liberty.

### `LIBOFFER-FR-009`

The server must read Liberty with no key. The operator must be able to turn Liberty off.
It is on by default.

## 7. Non-functional requirements

### `LIBOFFER-NFR-001`

The server must answer a search in less than 30 seconds when it must read the offers of
Liberty. Measure from the receipt of the request to the last byte.

`BOGOFFER-NFR-001` to `BOGOFFER-NFR-003` apply to Liberty.

## 8. Invariants

`BOGOFFER-INV-001` to `BOGOFFER-INV-003` apply to Liberty. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                         |
| -------------------- | ------------- | -------------------------------------------------------------- |
| `CLT-001`            | Clients       | The client is `pkg/liberty`.                                   |
| `CLT-002`, `ADR-0003` | Clients      | The client reads the page of an offer. It imports an HTML parser. |
| `DOM-007`            | Domain        | The end date is a civil date in Tbilisi.                       |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial. It stays at `v0`.  |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `offers`.        |
| `RSL-008`            | Resilience    | The limit of the calls to Liberty is a setting.                |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server read the terms from the page of the offer? | Temuri | Yes (2026-10-20). |
| 2   | Does the owner accept 67 calls for each read of Liberty, each hour for each language? The site calls itself a test version. | Temuri | Yes, for now (2026-10-20). |

## Retired identifiers

This file has no retired identifier.
