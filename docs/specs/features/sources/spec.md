---
id: SOURCES
title: The sources page
type: spec
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [BRD, MCP, LNG, BLD]
requirements: features/sources/requirements.md
---

# Specification: The sources page

## 1. Summary

Each vendor of `contexts.yaml` gains a kind of source and a link. A new view renders the
sources page at `/sources/` and `/ka/sources/` from the catalog of `WEBSITE-DD-004`, one
group for each context. The footer and the vendor list of each tool page link to it.

## 2. Coverage

| Requirement       | Where this specification realizes it                      |
| ----------------- | --------------------------------------------------------- |
| `SOURCES-FR-001`  | 4.3 Pages, `SOURCES-DD-002`, `SOURCES-SC-001`, `SC-003`   |
| `SOURCES-FR-002`  | 4.2, `SOURCES-DD-001`, `SOURCES-SC-004`                   |
| `SOURCES-FR-003`  | 4.2, `SOURCES-DD-001`, `SOURCES-SC-004`                   |
| `SOURCES-FR-004`  | `SOURCES-DD-002`, `SOURCES-SC-001`, `SC-005`              |
| `SOURCES-FR-005`  | 4.4 Copy, `SOURCES-SC-006`                                |
| `SOURCES-FR-006`  | 4.4 Copy, `SOURCES-DD-004`, `SOURCES-SC-006`              |
| `SOURCES-FR-007`  | `SOURCES-DD-003`, `SOURCES-SC-007`                        |
| `SOURCES-FR-008`  | `SOURCES-DD-003`, `SOURCES-SC-007`                        |
| `SOURCES-INV-001` | `SOURCES-DD-001`, `SOURCES-SC-002`, `SC-003`, `SC-004`    |
| `SOURCES-INV-002` | 4.3 Pages, `WEBSITE-SC-001`                               |

`WEBSITE-NFR-001`, `NFR-002`, and `NFR-005` hold for the new page. `WEBSITE-SC-028` and `SC-029` read each built page already. `SOURCES-SC-008` adds the page to the check of `WEBSITE-SC-031`.

## 3. Guideline compliance

| Rule                 | Guideline | How this specification obeys it                                     |
| -------------------- | --------- | ------------------------------------------------------------------- |
| `BRD-007`, `BRD-008` | Brand     | The view reads the tokens of `BRD-007` and the hue of each context. |
| `BRD-014`, `BRD-015` | Brand     | `src/i18n/en.ts` and `src/i18n/ka.ts` hold the copy, under one type. |
| `MCP-005`            | MCP tools | The copy of `SOURCES-FR-005` names the source and the time of each answer. |
| `BLD-001`            | Build     | `pnpm lint`, `pnpm test`, `pnpm build` in `website/`.                |
| `GIT-004`            | Git       | A commit of this feature has the scope `website`.                   |
| `LNG-006`, `LNG-007` | Language  | The test of `WEBSITE-SC-033` reads the new copy.                    |

`TST` does not govern `website/`. `WEBSITE-DD-012` gives the tests.

## 4. Design

### 4.1 Components

Every path is under `website/`.

| Part       | Path                                              | Change                                                   |
| ---------- | ------------------------------------------------- | -------------------------------------------------------- |
| Schema     | `src/lib/schemas.ts`                              | Add `kind` and `url` to a vendor. `SOURCES-DD-001`.       |
| Catalog    | `src/lib/catalog.ts`                              | Add the check of `SOURCES-DD-001`.                        |
| Sources    | `src/lib/sources.ts`                              | Create. `sourceGroups`. `SOURCES-DD-002`.                 |
| Content    | `src/content/contexts.yaml`                       | Add `kind` and `url` to each vendor.                      |
| Copy       | `src/i18n/en.ts`, `src/i18n/ka.ts`                | Add `sources`, `footer.sources`, `tool.sourcesLink`. 4.4.  |
| View       | `src/views/Sources.astro`                         | Create.                                                   |
| Pages      | `src/pages/sources.astro`, `src/pages/ka/sources.astro` | Create.                                             |
| Layout     | `src/layouts/Base.astro`, `src/views/Tool.astro`  | The links of `SOURCES-DD-003`.                            |
| Issue form | `.github/ISSUE_TEMPLATE/source_request.yml`, `.github/labels.yml` | Create the form. Add the label `source`. `SOURCES-DD-004`. |
| Tests      | `tests/unit/sources.test.ts`, `tests/unit/fixtures.ts`, `tests/unit/catalog.test.ts`, `tests/build/sources.test.ts`, `tests/browser/site.spec.ts` | Create the first and the fourth. Give each fixture vendor and each vendor written in `catalog.test.ts` a kind and a link. In the given data of `WEBSITE-SC-005`, the tool reads the three vendors of its context, so the check of `SOURCES-DD-001` holds. Add the page to the check of accessibility. |

### 4.2 Data model

```ts
// contexts.yaml: a vendor of a context gains two fields.
type Vendor = Localized & {
  code: string
  kind: "open-api" | "developer-api" | "unofficial-api" | "web-page" // SOURCES-FR-002
  url: string // https, a page for people or the documentation of the API. SOURCES-FR-003
}

// The result of sourceGroups.
type SourceGroup = {
  context: CatalogContext
  sources: Array<{ vendor: Vendor; tools: CatalogTool[] }> // in the order of contexts.yaml
}
```

| `kind`           | Official | English label              |
| ---------------- | -------- | -------------------------- |
| `open-api`       | Yes      | Official open API          |
| `developer-api`  | Yes      | Documented developer API   |
| `unofficial-api` | No       | Unofficial API             |
| `web-page`       | No       | Web page                   |

### 4.3 Declarations

**Pages.**

| English      | Georgian        | Source file                                          | Realizes                    |
| ------------ | --------------- | ---------------------------------------------------- | --------------------------- |
| `/sources/`  | `/ka/sources/`  | `src/pages/sources.astro`, `src/pages/ka/sources.astro` | `SOURCES-FR-001` to `FR-006`, `INV-002` |

The view `Sources.astro` holds, in order:

1. `<h1>` and the lead.
2. One `<section id="{context key}">` for each group, with the hue of the context, an `<h2>` with the name of the context, and one `<article id="{context key}-{code}">` for each source. An article holds an `<h3>` with the name of the vendor, the label of its kind, the sentence of its kind when the kind is not official, a link to `url` with the label "Visit the source", and the list of its tools, each a `<code>` linked to its tool page.
3. `<section id="how">`: the copy of `SOURCES-FR-005`.
4. `<section id="vendors">`: the copy of `SOURCES-FR-006`, then one link, `vendors.issue`, to `REPO_URL/issues/new?template=source_request.yml`. `SOURCES-DD-004`.

The sitemap integration of `WEBSITE` adds both addresses.

### 4.4 Copy

`sources` in each dictionary. The Georgian copy is a draft for the owner.

| Key                       | English                                                                                   |
| ------------------------- | ----------------------------------------------------------------------------------------- |
| `title`                   | Sources                                                                                   |
| `description`             | Where the data of Georgia MCP comes from, and whether each source is official.            |
| `lead`                    | Georgia MCP reads each answer from the organization that publishes the data. Each source below names its kind, a link to the vendor, and the tools that read it. |
| `kinds.<kind>`            | The labels of 4.2.                                                                         |
| `notes.unofficial-api`    | The vendor does not publish this API for reuse. It can change without notice.             |
| `notes.web-page`          | The server reads a page that the vendor writes for people. It can change without notice.  |
| `visit`                   | Visit the source                                                                           |
| `tools`                   | Tools                                                                                      |
| `how.heading`             | How the server reads a source                                                              |
| `how.paragraphs`          | The server calls a source only when a tool runs. It keeps an answer for a short time, from seconds to a few days, by the pace at which the source changes. It sends the vendor only the input of the call, never data about the person who asks. Each answer names its source and the time of its data. |
| `vendors.heading`         | For a vendor                                                                               |
| `vendors.text`            | To correct or remove your source, open a source request in the repository. A request is public. |
| `vendors.issue`           | Open a source request                                                                      |
| `footer.sources`          | Sources                                                                                    |
| `tool.sourcesLink`        | About these sources                                                                        |

## 5. Design decisions

### `SOURCES-DD-001`

**Realizes:** `SOURCES-FR-002`, `FR-003`, `INV-001`
**Decision:** The schema of a vendor in `src/lib/schemas.ts` adds `kind`, an enum of the four values of 4.2, and `url`, a URL that starts with `https://` and whose path holds no `/api`. A vendor with no `kind` or no `url` fails the build. `buildCatalog` adds one check: each vendor of a context is read by at least one tool of that context, else the problem `<context>.<code>: no tool reads the vendor`. The sentence of a kind that is not official comes from the dictionary, one sentence for each kind.
**Rationale:** The schema makes `SOURCES-INV-001` a build failure, not a review. One sentence for each kind keeps the Georgian copy short and the facts the same for each source of a kind. The new check keeps each source with at least one tool, so `SOURCES-FR-004` always has a tool to name.
**Alternatives:** A separate `sources.yaml`. The owner chose `contexts.yaml`. A sentence for each source. It doubles the copy to translate for no new fact.

### `SOURCES-DD-002`

**Realizes:** `SOURCES-FR-001`, `FR-004`
**Decision:** `src/lib/sources.ts` exports one pure function:

```ts
/** The sources of each context, with the tools that read each one, in the order of the home page. */
export function sourceGroups(catalog: Catalog): SourceGroup[]
```

A group follows `catalog.contexts`. A source lists the tools of its context whose `vendors` hold its code, in the order of `context.tools`. A vendor in two contexts gives two sources.
**Rationale:** The function takes the fixture catalog of the unit tests, and the view only renders.
**Alternatives:** The join inside the view. A unit test cannot reach it.

### `SOURCES-DD-003`

**Realizes:** `SOURCES-FR-007`, `FR-008`
**Decision:** The footer of `Base.astro` adds `<a href={localize("/sources/", lang)}>{copy.footer.sources}</a>` before the link of the privacy policy. The vendor section of `Tool.astro` adds, after its list, `<a class="sources-link" href={`${localize("/sources/", lang)}#${context.key}`}>{copy.tool.sourcesLink}</a>`.
**Rationale:** A link in words after the list reads well with a screen reader. The anchor of the group lands on the context of the tool.
**Alternatives:** A link on each vendor name. Each would open the same group. A link on the heading "Sources". A heading that is a link reads as navigation.

### `SOURCES-DD-004`

**Realizes:** `SOURCES-FR-006`
**Decision:** `.github/ISSUE_TEMPLATE/source_request.yml` is an issue form named "Source request", with the label `source` and the title prefix `[Source] `. It holds:

| Field        | Type     | Required | Holds                                                                       |
| ------------ | -------- | -------- | --------------------------------------------------------------------------- |
| `source`     | input    | Yes      | The vendor and the context, or the link of the source on the sources page.  |
| `request`    | dropdown | Yes      | Correct a fact on the sources page, Correct the data of a tool, Remove the source, Other. |
| `requester`  | dropdown | Yes      | I work for the vendor, I do not work for the vendor.                         |
| `details`    | textarea | Yes      | What is wrong or what to change, with a link that shows it.                  |
| `checks`     | checkboxes | Yes    | The request holds no key, token, or personal data.                           |

A markdown block at the top says that the issue is public. `.github/labels.yml` adds `source`, with the color `fbca04` and the description "A request about a source of the server.". The page links to `REPO_URL/issues/new?template=source_request.yml`, so the reader lands on the form.
**Rationale:** The owner wants one channel, in the open, with a type of its own. A form asks for the facts that the owner needs to act, and the label sorts the requests. The workflow of the labels creates the label from the file.
**Alternatives:** The email of the owner. The owner refused it. The bug report form. It asks for a client and a tool call, which a vendor does not have.

## 6. Scenarios

### `SOURCES-SC-001`

**Verifies:** `SOURCES-FR-001`, `SOURCES-FR-004`
**Layer:** unit

**Given** a fixture catalog with two contexts, one vendor code in both, and two tools in one context that read different vendors.
**When** the test calls `sourceGroups`.
**Then** the groups come in the order of the contexts. The shared vendor gives one source in each group. Each source lists only the tools whose vendors hold its code.

### `SOURCES-SC-002`

**Verifies:** `SOURCES-INV-001`
**Layer:** unit

**Given** a context with a vendor that no tool reads, and a vendor with the `url` `https://example.ge/api/rates`.
**When** the test calls `buildCatalog`, and parses the vendor with the schema.
**Then** `buildCatalog` throws a `CatalogError` with `fuel.vendor1: no tool reads the vendor`. The schema refuses the vendor.

### `SOURCES-SC-003`

**Verifies:** `SOURCES-FR-001`, `SOURCES-INV-001`
**Layer:** build

**Given** both sources pages.
**When** the test reads the groups and the articles.
**Then** the groups follow the order of `contexts.yaml`. The ids of the articles of each group equal the codes of the vendors of that context in `contexts.yaml`. The lead holds no digit.

### `SOURCES-SC-004`

**Verifies:** `SOURCES-FR-002`, `SOURCES-FR-003`
**Layer:** build

**Given** each article of both sources pages.
**When** the test reads its kind, its sentence, and its link.
**Then** the kind is one of the four labels of the language. An article of a kind that is not official holds the sentence of its kind. The link starts with `https://`, its host is not the host of `SITE_URL`, and its path holds no `/api`.

### `SOURCES-SC-005`

**Verifies:** `SOURCES-FR-004`
**Layer:** build

**Given** each article of both sources pages.
**When** the test reads its tool links.
**Then** the names equal the tools whose tool file lists the code in that context, and each link is the tool page in the language of the page.

### `SOURCES-SC-006`

**Verifies:** `SOURCES-FR-005`, `SOURCES-FR-006`
**Layer:** build

**Given** both sources pages.
**When** the test reads `#how` and `#vendors`.
**Then** `#how` holds the paragraph of the language. `#vendors` holds one link, to `REPO_URL/issues/new?template=source_request.yml`, and no `mailto:` link. `.github/ISSUE_TEMPLATE/source_request.yml` exists, with the label `source`, and `.github/labels.yml` holds the label `source`.

### `SOURCES-SC-007`

**Verifies:** `SOURCES-FR-007`, `SOURCES-FR-008`
**Layer:** build

**Given** each built page and `/404.html`, and each tool page.
**When** the test reads the footer and the vendor section.
**Then** the footer links to `/sources/` on an English page and to `/ka/sources/` on a Georgian page. The vendor section of a tool page links to the sources page of its language with the anchor of its context.

### `SOURCES-SC-008`

**Verifies:** `WEBSITE-NFR-005` for the sources page
**Layer:** browser

**Given** `/sources/` and `/ka/sources/`, in each theme.
**When** axe checks each page, in the test of `WEBSITE-SC-031`.
**Then** it finds no violation of WCAG 2.2 AA.

## 7. Build plan

| #   | Step                                                                                       | Realizes                       | Done |
| --- | ------------------------------------------------------------------------------------------ | ------------------------------ | ---- |
| 1   | Write the unit tests. Give each fixture vendor a kind and a link. Add `kind` and `url` to the schema, the check to `buildCatalog`, and write `src/lib/sources.ts` until they pass. | `SOURCES-DD-001`, `DD-002`, `SC-001`, `SC-002` | [x] |
| 2   | Fill `kind` and `url` of each vendor of `contexts.yaml` from section 3 of its feature.      | `SOURCES-FR-002`, `FR-003`     | [x] |
| 3   | Write the build tests. Write the copy of 4.4, `Sources.astro`, and both pages.              | `SOURCES-SC-003` to `SC-006`   | [x] |
| 4   | Write the build test of the links. Add the footer link and the link of the tool page. Add both pages to the check of accessibility. | `SOURCES-DD-003`, `SC-007`, `SC-008` | [x] |

## 8. Out of scope for this specification

- A check that a link of a vendor answers. It needs the network.
