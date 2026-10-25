---
id: CAVCINE
title: The cinemas, movies, sessions, and seats of Cavea
type: requirements
status: approved
created: 2026-10-25
updated: 2026-10-25
context: cinema
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The cinemas, movies, sessions, and seats of Cavea

## 1. Problem

A person asks "what is on at the cinema tonight", "where can I see Street Fighter in
English in IMAX", or "are there two free seats together for the 22:15 session". The
server has no answer. Cavea runs the largest network of cinemas in Georgia: 5 cinemas in
Tbilisi and Batumi. Cavea publishes its program on its web site only.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | The movies that Cavea shows now or soon, their sessions, prices, and free seats. |
| The operator           | Turns Cavea off with no change to the code.                                |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Cavea | The web site `cavea.ge`: the JSON methods under `/api/` that its pages call, and the data of the pages `/movies` and `/sessions`. | No. An unofficial API, and a web page for people. | The program: at each change. The seats: at each sale. | No key. No published terms and no published limit. `robots.txt` disallows `/api/` and allows the pages. The owner chose to read `/api/` (question 2). |

Observed behavior of the source on 2026-10-25:

- `/api/theaters` gives 8 cinemas. 5 are open: Tbilisi Mall, East Point, Galleria, City
  Mall Saburtalo (Tbilisi), and Grand Mall (Batumi). A closed cinema has `is_active`
  false. An open cinema can have no `is_active` field.
- `/movies` holds 25 movies now showing, and `/movies?filter=coming-soon` holds 10. Each
  movie has a name in Georgian and in English, genres, a duration in minutes, an age
  rating (`R`, `PG-13`, `PG`), a premiere date, directors, actors, and a description.
- The search `/api/movies/search` matches the English name only, from 2 letters. A text in
  Georgian matches nothing.
- `/sessions` holds every session of the next 3 days: 327 sessions. A session has one
  price in lari, the price of the class Standard.
- `/api/booking/sessions/all/<movie>` gives the sessions of one movie, with a price for
  each seat class: Economy, Standard, VIP, Status. It refuses a call with no `Origin`.
- A session has a hall, an audio language (`geo` or `eng`), an optional subtitle
  language, and the formats 3D, IMAX, Dolby Atmos, ScreenX, and Infinity Vision.
- Cavea writes the start of a session as the time in Tbilisi: `cag_5/27-10-2026/22:15`.
- `/api/sessions/availability` gives a level for each session: `green` (many seats),
  `yellow` (filling up), or another value (almost sold out). It answers for 30 sessions
  at most in one call.
- The seat map needs two calls: a POST gives a ticket for one session, valid for 15
  minutes, and two cookies. The GET with the ticket and the cookies gives each seat
  with its row, number, class, a mark for special needs, and a mark for a taken seat.
  Galleria hall 5 had 91 seats in 6 rows, and 28 were taken.
- The GET of the seat map needs the ticket and the cookie `cavea_vid`. With no ticket, or
  with no cookie, Cavea answers 403. The other cookies are not necessary.
- The probe did not hold a seat, so it did not show whether a seat that another buyer
  holds has the taken mark.
- The source accepts a User-Agent that names the server and its repository.

## 4. Out of scope

- A purchase, a reservation, or a hold of a seat.
- The bar, the gift cards, the offers, and the account of a person.
- The events, the festivals, and the trailers.
- A cinema chain other than Cavea. A later feature can add one to the context.
- The language `ru`.

## 5. Definitions

The glossary holds the terms of this feature, in the section Cinema.

## 6. Functional requirements

### `CAVCINE-FR-001`

The server must return each open cinema of Cavea with its code, name, city, and address.

**Examples:**

- Normal case: 5 cinemas. `east_point` is "Cavea East Point" in Tbilisi.
- Unwanted case: Rustaveli has `is_active` false. The server leaves it out.

### `CAVCINE-FR-002`

The server must return the movies of one movie status. The default is `now_showing`.
Each movie has its code, name, genres, duration, age rating, and premiere date.

**Examples:**

- Normal case: no status gives 25 movies, with `clayface-01` and its English name "Clayface".
- Limit case: `coming_soon` gives 10 movies, with `dune-part-three`.
- Unwanted case: `archived`. The server refuses it and names the two values.

### `CAVCINE-FR-003`

The server must return each movie that Cavea matches to a text of 2 to 100 characters,
with its movie status. It must send no call for a text out of that range.

**Why:** Cavea matches the English name only. The tool description tells the model to
search in English.

**Examples:**

- Normal case: `street` gives `street-fighter` and `the-end-of-oak-street`.
- Limit case: `du` gives `dune-part-three`, a movie coming soon.
- Unwanted case: `a`. The server refuses it.

### `CAVCINE-FR-004`

The server must return the sessions of one date that have not started. The date is in
Tbilisi. The default is today. A movie code, a cinema code, an audio language, and a
format each narrow the list. Each filter is optional.

**Examples:**

- Normal case: `street-fighter`, 2026-10-27, `en`. 4 sessions in 3 cinemas.
- Limit case: no filter on 2026-10-26. 152 sessions of every movie.
- Unwanted case: 2026-10-24, a date before today. The server refuses it.

### `CAVCINE-FR-005`

Each session must carry its code, movie, cinema, hall, start instant, audio language,
subtitle language, and formats.

### `CAVCINE-FR-006`

Each session of a request with a movie code must carry the price of each seat class.
Each session of a request with no movie code must carry the price of the class Standard.

**Why:** Cavea gives the price of each class for one movie only. The owner chose the page
of all sessions for a request with no movie (question 4).

**Examples:**

- Normal case: `street-fighter` at Galleria, 22:15. Economy 20, Standard 20, VIP 29,
  Status 22.
- Limit case: no movie. The same session carries Standard 20.

### `CAVCINE-FR-007`

Each session must carry its occupancy level. When Cavea does not give the level, the
server returns the sessions with no level, and the answer says so.

**Why:** A list of sessions with no level still answers "when". A failure would hide it.

### `CAVCINE-FR-008`

The server must return the sessions in the order of the start instant, then the cinema
code. It must return no more than the limit of the request. The default limit is 30. The
maximum is 100. It must state the count of sessions that matched.

### `CAVCINE-FR-009`

The server must return the seat map of one session that has not started. The map gives
each seat with its row, number, class, a mark for special needs, and a mark for a taken
seat. It gives the count of free seats of each class and the price of each class.

**Examples:**

- Normal case: `cag_5/27-10-2026/22:15`. 91 seats, 63 free: Standard 38 of 41, Status 25
  of 34, VIP 0 of 16.
- Unwanted case: `cag_5/24-10-2026/22:15`. The session is over. The server refuses it.
- Unwanted case: `xyz`. The server refuses it and names the form of a session code.

### `CAVCINE-FR-010`

The server must refuse a movie code, a cinema code, or a session code that Cavea does not
publish. The message names the tool that lists the codes.

### `CAVCINE-FR-011`

The server must give each name, genre, and description in the language of the request:
`ka` or `en`. The default is `en`.

**Examples:**

- Normal case: `en` gives "Cavea East Point" and the genre "Horror".
- Limit case: `ka` gives "კავეა ისთ ფოინთი" and "საშინელებათა".

### `CAVCINE-FR-012`

The server must treat an answer of Cavea that it cannot read as a failure of Cavea.

**Why:** Cavea can change its site with no notice. A wrong free seat is worse than no seat map.

**Examples:**

- Unwanted case: the page `/sessions` has no list of sessions.
- Unwanted case: a session code that does not give a date and a time.
- Unwanted case: a seat class other than the four of section 5.

### `CAVCINE-FR-013`

The server must read Cavea with no key. The operator must be able to turn Cavea off. It
is on by default.

### `CAVCINE-FR-014`

The server must leave out a session whose audio language or subtitle language is not
Georgian or English. The other sessions of the answer stay.

**Why:** Cavea can add a session in Russian. The language `ru` is out of scope, and one
such session must not fail the whole answer.

**Examples:**

- Normal case: 31 sessions, each `geo` or `eng`. The answer holds 31.
- Unwanted case: 31 sessions, one with the audio language `rus`. The answer holds 30,
  and the total counts 30.

## 7. Non-functional requirements

### `CAVCINE-NFR-001`

A list of cinemas, a list of movies, a search result, or a list of sessions must be less
than 15 minutes old, unless the answer marks it as stale. Measure from the read instant
to the receipt of the request.

### `CAVCINE-NFR-002`

The server must return a stale list only when Cavea fails, and only when the list is less
than 6 hours old. Measure from the read instant to the receipt of the request.

### `CAVCINE-NFR-003`

An occupancy level and a seat map must come from a call to Cavea that the server sends
after the receipt of the request. The server never returns a stale seat map.

**Why:** A seat is sold at any moment. An old seat map sends a person to a taken seat.

### `CAVCINE-NFR-004`

A seat map must send exactly 2 calls to Cavea, when the server holds the list of sessions
of that movie. Measure the calls at the source.

### `CAVCINE-NFR-005`

The server must send at most 4 calls to Cavea in each second, for any count of requests.
Measure the calls at the source.

**Why:** Cavea publishes no limit. A block of the server stops every person.

### `CAVCINE-NFR-006`

Every call to Cavea must carry a User-Agent that names Georgia MCP and the address of its
repository. Measure at the source.

## 8. Invariants

### `CAVCINE-INV-001`

Every price in an answer is a decimal amount in lari, greater than zero, as text.

### `CAVCINE-INV-002`

Every start instant in an answer carries the offset of Tbilisi, `+04:00`.

### `CAVCINE-INV-003`

The free seats of a class in a seat map equal the seats of that class with no taken mark.

## 9. Constraints from the guidelines

| Rule                   | Guideline     | Effect on this feature                                       |
| ---------------------- | ------------- | ------------------------------------------------------------ |
| `DOM-001`              | Domain        | The context is `cinema`.                                     |
| `DOM-006`, `MCP-009`   | Domain        | A price is a decimal string.                                 |
| `DOM-007`              | Domain        | A date and a start instant are in `Asia/Tbilisi`.            |
| `MCP-002`              | MCP tools     | Each tool name starts with `cinema_`.                        |
| `MCP-004`              | MCP tools     | Each tool is read-only. The seat map reads, and holds no seat. |
| `MCP-005`              | MCP tools     | Each answer names Cavea and the instant of the read.         |
| `MCP-007`, `MCP-008`   | MCP tools     | The tools take `language`, and a list takes `limit`.         |
| `CLT-003`, `CLT-009`   | Clients       | An unofficial API and a data stream of a page. The client says so. |
| `CLT-008`              | Clients       | The client keeps no ticket and no cookie between two seat maps. |
| `CFG-008`              | Configuration | The switch of Cavea is on by default.                        |
| `RSL-004`, `RSL-006`   | Resilience    | The time to live is 15 minutes. A list stays stale to 6 hours. |
| `RSL-008`              | Resilience    | The limit of `CAVCINE-NFR-005` is a setting.                 |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What are the key and the context? | Temuri | `CAVCINE` and `cinema` (2026-10-25). |
| 2   | `robots.txt` disallows `/api/`. Which source? | Temuri | `/api/` everywhere (2026-10-25). |
| 3   | Which price does a session carry? | Temuri | The seat map gives the price of each class (2026-10-25). |
| 4   | `/api/` has no list of all sessions, and no list of movies. | Temuri | Read the page when `/api/` does not give the data (2026-10-25). |
| 5   | How old can a value be? | Temuri | 15 minutes, 6 hours when Cavea fails. The seat map is live (2026-10-25). |

## Retired identifiers

This file has no retired identifier.
