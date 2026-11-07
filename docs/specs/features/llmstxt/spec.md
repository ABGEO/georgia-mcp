---
id: LLMSTXT
title: The catalog files
type: spec
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [MCP, LNG, BLD]
requirements: features/llmstxt/requirements.md
---

# Specification: The catalog files

## 1. Summary

Three static endpoints of the site in `website/` write `/llms.txt`, `/tools.json`, and one
Markdown copy for each tool, from the catalog of `WEBSITE-DD-004`. Each page links to the
catalog files in its head, and the robots file names them.

## 2. Coverage

| Requirement       | Where this specification realizes it                      |
| ----------------- | --------------------------------------------------------- |
| `LLMSTXT-FR-001`  | 4.3 Endpoints, `LLMSTXT-DD-001`, `LLMSTXT-SC-001`, `SC-002` |
| `LLMSTXT-FR-002`  | 4.3 Endpoints, `LLMSTXT-DD-001`, `LLMSTXT-SC-003`, `SC-004` |
| `LLMSTXT-FR-003`  | 4.3 Endpoints, `LLMSTXT-DD-001`, `LLMSTXT-SC-005`         |
| `LLMSTXT-FR-004`  | `LLMSTXT-DD-002`, `LLMSTXT-SC-006`                        |
| `LLMSTXT-INV-001` | `LLMSTXT-DD-001`, `LLMSTXT-SC-002`, `SC-004`, `SC-005`    |

`LLMSTXT-FR-003` has the limit case of a tool with no output schema. `MCP` gives every tool an output schema, and `definitionSchema` refuses a definition without one, so the build fails before it writes `/tools.json`. `LLMSTXT-SC-005` checks the normal case only.

## 3. Guideline compliance

| Rule                 | Guideline | How this specification obeys it                                        |
| -------------------- | --------- | ---------------------------------------------------------------------- |
| `MCP-005`            | MCP tools | A Markdown copy shows the recorded output, which `WEBSITE-INV-003` keeps with `source` and `as_of`. |
| `BLD-001`            | Build     | `pnpm lint`, `pnpm test`, `pnpm build` in `website/`.                   |
| `GIT-004`            | Git       | A commit of this feature has the scope `website`.                      |
| `LNG-006`, `LNG-007` | Language  | The test of `WEBSITE-SC-033` reads the copy. `SITE_SUMMARY` holds no em dash. |

`TST` does not govern `website/`. `WEBSITE-DD-012` gives the tests.

## 4. Design

### 4.1 Components

Every path is under `website/`.

| Part          | Path                                  | Change                                                  |
| ------------- | ------------------------------------- | ------------------------------------------------------- |
| Catalog files | `src/lib/catalog-files.ts`            | Create. `llmsText`, `toolMarkdown`, `toolsJson`. `LLMSTXT-DD-001`. |
| Endpoints     | `src/pages/llms.txt.ts`, `src/pages/tools.json.ts`, `src/pages/tools/[name].md.ts` | Create. 4.3. |
| Constants     | `src/site.ts`                         | Add `SITE_SUMMARY`.                                     |
| Layout        | `src/layouts/Base.astro`, `src/views/Tool.astro` | Add the links of `LLMSTXT-DD-002`.           |
| Static        | `public/robots.txt`                   | Add the comments of `LLMSTXT-DD-002`.                   |
| Image         | `.docker/nginx.conf`                  | Add the types of `LLMSTXT-DD-002`.                      |
| Tests         | `tests/unit/catalog-files.test.ts`, `tests/build/catalog-files.test.ts` | Create.          |

```ts
export const SITE_SUMMARY = "An MCP server for Georgian public data, read from the organizations that publish it."
```

### 4.2 Domain model

The feature reads `Catalog`, `CatalogTool`, and `CatalogContext` of `WEBSITE` 4.2, and adds
`ToolsJson` of `LLMSTXT-DD-001`. It has no domain of the server.

### 4.3 Declarations

**Endpoints.** Each is a static endpoint of Astro. The build writes it to `dist/`. Each address is in English only.

| Address              | Source file                             | Writes                                         | Realizes                    |
| -------------------- | --------------------------------------- | ---------------------------------------------- | --------------------------- |
| `/llms.txt`          | `src/pages/llms.txt.ts`                 | `llmsText(catalog)`                            | `LLMSTXT-FR-001`, `INV-001` |
| `/tools/<name>.md`   | `src/pages/tools/[name].md.ts`          | `toolMarkdown(tool, context)`, one file for each path of `toolPaths` | `LLMSTXT-FR-002`, `INV-001` |
| `/tools.json`        | `src/pages/tools.json.ts`               | `JSON.stringify(toolsJson(catalog), null, 2)`  | `LLMSTXT-FR-003`, `INV-001` |

### 4.4 Flow

`pnpm build` runs `getCatalog`, then each endpoint. A `CatalogError` stops the build before any file.

### 4.5 Errors

| Condition                         | Where         | Message                                      |
| --------------------------------- | ------------- | -------------------------------------------- |
| A check of `WEBSITE-DD-004` fails | `pnpm build`  | The `CatalogError` of `WEBSITE` 4.6.         |

## 5. Design decisions

### `LLMSTXT-DD-001`

**Realizes:** `LLMSTXT-FR-001`, `FR-002`, `FR-003`, `INV-001`
**Decision:** `src/lib/catalog-files.ts` exports three pure functions. Each reads the `Catalog` of `WEBSITE-DD-004`, so the three files hold the same set of tools as the tool pages. Each address is absolute, from `SITE_URL`. The text is English. Each description is joined to one line.

```ts
/** The text of /llms.txt. */
export function llmsText(catalog: Catalog): string
/** The Markdown copy of one English tool page. */
export function toolMarkdown(tool: CatalogTool, context: CatalogContext): string
/** The object of /tools.json. */
export function toolsJson(catalog: Catalog): ToolsJson

type ToolsJson = {
  name: "georgia-mcp"
  title: string       // SITE_NAME
  url: string         // SERVER_URL
  tools: Array<Definition & { page: string; markdown: string }>
}
```

`/llms.txt` follows the format of llmstxt.org:

```md
# Georgia MCP

> {SITE_SUMMARY}

Connect a client to {SERVER_URL}, over streamable HTTP, with no sign-in.

## Install

- [Install guides]({SITE_URL}/#install): the steps for each client, and for a local server.
- [Repository]({REPO_URL}): the source, the releases, and the container image.

## Tools

- [{title}]({SITE_URL}/tools/{name}.md): `{name}`. {description}
```

The tools come in the order of `catalog.tools`. The Markdown copy of a tool:

~~~md
# {name}

> {summary.en}

- Title: {title}
- Context: {context.name.en}
- Sources: {vendor.en}, ...
- Page: {SITE_URL}/tools/{name}/
- Server: {SERVER_URL}

## Description

{description}

## Input

- `{name}` ({type}, required|optional, default `{default}`): {description}

## Examples

### {question.en}

Input:

```json
{input}
```

Output, recorded at {recordedAt}:

```json
{output}
```
~~~

The input list comes from `inputRows` of `WEBSITE-DD-005`. A row with no default has no default part. A tool with no input says "The tool takes no input." The output is the recording of the example, so it holds `source` and `as_of` (`WEBSITE-INV-003`).

An entry of `/tools.json` holds the fields of `Definition`, picked by name, and no field of the site, plus `page` and `markdown`, the two absolute addresses.
**Rationale:** Pure functions take the fixture catalog of the unit tests. The catalog already checks the set of tools, so the files cannot drift from the pages. `SITE_SUMMARY` is the description of the plugins and the registry, and it names no subject. A list for the input avoids a table, whose cells would need the `|` and the line breaks of a description escaped.
**Alternatives:** A script after `astro build` that writes the files. It reads the data a second time, outside the checks of the catalog. A table for the input. A description with `|` breaks it. One section for each context in `/llms.txt`. The headings would list the subjects.

### `LLMSTXT-DD-002`

**Realizes:** `LLMSTXT-FR-004`
**Decision:** `Base.astro` writes two links into the head of each page, `/404.html` included:

```html
<link rel="alternate" type="text/markdown" title="llms.txt" href="/llms.txt" />
<link rel="alternate" type="application/json" title="Tools" href="/tools.json" />
```

`Tool.astro` passes `markdown` to `Base.astro` when `lang` is `en`, and `Base.astro` writes `<link rel="alternate" type="text/markdown" href="/tools/<name>.md" />`. A Georgian tool page writes no such link. `public/robots.txt` ends with two comment lines:

```
# llms.txt: https://geo-mcp.abgeo.cloud/llms.txt
# Tools: https://geo-mcp.abgeo.cloud/tools.json
```

`.docker/nginx.conf` serves the files:

- `charset utf-8` with `charset_types text/plain text/markdown application/json`, because a description holds Georgian text and `µg/m³`.
- `location ~ \.md$` with an empty `types` and `default_type text/markdown`, because the default types of nginx do not know `.md`.
- `gzip_types` adds `application/json` and `text/markdown`. `/tools.json` is about 140 KB, and about 15 KB compressed.
**Rationale:** `rel="alternate"` with a type is how a page names another form of itself. The robots file has no field for these files, so a comment names them for a person who reads it. The location adds no header, so it keeps the headers of the server block.
**Alternatives:** A field `Llms-txt:` in the robots file. No crawler reads it, and a strict parser could reject it. A link to the Markdown copy from a Georgian tool page. The copy is English, and `LLMSTXT-FR-004` names the English pages.

## 6. Scenarios

### `LLMSTXT-SC-001`

**Verifies:** `LLMSTXT-FR-001`
**Layer:** unit

**Given** the fixture catalog with `fuel_get_prices`, whose description holds a line break.
**When** the test calls `llmsText`.
**Then** the text starts with `# Georgia MCP`, then `> ` and `SITE_SUMMARY`. It holds `SERVER_URL`, a link to `SITE_URL/#install`, and a link to `REPO_URL`. It holds one line `- [<title>](SITE_URL/tools/fuel_get_prices.md): ` followed by the description on one line.

### `LLMSTXT-SC-002`

**Verifies:** `LLMSTXT-FR-001`, `LLMSTXT-INV-001`
**Layer:** build

**Given** a build of the site.
**When** the test reads `dist/llms.txt`.
**Then** the names of its tool lines equal the keys of `definitions.json`. Its summary line holds no English name of a context of `contexts.yaml`, and no digit. `dist/ka/llms.txt` does not exist.

### `LLMSTXT-SC-003`

**Verifies:** `LLMSTXT-FR-002`
**Layer:** unit

**Given** the fixture tool with one input that has a default, and a fixture tool with no input.
**When** the test calls `toolMarkdown` for each.
**Then** the first copy starts with `# <name>`, holds the address of the tool page and `SERVER_URL`, lists the input with its type and its default, and holds the output of the example with `source` and `as_of`. The second copy holds "The tool takes no input."

### `LLMSTXT-SC-004`

**Verifies:** `LLMSTXT-FR-002`, `LLMSTXT-INV-001`
**Layer:** build

**Given** a build of the site.
**When** the test lists `dist/tools/*.md` and `dist/ka/tools/`.
**Then** the names equal the keys of `definitions.json`. Each copy holds the description of its definition. No `.md` file exists under `dist/ka/`.

### `LLMSTXT-SC-005`

**Verifies:** `LLMSTXT-FR-003`, `LLMSTXT-INV-001`
**Layer:** build

**Given** a build of the site.
**When** the test parses `dist/tools.json`.
**Then** `name` is `georgia-mcp` and `url` is `SERVER_URL`. The names of `tools` equal the keys of `definitions.json`. Each entry, without `page` and `markdown`, deep equals its definition. `page` is `SITE_URL/tools/<name>/` and `markdown` is `SITE_URL/tools/<name>.md`.

### `LLMSTXT-SC-006`

**Verifies:** `LLMSTXT-FR-004`
**Layer:** build

**Given** each built page and `/404.html`.
**When** the test reads the head.
**Then** it links to `/llms.txt` and `/tools.json`. Each English tool page also links to `/tools/<name>.md`. No Georgian page links to a `.md` file. `dist/robots.txt` names `SITE_URL/llms.txt` and `SITE_URL/tools.json`.

## 7. Build plan

| #   | Step                                                                                       | Realizes                       | Done |
| --- | ------------------------------------------------------------------------------------------ | ------------------------------ | ---- |
| 1   | Write the unit tests of the catalog files. Add `SITE_SUMMARY`. Write `src/lib/catalog-files.ts` until they pass. | `LLMSTXT-DD-001`, `SC-001`, `SC-003` | [x] |
| 2   | Write the build tests of the catalog files. Write the three endpoints.                     | `LLMSTXT-DD-001`, `SC-002`, `SC-004`, `SC-005` | [x] |
| 3   | Write the build test of the links. Add the links to `Base.astro` and `Tool.astro`, the comments to `robots.txt`, and the types to `nginx.conf`. | `LLMSTXT-DD-002`, `SC-006` | [x] |

## 8. Out of scope for this specification

- A test of `nginx.conf`. The build tests read `dist/`, not the image.
