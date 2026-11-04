---
id: GEOSTAT
title: The statistical tables and the inflation of Geostat
type: requirements
status: approved
created: 2026-11-04
updated: 2026-11-04
context: stats
approved_by: Temuri
approved_on: 2026-11-04
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The statistical tables and the inflation of Geostat

## 1. Problem

A person asks "what is the inflation in Georgia", "what is the average salary in
Tbilisi", or "what is the population of Kutaisi". The server has no answer. Geostat
publishes these figures in 1,046 tables. A person must find the table on the site of
Geostat and select its cells by hand.

## 2. Users

| User                   | Need                                                                  |
| ---------------------- | --------------------------------------------------------------------- |
| A person with a client | The inflation of a month, or any figure that Geostat publishes.       |
| The operator           | Turns Geostat off with no change to the code.                         |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Geostat | The PxWeb API at `pc-axis.geostat.ge/PXWeb/api/v1` | Yes. The open API of the statistics office. | Each table has its own period: a month, a quarter, or a year. The consumer price index of a month appears early in the next month. | No key. No published limit. No `robots.txt`. The terms at `geostat.ge/en/page/monacemta-gamoyenebis-pirobebi` permit reuse with no prior permission, if the user names Geostat as the source. |

Observed behavior of the source on 2026-11-04:

- The source groups the tables in a tree of folders: 14 subjects, 182 folders, 1,046
  tables. The catalog in English is about 160 KB.
- A folder list takes about 100 ms. 40 calls in a row gave no refusal.
- The search of the source uses an index from 2024. Its paths give "not found" today.
- A table has dimensions. A dimension has a code and an ordered list of values. A value
  has a code (`"0"`, `"1"`) and a text (`"Tbilisi"`). The codes are the same in `ka`
  and in `en`. The language `ru` gives "not found".
- A table has no unit field. The title often holds the unit or the base period.
- Time is two dimensions in most tables: a year (`"2026"`) and a month (`"X"`) or a
  quarter (`"III"`, `"Annual"`). The values go from the oldest to the newest.
- A cell that the source did not publish has the text `..`.
- A query with an unknown dimension or value gives "Parameter error".
- One table gives 25,116 cells with no selection.
- The code of a table can change: `LFS_by_sex_and_urban_(Quarter_II,_2026).px`.

| Table (folder `Price Statistics/CPI/National Consumer Price Index`) | October 2026, Georgia, Total |
| ------------------------------------------------------------- | ------------------------------ |
| Consumer price index, same month of the previous year = 100   | `105.9`                        |
| Consumer price index, previous month = 100                    | `100.9`                        |
| Core inflation, percent change to the same month of the previous year | `4.0`                  |
| November 2026                                                 | `..`                           |

## 4. Out of scope

- The unit of a figure as a separate field.
- A chart, a sum, an average, or any calculation over cells, except inflation.
- The inflation of a period longer than one month in one answer.
- The harmonised index of consumer prices.
- A download of a full table.
- The language `ru`.

## 5. Definitions

The glossary holds the terms of this feature, in the section Stats.

## 6. Functional requirements

### `GEOSTAT-FR-001`

The server must return each table whose title or folder path holds every word of a
text, with its table code, title, folder path, and update date.

**Examples:**

- Normal case: `salary region`. The result holds `Social Statistics/Labour/Wages_Region.px`.
- Limit case: `CONSUMER price`. The match ignores the case of the letters.
- Unwanted case: `zzqx`. No table, and no failure.

### `GEOSTAT-FR-002`

The server must order the matched tables from the newest update date to the oldest.

**Why:** The catalog keeps old series beside the current ones. `GDP` matches tables
of 2019 and of 2026.

### `GEOSTAT-FR-003`

The server must return no more tables than the limit of the request, and state the
count of matched tables. The default limit is 20. The maximum is 100.

### `GEOSTAT-FR-004`

The server must give each title, dimension, and dimension value in the language of the
request: `ka` or `en`. The default is `en`. A search matches the text in that language.

**Examples:**

- Normal case: `ka` and `ინფლაცია`. The result holds the core inflation tables.
- Unwanted case: `ru`. The server refuses the request and names `ka` and `en`.

### `GEOSTAT-FR-005`

The server must return the title, the update date, and each dimension of a table with
each of its values, for a table code.

**Examples:**

- Normal case: `Social Statistics/Labour/Wages_Region.px`. Three dimensions: `Region`
  (12 values), `Year` (17 values), `Quarter` (5 values).
- Unwanted case: an unknown table code. The server tells the model to search the tables.

### `GEOSTAT-FR-006`

The server must return the cells of a table for a selection. A selection gives a list
of value codes for each dimension. A dimension that the selection omits takes every value.

### `GEOSTAT-FR-007`

The server must accept, for a dimension, a count of its last values in place of a list.

**Why:** The values go from the oldest to the newest. A model asks for the newest period
and does not know its code.

**Examples:**

- Normal case: `Year` with the last 1 value gives 2026 only.
- Limit case: the last 50 values of a dimension with 17 values give all 17.

### `GEOSTAT-FR-008`

The server must refuse a selection of more than 1,000 cells, state the count of cells,
and send no call to Geostat for the cells.

**Examples:**

- Normal case: 1 region, 1 group, the last year, and every month: 12 cells.
- Unwanted case: a selection that omits every dimension of the annual CPI table:
  25,116 cells. The server refuses it.

### `GEOSTAT-FR-009`

The server must give each cell with the text of its value on each dimension and its
figure. A figure that Geostat did not publish is empty, not zero.

### `GEOSTAT-FR-010`

The server must refuse an unknown dimension or value code, and name the accepted codes.

### `GEOSTAT-FR-011`

The server must return the annual inflation, the monthly inflation, and the average
inflation of a month, for a region and a price group.

**Examples:**

- Normal case: October 2026, Georgia, Total. Annual `5.9`, monthly `0.9`.
- Limit case: October 2026, Tbilisi, `Food and non-alcoholic beverages`.
- Unwanted case: the region `Rustavi`. The server refuses it and names the 7 regions.

### `GEOSTAT-FR-012`

The server must return the core inflation of a month when the region is Georgia and the
price group is `Total`.

**Why:** Geostat publishes the core inflation for the whole country only.

### `GEOSTAT-FR-013`

The server must use the newest month with a published annual inflation when the request
names no month.

**Examples:**

- Normal case: no month on 2026-11-04. The month is 2026-10.
- Unwanted case: the month 2026-11 on 2026-11-04. The server states that 2026-10 is the
  newest month.

### `GEOSTAT-FR-014`

The server must leave a figure of an inflation answer empty when Geostat did not
publish it for the month, the region, or the group.

**Why:** Geostat publishes the average inflation for the group `Total` only.

### `GEOSTAT-FR-015`

The server must treat an answer of Geostat that it cannot read as a failure of Geostat.

### `GEOSTAT-FR-016`

The server must read Geostat with no key. The operator must be able to turn Geostat off.
It is on by default.

### `GEOSTAT-FR-017`

The server must refuse a search before its first catalog of the language is ready, and
tell the model to try again in a few minutes. The other tools must answer in that time.

**Why:** The server reads the catalog in the background. A first read takes from 30
seconds to several minutes.

## 7. Non-functional requirements

### `GEOSTAT-NFR-001`

The catalog must be less than 24 hours old, unless the answer marks it as stale. Measure
from the read instant to the receipt of the request.

### `GEOSTAT-NFR-002`

The server must return a stale catalog only when Geostat fails, and only when the catalog
is less than 7 days old.

### `GEOSTAT-NFR-003`

A search must answer in less than 500 ms when its catalog is ready. Measure from the
receipt of the request to the last byte of the answer.

### `GEOSTAT-NFR-004`

A table description or a set of cells must be less than 1 hour old, unless the answer
marks it as stale. A stale value must be less than 24 hours old.

**Why:** Geostat publishes a new month on a working day. One hour bounds the delay.

### `GEOSTAT-NFR-005`

The server must send at most 10 calls to Geostat in each second, for any count of
requests. Measure the calls at the source.

### `GEOSTAT-NFR-006`

Every call to Geostat must carry a User-Agent that names Georgia MCP and the address of
its repository. Measure at the source.

### `GEOSTAT-NFR-007`

The server must start to read the catalog of `ka` and of `en` when it starts. It must
hold both catalogs within 15 minutes of the start when Geostat answers each folder list
within 3 attempts. Measure from the start of the process.

**Why:** On 2026-11-04, a full read took from 95 to 326 seconds. Some folder lists did
not answer in 10 seconds, with 1 caller and with 4.

### `GEOSTAT-NFR-008`

The server must send one folder list to Geostat at most 3 times when it fails, with 2
seconds or more between two attempts. It must not retry any other call.

**Why:** Every full read on 2026-11-04 had folder lists that failed, and each one
answered in 0.1 seconds at the next call. A catalog with no folder is wrong.

### `GEOSTAT-NFR-009`

The server must start a new read of a catalog within 15 minutes after a read fails, and
24 hours after a read succeeds.

## 8. Invariants

### `GEOSTAT-INV-001`

Every figure in an answer is the decimal text that Geostat publishes, or empty.

### `GEOSTAT-INV-002`

An inflation figure is the published index minus 100, with no rounding: `105.9` gives `5.9`.

### `GEOSTAT-INV-003`

Every answer names Geostat and the update date of each table that it read.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                         |
| -------------------- | ------------- | -------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `stats`.                                        |
| `DOM-006`, `MCP-009` | Domain, MCP   | A figure is a decimal string, never a float.                   |
| `MCP-002`            | MCP tools     | Each tool name starts with `stats_`.                           |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                      |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The search takes `limit`.           |
| `CLT-008`            | Clients       | The client does not keep the catalog. The cache keeps it.      |
| `CFG-008`            | Configuration | The switch of Geostat is on by default.                        |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale periods of section 7.          |
| `RSL-008`            | Resilience    | The limit of `GEOSTAT-NFR-005` is a setting.                   |
| `RSL-002`            | Resilience    | `GEOSTAT-NFR-008` names the only retry.                        |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `GEOSTAT` and `stats` (2026-11-04). |
| 2   | Which tools? | Temuri | A search, a table description, the cells of a table, and the inflation (2026-11-04). |
| 3   | How does the server find a table? | Temuri | It reads the catalog from the folders and keeps it for 24 hours (2026-11-04). |
| 4   | What happens to a selection of too many cells? | Temuri | The server refuses it above 1,000 cells (2026-11-04). |
| 5   | Do the published terms of Geostat permit reuse by a server? | Temuri | Yes, with Geostat named as the source. `GEOSTAT-INV-003` names it (2026-11-04). |
| 6   | When does the server read the catalog? | Temuri | In the background, at the start and each 24 hours. (2026-11-04). |

## Retired identifiers

This file has no retired identifier.
