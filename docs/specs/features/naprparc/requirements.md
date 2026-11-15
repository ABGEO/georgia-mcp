---
id: NAPRPARC
title: The land parcels of the NAPR cadastral map
type: requirements
status: approved
created: 2026-11-15
updated: 2026-11-15
context: parcels
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [DOM, MCP, RSL, CLT, CFG, TST]
---

# Requirements: The land parcels of the NAPR cadastral map

## 1. Problem

A person asks "who owns the parcel 01.15.05.066.020", "how large is the plot at these
coordinates", or "is the land at Chavchavadze Avenue 44 agricultural". The server has no
answer. The NAPR map shows each parcel of Georgia, but a person must find it on a map,
click it, and read a panel in Georgian.

## 2. Users

| User                   | Need                                                                         |
| ---------------------- | ---------------------------------------------------------------------------- |
| A person with a client | Find a parcel by its cadastral code, by a point, or by an address. Read its area, land type, owners, and registry documents. |
| The operator           | Turns NAPR off with no change to the code.                                    |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| NAPR | The methods of `maps.gov.ge`: `POST /map/portal/search`, `GET /lr/bo/mg/getinfo2`, `GET /lr/bo/mg/getinfo.alpha`, `GET /ar/mg/getinfo` | Yes. The public registry of Georgia. The methods serve the map page and have no documentation. | At each registration. A sale changes the owners on the day of the registration. | No key. No published terms found. `robots.txt` gives "Access Denied". |

Observed behavior of the source on 2026-11-09 and on 2026-11-15:

- A search for a cadastral code gives one parcel, or an empty list for an unknown code.
  A code of 4 groups (`53.20.41.012`) and of 5 groups (`01.15.05.066.020`) both exist.
  A group can have 4 digits: `05.31.07.1000`. A partial code gives no parcel.
- A search for a code of an apartment (`01.14.11.005.029.01.501`) gives no result.
- The details of a parcel: the area and its unit (`კვ.მ.`), the address, the land type,
  the ownership type, the owners, and the documents. The answer is the same for `lang=ka`
  and `lang=en`. Every text is Georgian.
- Land types seen: `სასოფლო-სამეურნეო`, `არასასოფლო სამეურნეო`. Ownership types seen:
  `საკუთრება`, `თანასაკუთრება`.
- An owner is a person, a company, a partnership (`ამხანაგობა "ჭავჭავაძე 46"`), or a
  public body. A co-owned parcel can list no owner.
- A document has a kind (`ამონაწერი საჯარო რეესტრიდან`, `საკადასტრო გეგმა`), a registration
  number, a date or no date, and a link to a PDF that opens with no login. One parcel had 13
  documents, and two of them repeated the same number and date.
- A point gives the parcels within about 800 m, nearest first, with the distance in metres
  and the polygon of each: 15 to 47 parcels, up to 165 KB. A point outside Georgia gives
  an empty list.
- The answer for a point is not valid JSON when an address holds a quotation mark:
  `"ცეკავშირის" მიმდებარედ`.
- An address search gives addresses, not parcels: at most 10 for a Georgian text, 50 for a
  Latin text. Each has a municipality and a polygon. The order is poor. `ჭავჭავაძის გამზირი
  44` puts Gori and Telavi before Tbilisi. `თბილისი, ჭავჭავაძის 44` gives no result.
- Each call answered in 0.1 to 2 seconds. Ten calls in one second all answered.

## 4. Out of scope

- A building, an apartment, or any unit under a parcel.
- The polygon of a parcel.
- The content of a document. The server gives the link, not the PDF.
- A history of owners, a mortgage, or an encumbrance.
- A search by the name of an owner.
- A map layer or an image of a map.

## 5. Definitions

The glossary holds the terms of this feature, in the section Parcels.

| Term | Meaning |
| ---- | ------- |
| NAPR | The National Agency of Public Registry. It keeps the cadastre of Georgia. Vendor code `napr`. |
| parcel | One plot of land in the cadastre, with a cadastral code. |
| cadastral code | The code of a parcel: 4 or 5 groups of digits with dots, `01.15.05.066.020`. |
| land type | `agricultural` or `non_agricultural`. |
| ownership type | `ownership` (one owner or more, each in full) or `co_ownership` (shared). |
| owner | A person or a body that the registry names as the holder of a parcel. |
| registry document | A document of the registry about a parcel: a kind, a registration number, a date, and a link. |
| document kind | `registry_extract` or `cadastral_plan`. |
| address | In `parcels`, the text of NAPR that names a settlement, a street, and a number. |
| municipality | The unit under which NAPR groups an address: `ქალაქი თბილისი`, `გორის მუნიციპალიტეტი`. |

## 6. Functional requirements

### `NAPRPARC-FR-001`

The server must return a parcel for a cadastral code, with its code, address, area, land
type, ownership type, owners, registry documents, and centre point.

**Examples:**

- Normal case: `01.15.05.066.020`. Freedom Square, 618 m², `non_agricultural`,
  `ownership`, owner `ქალაქ თბილისის თვითმმართველი ერთეული`, 2 documents.
- Limit case: `05.31.07.1000`. A code of 4 groups with a group of 4 digits.
- Unwanted case: `99.99.99.999.999`. The server states that NAPR has no such parcel and
  tells the model to search by a point or an address.

### `NAPRPARC-FR-002`

The server must refuse a cadastral code that is not 4 or 5 groups of digits with dots.
For a code of more than 5 groups, it must state that it covers parcels only and name the
first 5 groups as the code of the parcel.

**Examples:**

- Unwanted case: `01.14.11.005.029.01.501`. The server names `01.14.11.005.029`.
- Unwanted case: `01.15.05`. The server names the form `01.15.05.066.020`.

### `NAPRPARC-FR-003`

The server must return the parcels nearest a point, nearest first, with the cadastral code,
the address, and the distance in whole metres of each. The default count is 5. The
maximum is 20.

**Why:** The source gives up to 47 parcels with polygons. The model asks for the details
of one parcel with `NAPRPARC-FR-001`.

**Examples:**

- Normal case: 41.6934, 44.8015. `01.15.05.066.020` first, at 0 m.
- Limit case: the point is on a street. The first parcel has a distance above 0.
- Unwanted case: a point in Georgia with no parcel within the reach of the source. An
  empty list, and no failure.

### `NAPRPARC-FR-004`

The server must refuse a point outside the latitudes 41.0 to 43.6 and the longitudes
40.0 to 46.8. The message names the order: latitude, then longitude.

**Why:** A model that swaps the two numbers gives a point outside Georgia.

**Examples:**

- Unwanted case: 44.8015, 41.6934 (swapped).

### `NAPRPARC-FR-005`

The server must return the addresses that match a text, with the address, the
municipality, and the centre point of each, in the order of the source. The default count
is 5. The maximum is 20.

**Why:** The source finds addresses, not parcels. The model passes a centre point to
`NAPRPARC-FR-003`.

**Examples:**

- Normal case: `ჭავჭავაძის გამზირი 44` with the municipality `თბილისი`. The address in Tbilisi.
- Limit case: `Chavchavadze 44`. A Latin text, with matches across Georgia.
- Unwanted case: `zzqxv`. An empty list, and no failure.

### `NAPRPARC-FR-006`

The server must keep only the addresses whose municipality holds a text, when the request
gives one.

**Why:** The source order puts other cities first.

### `NAPRPARC-FR-007`

The server must give the land type, the ownership type, and the document kind as codes.
An unknown value from the source comes as `other`, with the text of the source.

### `NAPRPARC-FR-008`

The server must give each address, owner, and municipality as NAPR publishes it, in
Georgian, in every language of the request. It must state that NAPR publishes no
translation.

### `NAPRPARC-FR-009`

The server must give the area in square metres. It must give the area and the unit of the
source when the unit is not square metres.

### `NAPRPARC-FR-010`

The server must order the registry documents of a parcel by kind, then by date with the
newest first. A document with no date comes last in its kind. Two documents with the
same link are one document.

### `NAPRPARC-FR-011`

The server must return a parcel with an empty list of owners when NAPR names no owner.

**Examples:**

- Normal case: `01.15.05.066.015`, an underground passage, `co_ownership`, no owner.

### `NAPRPARC-FR-012`

The server must read the parcels at a point when an address holds a quotation mark that
the source does not escape.

**Examples:**

- Normal case: 41.6934, 44.8015. The list holds `01.15.04.028.003`.

### `NAPRPARC-FR-013`

The server must treat an empty detail of a parcel that the search found as a failure of
NAPR.

### `NAPRPARC-FR-014`

The server must read NAPR with no key. The operator must be able to turn it off. It is on
by default.

## 7. Non-functional requirements

### `NAPRPARC-NFR-001`

A parcel must be less than 1 hour old in the cache, unless the answer marks it as stale.
A stale parcel must be less than 7 days old. The parcels at a point must be less than
24 hours old. An address search must be less than 7 days old.

**Why:** A sale changes the owners on the day of the registration. A boundary and an
address change rarely.

### `NAPRPARC-NFR-002`

A parcel must cost at most 3 calls to NAPR. A point must cost 1 call. An address search
must cost at most 1 call, and 1 call for each address in the answer. Measure the calls at
the source.

### `NAPRPARC-NFR-003`

The server must send at most 10 calls to NAPR in each second, for any count of requests.
Measure the calls at the source.

**Why:** An address search of 20 addresses sends 21 calls. A cap of 1 call in each second
makes it wait 20 seconds.

### `NAPRPARC-NFR-004`

A tool call must answer in less than 10 seconds when NAPR answers each call in less than
2 seconds. Measure from the receipt of the request to the last byte.

### `NAPRPARC-NFR-005`

Every call to NAPR must carry a User-Agent that names Georgia MCP and the address of its
repository. Measure at the source.

## 8. Invariants

### `NAPRPARC-INV-001`

Every address, owner, and registry document in an answer is the one that NAPR publishes,
with no change of words.

### `NAPRPARC-INV-002`

Every answer of a parcel names its cadastral code and the address of the NAPR map.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                            |
| -------------------- | ------------- | ----------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `parcels`.                                         |
| `MCP-002`            | MCP tools     | Each tool name starts with `parcels_`.                            |
| `MCP-005`            | MCP tools     | Each answer carries `source` and `as_of`.                         |
| `MCP-007`, `MCP-008` | MCP tools     | The tools take `language`. The point and address searches take `limit`. |
| `CLT-007`            | Clients       | An answer that is not valid JSON fails, except `NAPRPARC-FR-012`. |
| `CFG-008`            | Configuration | The switch of NAPR is on by default.                              |
| `RSL-004`, `RSL-006` | Resilience    | The times to live and the stale period of `NAPRPARC-NFR-001`.     |
| `RSL-008`            | Resilience    | The limit of `NAPRPARC-NFR-003` is a setting.                     |
| `TST-006`            | Testing       | A recorded answer replaces the name of each private owner.        |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `NAPRPARC` and `parcels` (2026-11-15). |
| 2   | Do the owners and the documents stay in the answer? | Temuri | Yes. The registry is public data (2026-11-15). |
| 3   | How does a search treat an address? | Temuri | It returns addresses with a centre point. The model then asks for a point (2026-11-15). |
| 4   | What does a point return? | Temuri | The nearest parcels, 5 by default, 20 at most, with no polygon (2026-11-15). |
| 5   | Does a parcel carry its shape? | Temuri | The centre point only (2026-11-15). |
| 6   | Which documents does a parcel carry? | Temuri | All of them, newest first in each kind (2026-11-15). |
| 7   | Are the times to live of `NAPRPARC-NFR-001` and the cap of `NAPRPARC-NFR-003` correct? | Temuri | Yes: 1 hour, 7 days stale, and 10 calls in each second with 20 addresses at most (2026-11-15). |
| 8   | Is the link to a registry document stable over days? Stage 2 checks it. | Temuri | Stage 2 checks it and records the result (2026-11-15). |
| 9   | The centre point of a parcel needs a third call. Does `NAPRPARC-NFR-002` allow it? | Temuri | Yes. 3 calls for a parcel (2026-11-15). |

## Retired identifiers

This file has no retired identifier.
