---
id: NEAAIR
title: The air quality of the National Environmental Agency
type: requirements
status: approved
created: 2026-11-05
updated: 2026-11-05
context: air
approved_by: Temuri
approved_on: 2026-11-05
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The air quality of the National Environmental Agency

## 1. Problem

A person asks "is the air bad in Tbilisi today", "can my child play outside in Rustavi",
or "how was the PM2.5 on Tsereteli Avenue this week". The server has no answer. In winter,
smog covers Tbilisi on many days. The NEA publishes the hourly readings of its stations
on a map at `air.gov.ge`. A person must find the station and read the colors by hand.

## 2. Users

| User                   | Need                                                                 |
| ---------------------- | -------------------------------------------------------------------- |
| A person with a client | The air quality in a city now, and its change over the last days.    |
| The operator           | Turns the NEA off with no change to the code.                        |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| NEA | The data methods of the map at `air.gov.ge/api/`: `get_data_1hour` and `municipalities` | The agency is official: it belongs to the Ministry of Environmental Protection and Agriculture. The methods are unofficial: the NEA publishes no API, and its map calls them. | One reading per station, pollutant, and hour. A reading appears within one to two hours. | No key. No `robots.txt`. No published terms. |

Observed behavior of the source on 2026-11-05:

- One call returns 21 automatic stations, each with its pollutants and readings. A call for
  27 hours of every station is about 236 KB and takes about 0.8 s. 10 calls in a row gave
  no refusal.
- The call takes a start hour, an end hour, a station code or `all`, a municipality id or
  `all`, and a pollutant or `all`. A path without the final `/` gives a redirect.
- A station has a code (`TSRT`), an address in `ka` and in `en`, a municipality id,
  coordinates, and a list of pollutants. The city is the name of the municipality.
  The `en` name of a settlement is often empty.
- The pollutants are PM10, PM2.5, NO2, O3, SO2, and CO. CO is in mg/m³. The others are in μg/m³.
  A station measures two to six of them.
- Each pollutant carries five band limits. The map puts a value in a band when
  `from <= value < to`.
- A reading has a value with up to 13 decimals and a local time with no offset:
  `"2026-11-04T23:00:00"`. Some values are negative: CO `-0.8` in Poti.
- The time of a reading runs ahead of the clock by a fixed offset for each pollutant, at
  every station. At 00:38 Georgian time, NO2 and SO2 ended at 00:00, O3 and CO at 02:00,
  and PM10 and PM2.5 at 06:00. Each one moved forward by one hour at the next hour.
- Station `RST18` in Rustavi returned no reading for 27 hours.

| Pollutant | Good | Fair | Moderate | Poor | Very poor |
| --------- | ---- | ---- | -------- | ---- | --------- |
| PM10      | 0 to 20 | 20 to 35 | 35 to 50 | 50 to 100 | 100 to 1200 |
| PM2.5     | 0 to 10 | 10 to 20 | 20 to 25 | 25 to 50 | 50 to 800 |
| NO2       | 0 to 40 | 40 to 100 | 100 to 200 | 200 to 400 | 400 to 1000 |
| O3        | 0 to 80 | 80 to 120 | 120 to 180 | 180 to 240 | 240 to 600 |
| SO2       | 0 to 100 | 100 to 200 | 200 to 350 | 350 to 500 | 500 to 1250 |
| CO        | 0 to 5 | 5 to 7 | 7 to 10 | 10 to 15 | 15 to 50 |

## 4. Out of scope

- The 136 indicator stations. Each one gives one sample of two weeks, months late.
- A daily average. The daily method of the source gives "not found".
- A forecast.
- A health advice beyond the level that the NEA publishes.
- A reading older than 7 days.
- The meteorological parameters of a station.

## 5. Definitions

The glossary holds the terms of this feature, in the section Air.

## 6. Functional requirements

### `NEAAIR-FR-001`

The server must return the newest reading of each pollutant of each station in a city.

**Examples:**

- Normal case: `Tbilisi`. Five stations, `TSRT` with six pollutants.
- Limit case: `tbilisi` or `თბილისი`. The same five stations.
- Unwanted case: `Sochi`. The server refuses it and names each city with a station.

### `NEAAIR-FR-002`

The server must return the newest reading of each pollutant of one station for a station code.

**Examples:**

- Normal case: `TSRT`.
- Unwanted case: `XXXX`. The server refuses it and tells the model to ask for a city.

### `NEAAIR-FR-003`

The server must return every station with its newest readings when the request names no
city and no station.

**Why:** "Where is the air worst in Georgia" needs every station.

### `NEAAIR-FR-004`

The server must give each station with its code, address, city, coordinates, and station level.

### `NEAAIR-FR-005`

The server must give each reading with its pollutant, value, unit, time, and air quality level.

### `NEAAIR-FR-006`

The server must give a reading the air quality level whose band limits hold the value,
from the lower limit inclusive to the upper limit exclusive.

**Examples:**

- Normal case: PM10 `51.9` is `poor`.
- Limit case: PM10 `50.0` is `poor`. PM10 `1300` is `very_poor`.
- Unwanted case: CO `-0.8`. The reading has no level.

### `NEAAIR-FR-007`

The server must give a station the worst level of its newest readings as the station level.
A station with no level on any reading has no station level.

**Examples:**

- Normal case: `TSRT` with PM10 `poor` and the rest `good`. The station level is `poor`.
- Unwanted case: `RST18` with no reading. No station level.

### `NEAAIR-FR-008`

The server must take a reading as newest only when it is less than 24 hours old at the
receipt of the request. A station with no such reading appears with no reading.

**Why:** A broken station keeps its last reading for days. A model must not report it as current.

### `NEAAIR-FR-009`

The server must order the stations from the worst station level to the best. A station
with no station level comes last.

### `NEAAIR-FR-010`

The server must return the hourly readings of one station over a period, for a station code.

**Examples:**

- Normal case: `TSRT` from 2026-11-01 to 2026-11-04.
- Unwanted case: `XXXX`. The server refuses it and tells the model to ask for a city.

### `NEAAIR-FR-011`

The server must use the last 24 hours as the period when the request names none.

### `NEAAIR-FR-012`

The server must refuse a period longer than 7 days, a period that ends before it starts,
and a period that starts more than 7 days before the receipt of the request.

### `NEAAIR-FR-013`

The server must return the readings of one pollutant only when the request names it.

**Examples:**

- Normal case: `TSRT` and `PM2.5`.
- Limit case: `pm2.5` gives the same readings.
- Unwanted case: `PM1`. The server refuses it and names the six pollutants.

### `NEAAIR-FR-014`

The server must return no more readings for each pollutant than the limit of the request,
the newest first. The default limit is 24. The maximum is 100.

### `NEAAIR-FR-015`

The server must give each address and each city in the language of the request: `ka` or `en`.
The default is `en`. A city name matches in either language.

### `NEAAIR-FR-016`

The server must treat an answer of the NEA that it cannot read as a failure of the NEA.

### `NEAAIR-FR-017`

The server must read the NEA with no key. The operator must be able to turn the NEA off.
It is on by default.

## 7. Non-functional requirements

### `NEAAIR-NFR-001`

A reading must be less than 15 minutes old in the cache of the server, unless the answer
marks it as stale. Measure from the read instant to the receipt of the request.

**Why:** The NEA adds one reading per hour. 15 minutes bounds the delay.

### `NEAAIR-NFR-002`

The server must return stale readings only when the NEA fails, and only when they are
less than 3 hours old in the cache.

### `NEAAIR-NFR-003`

The list of cities must be less than 7 days old. Measure from the read instant to the
receipt of the request.

### `NEAAIR-NFR-004`

The server must send at most 2 calls to the NEA in each second, for any count of requests.
Measure the calls at the source.

### `NEAAIR-NFR-005`

Every call to the NEA must carry a User-Agent that names Georgia MCP and the address of
its repository. Measure at the source.

## 8. Invariants

### `NEAAIR-INV-001`

Every value in an answer is the decimal text of the value that the NEA publishes.

### `NEAAIR-INV-002`

Every time in an answer is the time that the NEA publishes, with the offset `+04:00`.

### `NEAAIR-INV-003`

Every answer names the NEA as the source.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                  |
| -------------------- | ------------- | ------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `air`.                                   |
| `DOM-006`, `MCP-009` | Domain, MCP   | A value is a decimal string, never a float.             |
| `MCP-002`            | MCP tools     | Each tool name starts with `air_`.                      |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.               |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The history takes `limit`.   |
| `CFG-008`            | Configuration | The switch of the NEA is on by default.                 |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale periods of section 7.   |
| `RSL-008`            | Resilience    | The limit of `NEAAIR-NFR-004` is a setting.             |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `NEAAIR` and `air` (2026-11-05). |
| 2   | Which tools? | Temuri | The newest readings of a city or a station, and the history of a station (2026-11-05). |
| 3   | Does the server state a level? | Temuri | Yes, for each reading and for each station (2026-11-05). |
| 4   | What do the PM times ahead of the clock mean? | Temuri | Report them as published (2026-11-05). A second call showed a fixed offset for each pollutant: 0, 2, or 6 hours (section 3). |
| 5   | Does the server report a negative value as published? | Temuri | Yes, with no level (2026-11-05). `NEAAIR-FR-006` holds it. |
| 6   | Does the absence of published terms permit reuse by a server? | Temuri | Yes, with the NEA named as the source (2026-11-05). `NEAAIR-INV-003` names it. |
| 7   | Does the server round a value? | Temuri | No. The value is the decimal text as published (2026-11-05). `NEAAIR-INV-001` holds it. |

## Retired identifiers

This file has no retired identifier.
