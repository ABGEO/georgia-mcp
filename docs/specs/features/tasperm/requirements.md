---
id: TASPERM
title: The applications of the Tbilisi Architecture Service
type: requirements
status: approved
created: 2026-11-15
updated: 2026-11-15
context: permits
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [DOM, MCP, RSL, CLT, CFG, TST]
---

# Requirements: The applications of the Tbilisi Architecture Service

## 1. Problem

A person asks "was a building permit issued for the parcel 01.18.04.020.005", "what did
the city decide on application AR11168188", or "what did Tbilisi approve last week". The
server has no answer. The Tbilisi Architecture Service publishes each application and its
decision, but a person must fill a form in Georgian and open each row.

## 2. Users

| User                   | Need                                                                         |
| ---------------------- | ---------------------------------------------------------------------------- |
| A person with a client | Find the applications of a parcel, of a number, or of a period. Read the category, the stage, the status, and the decision. |
| The operator           | Turns TAS off with no change to the code.                                     |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| TAS | The method `DocumentManager.getDocsForPublicInfo` of `docs.tbilisi.gov.ge/architect`, over DWR. The page `publicInformation.html` calls it. | Yes. The Tbilisi City Hall. The method serves the public search page and has no documentation. | At each application and each decision. | No key, no login, no cookie. No published terms found. |

Observed behavior of the source on 2026-11-09 and on 2026-11-15:

- The answer is JavaScript, not JSON: keys with no quotation marks, `new Date(ms)`,
  `\uXXXX` escapes, and `\u0000` padding.
- The source holds 526,693 applications of every kind: building permits, project
  approvals, signs, fences, and requests for public information. Tbilisi only.
- A filter by cadastral code gives the applications of a parcel: 2 for `01.15.05.066.020`,
  5 for `01.18.04.020.005`. An unknown code gives an empty list.
- A filter by number is a prefix: `AR11168188` gives 1, `AR1116818` gives 6.
- A filter by date of registration works: 87 applications on 2026-11-04.
- With no status filter, the answer holds drafts that TAS never reviewed: 1,328 since
  2026-11-01. The status set of the public page gives 1,053.
- The order is not by date, and a sort field changes nothing.
- A page of 500 applications answered in 1 second, 1.5 MB. A page of 5 answered in 0.1 to
  3.6 seconds.
- An application holds a number (`AR11168188`), an address, a date of registration, a
  deadline, a status code, a category and its parent, a class (`I` to `IV`), the stages
  (`პროექტის შეთანხმება`, `მშენებლობის ნებართვა`), an action, the name of the executor,
  the decision text in HTML, and the identifier of the applicant.
- Every text is Georgian.
- The page of an application is `https://tas.ge/?p=publicpage&documentId=<id>`.
- The detail method of that page returns the personal number, the phone, and the name of
  the applicant. This feature does not call it.

## 4. Out of scope

- The applicant: the name, the personal number, the phone, and the identifier.
- The detail method of the public page.
- A filter by category, by class, by architect, or by the number of a decision.
- The attached files and the drawings of an application.
- The cadastral codes that an application names, beyond the code of the search.
- Any city other than Tbilisi.

## 5. Definitions

The glossary holds the terms of this feature, in the section Permits.

| Term | Meaning |
| ---- | ------- |
| TAS | The Tbilisi Architecture Service of the Tbilisi City Hall. Vendor code `tas`. |
| application | One request to TAS with a number, `AR11168188`, and a status. |
| application number | The number of an application: `AR1` and digits. |
| date of registration | The day on which TAS received the application, in Tbilisi. |
| status | The state of an application. The codes are in `TASPERM-FR-005`. |
| category | The kind of object or work: `მრავალბინიანი საცხოვრებელი სახლი`, `აბრებისა და რეკლამების განთავსება`. |
| class | The class of risk of a construction, `I` to `IV`. |
| stage | A step of a construction that an application covers: project approval, construction permit. |
| decision text | The text with which TAS answers an application. |
| executor | The employee of TAS who handles an application. |

## 6. Functional requirements

### `TASPERM-FR-001`

The server must return the applications of a cadastral code, newest first by the date of
registration.

**Examples:**

- Normal case: `01.18.04.020.005`. 5 applications, the newest first.
- Limit case: `01.15.05.066.020`. 2 requests for public information of 2017.
- Unwanted case: `99.99.99.999.999`. An empty list, and no failure.

### `TASPERM-FR-002`

The server must return the applications registered in a period of at most 31 days, newest
first.

**Why:** The source holds more than 500,000 applications in no order. A bounded period
keeps the answer to one call.

**Examples:**

- Normal case: 2026-11-04 to 2026-11-04. 87 applications.
- Unwanted case: 2026-01-01 to 2026-03-01. The server refuses the period and names the
  limit of 31 days.
- Unwanted case: a start after the end. The server refuses it.

### `TASPERM-FR-003`

A search must take a cadastral code, a period, or both. The server must refuse a search
with neither.

### `TASPERM-FR-004`

When a search matches more than 500 applications, the server must state the count and
ask for a shorter period. It must not return a part of the matches as the newest.

**Why:** The source gives no order. The server can sort only the matches that it reads.

### `TASPERM-FR-005`

The server must give the status of each application as one code:

| Code | TAS status codes | Meaning |
| ---- | ---------------- | ------- |
| `approved` | 1 | TAS approved the application. |
| `refused` | 2 | TAS refused the application. |
| `deficient` | 7 | TAS found a deficiency. The applicant must correct it. |
| `conditional` | 12 | TAS approved the application with conditions. |
| `not_reviewed` | 11 | TAS left the application without review. |
| `revoked` | 27 | The decision lost its force. |
| `completed` | 28 | The procedure ended. |
| `awaiting_payment` | 25 | TAS waits for the fee. |
| `in_review` | 5, 6, 9, 20, 21, 26 | TAS reviews the application. |

The server must return only the applications with one of these TAS codes.

**Why:** The public page of TAS shows the same set. A draft never reached TAS.

### `TASPERM-FR-006`

The server must return an application for its full number, with the decision text as
plain text.

**Examples:**

- Normal case: `AR11168188`. `approved`, with the decision text.
- Limit case: `1168188` or `ar11168188`. The server gives `AR11168188`. It adds `AR1`
  to digits, as the page of TAS does.
- Unwanted case: `AR1116818X`. The server names the form `AR11168188`.
- Unwanted case: a number with no application. The server states it and tells the model
  to search by a cadastral code or a period.

### `TASPERM-FR-007`

Each application in an answer must carry its number, address, date of registration,
deadline, status, category and parent category, class, stages, action, executor, and the
address of its page on `tas.ge`. A field that TAS leaves empty or gives as `-` is absent.
Only the answer of `TASPERM-FR-006` carries the decision text.

**Why:** A decision text can run to several paragraphs. Fifty of them crowd the context of
the model.

### `TASPERM-FR-008`

A search must return at most `limit` applications, 10 by default and 50 at most, and the
count of all matches.

### `TASPERM-FR-009`

The server must name each status code in the `language` of the request, `en` or `ka`.
Every text of TAS stays in Georgian.

### `TASPERM-FR-010`

When TAS fails and the cache holds no answer, the server must state that TAS did not
answer and ask the model to try again later.

### `TASPERM-FR-011`

The server must not offer the tools when the operator turns TAS off.

## 7. Non-functional requirements

### `TASPERM-NFR-001`

An answer must be less than 1 hour old in the cache, unless it is marked as stale. A stale
answer must be less than 24 hours old.

**Why:** A decision changes the status on the day of the decision.

### `TASPERM-NFR-002`

A tool call must cost at most 1 call to TAS. Measure the calls at the source.

### `TASPERM-NFR-003`

The server must send at most 2 calls to TAS in each second, for any count of requests.
Measure the calls at the source.

### `TASPERM-NFR-004`

A tool call must answer in less than 10 seconds when TAS answers in less than 5 seconds.
Measure from the receipt of the request to the last byte.

### `TASPERM-NFR-005`

Every call to TAS must carry a User-Agent that names Georgia MCP and the address of its
repository. Measure at the source.

## 8. Invariants

### `TASPERM-INV-001`

No answer holds the name, the personal number, the phone, or the identifier of an
applicant.

**Why:** An applicant is often a private person.

### `TASPERM-INV-002`

Every address, category, stage, and decision text in an answer is the one that TAS
publishes, with no change of words. The decision text loses only its HTML.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                         |
| -------------------- | ------------- | -------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `permits`.                                      |
| `MCP-002`            | MCP tools     | Each tool name starts with `permits_`.                         |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                      |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The search takes `limit`.           |
| `CLT-007`            | Clients       | The answer is not JSON. The client parses the reply of DWR.    |
| `CFG-008`            | Configuration | The switch of TAS is on by default.                            |
| `RSL-004`, `RSL-006` | Resilience    | The time to live and the stale period of `TASPERM-NFR-001`.    |
| `RSL-008`            | Resilience    | The limit of `TASPERM-NFR-003` is a setting.                   |
| `TST-006`            | Testing       | A recorded answer replaces the identifier of each applicant.   |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `TASPERM` and `permits` (2026-11-15). |
| 2   | Which applications does a search return? | Temuri | All kinds, with the category (2026-11-15). |
| 3   | Which statuses? | Temuri | The set of the public page (2026-11-15). |
| 4   | Which fields beyond the number, address, date, status, and category? | Temuri | The decision text, the executor, the deadline, and the page on `tas.ge` (2026-11-15). |
| 5   | Does a search carry the decision text, or only the application of `TASPERM-FR-006`? | Temuri | Only the application of `TASPERM-FR-006` (2026-11-15). |
| 6   | Are the limits correct: a period of 31 days, 500 matches, 10 and 50 results, 1 hour and 24 hours stale, 2 calls in each second? | Temuri | Yes (2026-11-15). |

## Retired identifiers

This file has no retired identifier.
