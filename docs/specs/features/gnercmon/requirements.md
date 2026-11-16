---
id: GNERCMON
title: The monthly electricity balance and prices of GNERC
type: requirements
status: approved
created: 2026-11-17
updated: 2026-11-17
context: energy
approved_by: Temuri
approved_on: 2026-11-17
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The monthly electricity balance and prices of GNERC

## 1. Problem

A person asks "how much electricity did Georgia import in March", "how much of last year's
generation was hydro", or "what did balancing electricity cost this summer". The grid of
`GSEGRID` answers for one moment of the last 31 days only. GNERC publishes the monthly
balance and the monthly prices, but only as spreadsheets in Georgian on `data.gnerc.org`.

## 2. Users

| User                   | Need                                                                           |
| ---------------------- | ------------------------------------------------------------------------------ |
| A person with a client | The generation, import, export, and consumption of a month or a year, and the prices of a period. |
| The operator           | Turns GNERC off with no change to the code.                                    |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| GNERC | Four workbooks linked from `data.gnerc.org`: the balance (`excel/2015 - 2026 ფაქტიური ბალანსი.xlsx`) and the three prices | Official: the national regulator publishes them for download. GNERC documents no format. | Monthly. September 2026 appeared on 2026-10-21, with 2026 as nine months. | No key. `robots.txt` allows every path. No published terms. |

Observed behavior of the source on 2026-11-17:

- The balance is about 450 KB, with one sheet for each year from 2015 to 2026. Each sheet
  has a row for each item, a column for each month, and a column for the year total, in million kWh.
- The sheets from 2015 to 2021 use another layout, with other items. Out of scope.
- From 2022, each item carries a code: `1.1`, `5.2.1`. A code changes between years: hydro
  is `1.3` up to 2024 and `1.4` from 2025, when solar became `1.3`. The text of an item stays the same.
  The free market suppliers appear in 2024. Solar appears in 2025.
- The import by country has no code. A row "of which into Abkhazia" follows it.
- A month that GNERC has not published holds `0` or nothing: October to December 2026.
- A cell holds a binary float: `388.95957600000003`, `13.3652487`.
- Each price workbook holds one sheet, a row for each month and a column for each year from
  2015, in tetri per kWh. A cell can be text with non-breaking spaces: `" 12.304 "`.
- Every workbook answered 200 with `Last-Modified: Wed, 21 Oct 2026 09:45:18 GMT`, in 0.12 s from Tbilisi.
- The server of GNERC sends no intermediate certificate.
- The home page shows the yearly totals of each year from 2015, rounded to 0.1 million kWh. It is not a source of this feature.

## 4. Out of scope

- The generation of each power station. Its name changes between years.
- The monthly balance before 2022.
- The retail supply of each supplier, and the micro plants of net metering and net billing.
- A forecast, and a price of the day.

## 5. Definitions

The glossary holds the terms of this feature, in the section Energy.

## 6. Functional requirements

### `GNERCMON-FR-001`

The server must return the balance of a month that the request names, from January 2022.

### `GNERCMON-FR-002`

The server must return the balance of a year that the request names, from 2022, with the count of published months that it holds.

**Why:** The current year is partial. A model must not report nine months as a year.

**Example:** 2026 holds 9 months. Its total generation is 10308.382.

### `GNERCMON-FR-003`

The server must return the balance of the newest published month when the request names no month and no year.

### `GNERCMON-FR-004`

The server must refuse a request for the balance that:

- names a month and a year,
- names a month or a year before 2022,
- names a month after the newest published month, or a year after its year.

The refusal names the accepted period: from 2022-01 to the newest published month.

### `GNERCMON-FR-005`

The server must give these balance items, in million kWh:

| Group | Items |
| ----- | ----- |
| Generation | total, thermal, wind, solar, hydro |
| Station own use | total |
| Import | total; from Russia, Turkey, Azerbaijan, Armenia; the part into Abkhazia |
| Supply | total |
| Consumption | total; Abkhazia; suppliers, with the universal service, public service, and free market suppliers; direct consumers; stopped stations |
| Export | total; to Russia, Turkey, Azerbaijan, Armenia |
| Losses | total; transmission; distribution |
| Domestic consumption | total |
| Transit | total; Azerbaijan to Turkey, Russia to Turkey, Armenia to Turkey |

### `GNERCMON-FR-006`

The server must leave out an item that GNERC does not give for the year. It must not give zero for it.

**Examples:**

- Normal case: 2025-03 holds solar generation.
- Limit case: 2024-03 holds no solar generation, and holds the free market suppliers.
- Unwanted case: 2022-03 holds neither.

### `GNERCMON-FR-007`

The server must return the three prices of each published month of a period that the
request names, in tetri per kWh, from January 2015.

### `GNERCMON-FR-008`

The server must return the prices of the last 12 published months when the request names no period.

### `GNERCMON-FR-009`

The server must refuse a period that starts before 2015-01, ends before it starts, or holds more than 60 months.
The server must end the answer at the newest published month when the period ends after it.

**Examples:**

- Normal case: 2025-01 to 2025-12. Twelve months.
- Limit case: 2026-01 to 2026-12. Nine months, to 2026-09.
- Unwanted case: 2020-01 to 2025-12. The server refuses 72 months and names the limit of 60.

### `GNERCMON-FR-010`

The server must leave out a price that GNERC leaves empty for a published month, and give the other prices of that month.

### `GNERCMON-FR-011`

Every answer must give the newest published month of its data, and the instant that GNERC last changed that data, as GNERC reports it.

**Why:** GNERC publishes about three weeks after a month ends. A model must tell the person which month is the newest.

### `GNERCMON-FR-012`

The server must treat a workbook that it cannot read as a failure of GNERC. A workbook that lacks the total generation, the import, the supply, the consumption, or the export of a year is unreadable.

### `GNERCMON-FR-013`

The server must read GNERC with no key. The operator must be able to turn GNERC off. It is on by default.

## 7. Non-functional requirements

### `GNERCMON-NFR-001`

A balance or a price must be less than 6 hours old in the cache of the server, unless the
answer marks it as stale. Measure from the read instant to the receipt of the request.

**Why:** GNERC changes the workbooks once a month. A new month shows within 6 hours.

### `GNERCMON-NFR-002`

The server must return stale data only when GNERC fails, and only when it is less than 7 days old in the cache.

### `GNERCMON-NFR-003`

The server must send at most 3 calls to GNERC in each second, for any count of requests. Measure the calls at the source.

**Why:** A request for the prices reads three workbooks.

### `GNERCMON-NFR-004`

Every call to GNERC must carry a User-Agent that names Georgia MCP and the address of its repository. Measure at the source.

## 8. Invariants

### `GNERCMON-INV-001`

Every value is the value of GNERC rounded half away from zero to 3 decimals.

**Why:** A cell holds a binary float. 3 decimals keep 1 MWh and 0.001 tetri, as GNERC shows its prices.

### `GNERCMON-INV-002`

A month is `YYYY-MM`. An instant is a time of Tbilisi with the offset `+04:00`.

### `GNERCMON-INV-003`

Every answer names GNERC as the source.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                   |
| -------------------- | ------------- | -------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `energy`.                                 |
| `MCP-002`            | MCP tools     | Each tool name starts with `energy_`.                    |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                |
| `MCP-007`            | MCP tools     | The answer holds no text of GNERC. The tools take no `language`. |
| `MCP-008`            | MCP tools     | The prices are bounded: 60 months, as `NBGRATE` bounds its history. |
| `MCP-009`            | MCP tools     | A value is a decimal string. An instant is RFC 3339.     |
| `CFG-008`            | Configuration | The switch of GNERC is on by default.                    |
| `RSL-004`, `RSL-006` | Resilience    | The time to live and the stale period of section 7.      |
| `RSL-008`            | Resilience    | The limit of `GNERCMON-NFR-003` is a setting.            |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `GNERCMON` and `energy` (2026-11-17). |
| 2   | Which data? | Temuri | The prices and the balance totals. Not the stations, the retail supply, or net metering (2026-11-17). |
| 3   | How far back does the balance go? | Temuri | From 2022, the coded layout (2026-11-17). |
| 4   | How many decimals? | Temuri | 3 for the balance and the prices (2026-11-17). |
| 5   | One tool for each dataset, or one tool? | Temuri | One for each dataset (2026-11-17). |
| 6   | Which period does the balance take? | Temuri | A month or a year. No input gives the newest month (2026-11-17). |
| 7   | Which period do the prices take? | Temuri | A period of months, all three prices. No input gives the last 12 (2026-11-17). |
| 8   | How fresh? | Temuri | 6 hours, stale up to 7 days (2026-11-17). |
| 9   | Does `data.gnerc.org` answer the hosted server? | Temuri | Only with the intermediate certificate: curl on the hosted server failed with an SSL error (2026-11-17). |
| 10  | Does the absence of published terms permit reuse, with GNERC named as the source, as for GSE? | Temuri | Yes, with GNERC named as the source (2026-11-17). `GNERCMON-INV-003` names it. |
| 11  | Every balance item, or the 17 core items? | Temuri | Every item of `GNERCMON-FR-005`: 34 (2026-11-17). |

## Retired identifiers

This file has no retired identifier.
