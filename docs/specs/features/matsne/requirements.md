---
id: MATSNE
title: The legal acts of the Legislative Herald
type: requirements
status: approved
created: 2026-11-15
updated: 2026-11-15
context: law
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The legal acts of the Legislative Herald

## 1. Problem

A person asks "what does article 82 of the Tax Code say", "what is the notice period in
the Labour Code", or "what did the Tax Code say on 2025-01-01". The server has no answer.
The Legislative Herald publishes every legal act of Georgia with each of its versions.
A person must search its site, open a page of 3 MB, and find the article by hand.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | Find an act. Read one article of it, now or on a past date, in `ka` or `en`. |
| The operator           | Turns the Legislative Herald off with no change to the code.              |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| The Legislative Herald | The pages `matsne.gov.ge/{ka,en}/document/search` and `/document/view/{id}`, and the method `POST /document/suggest` | Yes. The official publisher of the acts of Georgia. A web page for people, with no API. | At each new act and at each consolidated version. The Tax Code had 3 versions in October 2026. | No key. No published terms found. `robots.txt` sets `Crawl-delay: 10` and does not block `/document/`. |

Observed behavior of the source on 2026-11-09 and on 2026-11-15:

- A bot filter guards the site since 2026-11-09 or later. A reader that is not a browser
  gets an "Access Denied" page of 840 bytes with the status 200, on every path. A browser
  gets every page.
- A search page gives 20 results and the total count. A result gives the identifier, the
  title, the document type, the issuer, the number, the signing date, and the status.
- The search has no order by relevance. "საგადასახადო კოდექსი" gives 29 active acts, and
  the Tax Code is not on the first page. With the type "Law", it gives the Tax Code only.
- The search in `/en/` matches the English title of a translated act: "Labour Code" finds
  the act 1155567.
- Suggest takes a Georgian text of 3 or more characters and gives at most 50 titles, each
  with its identifier and status. A text in English gives no title.
- A search of the text of the acts for "მინიმალური ხელფასი" gives 42 acts in 0.5 seconds.
- An act has consolidated versions, each with a number and a date. The Tax Code has 249,
  from 0 (2010-10-12) to 248 (2026-10-28). Two versions can share a date.
- The page of the Tax Code is 3.1 MB in `ka` and 3.0 MB in `en`. It marks 15 sections,
  45 chapters, 367 articles, and 4,378 paragraphs.
- An inserted article carries a superscript number: "მუხლი 133³" follows article 133.
- A short act (5441237) marks no part. Its text is 76,000 characters.
- The page `/en/` of an act with no translation shows the Georgian text, with no error.
- An unknown identifier, or an unknown version, gives the "Access Denied" page, even to a
  browser that gets every other page.

## 4. Out of scope

- The download of an act as a PDF.
- The search filters subject, registration code, publication date, and normative status.
- The date on which an amendment takes effect. A version carries its publication date only.
- The language `ru`, and the parallel texts.
- A comparison of two versions.
- The links from an amendment note to the act that amends.
- A crawl or a full copy of the acts.

## 5. Definitions

The glossary holds the terms of this feature, in the section Law.

### 5.1 Document types

| Code                         | Name at the Legislative Herald               |
| ---------------------------- | -------------------------------------------- |
| `law`                        | Law (this includes a code and an organic law) |
| `parliament_resolution`      | Resolution of the Parliament of Georgia      |
| `presidential_decree`        | Presidential Decree                          |
| `government_resolution`      | Resolution                                   |
| `government_decree`          | Decree                                       |
| `edict`                      | Edict                                        |
| `order`                      | Order                                        |
| `international_treaty`       | International Treaty                         |
| `constitutional_court_decision` | Decisions of the Constitutional Court of Georgia |
| `court_act`                  | Court Acts                                   |
| `self_government_ordinance`  | Self-government Ordinance                    |
| `self_government_decision`   | Self-government Decision                     |
| `revenue_service_order`      | Orders of the Revenue Service                |
| `individual_legal_act`       | Individual Legal Act                         |
| `eu_law`                     | European Union Law                           |
| `draft`                      | Draft Normative Acts                         |

## 6. Functional requirements

### `MATSNE-FR-001`

The server must return the acts whose title holds a text, with the act identifier, title,
document type, issuer, number, signing date, and act status of each.

**Examples:**

- Normal case: `შრომის კოდექსი` with the type `law`. The result holds 1155567.
- Unwanted case: `zzqxv`. No act, and no failure.

### `MATSNE-FR-002`

The server must search the text of the acts in place of the titles when the request asks
for it.

### `MATSNE-FR-003`

The server must filter a search by each of these, when the request gives it: document
type, act status, issuer, number, and a range of signing dates.

**Why:** The source has no order by relevance. A filter is the only way to bring an act
to the first page.

### `MATSNE-FR-004`

The server must put first each active act whose title equals the text of a title search,
when the text is Georgian. Case, quotation marks, and spaces at the ends do not count.

**Examples:**

- Normal case: `საქართველოს საგადასახადო კოდექსი` with no filter. The Tax Code (1043717)
  is the first result.
- Limit case: `საგადასახადო კოდექსი`. No title is equal. The order of the source stays.

### `MATSNE-FR-005`

The server must order a search by the signing date, the newest first, unless the request
asks for the oldest first.

### `MATSNE-FR-006`

The server must state the total count of matched acts. It must return a page of results
that the request names, with at most 20 acts on a page.

### `MATSNE-FR-007`

The server must return the metadata of an act for an act identifier: title, document type,
issuer, number, signing date, act status, and each version with its number and date.

**Examples:**

- Normal case: 1043717. 249 versions. The newest is 248, of 2026-10-28.
- Unwanted case: an unknown identifier. The server tells the model to search the acts.

### `MATSNE-FR-008`

The server must return the table of contents of an act when the request names no article.

### `MATSNE-FR-009`

The server must return the full text of each article that the request names, with its
number and title, in the order of the act. A request names at most 10 articles.

**Examples:**

- Normal case: 1043717, article `82`.
- Limit case: 1043717, article `133^3`. The article after 133 and its inserted siblings.
- Unwanted case: article `999` of the Tax Code. The server states that the act has no
  such article, and names the first and the last article.

### `MATSNE-FR-010`

The server must return the text of an act that marks no part in pieces of at most 50,000
characters, with the number of the piece and the count of pieces.

### `MATSNE-FR-011`

The server must read the newest version unless the request names a version number or a date.
For a date, it must read the newest version whose date is on or before that date.

**Examples:**

- Normal case: 1043717 and 2025-01-01. The newest version published on or before that date.
- Limit case: two versions share the date. The higher number wins.
- Unwanted case: a date before version 0. The server names the date of version 0.

### `MATSNE-FR-012`

The server must give a search and an act in the language of the request: `ka` or `en`.
The default is `en`.

### `MATSNE-FR-013`

The server must refuse an act in `en` when the Legislative Herald has no translation of it,
and tell the model to ask for `ka`.

**Why:** The source shows the Georgian text under an English page. A model must not
take it for a translation.

### `MATSNE-FR-014`

The server must state that an English text is a translation and that the Georgian text
is the only official text.

### `MATSNE-FR-015`

The server must refuse an unknown document type, act status, or article number form,
and name the accepted values.

### `MATSNE-FR-016`

The server must treat an "Access Denied" page or a page that it cannot read as a failure
of the Legislative Herald.

### `MATSNE-FR-017`

The server must read the Legislative Herald with no key. The operator must be able to turn
it off. It is on by default.

## 7. Non-functional requirements

### `MATSNE-NFR-001`

An answer must hold at most 60,000 characters of text from the source. If the requested
articles pass that size, the server returns the first whole articles that fit and names
the next one. Measure on the structured output.

**Why:** A whole act fills the context window of the model.

### `MATSNE-NFR-002`

A version with a number must be less than 24 hours old in the cache. The newest version
and a search must be less than 1 hour old, unless the answer marks them as stale.
A stale value must be less than 7 days old.

**Why:** A numbered version never changes. A new version or act can appear on any working day.

### `MATSNE-NFR-003`

The server must add no wait between two calls to the Legislative Herald below 10 calls
in each second. It must send at most 10 calls in each second, for any count of requests.
Measure the calls at the source.

**Why:** A wait slows each request of a person. The cap of 10 calls protects the source
from a burst, and it is the limit of `RSL-008`. The server does not follow the
`Crawl-delay` of `robots.txt`, because it does not crawl.

### `MATSNE-NFR-004`

A tool call must answer in less than 15 seconds when the source answers each call in less
than 5 seconds. Measure from the receipt of the request to the last byte.

**Why:** A page of 3 MB takes about 1 second to transfer from Tbilisi.

### `MATSNE-NFR-005`

Every call to the Legislative Herald must carry a User-Agent that names Georgia MCP and
the address of its repository. Measure at the source.

## 8. Invariants

### `MATSNE-INV-001`

Every text in an answer is the text that the Legislative Herald publishes, with no
summary and no change of words.

### `MATSNE-INV-002`

Every answer of an act names the act identifier, the version number, the version date,
and the address of the page at the Legislative Herald.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                       |
| -------------------- | ------------- | ------------------------------------------------------------ |
| `DOM-001`            | Domain        | The context is `law`.                                        |
| `MCP-002`            | MCP tools     | Each tool name starts with `law_`.                           |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                    |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The search takes `limit`.         |
| `CLT-002`            | Clients       | The client reads web pages and sends a browser handshake.    |
| `CLT-008`            | Clients       | The client does not cache. The cache of `RSL-003` keeps the acts. |
| `CFG-008`            | Configuration | The switch of the Legislative Herald is on by default.       |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale periods of `MATSNE-NFR-002`. |
| `RSL-008`            | Resilience    | The limit of `MATSNE-NFR-003` is a setting.                  |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `MATSNE` and `law` (2026-11-15). |
| 2   | How does the server pass the bot filter? | Temuri | With a browser handshake, as `GULFUEL` does. Stage 3 confirms it (2026-11-15). |
| 3   | How does a model read a long act? | Temuri | The table of contents, then articles by number (2026-11-15). |
| 4   | Which behaviors are in the first version? | Temuri | Past versions, English text, the exact title first, and the search of the text (2026-11-15). |
| 5   | `robots.txt` asks for 10 seconds between calls. Which limit? | Temuri | No wait between calls. A cap of 10 calls in each second (2026-11-15). |
| 6   | Does `en` with no translation refuse the request, or return the Georgian text with a mark? | Temuri | It refuses and names `ka` (2026-11-15). |
| 7   | What are the limits of one answer? | Temuri | 10 articles and 60,000 characters (2026-11-15). |

## Retired identifiers

This file has no retired identifier.
