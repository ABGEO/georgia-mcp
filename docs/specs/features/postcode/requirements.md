---
id: POSTCODE
title: The postal codes of Georgian Post
type: requirements
status: approved
created: 2026-10-24
updated: 2026-10-24
context: postal
approved_by: Temuri
approved_on: 2026-10-24
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The postal codes of Georgian Post

## 1. Problem

A person asks "what is the postal code of Ilia Chavchavadze Avenue in Tbilisi" or "what
is the postal code of Rustavi". The server has no answer. Georgian Post publishes its
postal codes in a search form on its web page only. It publishes no API and no open
data set of the codes.

## 2. Users

| User                   | Need                                                                |
| ---------------------- | ------------------------------------------------------------------- |
| A person with a client | The postal code of an address, a street, a town, or a village.      |
| The operator           | Turns Georgian Post off with no change to the code.                 |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Georgian Post | The search of the page `www.gpost.ge/help/postal-codes` | No. A web page, not an API. | Not published. A postal code changes rarely. | No key. No published terms and no published limit. `robots.txt` does not exist: the site answers with its home page. |

Observed behavior of the source on 2026-10-24:

- A search takes a text and gives every matching address with its postal code. It groups
  the addresses under a district: a city (`თბილისი`) or a municipality (`ასპინძის რაიონი`).
- Georgian Post gives a postal code for each street in Tbilisi, Batumi, and Kutaisi only.
  For another town or a village, the page tells the person to search its name only.
- One street can have two postal codes, one for each part. The part is in parentheses:
  `ილია ჭავჭავაძის გამზირი (N 2/2-ის გარდა)`.
- A postal code has 4 digits and can start with `0`.
- A search with no match gives a note with the rules above, and no address.
- The page asks for 3 letters or more.
- With English selected, the source gives each address and district in English
  (`City Rustavi`, `Rustavi City`). It matches a text in Georgian or in English in
  both languages.
- A search needs a token from a visit to the page in the same session. A search with no
  token fails with an error of the server.
- The source accepts a User-Agent that names the server and its repository.

| Text         | Addresses | Districts                     | Postal codes                       |
| ------------ | --------- | ----------------------------- | ---------------------------------- |
| `ამბერკი`     | 1         | Tbilisi                       | `0186`                             |
| `ჭავჭავაძის`  | 63        | Tbilisi, Batumi, Kutaisi      | 12, from `0102` to `6010`          |
| `რუსთავი`     | 12        | Rustavi, Aspindza, Tbilisi    | `3700` (city), `0500` (village), `0114` (Tbilisi) |
| `ქუჩა`        | 101       | Tbilisi                       | The broadest text of the probe.    |

The owner intends to ask Georgian Post for permission, or for an API.

## 4. Out of scope

- The address of a postal code.
- A post office, its address, or its hours.
- A postal code for one house number. The source gives a code for a street or a part of it.
- A check that a full address exists.
- A list or a download of every postal code.
- A list of suggested addresses while a person types.
- The language `ru`.

## 5. Definitions

The glossary holds the terms of this feature, in the section Postal.

## 6. Functional requirements

### `POSTCODE-FR-001`

The server must return each address that Georgian Post matches to a text, with its
postal code and its district.

**Examples:**

- Normal case: `ამბერკი`. One address, `ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა`, with
  the postal code `0186`.
- Limit case: `ჭავჭავაძის`. 63 addresses in 3 districts, with 12 postal codes.
- Unwanted case: `ზზზზზქქ`. No address, and no failure.

### `POSTCODE-FR-002`

The server must return the addresses in the order of Georgian Post.

**Why:** Georgian Post puts a settlement with the name of the text before a street
that contains it: `რუსთავი` gives the city of Rustavi first, then the streets of Tbilisi.

### `POSTCODE-FR-003`

The server must give each address and district in the language of the request: `ka` or
`en`. The default is `en`.

**Examples:**

- Normal case: `en` and `რუსთავი`. The first address is `City Rustavi`.
- Limit case: `ka` and `რუსთავი`. The first address is `ქალაქი რუსთავი`.
- Unwanted case: `ru`. The server refuses the request and names `ka` and `en`.

### `POSTCODE-FR-004`

The server must accept a text in Georgian or in English, in either language of the
answer.

**Examples:**

- Normal case: `Rustavi` with `en` gives the same postal codes as `რუსთავი` with `en`.
- Limit case: `Rustavi` with `ka` gives Georgian addresses.

### `POSTCODE-FR-005`

The server must refuse a text of fewer than 3 or more than 100 characters after it
removes the spaces at the ends. It must send no call to Georgian Post for it.

**Why:** Georgian Post asks for 3 letters or more.

**Examples:**

- Normal case: `ვაკე` is accepted.
- Limit case: `  ab  ` is refused.

### `POSTCODE-FR-006`

The server must return no more addresses than the limit of the request. The default
limit is 20. The maximum is 100.

**Why:** A broad text gives more than 100 addresses. A large answer fills the context
window of the model.

**Examples:**

- Normal case: `ჭავჭავაძის` with no limit gives the first 20 of 63 addresses.
- Limit case: the limit 100 and `ჭავჭავაძის` give 63 addresses.
- Unwanted case: the limit 500. The server refuses it and names the range.

### `POSTCODE-FR-007`

The server must state the count of addresses that Georgian Post matched, when that
count is greater than the count that the answer holds.

**Why:** A model must know that a narrower text gives the address that it looks for.

### `POSTCODE-FR-008`

The server must tell how to search again when Georgian Post matches no address. In
Tbilisi, Batumi, and Kutaisi, the person gives a street. For another town or a village,
the person gives its name only.

**Why:** A person often searches a street outside the three cities, and finds nothing.

### `POSTCODE-FR-009`

The server must treat an answer of Georgian Post that it cannot read as a failure of
Georgian Post.

**Why:** Georgian Post can change its page with no notice. A wrong postal code is worse
than no postal code.

**Examples:**

- Normal case: the page has the token, and the search answer has the address rows.
- Unwanted case: the page has no token. The server reports that Georgian Post did not
  answer.
- Unwanted case: a postal code of 3 digits, or an address with no postal code. The
  server reports that Georgian Post did not answer.

### `POSTCODE-FR-010`

The server must read Georgian Post with no key. The operator must be able to turn
Georgian Post off. It is on by default.

**Examples:**

- Normal case: no configuration. The server offers the search of postal codes.
- Unwanted case: Georgian Post is off. The server offers no tool of postal codes.

## 7. Non-functional requirements

### `POSTCODE-NFR-001`

A list of addresses must be less than 7 days old, unless the answer marks it as stale.
Measure from the read instant to the receipt of the request.

**Why:** A postal code changes rarely.

### `POSTCODE-NFR-002`

The server must return a stale list only when Georgian Post fails, and only when the
list is less than 30 days old. Measure from the read instant to the receipt of the
request.

### `POSTCODE-NFR-003`

A search must send no call to Georgian Post when the server read the same text in the
same language less than 7 days before. Measure the calls at the source.

### `POSTCODE-NFR-004`

A search that the server did not read before must send exactly 2 calls to Georgian Post:
one visit to the page and one search. Measure the calls at the source.

**Why:** The owner chose one visit for each search, and no suggestion step. The server
keeps no state of a visit between two searches.

### `POSTCODE-NFR-005`

The server must send at most 2 calls to Georgian Post in each second, for any count of
requests. Measure the calls at the source.

**Why:** Georgian Post publishes no limit. A block of the server stops every person.

### `POSTCODE-NFR-006`

Every call to Georgian Post must carry a User-Agent that names Georgia MCP and the
address of its repository. Measure at the source.

**Why:** Georgian Post can identify the caller and contact the owner.

## 8. Invariants

### `POSTCODE-INV-001`

Every postal code in an answer has exactly 4 digits, as text. A leading `0` stays.

### `POSTCODE-INV-002`

Every address in an answer carries its postal code and its district.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                   |
| -------------------- | ------------- | ------------------------------------------------------------------------ |
| `DOM-001`            | Domain        | The context is `postal`.                                                 |
| `DOM-005`            | Domain        | The check of a postal code belongs to the server, not the client.        |
| `MCP-002`            | MCP tools     | The tool name starts with `postal_`.                                     |
| `MCP-005`            | MCP tools     | Each answer names Georgian Post and the instant at which the server read it. |
| `MCP-007`, `MCP-008` | MCP tools     | The tool takes `language` and `limit`.                                   |
| `CLT-002` to `CLT-005` | Clients     | A web page source, under `ADR-0003`.                                     |
| `CLT-008`            | Clients       | The client keeps no token between two searches.                          |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`.         |
| `CFG-008`            | Configuration | The switch of Georgian Post is on by default.                            |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 7 days. The stale period ends at 30 days.            |
| `RSL-008`            | Resilience    | The limit of `POSTCODE-NFR-005` is a setting.                            |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `POSTCODE` and `postal` (2026-10-24). |
| 2   | Does the server keep the token of a visit for the next search? | Temuri | No. Each search visits the page (2026-10-24). |
| 3   | Does the server ask Georgian Post for suggestions first? | Temuri | No. The search takes the text of the person (2026-10-24). |
| 4   | Which languages? | Temuri | `ka` and `en`, the default `en` (2026-10-24). |
| 5   | How old can a list be? | Temuri | 7 days, and 30 days when Georgian Post fails (2026-10-24). |

## Retired identifiers

This file has no retired identifier.
