---
id: HOLIDAY
title: The public holidays of Georgia
type: requirements
status: approved
created: 2026-11-15
updated: 2026-11-15
context: holidays
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The public holidays of Georgia

## 1. Problem

A person asks "is May 26 a day off", "what are the public holidays in 2027", or
"when is the next long weekend". The server has no answer. A model guesses from its
training data. It misses the Orthodox Easter dates, which move each year.

## 2. Users

| User                   | Need                                                                |
| ---------------------- | ------------------------------------------------------------------- |
| A person with a client | The public holidays of a year, the state of one date, and the next holiday. |
| The operator           | Turns the source off with no change to the code.                    |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Nager.Date | `date.nager.at/api/v3/PublicHolidays/{year}/GE` | No. An open source project computes the list from its own rules. The official list is in the Labour Code of Georgia. | The project computes every year on request. It changes a rule when a contributor corrects it. | No key. The project states "No Rate Limits" on its page. It publishes no terms for the API. |

Observed behavior of the source on 2026-11-15:

- One call returns the holidays of one year for one country, in date order. A call
  takes about 0.15 s. The answer carries `cache-control: max-age=604800`.
- A holiday has a date (`2026-04-12`), a name in `ka` (`localName`), a name in `en`
  (`name`), and the type `Public`. Every holiday of Georgia is national.
- 2026 holds 17 holidays. Orthodox Easter is four days: Good Friday to Easter Monday.
  New Year is two days with the same name.
- The source supports the years 1975 to 2075. Another year gives the status 400. An
  unknown country gives the status 404.
- The method `v4` at `nagerholidays.com` gives the `en` name only.

The source holds no day off by government decree, for example 2025-08-29 (Decree 381).

## 4. Out of scope

- A count of working days between two dates.
- A year before 2025, or a year after the next year.
- A holiday of another country.
- A day off by government decree. See section 3.
- A religious feast day that is not a day off.

## 5. Definitions

The glossary holds the terms of this feature, in the section Holidays.

## 6. Functional requirements

### `HOLIDAY-FR-001`

The server must return every public holiday of a year, in date order.

**Examples:**

- Normal case: `2026`. 17 public holidays, from 2026-01-01 to 2026-11-23.
- Limit case: `2025`. The first year that the server supports.
- Unwanted case: `2024`, or `2028` in 2026. The server refuses it and names the supported years.

### `HOLIDAY-FR-002`

The server must use the current year in Georgia when the request names no year.

### `HOLIDAY-FR-003`

The server must support the years from 2025 to the year after the current year in Georgia.

**Why:** 2025 is the first year of the current list of the law. A person plans a year
ahead at most. A year further ahead can change before it comes.

**Examples:**

- Normal case: in 2026, the server supports 2025, 2026, and 2027.
- Limit case: on 2027-01-01, the server supports 2028.

### `HOLIDAY-FR-004`

The server must tell whether a date is a public holiday. When it is, the server must
give the holiday name.

**Examples:**

- Normal case: `2026-05-26`. A public holiday: Independence Day.
- Limit case: `2026-01-02`. A public holiday: New Year's Day, the second day.
- Unwanted case: `2026-02-30`. The server refuses it as an invalid date.

### `HOLIDAY-FR-005`

The server must use the current date in Georgia when the request names no date.

### `HOLIDAY-FR-006`

The server must return the next public holidays on or after a date, in date order.
The default date is the current date in Georgia.

**Examples:**

- Normal case: `2026-11-15`. The first is Saint George's Day on 2026-11-23.
- Limit case: `2026-11-23`. The first is Saint George's Day on 2026-11-23.
- Limit case: `2026-12-31`. The first is New Year's Day on 2027-01-01.
- Limit case: `2027-11-15` and the limit 20, in 2026. The server returns the holidays
  to 2027-12-31 only.

### `HOLIDAY-FR-007`

The server must return no more holidays than the limit of the request for
`HOLIDAY-FR-006`. The default limit is 1. The maximum is 20.

### `HOLIDAY-FR-008`

The server must give each public holiday with its date, its holiday name, and its weekday.

**Why:** "Is it a long weekend" needs the weekday.

### `HOLIDAY-FR-009`

The server must give each holiday name in the language of the request: `ka` or `en`.
The default is `en`.

### `HOLIDAY-FR-010`

The server must refuse a date outside the supported years for `HOLIDAY-FR-004` and
`HOLIDAY-FR-006`, and name the supported years.

### `HOLIDAY-FR-011`

The server must treat an answer of the source that it cannot read as a failure of the source.

### `HOLIDAY-FR-012`

The server must read the source with no key. The operator must be able to turn the
source off. It is on by default.

### `HOLIDAY-FR-013`

The server must tell the model that the holidays hold no day off by government decree.

**Why:** The source holds the holidays of the law only. A person who asks about a day
off by decree must not get a false "no".

## 7. Non-functional requirements

### `HOLIDAY-NFR-001`

The holidays of a year must be less than 7 days old in the cache of the server, unless
the answer marks them as stale. Measure from the read instant to the receipt of the request.

**Why:** The list of a year changes only when the law or the source changes.

### `HOLIDAY-NFR-002`

The server must return stale holidays only when the source fails, and only when they are
less than 30 days old in the cache.

### `HOLIDAY-NFR-003`

The server must send at most 2 calls to the source in each second, for any count of
requests. Measure the calls at the source.

### `HOLIDAY-NFR-004`

Every call to the source must carry a User-Agent that names Georgia MCP and the address
of its repository. Measure at the source.

## 8. Invariants

### `HOLIDAY-INV-001`

Every date in an answer is a date in Georgia, as `YYYY-MM-DD`.

### `HOLIDAY-INV-002`

Every answer names Nager.Date as the source.

### `HOLIDAY-INV-003`

Every answer holds only the public holidays that the source publishes. The server adds
no holiday and removes no holiday.

**Why:** A list that the server changes has no single source to cite.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                 |
| -------------------- | ------------- | ------------------------------------------------------ |
| `DOM-001`            | Domain        | The context is `holidays`.                             |
| `MCP-002`            | MCP tools     | Each tool name starts with `holidays_`.                |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.              |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The next holidays take `limit`. |
| `MCP-009`            | MCP tools     | A date is `YYYY-MM-DD`.                                |
| `CFG-008`            | Configuration | The switch of the source is on by default.             |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale periods of section 7.  |
| `RSL-008`            | Resilience    | The limit of `HOLIDAY-NFR-003` is a setting.           |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `HOLIDAY` and `holidays` (2026-11-15). |
| 2   | Which source? | Temuri | Nager.Date for now (2026-11-15). |
| 3   | Which tools? | Temuri | The holidays of a year, the state of a date, and the next holidays (2026-11-15). |
| 4   | Which languages and years? | Temuri | `ka` and `en`, from 2025 (2026-11-15). |
| 5   | Does the server tell the model that a day off by decree is missing? | Temuri | Yes (2026-11-15). `HOLIDAY-FR-013` holds it. |
| 6   | Which years? | Temuri | 2025 to the next year (2026-11-15). `HOLIDAY-FR-003` holds it. |
| 7   | What limit for the next holidays? | Temuri | Default 1, maximum 20 (2026-11-15). |
| 8   | What cache periods? | Temuri | 7 days, stale for 30 days (2026-11-15). |

An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
