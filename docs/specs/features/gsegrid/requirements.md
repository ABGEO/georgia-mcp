---
id: GSEGRID
title: The electricity grid of GSE
type: requirements
status: approved
created: 2026-11-17
updated: 2026-11-17
context: energy
approved_by: Temuri
approved_on: 2026-11-17
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The electricity grid of GSE

## 1. Problem

A person asks "how much of our power is hydro right now", "is Georgia importing electricity
tonight", or "how much did we export to Turkey yesterday at 13:00". The server has no
answer. GSE publishes the state of the grid every 3 minutes, but only as a chart on
`gse.com.ge`. A person must read the chart by hand, and the chart shows no past moment.

## 2. Users

| User                   | Need                                                                  |
| ---------------------- | --------------------------------------------------------------------- |
| A person with a client | The generation by type, the consumption, and the flows across each border, now or at a past moment. |
| The operator           | Turns GSE off with no change to the code.                             |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| GSE | The real time exports of `admin.gse.com.ge/storage/uploads/fact/`: one file of the grid for each 3 minutes, one file of the frequency for each minute | GSE is official: the state company that operates the transmission grid. The files are unofficial: GSE documents no format, and its own site reads them. | A file appears 1 to 2 minutes after its time. GSE keeps it for 31 days and never changes it. | No key. `robots.txt` allows every path. No published terms. |

Observed behavior of the source on 2026-11-17:

- The name of a file holds its time in Tbilisi, with the minute floored to 3 for the grid:
  `GSE_EXPORT_NEW_PERIOD_03M_17112026_0112.XML`, `GSE_EXPORT_PERIOD_01M_17112026_0114.XML`.
  A file that does not exist returns 404. A file of 31 days ago answered. A file of 32 days ago returned 404.
- A grid file is about 2.3 KB. It answers in less than 0.1 s from Tbilisi, and in 0.41 s from the hosted server. It holds the
  time (`17.11.2026 01:12:00`) and 13 values in MW, each the average of 3 minutes:

| Value | Name in the file |
| ----- | ---------------- |
| Hydro, thermal, wind, and solar generation | `TOTAL HYDRO STATIONS`, `TOTAL TERMAL STATIONS`, `TOTAL WIND PLANTS`, `TOTAL SUN PLANTS` |
| Total generation | `TOTAL GENERATION (HYDRO+TERMAL+WIND+SU` |
| Consumption | `TOTAL CONSUMPTION` |
| Flows of Azerbaijan, Armenia, and Turkey | `TOTAL AZERBAIJAN TIE LINES`, `TOTAL ARMENIA TIE LINES`, `TOTAL TURKEY TIE LINES` |
| Flows of Russia: four lines | `TOTAL RUSSIA TIE LINES`: `Russia`, `Salkhino`, `Java`, `Nakaduli` |

- A positive flow enters Georgia. In each of 9 files, the total generation plus every flow
  equals the consumption to 0.01 MW. The site of GSE computes the flow of Russia as the sum of its four lines.
- Each value carries a quality code. In 9 files: `ACT` 51 times, `SUB` 42, `NRE` 15, `BLS` 9.
  The consumption and the total generation were often `SUB` or `NRE`. The line `Java` was `BLS` with
  the same value, 18.3, in every file. GSE documents no code. The file names a DTD and a stylesheet; both return 404.
- A value has 6 decimals and comes from a single precision float: `18.299999`, `-0.000000`.
- A frequency file holds one value in Hz with a quality code: `50.020000`, `ACT`.

## 4. Out of scope

- The forecast of the next day. GSE publishes it as a binary spreadsheet that the standard library cannot read.
- The publications, news, and tenders of GSE.
- The values of a whole day as a series.
- A moment older than 31 days.
- The generation of a single power station. GSE publishes totals only. The monthly data of GNERC is a separate feature.
- The day-ahead market of GENEX. It shows one day of trades, 2026-06-29.

## 5. Definitions

The glossary holds the terms of this feature, in the section Energy.

## 6. Functional requirements

### `GSEGRID-FR-001`

The server must return the grid state of the newest grid slot when the request names no moment.

### `GSEGRID-FR-002`

The server must return the grid state of the grid slot that holds a moment of the request.

**Examples:**

- Normal case: `2026-11-16 13:01` in Tbilisi. The slot of 13:00.
- Limit case: `2026-11-16T09:01:00Z`. The same slot: 13:00 in Tbilisi.
- Unwanted case: a moment in the future, or more than 31 days before the request. The server refuses it and names the accepted period.

### `GSEGRID-FR-003`

The server must read a moment with no offset as a time of Tbilisi.

### `GSEGRID-FR-004`

The server must return the newest earlier grid slot when GSE publishes no file for the slot
of the request. It must search no further than 30 minutes before that slot.

**Why:** The newest slot appears 1 to 2 minutes late, and a slot can be missing. 30 minutes bounds the search.

**Examples:**

- Normal case: the request at 01:25. The slot of 01:24 is absent. The server returns the slot of 01:21.
- Unwanted case: no file in the 30 minutes. The server reports that GSE has no data for the moment.

### `GSEGRID-FR-005`

The server must give the time of the grid slot that it returns, and the length of the average: 3 minutes.

**Why:** FR-004 can return a slot before the moment of the request. The model must report the real time.

### `GSEGRID-FR-006`

The server must give the hydro, thermal, wind, and solar generation, the total generation,
and the consumption, each in MW.

### `GSEGRID-FR-007`

The server must give the flow of each country: Azerbaijan, Armenia, Russia, and Turkey.
It must give each line of Russia as GSE names it. The flow of Russia is the sum of its four lines.

**Example:** `Russia` 0, `Salkhino` 0, `Java` 18.3, `Nakaduli` 0.0. The flow of Russia is 18.3.

### `GSEGRID-FR-008`

The server must give the direction of each flow and of the net import: `import`, `export`, or `none` for zero.

**Why:** A model reads a sign wrongly. A word states the direction.

### `GSEGRID-FR-009`

The server must give the net import of the grid slot.

**Example:** Azerbaijan -23.05, Russia 30.12, Turkey 279.20, Armenia 0. The net import is 286.27, `import`.

### `GSEGRID-FR-010`

The server must give each value with its quality code and a meaning in English:

| Code | Meaning |
| ---- | ------- |
| `ACT` | measured |
| `SUB` | substituted by the operator |
| `NRE` | not received |
| `BLS` | not documented by GSE |
| Any other code | unknown |

A sum of FR-007 or FR-009 carries no quality code.

**Why:** Many values are not measured. A model must not report a substituted value as a measurement.

### `GSEGRID-FR-011`

The server must give the count of measured values and the count of all values of the grid slot.

### `GSEGRID-FR-012`

The server must give the frequency of the minute of the grid slot, with its quality code.
If GSE publishes no frequency for that minute, the answer holds no frequency and still holds the grid state.

### `GSEGRID-FR-013`

The server must treat an answer of GSE that it cannot read as a failure of GSE.

### `GSEGRID-FR-014`

The server must read GSE with no key. The operator must be able to turn GSE off. It is on by default.

## 7. Non-functional requirements

### `GSEGRID-NFR-001`

The newest grid state must be less than 2 minutes old in the cache of the server, unless the
answer marks it as stale. Measure from the read instant to the receipt of the request.

**Why:** GSE adds a slot each 3 minutes. 2 minutes bounds the delay.

### `GSEGRID-NFR-002`

The server must return a stale newest grid state only when GSE fails, and only when it is
less than 30 minutes old in the cache.

### `GSEGRID-NFR-003`

The server must send at most 5 calls to GSE in each second, for any count of requests.
Measure the calls at the source.

**Why:** FR-004 reads up to 11 files for one request.

### `GSEGRID-NFR-004`

Every call to GSE must carry a User-Agent that names Georgia MCP and the address of its
repository. Measure at the source.

## 8. Invariants

### `GSEGRID-INV-001`

Every value from a file is the decimal text that GSE publishes. A sum is the exact decimal sum of those values.

### `GSEGRID-INV-002`

Every time in an answer is a time of Tbilisi with the offset `+04:00`.

### `GSEGRID-INV-003`

Every answer names GSE as the source.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                 |
| -------------------- | ------------- | ------------------------------------------------------ |
| `DOM-001`            | Domain        | The context is `energy`.                               |
| `DOM-007`            | Domain        | The time of a slot is a time of Tbilisi.               |
| `MCP-002`            | MCP tools     | Each tool name starts with `energy_`.                  |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.              |
| `MCP-007`            | MCP tools     | GSE publishes the names in English. The tool takes no `language`. |
| `MCP-009`            | MCP tools     | A value is a decimal string. A time is RFC 3339.       |
| `CFG-008`            | Configuration | The switch of GSE is on by default.                    |
| `RSL-004`, `RSL-006` | Resilience    | The time to live and the stale period of section 7.    |
| `RSL-008`            | Resilience    | The limit of `GSEGRID-NFR-003` is a setting.           |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `GSEGRID` and `energy` (2026-11-17). |
| 2   | Which data? | Temuri | The grid and the frequency. The forecast and GNERC wait (2026-11-17). |
| 3   | How does an answer show a value that is not measured? | Temuri | Each value carries its quality code and a meaning (2026-11-17). |
| 4   | A series of a day, or one moment? | Temuri | One moment (2026-11-17). |
| 5   | Does `admin.gse.com.ge` answer the hosted server? | Temuri | Yes. A grid file returned 200, 2299 bytes, in 0.41 s (2026-11-17). |
| 6   | Does the server report `18.299999` and `-0.000000` as published, as `NEAAIR` does, or round to 0.1 MW? | Temuri | As published (2026-11-17). `GSEGRID-INV-001` holds it. |
| 7   | Does the absence of published terms permit reuse, with GSE named as the source, as for the NEA? | Temuri | Yes, with GSE named as the source (2026-11-17). `GSEGRID-INV-003` names it. |

## Retired identifiers

This file has no retired identifier.
