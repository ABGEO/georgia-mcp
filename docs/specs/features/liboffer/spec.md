---
id: LIBOFFER
title: The card offers of Liberty Bank for its Studio customers
type: spec
status: approved
created: 2026-10-20
updated: 2026-10-20
context: offers
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [ARC, DOM, MCP, CLT, RSL, CFG, TST]
requirements: features/liboffer/requirements.md
---

# Specification: The card offers of Liberty Bank for its Studio customers

## 1. Summary

This feature adds the vendor client module `pkg/liberty` and the driven adapter
`adapter/liberty` of the context `offers`. The tools of `BOGOFFER` answer for Liberty with
no change. A read builds the categories, the kinds, and the cards of each offer from one
list for each category and each type of Liberty. The terms come from the page of the offer.

## 2. Coverage

| Requirement        | Where this specification realizes it            |
| ------------------ | ----------------------------------------------- |
| `LIBOFFER-FR-001`  | 4.1 `adapter/liberty`. `LIBOFFER-DD-001`        |
| `LIBOFFER-FR-002`  | 4.1 `adapter/liberty`                           |
| `LIBOFFER-FR-003`  | `LIBOFFER-DD-003`                               |
| `LIBOFFER-FR-004`  | `LIBOFFER-DD-001`, `LIBOFFER-DD-002`            |
| `LIBOFFER-FR-005`  | `LIBOFFER-DD-001`, `LIBOFFER-DD-002`            |
| `LIBOFFER-FR-006`  | `LIBOFFER-DD-001`, `LIBOFFER-DD-002`            |
| `LIBOFFER-FR-007`  | 4.1 `pkg/liberty` `OfferPage`, `adapter/liberty` |
| `LIBOFFER-FR-008`  | 4.1 `adapter/liberty`                           |
| `LIBOFFER-FR-009`  | 4.3 configuration                               |
| `LIBOFFER-NFR-001` | 4.3 configuration. `LIBOFFER-DD-004`. `LIBOFFER-SC-008` |

## 3. Guideline compliance

| Rule                 | How this specification obeys it                                                  |
| -------------------- | -------------------------------------------------------------------------------- |
| `CLT-001`            | `pkg/liberty` is a module: `github.com/abgeo/georgia-mcp/pkg/liberty`.           |
| `CLT-002`, `ADR-0003` | `pkg/liberty` uses the standard library, and `golang.org/x/net/html` for the page of an offer. |
| `CLT-003`            | The types of the API carry JSON tags. The type of the page carries the text as the page shows it, with no tag. |
| `CLT-005`            | The client sets `Accept: application/json` on a call of the API, and `Accept: text/html` on the page. |
| `CLT-009`, `CLT-011` | The package comment and the README say that the API is unofficial. `v0`.         |
| `DOM-005`, `DOM-011` | `adapter/liberty` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `RSL-003`, `RSL-008` | The `OfferCache` of `BOGOFFER`. The configuration gives the limit of the calls.  |
| `CFG-003`, `CFG-005` | Each variable has the prefix `GEOMCP_OFFERS_LIBERTY_` and a default in its tag.  |

## 4. Design

### 4.1 Components

`O` is `internal/app/georgia-mcp/offers`.

| Layer          | Path                          | Types and functions                                     |
| -------------- | ----------------------------- | ------------------------------------------------------- |
| Vendor client  | `pkg/liberty/`                | `Client`, `New`, `Option`, `Categories`, `Types`, `Offers`, `OfferPage`, `Category`, `Type`, `OffersQuery`, `Offer`, `Page`, `APIError`, `ErrUnexpectedStatus`, `ErrNoSuccess`, `ErrNoTerms`. |
| Driven adapter | `O/adapter/liberty/`          | `Adapter`, `NewAdapter`, the tables of `LIBOFFER-DD-002`. |
| Module         | `O/module.go`, `O/config.go` (change) | `LibertyConfig`, `LibertyHTTPConfig`. The source of Liberty. |
| Configuration  | `configs/.env.example` (change) | The variables of 4.3.                                 |
| Workspace      | `go.work` (change)            | `use ./pkg/liberty`.                                    |

**`pkg/liberty`.**

```go
const DefaultBaseURL = "https://studioliberty.ge"

var (
    ErrUnexpectedStatus = errors.New("liberty: unexpected status")
    ErrNoSuccess        = errors.New("liberty: the answer reports no success")
    ErrNoTerms          = errors.New("liberty: the page has no block of terms")
)

func New(opts ...Option) *Client // WithBaseURL, WithHTTPClient, WithUserAgent
func (c *Client) Categories(ctx context.Context, language string) ([]Category, error)
func (c *Client) Types(ctx context.Context, language string) ([]Type, error)
func (c *Client) Offers(ctx context.Context, q OffersQuery) ([]Offer, error) // one page
func (c *Client) OfferPage(ctx context.Context, id int, language string) (Page, error)

type Category struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
}

type Type struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
}

type OffersQuery struct {
    CategoryID int    // 0: every category
    TypeID     int    // 0: every type
    Page       int    // from 1
    Language   string // "ka" or "en"
}

type Offer struct {
    ID            int      `json:"id"`
    Title         string   `json:"title"`
    ShortContent  string   `json:"shortContent"`
    URL           string   `json:"url"`
    Labels        []string `json:"labels"`
    RemainingDays string   `json:"remainingDays"` // "72 Days left"
}

type Page struct {
    Title     string   // the text of h1.featured-hero__title
    Timer     string   // the text of span.featured-card__timer-text
    Paragraphs []string // the text of each p in div.featured-hero__text, in order
}
```

`Categories`, `Types`, and `Offers` send `POST /_mvcapi/offersapi/<loadcategories,
loadtypes, loadOffers>` with a JSON body: `{"langAbbr":"en"}`, and for `Offers`
`{"categoryId":…,"typeId":…,"page":…,"sortBy":"","langAbbr":…}` with `null` for a zero id.
Each answer is `{"data":{"list":[…]},"succeeded":true,"message":""}`. `succeeded` false
returns an error that wraps `ErrNoSuccess`. `OfferPage` sends
`GET /<language>/studio-lifestyle/offers/details/<id>` and parses the body with
`golang.org/x/net/html`. A page with no `div.featured-hero__text` returns `ErrNoTerms`.
The body limit is 2 MiB. The User-Agent is `georgia-mcp-liberty/<version> (+…)`.

**`adapter/liberty`.** `Bank` returns `{Code: "liberty", Name: "Liberty Bank"}`. `Cards`
returns the cards of `LIBOFFER-DD-002`. `Offers` follows `LIBOFFER-DD-001`. The mapping:

| Field of the vendor      | Domain value                                                         |
| ------------------------ | -------------------------------------------------------------------- |
| `id`                     | `ID`, as a decimal text.                                             |
| `title`                  | `Title`, trimmed with `strings.TrimSpace`. It removes U+00A0.        |
| `labels`                 | `Benefit`: each label trimmed, the empty ones dropped, joined by "; ". No label: `shortContent`, trimmed. |
| `remainingDays`          | `Period.End`. `LIBOFFER-DD-003`.                                     |
| the lists that hold it   | `Categories`, `BankCategories`, `Kinds`, `Cards`.                    |
| `id`                     | `URL`: `https://studioliberty.ge/<ka or en>/studio-lifestyle/offers/details/<id>`. |

Every offer has the tiers `[premium]` and no partner. The adapter leaves `Terms` empty.
`Terms` reads `OfferPage` and joins the trimmed, non-empty paragraphs with a newline. Each
error wraps `kernel.ErrSourceUnavailable`.

### 4.2 Domain model

No change. `LIBOFFER` uses the model of `BOGOFFER` 4.2.

### 4.3 Declarations

**Vendor endpoint.**

| Client         | Method and path                                       | Key  | Official                  | Realizes                               |
| -------------- | ----------------------------------------------------- | ---- | ------------------------- | -------------------------------------- |
| `pkg/liberty`  | `POST /_mvcapi/offersapi/loadcategories`              | None | No, an unofficial API     | `LIBOFFER-FR-004`                      |
| `pkg/liberty`  | `POST /_mvcapi/offersapi/loadtypes`                   | None | No, an unofficial API     | `LIBOFFER-FR-005`, `LIBOFFER-FR-006`   |
| `pkg/liberty`  | `POST /_mvcapi/offersapi/loadOffers`                  | None | No, an unofficial API     | `LIBOFFER-FR-001` to `LIBOFFER-FR-006` |
| `pkg/liberty`  | `GET /<language>/studio-lifestyle/offers/details/<id>` | None | No, a web page source    | `LIBOFFER-FR-007`                      |

A real answer, recorded on 2026-10-20, cut to one offer:

```
POST https://studioliberty.ge/_mvcapi/offersapi/loadOffers
{"categoryId":null,"typeId":null,"page":1,"sortBy":"","langAbbr":"en"}
200 application/json
{"data":{"list":[{"id":727,"title":"Hyatt Regency Bangkok Sukhumvit   ",
  "shortContent":"","imageUrl":"https://CDN.lb.ge/lbstudio/…/hyatt-regency-bangkok-sukhumvit.jpg",
  "url":"https://StudioLiberty.ge/en/studio-lifestyle/offers/details/727-hyatt-regency-bangkok-sukhumvit - ",
  "wealthUrl":"","labels":["Special Hotel Offer"],"remainingDays":"72 Days left"}]},
 "succeeded":true,"message":""}
```

**Cache.** The `OfferCache` of `BOGOFFER` 4.3, with the same keys and durations.

**Configuration.**

| Variable                            | Type     | Default                    | Secret | Realizes           |
| ----------------------------------- | -------- | -------------------------- | ------ | ------------------ |
| `GEOMCP_OFFERS_LIBERTY_ENABLED`     | bool     | `true`                     | No     | `LIBOFFER-FR-009`  |
| `GEOMCP_OFFERS_LIBERTY_BASE_URL`    | URL      | `https://studioliberty.ge` | No     | `LIBOFFER-FR-001`  |
| `GEOMCP_OFFERS_LIBERTY_TIMEOUT`     | duration | `10s`                      | No     | `LIBOFFER-NFR-001` |
| `GEOMCP_OFFERS_LIBERTY_RATE`        | float    | `5`                        | No     | `LIBOFFER-NFR-001` |
| `GEOMCP_OFFERS_LIBERTY_BURST`       | int      | `5`                        | No     | `LIBOFFER-NFR-001` |

## 5. Design decisions

### `LIBOFFER-DD-001`

**Realizes:** `LIBOFFER-FR-001`, `LIBOFFER-FR-004` to `LIBOFFER-FR-006`
**Decision:** A read of Liberty in one language:

1. `Categories` and `Types`.
2. `Offers` with no filter, from page 1 until a page holds less than 16 offers, at most 30 pages.
3. The same pages for each category id and each type id. At most 4 lists run at once.
4. Each offer of step 2 gets the values of each list that holds its id.

Any failed call fails the read. On 2026-10-20 a read needs 67 calls.
**Rationale:** The list holds no category, no type, and no date. The page of each offer
holds no category either.
**Alternatives:** A guess of the kind from a label. The owner rejected a parse of a text.

### `LIBOFFER-DD-002`

**Realizes:** `LIBOFFER-FR-004` to `LIBOFFER-FR-006`
**Decision:** Fixed tables by id. An id that is not in a table gives the value of the last
row, and a warning one time for each id.

| Category id | Name (en, 2026-10-20) | Category        |
| ----------- | --------------------- | --------------- |
| 1           | Auto-moto             | `auto`          |
| 3           | Cafe / Restaurant     | `dining`        |
| 4           | Shopping              | `shopping`      |
| 5           | Household appliances  | `electronics`   |
| 6           | Leisure               | `leisure`       |
| 7           | Education             | `education`     |
| 8           | For Kids              | `kids`          |
| 9           | Gifts                 | `shopping`      |
| 10          | Fitness               | `health_beauty` |
| 11          | For the house         | `home`          |
| 14          | Developers            | `real_estate`   |
| 13, other   | Other                 | `other`         |

| Type id | Name                   | Value                                     |
| ------- | ---------------------- | ----------------------------------------- |
| 1       | Discount               | the kind `discount`                       |
| 2       | Cashback               | the kind `cashback`                       |
| 3       | Visa                   | the card `visa`, "Visa"                   |
| 4       | MasterCard             | the card `mastercard`, "Mastercard"       |
| 5       | Top\|Card Black Edition | the card `top_card_black`, "Top|Card Black Edition" |
| other   |                        | the card `type_<id>`, with the name of the type |

The bank category is the title of the category in the language, trimmed.
**Rationale:** An id is stable. A title has a trailing space in Georgian: "ავტო-მოტო ".
**Alternatives:** A match on the title.

### `LIBOFFER-DD-003`

**Realizes:** `LIBOFFER-FR-003`, `DOM-007`
**Decision:** The adapter reads the first run of digits of `remainingDays` as N. The end
date is the date of the read instant in Tbilisi plus N days. No digit: a zero end date.
**Rationale:** Offer 727 had "72 Days left" on 2026-10-20 and the end 2026-12-31 in its
terms. The cache keeps the date of the read, so the date stays right for 25 hours.
**Alternatives:** A parse of the date from the terms. One page call for each offer, and a
text with no fixed form.

### `LIBOFFER-DD-004`

**Realizes:** `LIBOFFER-NFR-001`, `BOGOFFER-NFR-002`
**Decision:** The rate is 5 calls each second, with a burst of 5.
**Rationale:** 67 calls take about 14 seconds at this rate, inside 30 seconds.
**Alternatives:** 1 call each second. About 70 seconds.

## 6. Scenarios

`pkg/liberty/testdata/` holds, recorded on 2026-10-20 in `en`: `categories.json`,
`types.json`, `offers-1.json` to `offers-13.json`, `category-6-1.json` to
`category-6-5.json`, `type-2-1.json` to `type-2-9.json`, `type-3-1.json` to
`type-3-10.json`, and `details-727.html`. A test serves an empty list for every other
filter.

### `LIBOFFER-SC-001`

**Verifies:** `LIBOFFER-FR-001`
**Layer:** unit

**Given** `httptest` that records the request and serves `offers-1.json`.
**When** `Offers` runs with page 1 and `en`.
**Then** the path is `/_mvcapi/offersapi/loadOffers`, the body has `"categoryId":null`,
`"page":1`, and `"langAbbr":"en"`, and the list holds 16 offers.

### `LIBOFFER-SC-002`

**Verifies:** `LIBOFFER-FR-008`
**Layer:** unit

**Given** an answer with the status 200 and `"succeeded":false`.
**When** `Offers` runs.
**Then** the error wraps `ErrNoSuccess`.

### `LIBOFFER-SC-003`

**Verifies:** `LIBOFFER-FR-007`
**Layer:** unit

**Given** `httptest` serves `details-727.html`, then a page with no block of terms.
**When** `OfferPage` runs for 727 and `en`, then again.
**Then** the first gives the title "Hyatt Regency Bangkok Sukhumvit" and 13 paragraphs. The
last is "• The offer is valid until December 31, 2026". The second returns `ErrNoTerms`.

### `LIBOFFER-SC-004`

**Verifies:** `LIBOFFER-FR-001`, `LIBOFFER-FR-002`, `LIBOFFER-FR-003`
**Layer:** integration

**Given** `httptest` serves the files, and the read instant is 2026-10-20 12:00 in Tbilisi.
**When** the adapter reads the offers in `en`.
**Then** the catalog holds 206 offers, each with the tiers `[premium]`. Offer `727` has the
title "Hyatt Regency Bangkok Sukhumvit", the benefit "Special Hotel Offer", no start date,
and the end date 2026-12-31.

### `LIBOFFER-SC-005`

**Verifies:** `LIBOFFER-FR-004`, `LIBOFFER-FR-005`, `LIBOFFER-FR-006`
**Layer:** integration

**Given** the same files.
**When** the adapter reads the offers.
**Then** each offer of the 5 pages of category 6 has `leisure` and the bank category
"Leisure". Each offer of type 2 has the kind `cashback`. Each offer of type 3 has the card
`{visa, Visa}`. An offer in no type has the kind `[other]` and no card.

### `LIBOFFER-SC-006`

**Verifies:** `LIBOFFER-FR-003`
**Layer:** unit

**Given** the texts "72 Days left", "დარჩა 3 დღე", and "Last day", on 2026-10-20.
**When** the adapter maps them.
**Then** the end dates are 2026-12-31, 2026-10-23, and a zero date.

### `LIBOFFER-SC-007`

**Verifies:** `LIBOFFER-FR-007`
**Layer:** integration

**Given** `httptest` serves the files.
**When** a client calls `offers_get_offer` with `liberty`, `727`, `en`.
**Then** the terms hold one line for each paragraph and no blank line.

### `LIBOFFER-SC-008`

**Verifies:** `LIBOFFER-NFR-001`
**Layer:** live

**Given** the real Liberty and an empty cache.
**When** a search runs for `liberty`.
**Then** the answer arrives in less than 30 seconds.

### `LIBOFFER-SC-009`

**Verifies:** `BOGOFFER-FR-017`, `BOGOFFER-FR-010`
**Layer:** integration

**Given** BOG, TBC, and Liberty from `httptest`, and Liberty answers 500 for type 2.
**When** a client calls the search with `categories: ["leisure"]`.
**Then** the answer holds `bog` and `tbc`, and names Liberty Bank in `unavailable`.

## 7. Build plan

| #   | Step                                                                     | Realizes                                | Done |
| --- | ------------------------------------------------------------------------ | --------------------------------------- | ---- |
| 1   | `pkg/liberty` module, recorded answers and page, README, example. `go.work`. | `LIBOFFER-FR-001`, `LIBOFFER-FR-007` | [ ]  |
| 2   | `adapter/liberty`: the read of `LIBOFFER-DD-001`, the tables, the terms. | `LIBOFFER-FR-001` to `LIBOFFER-FR-008`  | [ ]  |
| 3   | `config.go`, `module.go`, `configs/.env.example`.                        | `LIBOFFER-FR-009`, `LIBOFFER-NFR-001`   | [ ]  |
| 4   | Move the terms of requirements 5 to the glossary.                        | `DOM-002`                               | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
