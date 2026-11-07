---
id: LLMSTXT
title: The catalog files
type: requirements
status: approved
created: 2026-11-08
updated: 2026-11-08
context: website
approved_by: Temuri
approved_on: 2026-11-08
constrained_by: [MCP, LNG]
---

# Requirements: The catalog files

## 1. Problem

An agent or a crawler that wants the tools of Georgia MCP must parse the pages of the
site, or connect to the server and open a session. Neither gives one short text that a
model reads in full, nor the input schemas before the first call.

## 2. Users

| User                  | Need                                                                  |
| --------------------- | --------------------------------------------------------------------- |
| An agent or a crawler | Read the tools and the server address without parsing a page.         |
| A directory           | Read the tool definitions without a session with the server.          |

## 3. Source

| Vendor      | Source                                                        | Official                 | Publication period         | Terms of use |
| ----------- | ------------------------------------------------------------- | ------------------------ | -------------------------- | ------------ |
| Georgia MCP | The data of the site of `WEBSITE`: the tool definitions, the tool summaries, and the tool examples | Yes. The server is ours. | At each build of the site. | None. |

## 4. Out of scope

- A Georgian catalog file or a Georgian Markdown copy. The tool definitions exist in English only.
- A Markdown copy of the home page or of a legal page.
- The site version in a catalog file.

## 5. Definitions

The terms are in the glossary, in the section "The website": catalog file, Markdown copy.
`SERVER_URL` is the address of the server of `WEBSITE`.

## 6. Functional requirements

### `LLMSTXT-FR-001`

The site must serve a catalog file at `/llms.txt`, in English, as plain text in Markdown. It names the server, says in one sentence what the server is, and gives `SERVER_URL`. It links to the install guides and the repository. It lists each tool with its title, the address of its Markdown copy, and its description.

**Why:** A model reads the tools from one short file, with no page to parse.

**Examples:**

- Normal case: the file lists `fuel_get_prices` as a link to `/tools/fuel_get_prices.md`, followed by the description that the server gives.
- Limit case: the server adds a tool. The next build lists it, with no change to the copy.
- Unwanted case: the sentence that says what the server is lists or counts the subjects. A build test fails.

### `LLMSTXT-FR-002`

The site must serve a Markdown copy of each English tool page at `/tools/<name>.md`, as plain text. It holds the tool summary, the tool definition, the tool example, the address of the tool page, and `SERVER_URL`.

**Why:** The format of `/llms.txt` links to a Markdown copy of each page, so a model reads the page without its markup.

**Examples:**

- Normal case: `/tools/fuel_get_prices.md` holds the summary, the inputs with their descriptions, one question, its input, and its output with `source` and `as_of`.
- Limit case: a tool with no input lists no input and says so.

### `LLMSTXT-FR-003`

The site must serve a catalog file at `/tools.json`, as JSON. It holds the server name, `SERVER_URL`, and each tool with its tool definition, its output schema, and its annotations, as the server lists them, plus the address of its tool page and of its Markdown copy.

**Why:** An agent plans its calls before it connects, and a directory reads the tools without a session.

**Examples:**

- Normal case: the entry of `fuel_get_prices` has the same input schema as the tool list of the server.
- Limit case: a tool with no output schema has no output schema in its entry.

### `LLMSTXT-FR-004`

The head of each page must link to `/llms.txt` and `/tools.json`. The head of each tool page in English must also link to its Markdown copy. The robots file of the site must name both catalog files.

**Why:** A crawler that reads one page or the robots file finds the catalog.

## 7. Non-functional requirements

This feature has no non-functional requirement.

## 8. Invariants

### `LLMSTXT-INV-001`

The set of tools in `/llms.txt`, the set in `/tools.json`, and the set of Markdown copies are each equal to the set of tools that the server registers. The catalog files and the Markdown copies exist in English only, outside `WEBSITE-INV-002`.

**Why:** The tool definitions exist in English only. A catalog that the server does not match tells an agent a false thing.

## 9. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                      |
| --------- | --------- | ----------------------------------------------------------- |
| `MCP-005` | MCP tools | The output of each example in a Markdown copy shows `source` and `as_of`. |
| `LNG-006`, `LNG-007` | Language | The copy of the catalog files. |

## 10. Open questions

None.
