---
id: WEBSITE
title: The website
type: requirements
status: approved
created: 2026-10-20
updated: 2026-10-24
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
A directory of apps, such as the one of OpenAI or of Anthropic, asks for a privacy
policy and terms of service that name this server.

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
- A blog, a changelog, or documentation beyond the tools, the privacy policy, and the terms of service.
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

### `WEBSITE-FR-005`

The home page must show each context with its name, its count of tools, and the names of up to three vendors that it reads. When the context reads more than three vendors, the home page must also show the count of the rest.

**Why:** Cards of equal height keep the grid of contexts even.

**Examples:**

- Normal case: "Fuel prices, 1 tool, Wissol, Rompetrol, SOCAR", then "+N more" for the other companies.
- Limit case: a context with three vendors shows three names and no count.
- Limit case: a context with one vendor shows one vendor name.

### `WEBSITE-FR-007`

The site must give an install guide for each of these clients: Claude on the web and on the desktop, Claude Code, ChatGPT, Cursor, and Visual Studio Code.

**Why:** These are the clients that the owner and the readers use.

**Examples:**

- Normal case: the guide of claude.ai links to the page of Georgia MCP in the directory of Claude, where the reader connects it in one step.
- Limit case: the reader wants a custom connector. The guide of claude.ai still gives the address of the server.

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

The site must answer an unknown address with a page in English that links to the home page.

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

### `WEBSITE-FR-024`

A tool page must show each tool example of the tool, with the question, the input, and the output.

**Why:** The output carries its `as_of` (`WEBSITE-INV-003`). A second date beside it confused the owner.

### `WEBSITE-FR-027`

The build must fail when a context has no chat example, when a chat example has no text in one language, or when a chat example names a tool of another context or a tool that the server does not register.

**Why:** A chat example is written by hand. The build catches the gap that a person misses.

### `WEBSITE-FR-028`

The home page must show the chat examples of several tools, one after another, in a random order. Each chat example shows a question and the answer of a model, as text in the language of the page.

**Why:** One example at a time reads like a real chat. A random order shows a new tool to a reader who comes back.

**Examples:**

- Normal case: a random question types out, the tool name shows, the answer types out. After a pause, the question and the answer of another tool join below it, as in one conversation.
- Limit case: the reader asks the browser for reduced motion. One chat example shows in full, with no typing and no automatic change.
- Unwanted case: the home page shows the input or the output of a tool as JSON.

### `WEBSITE-FR-030`

The home page must hold these sections, in this order: the introduction, the chat examples, the contexts, the install guides, and the tools.

**Why:** The chat examples show what the server does before the reader reads the list of subjects.

### `WEBSITE-FR-032`

Beside the chat examples, the home page must explain in three steps how a question reaches a source and comes back. It must also give the count of tools, of contexts, and of vendors.

**Why:** A reader who does not know MCP sees what happens between the question and the answer.

**Examples:**

- Normal case: "You ask", "The model calls a tool", "You get the answer", then the counts of the tools, the subjects, and the sources.
- Limit case: the owner adds a tool. The count of tools grows with no change to the copy.

### `WEBSITE-FR-033`

A tool page must list each vendor that the tool reads, with its name in the language of the page. When the tool takes the code of a vendor in its input, the page also gives the code.

**Why:** A reader sees which banks or fuel companies a tool can answer for before a call, and which code to name.

**Examples:**

- Normal case: `fuel_get_prices` lists Wissol `wissol`, Rompetrol `rompetrol`, and 5 more.
- Limit case: `rates_get_official_rates` lists the National Bank of Georgia, with no code: the tool takes none.
- Unwanted case: the server reads a new bank, and the page of the tool does not list it. The build fails (`WEBSITE-INV-004`).

### `WEBSITE-FR-034`

The site must have a privacy policy page at `/privacy/` and `/ka/privacy/`. It covers the server at `SERVER_URL` and the site.

**Why:** A directory of apps reads the policy and checks that it names the app. The general policy of the author names no app.

**Examples:**

- Normal case: the page names Georgia MCP, the operator, the effective date, and the address of contact.
- Limit case: a reader runs the server on their own computer. The page says that the policy does not cover that server.

### `WEBSITE-FR-035`

The privacy policy must name each record that the server and the site keep: the access log of each request, and the record of each tool call. It must say that the server keeps no input of a tool call and no content of a request or of a response.

**Why:** The reviewer of a directory checks what the server collects. The logs are the only records.

**Examples:**

- Normal case: the page lists the time, the method, the path, the status, the duration, the IP address, and the user agent of a request.
- Unwanted case: the page claims that the server keeps no IP address.

### `WEBSITE-FR-036`

The site must have a terms of service page at `/terms/` and `/ka/terms/`, for the server at `SERVER_URL` and the site.

### `WEBSITE-FR-037`

The footer of each page must link to the privacy policy page and the terms of service page in the language of the page.

**Examples:**

- Normal case: the footer of `/ka/tools/fuel_get_prices/` links to `/ka/privacy/` and `/ka/terms/`.
- Limit case: the page of `WEBSITE-FR-021` links to `/privacy/` and `/terms/`.

### `WEBSITE-FR-038`

The site must answer `/.well-known/openai-apps-challenge` with the verification token that OpenAI gives the owner, as plain text.

**Why:** OpenAI checks that the owner controls the host of the server before it lists the app. The site and the server share the host.

**Examples:**

- Normal case: a request to the address answers 200 with the token alone.
- Limit case: OpenAI gives a new token. The owner replaces the file, and the next build serves it.

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

### `WEBSITE-NFR-005`

Each page must conform to WCAG 2.2 level AA. Each control must work from the keyboard alone.
The chat examples of the home page are the one exception: they do not meet success criterion 2.2.2, and they have no control that pauses them. With `prefers-reduced-motion: reduce`, they do not move.
Measure with an automatic accessibility check on each built page, and by hand on one page of each kind.

**Why:** The owner wants the chat examples with no control. A reader who asks the browser for reduced motion gets a still chat example.

## 8. Invariants

### `WEBSITE-INV-001`

The set of tool pages in one language is equal to the set of tools that the server registers.

### `WEBSITE-INV-002`

The set of pages in English is equal to the set of pages in Georgian.

**Why:** `BRD-015`.

### `WEBSITE-INV-003`

The output of a tool example always shows its `source` and its `as_of`.

**Why:** `MCP-005`. A reader checks a number by its source and its date.

### `WEBSITE-INV-004`

When a tool definition lists the vendors of the server, the vendors of the tool page are the same set, by code.

**Why:** A list on the site that the server does not match tells the reader a false thing.

## 9. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                      |
| --------- | --------- | ----------------------------------------------------------- |
| `BRD-002` to `BRD-013` | Brand | The look, the mark, the colors, the type, and the motion.  |
| `BRD-014` | Brand     | The copy of each page.                                      |
| `BRD-015` | Brand     | The two languages, and the real outputs.                    |
| `MCP-005` | MCP tools | The `source` and the `as_of` of each tool example.          |
| `LOG-008` to `LOG-010` | Logging | The records that the privacy policy names.          |

## 10. Open questions

| #   | Question                                                                                     | Owner | Answer |
| --- | -------------------------------------------------------------------------------------------- | ----- | ------ |
| 1   | What is the public address of the server? The install guides of `WEBSITE-FR-007` need it.    | Owner | `https://geo-mcp.abgeo.cloud/mcp`. |
| 2   | Where does the site run, and at which domain?                                                | Owner | `https://geo-mcp.abgeo.cloud`, on the host of the server. |
| 3   | Must the install guide also show a local server, with the binary or the container image?     | Owner | Yes, both. `WEBSITE-FR-009`. |
| 4   | The refresh of `WEBSITE-FR-018` calls `rates_get_commercial_rates` for TBC Bank, which needs a key. Does the refresh use the key of the owner? | Owner | Yes, for now. |
| 5   | The binary installs with `go install` only, and the image builds from the source only. Does a release publish a binary for each platform and an image to a registry? The guides of `WEBSITE-FR-009` depend on it. | Owner | Yes, later. The guides show the release binary and `ghcr.io/abgeo/georgia-mcp` now. |
| 6   | How long does the host keep the logs? `WEBSITE-FR-035` needs the period. | Owner | Open. The policy says "only as long as needed to run the service" until the owner gives a period. |

## Retired identifiers

| ID             | Retired    | Reason                                                                 |
| -------------- | ---------- | ---------------------------------------------------------------------- |
| WEBSITE-FR-016 | 2026-10-20 | The build reads a file, not the server. `WEBSITE-FR-023` replaces it.  |
| WEBSITE-FR-014 | 2026-10-20 | The page shows no date of the record. `WEBSITE-FR-024` replaces it.    |
| WEBSITE-FR-006 | 2026-10-20 | The home page shows chat examples. `WEBSITE-FR-025` replaces it.       |
| WEBSITE-FR-025 | 2026-10-21 | The chat examples play one at a time. `WEBSITE-FR-028` replaces it.    |
| WEBSITE-FR-004 | 2026-10-21 | The chat examples come first. `WEBSITE-FR-030` replaces it.            |
| WEBSITE-FR-026 | 2026-10-21 | A chat example links to no tool page. The owner removed the link.      |
| WEBSITE-FR-029 | 2026-10-21 | The reader does not move to the next example. `WEBSITE-FR-031` replaces it. |
| WEBSITE-FR-031 | 2026-10-21 | The chat examples have no pause control. The owner removed it.         |
| WEBSITE-NFR-004 | 2026-10-21 | The chat examples are an exception to WCAG 2.2.2. `WEBSITE-NFR-005` replaces it. |
