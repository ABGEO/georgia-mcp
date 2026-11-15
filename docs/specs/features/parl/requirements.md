---
id: PARL
title: The bills, votes, and questions of the Parliament of Georgia
type: requirements
status: approved
created: 2026-11-15
updated: 2026-11-15
context: parliament
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The bills, votes, and questions of the Parliament of Georgia

## 1. Problem

A person asks "where is the bill on the state budget for 2027", "how did the parliament
vote on it", "how did my MP vote", or "which questions did an MP send to a ministry".
The server has no answer. The Legislative Herald holds an act after its adoption only.
The Parliament publishes each bill, each stage of its review, each vote with the choice
of each member, and each question of a member, on a site that a person must search by hand.

## 2. Users

| User                   | Need                                                                     |
| ---------------------- | ------------------------------------------------------------------------ |
| A person with a client | Find a bill and its stage. Read a vote and the choice of each member. Find the questions of a member. |
| The operator           | Turns the Parliament off with no change to the code.                     |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| The Parliament | The JSON methods of `info.parliament.ge`, described at `/v2/api-docs?group=parl.infdep.ge` | Yes. The IT department of the Parliament publishes it, with a Swagger page. | At each new bill, stage, vote, and question. 12 votes from 2026-11-10 to 2026-11-12. | No key. No published terms. The definition names the license GPLv3. No `robots.txt`. |

Observed behavior of the source on 2026-11-15:

- The source holds 26,312 bills, 18,588 votes, and 14,183 questions. 573 votes in 2026.
- Every text is Georgian. Only the records of the members carry an English name.
- A list method needs the inputs `start` and `limit`. With no `limit`, or with an unknown
  input, it answers an HTML page with the status 404.
- A list method gives the total count and a page. A bill page with a `limit` above 50
  gives an empty body with the status 200. A vote or question page of 100 works.
- A date input is `dd-MM-yyyy`. A date in another form gives an empty list, with no error.
- An unknown bill or vote identifier gives `[null]` with the status 200.
- A search of the bill titles gives the newest first. The general search of the bills
  gives an order with no date: "ბიუჯეტ" gives a bill of 2013 first.
- A search of the votes by the title of the bill gives the newest first.
- A text in English, "budget", finds no bill.
- A bill holds its package: the registration number, the initiator, the author, and the
  registration date. A package can hold more than one bill.
- A bill has review stages. The change to the Budget Code (32449) has 17: the initiated
  version, 15 committees, and the first reading.
- A stage with a vote holds its yes and no counts. A filter by stage gives its votes.
- A vote holds its outcome, its quorum, its reading, its bill, and three counts: yes, no,
  and "without interaction". Vote 31510: 77 yes, 0 no, 24 without interaction.
- The choices of one vote come in one page, each with the member and the faction:
  101 members in 2026, 151 in 2015. A choice is "დიახ", "არა", or "არ ესწრება".
  Vote 31507: 79, 10, and 12, equal to its counts of yes, no, and without interaction.
- In 2026, 562 votes adopted the item and 12 did not.
- A question holds its type, number, registration date, request, addressee, member,
  faction, deadline, and response date. The question and the answer are PDF files.
- The site shows a bill at `/#law-drafting/{id}` and a question at `/#mpqs/{id}`.
  A vote has no page of its own.

## 4. Out of scope

- The members, the committees, the factions, the agendas, the calendar, and open data.
- The texts of a bill, a question, and an answer. Each is a PDF file.
- The record of one member across votes. The source refuses a filter by member.
- The notes on a bill. The source lets a person write them.
- The export to a spreadsheet.

## 5. Definitions

The glossary holds the terms of this feature, in the section Parliament.

| Term | Meaning |
| ---- | ------- |
| the Parliament | The Parliament of Georgia, as the publisher of `info.parliament.ge`. Vendor code `parliament`. |
| bill | One draft that the Parliament reviews, with a bill identifier: `32449`. |
| bill identifier | The number of a bill at the Parliament. |
| registration number | The number of a package of bills: `372/2-XIმპ`. |
| initiator | The body that submits a bill: "საქართველოს მთავრობა". |
| author | The body that drafted a bill: "საქართველოს ფინანსთა სამინისტრო". |
| review stage | One step in the review of a bill: a committee hearing or a reading. |
| reading | A plenary review stage. Section 5.1 gives the codes. |
| vote | One plenary vote on a review stage, with a vote identifier: `31510`. |
| outcome | `adopted` or `rejected`. |
| choice | What one member did in a vote: `yes`, `no`, `absent`, or `other`. |
| faction | A group of members in the Parliament: "ფრაქცია „ქართული ოცნება“". |
| member question | A written question of a member to a body of the state. |
| addressee | The body and the person that a member question goes to. |
| question status | `answered`, `pending`, or `overdue`. |

### 5.1 Readings

| Code | Name at the Parliament |
| ---- | ---------------------- |
| `first_reading` | I მოსმენა |
| `second_reading` | II მოსმენა |
| `third_reading` | III მოსმენა |
| `single_reading` | ერთი მოსმენა |
| `plenary_draft` | სხდომაზე განსახილველი პროექტი |
| `alternative_proposal` | ალტერნატიული წინადადება |
| `other` | Any other name. The answer keeps the name of the source. |

## 6. Functional requirements

### `PARL-FR-001`

The server must return the bills whose title holds a text, with the bill identifier,
title, bill type, registration number, registration date, initiator, author, and the
name of the newest review stage of each.

**Examples:**

- Normal case: `ბიუჯეტ`. 1088 bills. The newest is first.
- Unwanted case: `zzqxv`. No bill, and no failure.

### `PARL-FR-002`

The server must filter a bill search by a range of registration dates and by a
registration number, when the request gives them. A search needs a text, a range, or
a number.

### `PARL-FR-003`

The server must return the metadata of a bill for a bill identifier, with every review
stage in the order of the review. A stage gives its name, its reading code when it is a
reading, its dates, its session number, its yes and no counts, and the vote identifiers
of its votes.

**Why:** A person asks where a bill is. A model asks for a vote by its identifier.

**Examples:**

- Normal case: 32449. 17 stages. The last is `first_reading`, with the vote 31510.
- Unwanted case: an unknown identifier. The server tells the model to search the bills.

### `PARL-FR-004`

The server must return the votes in a range of dates, or the votes on bills whose title
holds a text, newest first. A vote gives the vote identifier, date, bill identifier, bill
title, registration number, reading, outcome, quorum, and the counts of yes, no, and
without interaction.

**Examples:**

- Normal case: 2026-11-10 to 2026-11-12. 12 votes.
- Limit case: a range with no plenary sitting. No vote, and no failure.

### `PARL-FR-005`

The server must return a vote for a vote identifier, with the choice, the name, and the
faction of each member, and the count of each choice in each faction.

**Examples:**

- Normal case: 31510. 101 members. 77 `yes` and 24 `absent`.
- Unwanted case: an unknown identifier. The server tells the model to list the votes.

### `PARL-FR-006`

The server must return the member questions whose text holds a text, or that a member
sent, or that the Parliament registered in a range of dates, newest first. A question
gives its identifier, type, registration number, registration date, request, member,
faction, addressee, deadline, response date, and question status.

### `PARL-FR-007`

The server must set the question status to `answered` when the question has a response
date, to `overdue` when it has none and its deadline is before the current day in
Tbilisi, and to `pending` in each other case.

**Examples:**

- Normal case: 14583, registered 2026-11-13, deadline 2026-11-28, no response. `pending`.
- Limit case: no response, and the deadline is the current day. `pending`.

### `PARL-FR-008`

The server must state the total count of a search or a list. It must return the page
that the request names, with at most 50 items on a page.

**Why:** The source gives no bill page above 50.

### `PARL-FR-009`

The server must give each reading, outcome, and choice as a code, with the Georgian name
of the source beside it. An unknown name becomes `other`.

**Why:** A code lets a model answer in any language. The Georgian name keeps the source.

### `PARL-FR-010`

The server must tell the model to search in Georgian when a text with no Georgian letter
finds nothing.

**Why:** "budget" finds no bill. The model must not conclude that no bill exists.

### `PARL-FR-011`

The server must refuse a range whose start is after its end, and a search with no
criterion, and name the accepted inputs.

### `PARL-FR-012`

The server must treat an HTML page, an empty body, or a body that it cannot read as a
failure of the Parliament.

### `PARL-FR-013`

The server must read the Parliament with no key. The operator must be able to turn it
off. It is on by default.

## 7. Non-functional requirements

### `PARL-NFR-001`

A vote, and a bill read by its identifier, must be less than 1 hour old in the cache.
A search or a list must be less than 15 minutes old. A stale value must be less than
24 hours old, and the answer marks it as stale.

**Why:** A plenary sitting adds votes during the day. A cast vote never changes.

### `PARL-NFR-002`

The server must add no wait between two calls to the Parliament below 10 calls in each
second. It must send at most 10 calls in each second, for any count of requests.
Measure the calls at the source.

**Why:** The limit of `RSL-008`. A bill with many votes needs several calls.

### `PARL-NFR-003`

A tool call must answer in less than 10 seconds when the source answers each call in
less than 3 seconds. Measure from the receipt of the request to the last byte.

**Why:** The source answered in less than 0.1 seconds on 2026-11-15. A bill reads its
stages and the votes of each stage.

### `PARL-NFR-004`

Every call to the Parliament must carry a User-Agent that names Georgia MCP and the
address of its repository. Measure at the source.

## 8. Invariants

### `PARL-INV-001`

Every text in an answer is the text that the Parliament publishes, with no summary and
no change of words.

### `PARL-INV-002`

Every bill and every vote in an answer names the address of the page of its bill at the
Parliament. Every member question names the address of its page.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                       |
| -------------------- | ------------- | ------------------------------------------------------------ |
| `DOM-001`            | Domain        | The context is `parliament`.                                 |
| `MCP-002`            | MCP tools     | Each tool name starts with `parliament_`.                    |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                    |
| `MCP-007`            | MCP tools     | The source is Georgian only. The tools take no `language`.   |
| `MCP-008`            | MCP tools     | Each search and list takes `limit`, at most 50.              |
| `MCP-009`            | MCP tools     | A date input and output is `YYYY-MM-DD`.                     |
| `CFG-008`            | Configuration | The switch of the Parliament is on by default.               |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale period of `PARL-NFR-001`.    |
| `RSL-008`            | Resilience    | The limit of `PARL-NFR-002` is a setting.                    |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `PARL` and `parliament` (2026-11-15). |
| 2   | Does a Georgian-only source take `language`? | Temuri | No. `MCP-007` changed with no ADR (2026-11-15). |
| 3   | What does a vote return? | Temuri | Every member, and the count of each choice in each faction (2026-11-15). |
| 4   | What does a bill return? | Temuri | Every review stage, with the vote identifiers of each (2026-11-15). |
| 5   | Is "without interaction" the same as `absent`? | Temuri | Yes. The choices of the votes 7102 (2015) and 31507 (2026) match their counts (2026-11-15). |

## Retired identifiers

This file has no retired identifier.
