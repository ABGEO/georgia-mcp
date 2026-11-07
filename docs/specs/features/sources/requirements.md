---
id: SOURCES
title: The sources page
type: requirements
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [BRD, MCP, LNG]
---

# Requirements: The sources page

## 1. Problem

A reader, a reviewer of a directory, or a vendor asks where the data of Georgia MCP comes
from. The site names the vendors of each tool, but not whether a source is official, where
it lives, or how an organization asks for a change. Section 3 of each feature holds these
facts, and a reader of the site does not see them.

## 2. Users

| User                   | Need                                                                      |
| ---------------------- | ------------------------------------------------------------------------- |
| A person who chats     | Know where an answer comes from, and whether the source is official.      |
| A reviewer of a directory | Check each source of the server on one page.                           |
| A vendor               | Find its own source, and the way to ask for a change or a removal.        |

## 3. Source

| Vendor      | Source                                                              | Official                 | Publication period         | Terms of use |
| ----------- | ------------------------------------------------------------------- | ------------------------ | -------------------------- | ------------ |
| Georgia MCP | Section 3 of each feature of a context, copied by the owner into the content of the site of `WEBSITE` | Yes. The project is ours. | At each build of the site. | None. |

## 4. Out of scope

- The address of the API of a source. The page links to a page for people.
- The key, the terms, the limits, and the publication period of a source.
- A check that the page agrees with section 3 of each feature. The owner keeps them in step.

## 5. Definitions

The terms are in the glossary: vendor, source, and, in the section "The website", sources page and kind of source.

## 6. Functional requirements

### `SOURCES-FR-001`

The site must have a sources page at `/sources/` and `/ka/sources/`. It lists each source once for each context that reads it, grouped by context in the order of the home page, with the name of the vendor in the language of the page.

**Why:** A reader finds a source under the subject that they asked about. A vendor that feeds two contexts can have a different source in each.

**Examples:**

- Normal case: the group "Fuel prices" lists Wissol, Gulf, and the other fuel companies.
- Limit case: Bank of Georgia appears under "Exchange rates" and under "Card offers", each with its own kind.
- Unwanted case: the copy of the page counts the sources or the subjects. A build test fails.

### `SOURCES-FR-002`

Each source must show its kind of source: an official open API, a documented developer API, an unofficial API, or a web page. A source that is not official must say so in words.

**Why:** A reader weighs an answer by its source. An unofficial source can change without notice.

**Examples:**

- Normal case: the NBG shows "Official open API".
- Limit case: the TTC shows "Unofficial API", with a sentence that the TTC does not publish it for reuse.

### `SOURCES-FR-003`

Each source must link to a public page of the vendor: the page that a person reads, or the documentation of the API.

**Why:** A reader checks a value at its origin.

**Examples:**

- Normal case: Gulf links to its page of fuel prices.
- Unwanted case: a link to the raw address of an API. A build test fails on a link whose path holds `/api`.

### `SOURCES-FR-004`

Each source must name the tools that read it, each linked to its tool page in the language of the page.

**Why:** A reader goes from a source to the tools that answer from it.

**Examples:**

- Normal case: Wissol lists `fuel_get_prices`.
- Limit case: the TTC lists each transit tool.

### `SOURCES-FR-005`

The sources page must say how the server reads a source: it calls the source when a tool runs, keeps an answer for a short time that depends on the source, sends the vendor only the input of the call, and names the source and the time of the data in each answer.

**Why:** A reader and a vendor know the load on a source and what reaches it.

### `SOURCES-FR-006`

The sources page must tell a vendor how to ask for a correction or a removal of its source: a form of the repository for a source request, and nothing else. The form asks for the source, the kind of request, and the details.

**Why:** An organization that did not choose to feed the server has a direct way to ask. One public channel keeps each request and its answer in one place.

### `SOURCES-FR-007`

The footer of each page must link to the sources page in the language of the page.

### `SOURCES-FR-008`

The list of vendors on each tool page must link to the sources page, to the group of the context of the tool.

**Why:** A reader of a tool goes to the facts of its sources in one step.

## 7. Non-functional requirements

This feature has no non-functional requirement of its own. `WEBSITE-NFR-001`, `NFR-002`, and `NFR-005` hold for the sources page, as for each page.

## 8. Invariants

### `SOURCES-INV-001`

The sources on the page are, for each context, the vendors that the site lists for that context, by code. No source lacks a kind or a link.

**Why:** A list that the tools do not match tells the reader a false thing.

### `SOURCES-INV-002`

The set of pages in English is still equal to the set of pages in Georgian. `WEBSITE-INV-002` holds for the sources page.

## 9. Constraints from the guidelines

| Rule                 | Guideline | Effect on this feature                                  |
| -------------------- | --------- | ------------------------------------------------------- |
| `BRD-014`, `BRD-015` | Brand     | The copy of the page, in English and in Georgian.       |
| `MCP-005`            | MCP tools | `SOURCES-FR-005` names the source and the time of each answer. |
| `LNG-006`, `LNG-007` | Language  | The copy holds no em dash and no filler word.           |

## 10. Open questions

None.
