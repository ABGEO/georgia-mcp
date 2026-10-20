---
id: WEBSITE
title: The website
type: requirements
status: approved
created: 2026-10-20
updated: 2026-10-20
context: website
approved_by: Temuri
approved_on: 2026-10-20
constrained_by: [BRD, MCP, LNG]
---

# Requirements: The website

## 1. Problem

A person who hears of Georgia MCP has no page to read. They cannot see what the
tools answer, which vendors feed them, or how to connect a client. A Georgian
reader has no text in Georgian. A link to the project shows no preview.

## 2. Users

| User              | Need                                                                  |
| ----------------- | --------------------------------------------------------------------- |
| A person who chats | See what they can ask, and connect their client in a few steps.       |
| A developer       | Read the name, the input, and a real output of each tool.             |
| A Georgian reader | Read the site in Georgian.                                            |

## 3. Source

| Vendor        | Source                                                    | Official | Publication period         | Terms of use |
| ------------- | --------------------------------------------------------- | -------- | -------------------------- | ------------ |
| Georgia MCP   | The tool list of the server: name, title, description, input schema | Yes. The server is ours. | At each build of the site. | None. |
| Each vendor   | A tool example: one output of the server, recorded and kept in the repository | As the feature of the tool states. | When the owner refreshes the examples. | As the feature of the tool states. Most vendors publish no terms. |

## 4. Out of scope

- A tool runner in the browser. A visitor cannot call a tool from the site.
- A search over the site.
- A blog, a changelog, or documentation beyond the tools.
- An analytics script, a tracking cookie, or a cookie banner.
- A language other than English and Georgian.
- A redirect by the language of the browser.

## 5. Definitions

The terms are in the glossary, in the section "The website".

## 6. Functional requirements

### `WEBSITE-FR-001`

The site must serve each page in English at a path with no language prefix, and in Georgian at the same path under `/ka/`.

**Examples:**

- Normal case: `/` and `/ka/`. `/tools/fuel_get_prices/` and `/ka/tools/fuel_get_prices/`.
- Unwanted case: `/en/` answers with the page of `WEBSITE-FR-021`.

### `WEBSITE-FR-002`

Each page must link to the same page in the other language.

**Why:** A reader who switches the language must stay on the same subject.

### `WEBSITE-FR-003`

The site must not choose the language from the browser or the location of the visitor.

**Why:** A shared link opens the same page for every reader.

### `WEBSITE-FR-004`

The home page must hold these sections, in this order: the introduction, the
contexts, the tool examples, the install guides, the tools, and the author.

### `WEBSITE-FR-005`

The home page must show each context with its name, its count of tools, and the name of each vendor that it reads.

**Examples:**

- Normal case: "Fuel prices, 1 tool, Wissol, Rompetrol, SOCAR".
- Limit case: a context with one vendor shows one vendor name.

### `WEBSITE-FR-006`

The home page must show one tool example for each context at least.

### `WEBSITE-FR-007`

The site must give an install guide for each of these clients: Claude on the web and on the desktop, Claude Code, ChatGPT, Cursor, and Visual Studio Code.

**Why:** These are the clients that the owner and the readers use.

### `WEBSITE-FR-008`

Each install guide must show the address of the server, and a control that copies it.

**Examples:**

- Normal case: the guide shows `https://geo-mcp.abgeo.cloud/mcp`. The reader presses the control. The clipboard holds the address.
- Unwanted case: the browser refuses the clipboard. The address stays selected, so the reader can copy it by hand.

### `WEBSITE-FR-009`

The site must give an install guide for a server on the computer of the reader: one with the binary, and one with the container image.

**Why:** A reader who does not want a public server runs their own.

**Examples:**

- Normal case: the binary guide downloads `georgia-mcp` from the releases of the repository, and connects Claude Code to `georgia-mcp stdio`.
- Normal case: the container guide runs `docker run -p 8080:8080 ghcr.io/abgeo/georgia-mcp`, and gives the address `http://localhost:8080/mcp`.
- Limit case: the reader has Go. The binary guide also gives `go install github.com/abgeo/georgia-mcp/cmd/georgia-mcp@latest`.
- Unwanted case: the reader wants the commercial rates of TBC Bank. The guide names the setting of the key, and says that the other tools work without it.

### `WEBSITE-FR-010`

The site must serve no page at `/mcp` or under it.

**Why:** The site and the server share the host `geo-mcp.abgeo.cloud`. The server answers at `/mcp`.

### `WEBSITE-FR-011`

The site must list every tool that the server registers, grouped by context, with its name and its tool summary.

### `WEBSITE-FR-012`

The site must have one tool page for each tool, at `/tools/<tool name>/` and `/ka/tools/<tool name>/`.

### `WEBSITE-FR-013`

A tool page must show the tool definition. For each input, it shows the name, the type, the description, whether it is required, and its default.

**Examples:**

- Normal case: `rates_get_official_rates` shows its inputs `currencies` and `date`.
- Limit case: a tool with no input shows the sentence "This tool takes no input."

### `WEBSITE-FR-014`

A tool page must show each tool example of the tool, with the question, the input, the output, and the date of the record.

**Why:** A reader must see that the output is a record, and how old it is.

### `WEBSITE-FR-015`

A Georgian tool page must show the tool summary and the question of each tool example in Georgian. It shows the tool definition and the output as the server returns them.

**Why:** The model reads the definition in English. A translation would show the reader a text that the model never sees.

### `WEBSITE-FR-017`

The build must fail when one of these is true:

- The server registers a tool that has no tool example.
- The server registers a tool that has no tool summary in each language.
- A tool example names a tool that the server does not register.
- The output of a tool example does not match the output schema of its tool.

**Examples:**

- Normal case: every tool has an example and two summaries. The build passes.
- Unwanted case: a new tool `transit_list_vehicles` has no example. The build fails and names the tool.

### `WEBSITE-FR-018`

The owner must refresh every tool example with one command.

**Why:** A recorded output grows old. One command keeps the cost of a refresh low.

### `WEBSITE-FR-019`

Each page must end with the byline of `BRD-011` and a link to the repository on GitHub.

### `WEBSITE-FR-020`

The site must keep the theme that the reader selects from page to page in the same browser.

**Examples:**

- Normal case: the reader selects dark on the home page. Each tool page opens dark.
- Limit case: the browser keeps no data. Each page follows the theme of the system.

### `WEBSITE-FR-021`

The site must answer an unknown address with a page in English and in Georgian that links to both home pages.

### `WEBSITE-FR-022`

Each page must carry a title, a description, and a preview image in its language, for a search engine and for a link preview.

**Examples:**

- Normal case: a link to `/ka/tools/fuel_get_prices/` in a chat shows the Georgian title and the image.

### `WEBSITE-FR-023`

The build of the site must read each tool definition from a file in the repository. The command of `WEBSITE-FR-018` writes the file from the server.

**Why:** The build of the site needs no server. One command refreshes the definitions and the tool examples together.

**Examples:**

- Normal case: the owner adds a tool to the server and runs the command. The file holds the new tool, and the build fails until the tool has an example and two summaries.
- Limit case: the owner changes a tool and does not run the command. The site shows the old definition.

## 7. Non-functional requirements

### `WEBSITE-NFR-001`

Each page must show all of its text with scripts turned off in the browser.
Measure on the built file of each page, before a browser runs it.

**Why:** A crawler and a link preview read the file, not the result of a script.

### `WEBSITE-NFR-002`

The home page must transfer 60 KB of script at most, compressed. A tool page must transfer 30 KB at most.
Measure the compressed size of the script files that each built page loads.

**Why:** `ADR-0005`. Only the interactive parts send a script.

### `WEBSITE-NFR-003`

The home page must transfer 500 KB at most in total, fonts included, compressed, on its first view with an empty cache.
Measure in a browser at a width of 390 px.

### `WEBSITE-NFR-004`

Each page must conform to WCAG 2.2 level AA. Each control must work from the keyboard alone.
Measure with an automatic accessibility check on each built page, and by hand on one page of each kind.

## 8. Invariants

### `WEBSITE-INV-001`

The set of tool pages in one language is equal to the set of tools that the server registers.

### `WEBSITE-INV-002`

The set of pages in English is equal to the set of pages in Georgian.

**Why:** `BRD-015`.

### `WEBSITE-INV-003`

The output of a tool example always shows its `source` and its `as_of`.

**Why:** `MCP-005`. A reader checks a number by its source and its date.

## 9. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                      |
| --------- | --------- | ----------------------------------------------------------- |
| `BRD-002` to `BRD-013` | Brand | The look, the mark, the colors, the type, and the motion.  |
| `BRD-014` | Brand     | The copy of each page.                                      |
| `BRD-015` | Brand     | The two languages, and the real outputs.                    |
| `MCP-005` | MCP tools | The `source` and the `as_of` of each tool example.          |

## 10. Open questions

| #   | Question                                                                                     | Owner | Answer |
| --- | -------------------------------------------------------------------------------------------- | ----- | ------ |
| 1   | What is the public address of the server? The install guides of `WEBSITE-FR-007` need it.    | Owner | `https://geo-mcp.abgeo.cloud/mcp`. |
| 2   | Where does the site run, and at which domain?                                                | Owner | `https://geo-mcp.abgeo.cloud`, on the host of the server. |
| 3   | Must the install guide also show a local server, with the binary or the container image?     | Owner | Yes, both. `WEBSITE-FR-009`. |
| 4   | The refresh of `WEBSITE-FR-018` calls `rates_get_commercial_rates` for TBC Bank, which needs a key. Does the refresh use the key of the owner? | Owner | Yes, for now. |
| 5   | The binary installs with `go install` only, and the image builds from the source only. Does a release publish a binary for each platform and an image to a registry? The guides of `WEBSITE-FR-009` depend on it. | Owner | Yes, later. The guides show the release binary and `ghcr.io/abgeo/georgia-mcp` now. |

## Retired identifiers

| ID             | Retired    | Reason                                                                 |
| -------------- | ---------- | ---------------------------------------------------------------------- |
| WEBSITE-FR-016 | 2026-10-20 | The build reads a file, not the server. `WEBSITE-FR-023` replaces it.  |
