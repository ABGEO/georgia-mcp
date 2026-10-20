---
id: TBCOFFER
title: The card offers of TBC Bank
type: requirements
status: approved
created: 2026-10-20
updated: 2026-10-20
context: offers
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The card offers of TBC Bank

## 1. Problem

A person asks "what does TBC Concept give at resorts" or "is the cashback at BOG or at
TBC better for clothes". The server answers for BOG only. TBC publishes offers for every
customer, for its premium package Concept, and for young customers.

## 2. Users

| User                   | Need                                                           |
| ---------------------- | -------------------------------------------------------------- |
| A person with a client | The current offers of TBC in the answer of every bank.         |
| The operator           | Turns TBC off with no change to the code.                      |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| TBC Bank | `apigw.tbcbank.ge/api/v1/marketing` | No. An unofficial API. The pages `tbcbank.ge/ka/offers` and `tbcconcept.ge/ge/offers` read it. `apigw.tbcconcept.ge` gives the same answers. The server reads `apigw.tbcbank.ge`. | Not published. | No key. No published terms and no published limit. |

Observed behavior of the source on 2026-10-20:

- `POST /entries/offer` with no segment returns every offer: 518. The body gives a filter
  list, a locale (`ka-GE` or `en-US`), a page index from 0, and a page size. A size of
  1000 returns every offer in one page.
- An offer has an id, a title, a description, a slug, a start date, an end date, a
  partner with a title, and a list of segments. It has no category and no benefit kind.
- 55 offers have an end date before the day of the call. 426 have no start date. 323
  have no end date. A date has no zone: `2026-10-21T23:59:00`.
- A segment has an id, a label, and a flag `isHidden`. The label changes with the locale.
  The id does not.

| Segment id               | Label (en)    | Offers |
| ------------------------ | ------------- | ------ |
| `64mvOqr0aBsBrvFAbtHoz9` | For Everyone  | 379    |
| `5I27EbKG2HykRaU760nvck` | Concept       | 121    |
| `1opQpNUWlkhEPLDyVjeGm5` | For Youth     | 16     |
| `6WK1EC8bbUzUG02i51urvP` | Expats        | 227, always with a second segment |
| `6gyl5FRPnrS66J79o230lD`, `5FSNwGngwsxY6Giq6U8Ani` | Emigrants, Installment | 1 each, alone |

- `GET /filters?type=offer&segment=<code>` gives the filters of the segments `Concept`,
  `ForEveryone`, and `ForYouth`, with a code and a name in the locale: the groups
  `Category`, `ProductType`, `OfferType`, and `CardType`. With no segment, it fails.
- A list with the filter `Category:Leisure` and no segment returns the 27 leisure offers.
  A list with a filter is the only way to know the category, the kind, and the card of an offer.
- `OfferType` holds `Discount`, `Cashback`, and `PartnerOffers`. One offer can have two.
- `GET /entries/offer/<slug>/<locale>` returns one offer with its terms as a tree of
  paragraphs and list items.
- The page of an offer is `tbcbank.ge/<ka or en>/offers/all-offers/<slug>`, or
  `tbcconcept.ge/<ge or en>/offers/<slug>` for a Concept offer.

## 4. Out of scope

- The segment Expats. Every offer of Expats has a second segment.
- An offer with no segment of 5.1 of `BOGOFFER`: Emigrants and Installment.
- A subcategory of Travel: airport services, car rental, hotels, roaming.

## 5. Definitions

| Term    | Meaning                                                         |
| ------- | --------------------------------------------------------------- |
| TBC     | TBC Bank, a bank. Bank code `tbc`.                              |
| Concept | The premium package of TBC.                                     |

## 6. Functional requirements

### `TBCOFFER-FR-001`

The server must return the current offers of TBC in the answer of every bank. Each offer
has its title, its partner, its description as the benefit text, and its offer period.

**Examples:**

- Normal case: `tbc` on 2026-10-20. The answer holds "Oktopus" with the partner
  "Oktopus Fitness Club" and the period 2026-10-20 to 2026-10-21.
- Limit case: an offer with no start date and no end date. It is in the answer with no dates.

### `TBCOFFER-FR-002`

The server must give each offer of TBC its tiers by the segment: For Everyone is
`standard`, Concept is `premium`, For Youth is `youth`.

**Examples:**

- Normal case: an offer of Concept and Expats has the tiers `[premium]`.
- Unwanted case: an offer of Installment only. The server leaves it out.

### `TBCOFFER-FR-003`

The server must give each offer of TBC the offer category of each category filter that
lists it, and the name of each filter as a bank category.

**Examples:**

- Normal case: an offer in `Leisure` has `[leisure]`.
- Limit case: an offer in `Clothes` and `OnlinePartners` has `[shopping]` and two bank categories.
- Unwanted case: an offer in no category filter has `[other]`.

### `TBCOFFER-FR-004`

The server must give each offer of TBC the benefit kind of each offer type that lists it:
`Discount` is `discount`, `Cashback` is `cashback`. It must give `other` when no type of
the two lists it.

**Why:** `PartnerOffers` says that a partner gives the offer. It is not a kind of benefit.

### `TBCOFFER-FR-005`

The server must give each offer of TBC each card of a product type filter or a card type
filter that lists it.

**Examples:**

- Normal case: an offer in `ProductType:TBCConceptCard` and `CardType:MasterCard` has the
  cards `concept_card` and `mastercard`.
- Limit case: the code `CreditCard` of `CardType` is the card `visa`. The code `CreditCard`
  of `ProductType` is the card `credit_card`.

### `TBCOFFER-FR-006`

The server must give the address of the page of an offer: `tbcconcept.ge` when its only
tier is `premium`, else `tbcbank.ge`.

### `TBCOFFER-FR-007`

The server must return the terms of an offer of TBC from the page of the offer in the API.

**Examples:**

- Normal case: `tbc`, `7tD3DL0YuO2V6f1Efr4sSV` in `en`. The terms hold the line
  "The offer applies only to 6-month memberships."

### `TBCOFFER-FR-008`

The server must read TBC with no key. The operator must be able to turn TBC off. It is on
by default.

## 7. Non-functional requirements

### `TBCOFFER-NFR-001`

The server must answer a search in less than 30 seconds when it must read the offers of
TBC. Measure from the receipt of the request to the last byte.

**Why:** A read of TBC needs about 40 calls: one list, and one list for each filter.

`BOGOFFER-NFR-001` to `BOGOFFER-NFR-003` apply to TBC.

## 8. Invariants

`BOGOFFER-INV-001` to `BOGOFFER-INV-003` apply to TBC. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                         |
| -------------------- | ------------- | -------------------------------------------------------------- |
| `DOM-001`, `CLT-001` | Domain        | TBC is a vendor of `rates` too. The client is `pkg/tbcoffers`. `ADR-0004`. |
| `DOM-005`            | Domain        | The segments, the filters, and the tree of the terms stop at the adapter. |
| `DOM-007`            | Domain        | A date of TBC is a civil date in Tbilisi.                      |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial. It stays at `v0`.  |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `offers`.        |
| `RSL-008`            | Resilience    | The limit of the calls to TBC is a setting.                    |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Does the server read `apigw.tbcbank.ge` or `apigw.tbcconcept.ge`? Both give the same 518 offers. | Temuri | `apigw.tbcbank.ge` (2026-10-20). |
| 2   | Does the owner accept about 40 calls for each read of TBC, each hour for each language? | Temuri | Yes, for now (2026-10-20). |

## Retired identifiers

This file has no retired identifier.
