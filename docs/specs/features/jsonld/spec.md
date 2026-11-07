---
id: JSONLD
title: The structured data
type: spec
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [LNG, BLD]
requirements: features/jsonld/requirements.md
---

# Specification: The structured data

## 1. Summary

The view of the home page in `website/` writes one block of JSON-LD into the head of `/`
and `/ka/`. The block is a graph of four nodes: the site, the application, the source code,
and the author. A pure function builds it from the constants, the copy, and the site version.

## 2. Coverage

| Requirement      | Where this specification realizes it                 |
| ---------------- | ---------------------------------------------------- |
| `JSONLD-FR-001`  | `JSONLD-DD-001`, `JSONLD-SC-001`, `JSONLD-SC-003`    |
| `JSONLD-FR-002`  | `JSONLD-DD-001`, `JSONLD-DD-002`, `JSONLD-SC-001`, `SC-002`, `SC-004` |
| `JSONLD-FR-003`  | `JSONLD-DD-001`, `JSONLD-SC-001`                     |
| `JSONLD-FR-004`  | `JSONLD-DD-001`, `JSONLD-SC-001`                     |
| `JSONLD-NFR-001` | `JSONLD-DD-003`, `JSONLD-SC-003`                     |
| `JSONLD-INV-001` | `JSONLD-DD-001`, `JSONLD-SC-004`                     |

`JSONLD-NFR-001` also names the validator of schema.org. The owner runs it by hand after the first deploy. No test calls it.

## 3. Guideline compliance

| Rule                 | Guideline | How this specification obeys it                                   |
| -------------------- | --------- | ----------------------------------------------------------------- |
| `BLD-001`            | Build     | `pnpm lint`, `pnpm test`, `pnpm build` in `website/`.              |
| `GIT-004`            | Git       | A commit of this feature has the scope `website`.                 |
| `LNG-006`, `LNG-007` | Language  | The description is the page description, which `WEBSITE-SC-033` reads. |
| `WEBSITE-NFR-002`    | Website   | The block is about 1.5 KB, 0.6 KB compressed. The check of `WEBSITE-SC-029` counts it in the 60 KB of the home page. `JSONLD-DD-003`. |

`TST` does not govern `website/`. `WEBSITE-DD-012` gives the tests.

## 4. Design

### 4.1 Components

Every path is under `website/`.

| Part            | Path                                   | Change                                               |
| --------------- | -------------------------------------- | ---------------------------------------------------- |
| Structured data | `src/lib/structured-data.ts`           | Create. `structuredData`. `JSONLD-DD-001`.            |
| Release         | `src/lib/release.ts`                   | Add `releaseVersion`. `JSONLD-DD-002`.                |
| Constants       | `src/site.ts`                          | Add `AUTHOR_NAME`, `AUTHOR_GITHUB_URL`, `LICENSE_URL`. |
| Layout          | `src/layouts/Base.astro`               | Add the prop `structuredData`. `JSONLD-DD-003`.       |
| View            | `src/views/Home.astro`                 | Pass `structuredData` to `Base`.                      |
| Tests           | `tests/unit/structured-data.test.ts`, `tests/build/structured-data.test.ts` | Create. |

```ts
export const AUTHOR_NAME = "Temuri Takalandze"
export const AUTHOR_GITHUB_URL = "https://github.com/ABGEO"
export const LICENSE_URL = "https://www.gnu.org/licenses/gpl-3.0.html"
```

### 4.2 Domain model

The feature has no domain of the server. It reads `Lang` of `WEBSITE` 4.2, the constants of `src/site.ts`, and the site version of `WEBSITE-DD-025`.

### 4.3 Declarations

| Page  | Head                                                     | Realizes                    |
| ----- | -------------------------------------------------------- | --------------------------- |
| `/`   | One `<script type="application/ld+json">`, `structuredData("en", PUBLIC_VERSION, copy.meta.description)` | `JSONLD-FR-001` to `FR-004` |
| `/ka/` | One `<script type="application/ld+json">`, `structuredData("ka", PUBLIC_VERSION, copy.meta.description)` | `JSONLD-FR-001` to `FR-004` |

No other page writes the block.

### 4.4 Flow

The build renders each home page. `Home.astro` calls `structuredData`, and `Base.astro` writes the result into the head.

### 4.5 Errors

None. The inputs are constants and the copy, and `astro check` types them.

## 5. Design decisions

### `JSONLD-DD-001`

**Realizes:** `JSONLD-FR-001` to `FR-004`, `INV-001`
**Decision:** `src/lib/structured-data.ts` exports one pure function:

```ts
/** The JSON-LD graph of a home page. */
export function structuredData(lang: Lang, version: string, description: string): StructuredData

type StructuredData = { "@context": "https://schema.org"; "@graph": Array<Record<string, unknown>> }
```

The graph holds four nodes. Each node has an `@id` under `SITE_URL`, and a node names another by `{ "@id": ... }`, so the author is one entity. `{home}` is `SITE_URL/` for `en` and `SITE_URL/ka/` for `ka`.

| `@id`               | `@type`              | Fields                                                                                  |
| ------------------- | -------------------- | --------------------------------------------------------------------------------------- |
| `SITE_URL/#author`  | `Person`             | `name` `AUTHOR_NAME`, `url` `AUTHOR_URL`, `sameAs` `[AUTHOR_GITHUB_URL]`                |
| `{home}#website`    | `WebSite`            | `name` `SITE_NAME`, `url` `{home}`, `inLanguage` `lang`, `publisher` the author          |
| `SITE_URL/#app`     | `SoftwareApplication` | `name` `SITE_NAME`, `description`, `inLanguage` `lang`, `applicationCategory` `DeveloperApplication`, `url` `{home}`, `license` `LICENSE_URL`, `offers` `{ "@type": "Offer", "price": "0", "priceCurrency": "GEL" }`, `author` the author, `softwareVersion` when `releaseVersion(version)` gives one |
| `SITE_URL/#source`  | `SoftwareSourceCode` | `codeRepository` `REPO_URL`, `programmingLanguage` `Go`, `license` `LICENSE_URL`, `author` the author, `targetProduct` the application |

The description is the page description of `WEBSITE-FR-022`, so the block and the page agree. It names no subject.
**Rationale:** One graph with `@id` references states the author once. The page description already passes the copy test. `DeveloperApplication` is the category of schema.org that a search engine reads for a tool of developers. The licence is GPL 3.0, from `LICENSE`.
**Alternatives:** Four separate blocks. Each would repeat the author. `operatingSystem`. The server runs as a remote address, a binary, and an image, and no single value is true. A licence given as an SPDX code. schema.org asks for a URL or a creative work.

### `JSONLD-DD-002`

**Realizes:** `JSONLD-FR-002`
**Decision:** `src/lib/release.ts` adds `releaseVersion`, with the same pattern as `releaseUrl`:

```ts
/** The version of a release without its "v", or undefined when the site version names no release. */
export function releaseVersion(version: string): string | undefined
```

`releaseUrl` uses it, so one pattern decides both.
**Rationale:** The footer and the block decide "a release" the same way.
**Alternatives:** A second pattern in `structured-data.ts`. Two copies can drift.

### `JSONLD-DD-003`

**Realizes:** `JSONLD-NFR-001`
**Decision:** `Base.astro` takes an optional prop `structuredData`. When it is set, it writes `<script type="application/ld+json" set:html={json}>`, where `json` is `JSON.stringify(structuredData)` with each `<` replaced by `<`. The block stays in the count of `WEBSITE-SC-029`.
**Rationale:** The replacement keeps a `</script>` in a text from closing the tag, and JSON reads `<` as `<`. The block is inline, so the page sends no extra request. It adds about 0.6 KB compressed to a budget of 60 KB.
**Alternatives:** A separate file. A search engine reads JSON-LD only from the page. Leaving the block out of the count of `WEBSITE-SC-029`. It changes a test of `WEBSITE` for 0.6 KB.

## 6. Scenarios

### `JSONLD-SC-001`

**Verifies:** `JSONLD-FR-001` to `FR-004`
**Layer:** unit

**Given** the language `en`, the site version `v1.2.3`, and a description.
**When** the test calls `structuredData`.
**Then** the graph holds one node of each of `Person`, `WebSite`, `SoftwareApplication`, and `SoftwareSourceCode`. The application has the description, `softwareVersion` `1.2.3`, `license` `LICENSE_URL`, an offer of price `0` in `GEL`, and the author by `@id`. The source has `codeRepository` `REPO_URL`, `programmingLanguage` `Go`, and the application as `targetProduct`. The person has `AUTHOR_NAME`, `AUTHOR_URL`, and `AUTHOR_GITHUB_URL`. Each `@id` reference names a node of the graph. With `ka`, the site has `url` `SITE_URL/ka/` and `inLanguage` `ka`.

### `JSONLD-SC-002`

**Verifies:** `JSONLD-FR-002`
**Layer:** unit

**Given** the site versions `v1.1.2-8-g8def960` and `dev`.
**When** the test calls `structuredData` and `releaseVersion` for each.
**Then** the application has no `softwareVersion`, and `releaseVersion` gives `undefined`. `releaseVersion("v1.2.3")` gives `1.2.3`.

### `JSONLD-SC-003`

**Verifies:** `JSONLD-FR-001`, `JSONLD-NFR-001`
**Layer:** build

**Given** each built page.
**When** the test reads the blocks of the type `application/ld+json`.
**Then** each home page holds one block that parses as JSON, with a `WebSite` whose `url` is the canonical address of the page and whose `inLanguage` is the language of the page. No other page holds a block.

### `JSONLD-SC-004`

**Verifies:** `JSONLD-INV-001`, `JSONLD-FR-002`
**Layer:** build

**Given** both home pages, built with no `PUBLIC_VERSION`.
**When** the test compares the block with the page.
**Then** the name of the application is `SITE_NAME`, and the title of the page holds it. The description equals the `<meta name="description">` of the page and holds no name of a context in the language of the page. The application has no `softwareVersion`, and the footer shows `dev`.

## 7. Build plan

| #   | Step                                                                                       | Realizes                       | Done |
| --- | ------------------------------------------------------------------------------------------ | ------------------------------ | ---- |
| 1   | Write the unit tests. Add the constants and `releaseVersion`. Write `src/lib/structured-data.ts` until they pass. | `JSONLD-DD-001`, `DD-002`, `SC-001`, `SC-002` | [x] |
| 2   | Write the build tests. Add the prop to `Base.astro` and pass it from `Home.astro`.         | `JSONLD-DD-003`, `SC-003`, `SC-004` | [x] |

## 8. Out of scope for this specification

- A call to the validator of schema.org from a test. It needs the network.
