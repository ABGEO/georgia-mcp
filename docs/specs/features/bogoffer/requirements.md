---
id: BOGOFFER
title: The card offers of Bank of Georgia
type: requirements
status: approved
created: 2026-10-20
updated: 2026-10-20
context: offers
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The card offers of Bank of Georgia

## 1. Problem

A person asks "what offers does SOLO have this month" or "which bank gives a better
offer for a resort". A model knows no current offer. Each bank of Georgia publishes
its offers on its own site, in its own categories, for its own customer groups.

A bank gives an offer to some of its customers only: a premium package, a private
banking customer, a holder of one card. A model that does not know the group of an
offer tells a person about an offer that the person cannot use.

The banks describe a benefit in free text: "30% ქეშბექი, მაქს. 2000₾". No two banks
use one form. The server does not read a number from the text. The model reads it.

This feature adds the context `offers`, its two tools, and the first bank, Bank of
Georgia. `TBCOFFER` and `LIBOFFER` add two more banks to the same tools.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | The current offers of a bank, by category, customer group, card, and period. |
| A person with a client | The full terms of one offer.                                               |
| The operator           | Turns BOG off with no change to the code.                                  |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| BOG | `bankofgeorgia.ge/api/bog-b/offers-hub/get-offers` | No. An unofficial API. The page `bankofgeorgia.ge/ka/offers-hub/catalog` reads it. | Not published. Each offer carries its start date and its end date. | No key. No published terms and no published limit. `robots.txt` forbids two query parameters only. |

Observed behavior of the source on 2026-10-20:

- A `POST` with the body `{}` returns one page of offers. The query gives `pageNumber`
  from 0 and `pageSize`. A size of 100 works. A size of 500 returns an error.
- 239 offers in 3 pages of 100. No offer has an end date before the day of the call.
  7 offers start after it.
- Each offer has a campaign id, a title, a short text, a long text in HTML, a start and
  an end instant in milliseconds, a category id and name, product codes, segment types,
  a benefit name, cities, and brand names.
- `segmentTypes` holds `RETAIL`, `SOLO`, and `WM`, separated by `;`. `WM` is Wealth
  Management, the private banking of BOG. 122 offers are `SOLO;WM`, 100 are
  `RETAIL;SOLO;WM`, 9 are `WM`.
- `productCodes` holds 13 codes of cards and packages: `SOLO`, `PLUS`, `AMEX`, `VISA_CREDIT_INFINITE`.
- `benefitName` is `DISCOUNT`, `CASHBACK`, `PLUS`, `MR`, `SCOOLA`, or `OTHER`. 76 offers
  have `OTHER`. The short text of many of them names a cashback: "30% ქეშბექი, მაქს. 2000₾".
- The header `Accept-Language: en` gives the texts and the category name in English.
  With no header, they are in Georgian.
- An error is a status 500 with a JSON body that holds a Georgian message.
- The page of one offer is `bankofgeorgia.ge/<language>/offers-hub/details/<campaign id>`.

## 4. Out of scope

- A number that the server reads from a benefit text: a percent, a multiplier, or a cap.
- A comparison of the banks by the server. The model compares the offers.
- A filter by city. Two of the three banks give no city in the list of offers.
- An offer for a business customer, and a personal offer that needs a login.
- The state of an offer that a customer must activate in the application.
- A past offer. The banks do not publish one.

## 5. Definitions

| Term          | Meaning                                                                                       |
| ------------- | --------------------------------------------------------------------------------------------- |
| bank          | A commercial bank of Georgia that publishes card offers.                                      |
| bank code     | The short lowercase name of a bank in a request: `bog`.                                       |
| offer         | A benefit that a bank gives to its customers at a partner, for a period.                      |
| partner       | The company where a customer gets the benefit of an offer: a shop, a hotel, a spa.            |
| benefit text  | The text of the bank that states the benefit of an offer: "20% Discount, unlimited".          |
| benefit kind  | The class of a benefit that the bank states in a field, not in a text. 5.3 lists the kinds.   |
| tier          | A group of customers of a bank that an offer is for. 5.1 lists the tiers.                     |
| card          | A card or a package of a bank that an offer names as a condition: "Visa Infinite", "PLUS".    |
| card code     | The short lowercase name of a card in a request: `visa_infinite`. Each bank has its own codes. |
| offer category | The class of the partner of an offer that every bank shares. 5.2 lists the categories.       |
| bank category | The name that a bank gives to the category of an offer: "Health and Care".                    |
| offer period  | The dates from the start date to the end date of an offer, both included.                     |
| terms         | The full text of the conditions of an offer.                                                  |
| BOG           | Bank of Georgia, a bank. Bank code `bog`.                                                     |

### 5.1 The tiers

| Tier       | Meaning                                                    | BOG       |
| ---------- | ---------------------------------------------------------- | --------- |
| `standard` | Every customer of the bank.                                | `RETAIL`  |
| `premium`  | The customers of the premium package of the bank.         | `SOLO`    |
| `private`  | The customers of the private banking of the bank.         | `WM`      |
| `youth`    | The young customers of the bank: pupils and students.     | None      |

### 5.2 The offer categories

| Category        | Partners                                       |
| --------------- | ---------------------------------------------- |
| `dining`        | Restaurants, cafes, bars, food delivery.       |
| `leisure`       | Resorts, hotels in Georgia, spas of a resort.  |
| `travel`        | Flights, hotels abroad, airports, car rental, roaming. |
| `health_beauty` | Clinics, pharmacies, beauty salons, fitness.   |
| `auto`          | Cars, car service, fuel.                       |
| `transport`     | Public transport, taxis.                       |
| `shopping`      | Clothes, accessories, gifts, groceries, online shops. |
| `electronics`   | Electronics and appliances.                    |
| `home`          | Furniture, home goods, repair.                 |
| `education`     | Schools, courses, trainings.                   |
| `kids`          | Goods and services for children.               |
| `entertainment` | Cinemas, events, games.                        |
| `real_estate`   | Developers and apartments.                     |
| `other`         | A partner that no other category covers.       |

### 5.3 The benefit kinds

| Kind       | Meaning                                                      |
| ---------- | ------------------------------------------------------------ |
| `discount` | The partner takes a lower price.                             |
| `cashback` | The bank returns a part of the payment as money.             |
| `points`   | The bank gives loyalty points for the payment.               |
| `other`    | The bank states no kind, or a kind that is not in this table. |

## 6. Functional requirements

### `BOGOFFER-FR-001`

The server must return the current offers of a bank. Each offer has its id, its title,
its benefit text, its offer period, and the address of its page.

**Examples:**

- Normal case: `bog` on 2026-10-20. The answer holds "Ronny's Pizza" with the benefit
  text "20% Discount, unlimited" and the period 2026-05-29 to 2027-05-22.
- Unwanted case: an offer with no title or no id. The server leaves it out.

### `BOGOFFER-FR-002`

The server must give the benefit text of an offer as the bank publishes it. It must not
derive a number from the text.

**Why:** No form of the text holds for every bank. A wrong number misleads more than a text.

### `BOGOFFER-FR-003`

The server must give each offer its tiers. It must leave out an offer that has no tier of 5.1.

**Examples:**

- Normal case: `SOLO;WM` gives `premium` and `private`.
- Limit case: `RETAIL;SOLO;WM` gives `standard`, `premium`, and `private`.
- Unwanted case: a new segment `VIP` only. The server leaves the offer out.

### `BOGOFFER-FR-004`

The server must give each offer the cards that it names, each with its card code and its name.

**Examples:**

- Normal case: `SOLO;VISA_CARDS` gives `solo` ("SOLO") and `visa` ("Visa").
- Unwanted case: an unknown code `GOLD_CARD`. The card has the code `gold_card` and the name `GOLD_CARD`.

### `BOGOFFER-FR-005`

The server must give each offer its benefit kinds. It must give `other` when the bank
states no kind of 5.3.

**Examples:**

- Normal case: `CASHBACK` gives `cashback`. `PLUS` and `MR` give `points`.
- Limit case: `OTHER` with the text "30% Cashback, max. 2000 GEL" gives `other`.

### `BOGOFFER-FR-006`

The server must give each offer its offer categories and its bank category.

**Examples:**

- Normal case: "Health and Care" gives `health_beauty`.
- Unwanted case: a new category of the bank. It gives `other`.

### `BOGOFFER-FR-007`

The server must leave out an offer whose end date is before the current date in Tbilisi.

### `BOGOFFER-FR-008`

The server must return only the offers whose offer period overlaps the requested dates.
The default is the current date in Tbilisi for both dates.

**Why:** "This month" is a range. "Now" is one date.

**Examples:**

- Normal case: 2026-10-01 to 2026-10-31. The answer holds an offer that ends on 2026-10-31,
  and an offer that starts on 2026-10-30.
- Limit case: no dates. An offer that starts on 2026-10-30 is not in the answer of 2026-10-20.
- Unwanted case: an end date before the start date, or a range of more than 366 days.
  The server refuses it.

### `BOGOFFER-FR-009`

The server must treat an offer with no start date as started, and an offer with no end
date as open, when it compares the offer period with the requested dates.

### `BOGOFFER-FR-010`

The server must return only the offers that match each filter of the request: the banks,
the tiers, the offer categories, the benefit kinds, and the card codes. An offer matches
a filter when it has one of the requested values.

**Examples:**

- Normal case: `bog`, `premium`, `leisure`. The answer holds the leisure offers for SOLO.
- Limit case: no filter. The answer holds the current offers of every bank.
- Unwanted case: the tier `vip`. The server refuses it, and the message lists the tiers.

### `BOGOFFER-FR-011`

The server must match an offer that names no card to any requested card code.

**Why:** An offer that names no card is for every card of its tiers.

### `BOGOFFER-FR-012`

The server must return only the offers whose title, partner, benefit text, or bank
category holds the requested text, with no regard to the case of a letter.

**Examples:**

- Normal case: "pizza" finds "Ronny's Pizza".
- Unwanted case: an empty text, or more than 100 characters. The server refuses it.

### `BOGOFFER-FR-013`

The server must return at most the requested count of offers for each bank, and the
count of every matching offer of the bank.

**Why:** A model must know that more offers exist.

### `BOGOFFER-FR-014`

The server must return the terms of one offer, given by its bank code and its id.

**Examples:**

- Normal case: `bog`, `16902`. The answer holds the models and the prices of Mazda.
- Unwanted case: an id that the bank does not list now. The server reports that it
  knows no such offer, and names the tool that searches the offers.

### `BOGOFFER-FR-015`

The server must give each text in the requested language: `ka` or `en`. The default is `en`.

### `BOGOFFER-FR-016`

The server must give, for each bank, the instant at which it read the offers.

**Why:** The source publishes no instant.

### `BOGOFFER-FR-017`

The server must report a bank that does not answer, and return the offers of the other banks.

### `BOGOFFER-FR-018`

The server must return offers that it read less than 25 hours before, marked as stale,
when a bank does not answer.

**Why:** A bank adds offers a few times in a week. Offers of yesterday help more than none.

### `BOGOFFER-FR-019`

The server must refuse a bank code that it does not know or cannot read. The message must
name the codes that it accepts.

### `BOGOFFER-FR-020`

The server must read BOG with no key. The operator must be able to turn BOG off. It is on
by default.

### `BOGOFFER-FR-021`

The server must tell the model, in the description of each tool, to give the person the
address of the page of each offer that it names.

**Why:** A benefit text is short. The page of the bank holds the conditions and the current state.

### `BOGOFFER-FR-022`

The server must skip the requested count of matching offers of each bank before it returns
offers, in the order of the answer. It must give the count to skip for the next page of a
bank when more matching offers remain.

**Why:** A bank has more offers than one answer holds. A person who asks for every offer of
SOLO must get each one.

**Examples:**

- Normal case: 130 matching offers, the limit 30, the offset 30. The answer holds offers
  31 to 60, and the offset 60 for the next page.
- Limit case: the offset 120. The answer holds the last 10 offers and no next offset.
- Unwanted case: the offset -1. The server refuses it.

## 7. Non-functional requirements

### `BOGOFFER-NFR-001`

An offer in an answer must be less than 1 hour old, unless the answer marks it as stale.
Measure from the read instant to the receipt of the request.

### `BOGOFFER-NFR-002`

The server must read the full list of a bank at most one time in each hour for each
language while the bank answers, for any count of requests. Measure the calls at the source.

### `BOGOFFER-NFR-003`

The server must answer a search in less than 500 ms when the offers of each requested bank
are in the memory of the server. Measure from the receipt of the request to the last byte.

### `BOGOFFER-NFR-004`

The server must answer a search in less than 10 seconds when it must read the offers of BOG.
Measure from the receipt of the request to the last byte.

## 8. Invariants

### `BOGOFFER-INV-001`

Every offer in an answer carries its bank, one tier or more, one offer category or more,
one benefit kind or more, and the address of its page at the bank.

**Why:** A person must be able to open the offer at the bank and check its conditions.

### `BOGOFFER-INV-002`

A benefit text and a terms text have the words of the bank. The server removes markup and
the white space at the start and the end only.

### `BOGOFFER-INV-003`

No offer in an answer has an end date before the current date in Tbilisi.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                               |
| -------------------- | ------------- | -------------------------------------------------------------------- |
| `DOM-001`, `CLT-001` | Domain        | The context is `offers`. BOG is a vendor of `rates` too. `ADR-0004`. |
| `DOM-007`            | Domain        | A start date and an end date are civil dates in Tbilisi.             |
| `MCP-002`            | MCP tools     | The tools are `offers_search_offers` and `offers_get_offer`.         |
| `MCP-005`            | MCP tools     | Each answer names the bank and the instant at which the server read it. |
| `MCP-007`, `MCP-008` | MCP tools     | Each tool takes `language`. The search takes `limit`.               |
| `MCP-009`            | MCP tools     | A date is `YYYY-MM-DD`.                                              |
| `CLT-009`, `CLT-011` | Clients       | The client says that the API is unofficial. It stays at `v0`.        |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `offers`.              |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 1 hour, the stale age 24 hours.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Which tools does the context offer? | Temuri | A search and a get. No comparison tool (2026-10-20). |
| 2   | Does the server read a number from a benefit text? | Temuri | No. The model reads the text (2026-10-20). |
| 3   | How old can an offer be? | Temuri | 1 hour, and 24 hours when the bank fails (2026-10-20). |
| 4   | Does the answer hold the offers of every customer, and the cards as a filter? | Temuri | Yes, both (2026-10-20). |
| 5   | Does the owner accept `ADR-0004`: a vendor in two contexts? | Temuri | Yes (2026-10-20). |
| 6   | Does the owner accept an unofficial API with no published terms? The page of BOG reads it with no key. | Temuri | Yes, for BOG, TBC, and Liberty (2026-10-20). |

## Retired identifiers

This file has no retired identifier.
