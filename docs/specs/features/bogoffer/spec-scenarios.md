---
id: BOGOFFER
title: The card offers of Bank of Georgia, the scenarios
type: spec
status: approved
created: 2026-10-20
updated: 2026-10-20
context: offers
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [TST]
requirements: features/bogoffer/requirements.md
---

# Specification: The card offers of Bank of Georgia, the scenarios

`pkg/bogoffers/testdata/` holds the answers that the tests serve:

| File                  | Content                                                                   |
| --------------------- | ------------------------------------------------------------------------- |
| `offers-en-0.json`    | The real page 0 of `get-offers` with `Accept-Language: en`, recorded on 2026-10-20. 100 offers. |
| `offers-en-1.json`, `offers-en-2.json` | Pages 1 and 2 of the same read. 139 offers.             |
| `offers-ka-0.json`    | The real page 0 in Georgian.                                              |
| `offers-edge.json`    | One page of 6 offers: one with `segmentTypes` `VIP`, one with the category 999, one with the product `GOLD_CARD`, one with no title, one with `endDate` `null`, one with `benefitName` `OTHER`. |
| `error.json`          | The body of a status 500, recorded on 2026-10-20.                         |

The clock of every test of `app` and of the tool is 2026-10-20 12:00 in Tbilisi.
An integration test serves the files from `httptest` and calls the tool through the
in-memory MCP transport.

## The client

### `BOGOFFER-SC-001`

**Verifies:** `BOGOFFER-FR-001`, `BOGOFFER-FR-015`
**Layer:** unit

**Given** an `httptest` server that records the request and serves `offers-en-0.json`.
**When** `Offers` runs with `{Language: "en", PageNumber: 0, PageSize: 100}`.
**Then** the method is `POST`, the path is `/get-offers`, the query holds `pageNumber=0`
and `pageSize=100`, the body is `{}`, `Accept-Language` is `en`, the User-Agent starts with
`georgia-mcp/`, and the page holds 100 offers with `totalPageCount` 3.

### `BOGOFFER-SC-002`

**Verifies:** `BOGOFFER-FR-017`
**Layer:** unit

**Given** the vendor answers 500 with `error.json`.
**When** `Offers` runs.
**Then** the error is an `*APIError` with 500. `errors.Is(err, ErrUnexpectedStatus)` holds.

### `BOGOFFER-SC-003`

**Verifies:** `BOGOFFER-FR-001`
**Layer:** live

**Given** the real BOG.
**When** `Offers` runs for page 0 with the size 100.
**Then** the page holds one offer or more, and each has a campaign id and a title.

## The domain

### `BOGOFFER-SC-004`

**Verifies:** `BOGOFFER-INV-001`
**Layer:** unit

**Given** a table of offers: valid; no tier; no category; no kind; an empty title; an
address with `http`; the kinds `other` and `cashback`.
**When** `NewOffer` runs for each.
**Then** the first and the last pass. The last has the kinds `[cashback]`. Each other one
returns `ErrInvalidOffer`.

### `BOGOFFER-SC-005`

**Verifies:** `BOGOFFER-FR-008`, `BOGOFFER-FR-009`
**Layer:** unit

**Given** a table of periods and ranges:

| Period                     | Range                    | Overlaps |
| -------------------------- | ------------------------ | -------- |
| 2026-05-29 to 2027-05-22   | 2026-10-20 to 2026-10-20 | true     |
| 2026-10-30 to 2026-12-31   | 2026-10-20 to 2026-10-20 | false    |
| 2026-10-30 to 2026-12-31   | 2026-10-01 to 2026-10-31 | true     |
| 2026-09-01 to 2026-10-01   | 2026-10-01 to 2026-10-31 | true     |
| zero to 2026-10-21         | 2026-10-20 to 2026-10-20 | true     |
| 2026-01-01 to zero         | 2030-01-01 to 2030-01-01 | true     |

**When** `Overlaps` runs for each row.
**Then** each result is the value of the last column.

### `BOGOFFER-SC-006`

**Verifies:** `BOGOFFER-FR-010`
**Layer:** unit

**Given** the texts `vip`, `Leisure`, `money`, and `Visa Gold`.
**When** `ParseTier`, `ParseCategory`, `ParseBenefitKind`, and `ParseCardCode` run.
**Then** each returns its sentinel of 4.2.

## The adapter

### `BOGOFFER-SC-007`

**Verifies:** `BOGOFFER-FR-001`, `BOGOFFER-FR-002`, `BOGOFFER-FR-006`, `BOGOFFER-INV-002`
**Layer:** integration

**Given** `httptest` serves the three pages in English.
**When** `Offers` of `bog.Adapter` runs with `en`.
**Then** the catalog holds 239 offers. Offer `17409` has the title "Ronny's Pizza", the
benefit "20% Discount, unlimited", the categories `[dining]`, the bank categories
`[Restaurants]`, the period 2026-05-29 to 2027-05-22, and the address
`https://bankofgeorgia.ge/en/offers-hub/details/17409`. Its terms hold no `<p>`.

### `BOGOFFER-SC-008`

**Verifies:** `BOGOFFER-FR-003`, `BOGOFFER-FR-004`, `BOGOFFER-FR-005`
**Layer:** unit

**Given** `httptest` serves `offers-edge.json`.
**When** `Offers` runs.
**Then** the offer with `VIP` and the offer with no title are absent. The offer with the
category 999 has `[other]`. The card `GOLD_CARD` is `{gold_card, GOLD_CARD}`. The offer
with `OTHER` has the kinds `[other]`. The offer with a `null` end has a zero end. The log
holds one warning for `VIP`, one for 999, and one for `GOLD_CARD`.

### `BOGOFFER-SC-009`

**Verifies:** `BOGOFFER-FR-017`
**Layer:** unit

**Given** page 0 answers, and page 1 answers 500.
**When** `Offers` runs.
**Then** the error wraps `kernel.ErrSourceUnavailable`.

### `BOGOFFER-SC-010`

**Verifies:** `BOGOFFER-FR-017`
**Layer:** unit

**Given** a page with the status 200 and `"code":15`.
**When** `Offers` runs.
**Then** the error wraps `kernel.ErrSourceUnavailable`.

## The use cases

### `BOGOFFER-SC-011`

**Verifies:** `BOGOFFER-FR-007`, `BOGOFFER-FR-008`, `BOGOFFER-INV-003`
**Layer:** unit

**Given** a fake source with 4 offers: A ends 2026-10-19, B 2026-10-20 to 2026-10-31,
C starts 2026-10-30, D has no dates.
**When** `Search` runs with no dates, then with 2026-10-01 to 2026-10-31.
**Then** the first answer holds B and D. The second holds B, C, and D. A is in neither.

### `BOGOFFER-SC-012`

**Verifies:** `BOGOFFER-FR-010`, `BOGOFFER-FR-011`
**Layer:** unit

**Given** a fake source with: E `premium`, `leisure`, cards `[solo]`; F `standard`,
`leisure`, no card; G `premium`, `dining`, cards `[amex]`.
**When** `Search` runs with the tier `premium`, the category `leisure`, and the card `solo`.
**Then** the answer holds E only. With the card `visa` and no other filter, it holds F only.

### `BOGOFFER-SC-013`

**Verifies:** `BOGOFFER-FR-012`
**Layer:** unit

**Given** a fake source with "Ronny's Pizza" and "Mazda".
**When** `Search` runs with the text `PIZZA`.
**Then** the answer holds "Ronny's Pizza" only.

### `BOGOFFER-SC-014`

**Verifies:** `BOGOFFER-FR-013`
**Layer:** unit

**Given** a fake source with 25 matching offers.
**When** `Search` runs with the limit 10.
**Then** the bank has 10 offers, sorted by end date then title, and the total 25.

### `BOGOFFER-SC-015`

**Verifies:** `BOGOFFER-FR-014`
**Layer:** unit

**Given** a fake source whose catalog holds offer `16902` with terms, and a call counter on `Terms`.
**When** `Get` runs for `bog`, `16902`, then for `bog`, `123`.
**Then** the first returns the terms and the `ReadAt` of the catalog, with 0 calls to
`Terms`. The second returns `ErrOfferNotFound`.

### `BOGOFFER-SC-016`

**Verifies:** `BOGOFFER-FR-017`, `BOGOFFER-FR-019`
**Layer:** unit

**Given** two fake sources: `bog` answers, `xyz` fails with `kernel.ErrSourceUnavailable`.
**When** `Search` runs with no bank, then with `xyz` only, then with `abc`.
**Then** the first holds `bog` and names `xyz` in `Unavailable`. The second returns an error
that wraps `kernel.ErrSourceUnavailable`. The third returns `*UnknownBankError` with the
known codes `[bog, xyz]`.

### `BOGOFFER-SC-017`

**Verifies:** `BOGOFFER-FR-008`
**Layer:** unit

**Given** a fake source.
**When** `Search` runs with 2026-10-31 to 2026-10-01, then with 2026-01-01 to 2027-01-03.
**Then** each returns `ErrInvalidRange`.

### `BOGOFFER-SC-023`

**Verifies:** `BOGOFFER-FR-022`
**Layer:** unit

**Given** a fake source with 25 matching offers.
**When** `Search` runs with the limit 10 and the offset 0, then 10, then 20, then 30.
**Then** the pages hold 10, 10, 5, and 0 offers, with the next offsets 10, 20, 0, and 0.
The three pages hold the 25 ids with no repeat, in the order of the answer of the offset 0
with the limit 30. The total is 25 each time.

## The cache

### `BOGOFFER-SC-018`

**Verifies:** `BOGOFFER-NFR-001`, `BOGOFFER-FR-018`
**Layer:** unit

**Given** a fake source that answers at 10:00 and fails with `kernel.ErrSourceUnavailable`
after it, and a fake clock.
**When** `Offers` runs at 10:30, 11:30, and 11:30 of the next day.
**Then** the first is fresh with no new call. The second calls the source and returns the
list of 10:00 with `Stale` true. The third returns the error.

### `BOGOFFER-SC-019`

**Verifies:** `BOGOFFER-NFR-002`, `BOGOFFER-FR-015`
**Layer:** integration

**Given** `httptest` that counts the calls.
**When** 20 searches run at once in `en`, then one in `ka`.
**Then** the server sends 3 calls for `en` and 3 calls for `ka`.

### `BOGOFFER-SC-020`

**Verifies:** `BOGOFFER-NFR-003`
**Layer:** integration

**Given** the catalog of 239 offers in the cache.
**When** 100 searches run with the text `a` and every filter empty.
**Then** each answer arrives in less than 500 ms at the tool.

### `BOGOFFER-SC-021`

**Verifies:** `BOGOFFER-NFR-004`
**Layer:** live

**Given** the real BOG and an empty cache.
**When** a search runs.
**Then** the answer arrives in less than 10 seconds.

## The tools

### `BOGOFFER-SC-022`

**Verifies:** `BOGOFFER-FR-013`, `BOGOFFER-FR-016`, `BOGOFFER-FR-021`, `BOGOFFER-INV-001`, `MCP-004`, `MCP-005`
**Layer:** integration

**Given** `httptest` serves the pages of BOG.
**When** a client lists the tools, calls `offers_search_offers` with `banks: ["bog"]`,
`tiers: ["premium"]`, `categories: ["leisure"]`, `limit: 5`, then calls `offers_get_offer`
with the first id, then calls the search with `tiers: ["vip"]`.
**Then** each tool has the annotations of `MCP-004` and the description names the card
code `visa_infinite`. A search with `offset: -1` returns `IsError` with the message of 4.5,
and a search with `offset: 5` passes 5 to the use case. The search holds `bog` with `source` "Bank of Georgia", an `as_of`,
5 offers or less, and a `total`. Each offer has the tier `premium` and a `url` that
starts with `https://bankofgeorgia.ge/en/offers-hub/details/`. Each description tells the
model to give the person the url. The get returns the
terms of that id. The third call returns `IsError` with the message of 4.5.

## Retired identifiers

This file has no retired identifier.
