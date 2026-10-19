---
id: TTCTRANS
title: The public transport of Tbilisi
type: requirements
status: approved
created: 2026-10-19
updated: 2026-10-19
context: transit
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The public transport of Tbilisi

## 1. Problem

A person asks "when does the next bus come to my stop", "how do I get from Vake Park to
Rustaveli", or "when does the 385 leave on Saturday". The server has no answer. The TTC
publishes stops, routes, arrivals, timetables, and trip plans in its web application only.

## 2. Users

| User                   | Need                                                                     |
| ---------------------- | ------------------------------------------------------------------------ |
| A person with a client | A stop, the next arrivals at it, a route, a timetable, and a trip plan.  |
| The operator           | Sets the key of the TTC. Turns the TTC off with no change to the code.   |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| TTC | `transit.ttc.com.ge/pis-gateway/api` | No. An unofficial API. The web application `transit.ttc.com.ge` reads it. | Stops and routes: rarely. Timetables: each week. Arrivals and vehicles: each few seconds. | A key in a header. The web page publishes the key, and the TTC can change it. No published terms and no published limit. `robots.txt` does not exist. |

Observed behavior of the source on 2026-10-19:

- The source answers 400 to a call with no key. Each call takes the language `ka` or
  `en`. Some names of minibus routes and stops stay in Georgian with `en`.
- 2,758 stops: 2,729 of bus, 23 of metro, 6 of cable car. A stop has an identifier
  (`1:1000`), a name, and coordinates. Only a bus stop has a code (`1000`). 787 names
  repeat, for example a stop on each side of a street.
- 282 routes: 277 of bus, 2 of metro, 3 of cable car. A route has an identifier, a
  number, a name, and a color. The numbers `1` and `2` each name one metro line and one
  cable car.
- The source gives a minibus route the mode of a bus. 166 routes have an identifier that
  starts with `1:minibus` and the color `0033B4`. A route has one or two directions.
- An arrival gives a route number, a headsign, a flag of live tracking, a count of
  minutes from live tracking, and a count of minutes from the timetable. The count from
  the timetable is often wrong: `-131` for an arrival that live tracking gives in 3
  minutes. The metro and the cable car have no live tracking.
- An unknown stop gives an empty list of arrivals. An unknown route gives 500.
- A timetable covers the 7 days of the current week. It groups the days with the same
  times: Monday to Friday, then Saturday and Sunday. A time has the form `6:45`.
- A trip plan takes two pairs of coordinates and gives up to 3 itineraries. The source
  gives each instant in UTC. A point outside Tbilisi gives 400. The route of a leg has a
  number and no identifier.
- A place search takes a text and gives places from OpenStreetMap in Tbilisi.

## 4. Out of scope

- The live position of a vehicle. A later feature can add it.
- The shape of a route on a map, the interchanges of a route, and map tiles.
- An address from coordinates.
- The steps of a walk, one turn at a time. A leg of a walk gives its distance and time.
- The language `ru`.
- A fare, a ticket, or a travel card.
- A stop or a route outside Tbilisi.

## 5. Definitions

The glossary holds the terms of this feature, in the section Transit.

## 6. Functional requirements

### `TTCTRANS-FR-001`

The server must return the stops whose name or stop code matches a text.

**Why:** A person names a stop as the sign shows it, in Georgian or in English.

**Examples:**

- Normal case: `godziashvili` in `en`. The answer holds "Vaso Godziashvili Park", `1:1000`.
- Limit case: `1000`. The answer holds the stop with the stop code `1000`.
- Limit case: `ვაკის პარკი` in `en`. The answer holds each "Vake Park", in English.
- Unwanted case: an empty text. The server refuses it.

### `TTCTRANS-FR-002`

The server must return the stops within a distance of a point, nearest first, each with
its distance in metres.

**Examples:**

- Normal case: 41.7251, 44.7554 and 300 m. The first stop is `1:1000` at 3 m.
- Limit case: no distance in the request. The server uses 500 m.
- Unwanted case: a distance above 2,000 m, or a point outside Tbilisi. The server refuses it.

### `TTCTRANS-FR-003`

The server must limit a search of stops to one kind of stop when the request names one.

**Examples:**

- Normal case: `isani` and `metro`. The answer holds the metro station only.
- Unwanted case: the kind `minibus`. The server refuses it, and lists the kinds of a stop.

### `TTCTRANS-FR-004`

The server must return the next arrivals at a stop, soonest first. It must accept the
stop as an identifier or as a stop code.

**Examples:**

- Normal case: `1000`. The first arrival is the `385` to "Vake - Saburtalo" in 2 minutes.
- Limit case: `1:gondola_1`, a station with no stop code. The server accepts it.
- Unwanted case: `999999`. The server refuses it, and names the tool that searches stops.

### `TTCTRANS-FR-005`

The server must give each arrival its route number, its kind, its headsign, its minutes
from now, and its expected instant.

### `TTCTRANS-FR-006`

The server must mark each arrival as live or as from the timetable. It must use the
minutes from live tracking for a live arrival, and the minutes from the timetable for
another arrival.

**Why:** The minutes from the timetable of a live arrival are often wrong.

**Examples:**

- Normal case: live, 3 minutes from tracking, -131 from the timetable. The arrival is
  live, in 3 minutes.
- Limit case: the metro, not live, 3 minutes. The arrival is from the timetable, in 3 minutes.
- Unwanted case: not live, -5 minutes. The server leaves the arrival out.

### `TTCTRANS-FR-007`

The server must limit the arrivals to one route when the request names one.

**Examples:**

- Normal case: stop `1000`, route `385`. The answer holds the arrivals of the `385` only.
- Unwanted case: a route that does not stop there. The answer is empty, and says so.

### `TTCTRANS-FR-008`

The server must return the routes whose route number or name matches a text, or every
route when the request has no text.

**Examples:**

- Normal case: `385`. The answer holds the bus "Vake-Saburtalo".
- Limit case: `1`. The answer holds the metro line `1` and the cable car `1`.

### `TTCTRANS-FR-009`

The server must give a minibus route the kind `minibus`, not `bus`.

**Why:** A person in Tbilisi calls a minibus a "marshrutka", not a bus.

**Examples:**

- Normal case: `472`. The kind is `minibus`.
- Limit case: `385`. The kind is `bus`.

### `TTCTRANS-FR-010`

The server must return a route with each of its directions: the headsign and the ordered
stops. It must accept the route as an identifier or as a route number.

**Examples:**

- Normal case: `385`. One direction, "Vake - Saburtalo", from "Tandzia Street", 22 stops.
- Limit case: the metro line `1` in two directions, each with 16 stations.
- Unwanted case: `1` with no kind. The server refuses it, and lists the two routes.

### `TTCTRANS-FR-011`

The server must return the times at which the vehicles of one direction of a route leave
one stop, on one date.

**Why:** The full timetable of a route holds thousands of times. One stop answers the question.

**Examples:**

- Normal case: `385`, stop `1:1001`, 2026-10-19. The times start with `06:45`, `06:55`.
- Limit case: no date. The server uses the current date in Tbilisi.
- Unwanted case: a stop that is not on the direction. The server refuses it.

### `TTCTRANS-FR-012`

The server must refuse a date of a timetable outside the week that the TTC publishes. The
message must give the first and the last date of that week.

**Examples:**

- Unwanted case: 2026-10-26. The server refuses it, and gives 2026-10-19 to 2026-10-25.

### `TTCTRANS-FR-013`

The server must return up to 3 itineraries between two points, each given as coordinates.

**Examples:**

- Normal case: 41.7251, 44.7554 to 41.7151, 44.8271. 3 itineraries, the first of 61 minutes.
- Limit case: the TTC finds no ride. The answer holds an itinerary of a walk only.
- Unwanted case: a point outside Tbilisi. The server refuses it.

### `TTCTRANS-FR-014`

The server must give each itinerary its start and end instants, its duration in minutes,
its distance of walk in metres, its count of transfers, and its legs.

### `TTCTRANS-FR-015`

The server must give each leg its kind or `walk`, its start and end instants, its stop of
start and its stop of end, and its distance in metres. A ride must also give its route
number, its route name, and its count of stops.

**Examples:**

- Normal case: a ride on the `310` from "Vaso Godziashvili Park" to "Mikheil Javakhishvili
  Street", 21:42 to 21:59, 4,913 m.
- Limit case: a walk of 3 m from the start point to the first stop.

### `TTCTRANS-FR-016`

The server must plan a trip to leave now, to leave at a time, or to arrive by a time.
The default is to leave now.

**Examples:**

- Normal case: arrive by 2026-10-20 09:00. The last leg ends at 09:00 or before.
- Limit case: a time with no date. The server uses the current date in Tbilisi.

### `TTCTRANS-FR-017`

The server must plan a trip with only the kinds that the request names. A walk is always
part of a trip.

**Examples:**

- Normal case: `metro` and `bus`. No leg rides a cable car.
- Unwanted case: `minibus` with no `bus`. The server refuses it: the TTC plans both
  together. The message lists the kinds that a trip plan accepts.

### `TTCTRANS-FR-018`

The server must plan a trip for one preference: the shortest time, the least walk, or the
fewest transfers. The default is the shortest time.

### `TTCTRANS-FR-019`

The server must return the places in Tbilisi that match a text, each with its name, its
street, its house number, its district, and its coordinates.

**Examples:**

- Normal case: `Rustaveli`. The answer holds the metro station and the theatre.
- Limit case: a text with no match. The answer is empty, and says so.

### `TTCTRANS-FR-020`

The server must give each name in the requested language. If the TTC has no name in that
language, the server must give the name that the TTC has.

### `TTCTRANS-FR-021`

The operator must be able to turn the TTC off. It is on by default. When the TTC is off,
the server must offer no tool of transport.

### `TTCTRANS-FR-022`

The server must read the key of the TTC from its configuration. If the TTC is on and the
key is empty, the server must not start, and must name the missing setting.

### `TTCTRANS-FR-023`

The server must log an error that names the setting of the key when the TTC rejects the
key.

## 7. Non-functional requirements

### `TTCTRANS-NFR-001`

A list of stops or of routes must be less than 24 hours old, unless the answer marks it
as stale. Measure from the read instant to the receipt of the request.

### `TTCTRANS-NFR-002`

An arrival must be less than 30 seconds old. Measure from the read instant to the receipt
of the request.

### `TTCTRANS-NFR-003`

A search of stops, of nearby stops, or of routes must send no call to the TTC when the
list is less than 24 hours old. Measure the calls at the source.

**Why:** One list of stops is 325 KB. A search per request must not read it.

### `TTCTRANS-NFR-004`

The server must send at most 5 calls to the TTC in each second, for any count of
requests. Measure the calls at the source.

**Why:** The TTC publishes no limit. A block of the server stops every person.

## 8. Invariants

### `TTCTRANS-INV-001`

Every stop in an answer carries its identifier. Every tool that takes a stop accepts that
identifier.

### `TTCTRANS-INV-002`

Every route in an answer carries its route number and its kind.

### `TTCTRANS-INV-003`

Every collection in an answer has no more items than the limit of the request.

| Collection     | Default | Maximum |
| -------------- | ------- | ------- |
| Stops          | 10      | 50      |
| Arrivals       | 10      | 30      |
| Routes         | 20      | 100     |
| Places         | 5       | 20      |

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                                 |
| -------------------- | ------------- | ---------------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `transit`.                                              |
| `DOM-005`            | Domain        | The kind `minibus`, the zone of an instant, and the choice of minutes belong to the server. |
| `DOM-007`, `MCP-009` | Domain        | An instant carries the offset of Tbilisi. A time of a timetable is `HH:MM`. |
| `MCP-002`            | MCP tools     | Each tool name starts with `transit_`.                                 |
| `MCP-005`            | MCP tools     | Each answer names the TTC and the instant at which the server read it. |
| `MCP-007`, `MCP-008` | MCP tools     | Each tool takes `language`. Each collection takes `limit`.             |
| `CLT-009`, `CLT-011` | Clients       | The client holds no key, says that the API is unofficial, and stays at `v0`. |
| `CFG-003`, `CFG-005` | Configuration | The settings have the prefix `GEOMCP_TRANSIT_`. The start fails on a missing key. |
| `RSL-004`, `RSL-006` | Resilience    | Stops and routes live 24 hours in the cache. Arrivals and trip plans have no stale value. |
| `RSL-008`            | Resilience    | The limit of `TTCTRANS-NFR-004` is a setting.                          |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What happens when the TTC is on and no key is set? | Temuri | The server does not start (2026-10-19). |
| 2   | What does a client see when the TTC is off? | Temuri | No tool of transport (2026-10-19). |
| 3   | Does an arrival from the timetable appear? | Temuri | Yes, marked as not live (2026-10-19). |
| 4   | How does an arrival give its time? | Temuri | Minutes from now and an instant (2026-10-19). |
| 5   | Is a minibus a kind of its own? | Temuri | Yes (2026-10-19). |

## Retired identifiers

This file has no retired identifier.
