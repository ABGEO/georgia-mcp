---
id: JSONLD
title: The structured data
type: requirements
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [LNG]
---

# Requirements: The structured data

## 1. Problem

A search engine or a model that reads the home page sees text, not facts. It cannot tell
the name of the site, that Georgia MCP is free software, its licence, its version, its
repository, or its author, without guessing from the copy.

## 2. Users

| User                     | Need                                                                     |
| ------------------------ | ------------------------------------------------------------------------ |
| A search engine          | The name of the site for its results, and the facts of the software.     |
| A model or a crawler     | The licence, the version, the repository, and the author, as fields.     |

## 3. Source

| Vendor      | Source                                                                 | Official                 | Publication period         | Terms of use |
| ----------- | ---------------------------------------------------------------------- | ------------------------ | -------------------------- | ------------ |
| Georgia MCP | The constants and the copy of the site of `WEBSITE`, the site version of `WEBSITE-FR-039`, and the licence of the repository | Yes. The project is ours. | At each build of the site. | None. |

## 4. Out of scope

- Structured data on a tool page, a legal page, or the page of `WEBSITE-FR-021`.
- A rating, a review, or a count of downloads. The project has none to state.
- A rich result of a search engine. A software application gets one only with a rating or a review.

## 5. Definitions

The term is in the glossary, in the section "The website": structured data.
The vocabulary is schema.org. Each type below is a type of that vocabulary.

## 6. Functional requirements

### `JSONLD-FR-001`

Each home page must carry structured data of the type `WebSite`: the name of the site, its address, and the language of the page.

**Why:** A search engine shows the name of the site in its results from this type.

**Examples:**

- Normal case: `/` states the name "Georgia MCP", the address `https://geo-mcp.abgeo.cloud/`, and the language `en`.
- Limit case: `/ka/` states the same name, the address `https://geo-mcp.abgeo.cloud/ka/`, and the language `ka`.

### `JSONLD-FR-002`

Each home page must carry structured data of the type `SoftwareApplication` for the server: its name, its description in the language of the page, its category, its address, its licence, its price of zero, and its author. When the site version names a release, it must also state that version.

**Why:** A model reads the facts of the software as fields, not from the copy.

**Examples:**

- Normal case: a build of `v1.2.3` states the version `1.2.3`, the licence GPL 3.0, and the price 0 in GEL.
- Limit case: a build of `v1.1.2-8-g8def960` or `dev` states no version.
- Unwanted case: the description lists or counts the subjects of the server. A build test fails.

### `JSONLD-FR-003`

Each home page must carry structured data of the type `SoftwareSourceCode` for the repository: its address, its language of programming, its licence, and its author. It names the application of `JSONLD-FR-002` as its target.

**Why:** A crawler of code links the site to its source.

### `JSONLD-FR-004`

The author in `JSONLD-FR-002` and `FR-003` must be one `Person`: Temuri Takalandze, with the address of the site of the author and the profile on GitHub.

**Why:** One author, named the same way in each place, is one entity for a search engine.

## 7. Non-functional requirements

### `JSONLD-NFR-001`

The structured data of each home page must parse, and must report no error in the Schema Markup Validator of schema.org.
Measure the parse on each built home page. Measure the validator by hand on both home pages after the first deploy.

**Why:** Data that does not parse is ignored without a warning.

## 8. Invariants

### `JSONLD-INV-001`

The structured data of a home page agrees with the page: the same name as the page title states, the same description as the description of the page, and the version of the footer without its `v`.

**Why:** A search engine ignores or penalizes data that the page does not show.

## 9. Constraints from the guidelines

| Rule                 | Guideline | Effect on this feature                       |
| -------------------- | --------- | -------------------------------------------- |
| `LNG-006`, `LNG-007` | Language  | The description holds no em dash and no filler word. |

## 10. Open questions

None.
