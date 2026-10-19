---
id: TTCTRANS
title: The public transport of Tbilisi, the scenarios
type: spec
status: approved
created: 2026-10-19
updated: 2026-10-19
context: transit
approved_by: Temuri
approved_on: 2026-10-19
constrained_by: [TST]
requirements: features/ttctrans/requirements.md
---

# Specification: The public transport of Tbilisi, the scenarios

`pkg/ttc/testdata/` holds real answers of the TTC, recorded on 2026-10-19, cut to the
items that a scenario names:

| File                       | Content                                                                        |
| -------------------------- | ------------------------------------------------------------------------------ |
| `stops-en.json`, `stops-ka.json` | `GET /v2/stops`: the stops of `pattern-stops-385.json`, `1:1000`, `1:1054`, `1:1055`, the stop ends of `plan.json`, the 16 stations of the metro line `1`, `1:gondola_1`, each "M/S Isani", and each "Vake Park". |
| `routes-en.json`, `routes-ka.json` | `GET /v3/routes`: `385`, `310`, `472`, `532`, the metro lines `1` and `2`, the cable cars `1` and `2`. |
| `route-385.json`, `route-metro-1-en.json`, `route-metro-1-ka.json` | `GET /v3/routes/{id}`. |
| `pattern-stops-385.json`   | `GET /v3/routes/1:R29925/stops?patternSuffix=1:01`: 22 stops.                  |
| `schedule-385.json`        | `GET /v3/routes/1:R29925/schedule?patternSuffix=1:01`: three groups, cut to the first 4 stops and the last stop. |
| `schedule-metro-1.json`    | The schedule of direction 1 of the metro line `1`. `1:metro_1_10` ends with `24:10`. |
| `arrival-times.json`       | `GET /v2/stops/1:1000/arrival-times`: 12 arrivals, `472` live in 5 with `-119`. |
| `arrival-times-385.json`   | The same with `routeId=1:R29925&maxNumberOfArrivalTimes=5`.                    |
| `arrival-times-metro.json` | `GET /v2/stops/1:metro_1_1/arrival-times`: not live, 3 minutes.                |
| `plan.json`                | `GET /v2/plan` from 41.7251,44.7554 to 41.7151,44.8271: 3 itineraries.         |
| `geocode.json`             | `GET /v2/geocode?query=Rustaveli`.                                              |

A test writes any other case as a copy of one of these files with the change that the
scenario names. An integration test serves the files from `httptest` and calls the tool
through the in-memory MCP transport. `T` is `internal/app/georgia-mcp/transit`.

## The client

| ID | Verifies | Layer | Given, When, Then |
| -- | -------- | ----- | ----------------- |
| `TTCTRANS-SC-001` | `TTCTRANS-FR-001` | unit | `Stops(ctx, "en")` against `stops-en.json`. The path is `/v2/stops`, `locale=en`, `X-api-key` holds the key of `WithAPIKey`, `Accept` is `application/json`, the User-Agent starts with `georgia-mcp-ttc/`. `1:1000` has the code `1000`. `1:gondola_1` has a nil code. |
| `TTCTRANS-SC-002` | `TTCTRANS-FR-008` | unit | `Routes(ctx, "en")`. The query holds `modes=BUS,SUBWAY,GONDOLA`. `1:minibusR25521` has the number `472` and the mode `BUS`. |
| `TTCTRANS-SC-003` | `TTCTRANS-FR-010`, `TTCTRANS-FR-011` | unit | `Route`, `PatternStops`, and `Schedule` with `1:R29925` and `1:01`. The raw paths are `/v3/routes/1:R29925`, `/stops`, `/schedule`, with `patternSuffix=1:01`. |
| `TTCTRANS-SC-004` | `TTCTRANS-FR-007` | unit | `ArrivalTimes` with no route sends `ignoreScheduledArrivalTimes=false` and no `routeId`. With `1:R29925` and 5, it sends `routeId=1:R29925` and `maxNumberOfArrivalTimes=5`. |
| `TTCTRANS-SC-005` | `TTCTRANS-FR-016` | unit | `Plan` with `departAt`, `2026-10-20`, `09:00`. The query holds `fromPlace=41.7251,44.7554`, `departMode=departAt`, `date=2026-10-20`, `time=09:00`, `modes=WALK,BUS`. `plan.json` decodes 3 itineraries. |
| `TTCTRANS-SC-006` | `TTCTRANS-FR-019` | unit | `Geocode` with `Rustaveli`. The query holds `query`, `locale`, and `bbox`. The first feature has the coordinates `[44.7896467, 41.703497]`. |
| `TTCTRANS-SC-007` | `TTCTRANS-FR-023` | unit | The server answers 401 with `Unauthorized`. The error is an `*APIError` with 401, and `errors.Is(err, ErrUnexpectedStatus)` holds. |
| `TTCTRANS-SC-008` | `TTCTRANS-FR-001` | live | The key in `GEOMCP_TRANSIT_TTC_API_KEY`. `Stops(ctx, "en")` returns more than 2,000 stops, and `1:1000` is "Vaso Godziashvili Park". The test skips with no key. |

## The domain

| ID | Verifies | Layer | Given, When, Then |
| -- | -------- | ----- | ----------------- |
| `TTCTRANS-SC-009` | `TTCTRANS-FR-003` | unit | `ParseStopKind` accepts `bus`, `metro`, `cable_car`, and refuses `minibus`, `BUS`, `walk`, and the empty text with `ErrInvalidKind`. `ParseRouteKind` accepts `minibus`. A fuzz test finds no panic. |
| `TTCTRANS-SC-010` | `TTCTRANS-FR-017` | unit | `ParseTripKinds`: none gives every kind. `bus, minibus, metro` passes. `minibus` alone and `bus, metro` give `ErrInvalidTripKinds`. `bus, bus, minibus` gives `bus, minibus`. |
| `TTCTRANS-SC-011` | `TTCTRANS-FR-002` | unit | `NewPoint(95, 44.8)` and NaN give `ErrInvalidPoint`. `NewPoint(40, 40).InTbilisi()` is false. The distance from 41.7251, 44.7554 to `1:1000` rounds to 3 m. |
| `TTCTRANS-SC-012` | `TTCTRANS-FR-011` | unit | `ParseClockTime`: `6:45` gives `06:45`, `24:05` gives `24:05`, `48:00`, `6:60`, and `645` fail with `ErrInvalidClockTime`. A fuzz test: each accepted text formats to `HH:MM` and parses back. |
| `TTCTRANS-SC-013` | `TTCTRANS-FR-020` | unit | `Name{Georgian: "ვაკის პარკი"}.In(en)` gives the Georgian name. A name with both gives the requested one. |
| `TTCTRANS-SC-014` | `TTCTRANS-FR-014` | unit | An itinerary of walk, ride, ride, walk has 1 transfer. A walk only has 0. Its duration is the end minus the start. |

## The driven adapter

| ID | Verifies | Layer | Given, When, Then |
| -- | -------- | ----- | ----------------- |
| `TTCTRANS-SC-015` | `TTCTRANS-FR-001`, `TTCTRANS-FR-020` | unit | `Stops` joins `stops-ka.json` and `stops-en.json`. `1:1000` has "ვასო გოძიაშვილის ბაღი" and "Vaso Godziashvili Park". `1:metro_1_1` is `metro`. `1:gondola_1` is `cable_car` with no code. A copy with the mode `TRAM` fails with `ErrSourceUnavailable`. |
| `TTCTRANS-SC-016` | `TTCTRANS-FR-009` | unit | `Routes`: `472` is `minibus`, `385` is `bus`, the line `1` of `SUBWAY` is `metro`, of `GONDOLA` is `cable_car`. |
| `TTCTRANS-SC-017` | `TTCTRANS-FR-006` | unit | `Arrivals` with `arrival-times.json`: `472` is live in 5 minutes, not -119. `arrival-times-metro.json`: not live, 3 minutes. A copy with one arrival not live at -5 leaves it out. `Expected` is `ReadAt` plus the minutes. |
| `TTCTRANS-SC-018` | `TTCTRANS-FR-010` | unit | `Directions(1:Metro_Metro_1)`: directions 1 and 0, with the headsigns in both languages. `Directions(1:R29925)`: one direction, 22 stops, first and last `1:1001`. |
| `TTCTRANS-SC-019` | `TTCTRANS-FR-011` | unit | `Timetable(1:R29925, 1)` gives one day for each of 7 dates, 2026-10-19 to 2026-10-25. Direction 0 of `385` gives `ErrUnknownDirection`. The metro keeps `24:10` as the last time of `1:metro_1_10` on 2026-10-19. |
| `TTCTRANS-SC-020` | `TTCTRANS-FR-013`, `TTCTRANS-FR-015` | unit | `Plan` with `plan.json`: the first ride starts at `2026-10-19T21:42:51+04:00`. A 400 gives `app.ErrTripNotFound`. A 500 gives `kernel.ErrSourceUnavailable`. |
| `TTCTRANS-SC-021` | `TTCTRANS-FR-023` | unit | Each port method meets a 401. The log holds one error record `the TTC refused the key` with `variable=GEOMCP_TRANSIT_TTC_API_KEY`, and no record holds the key. The error wraps `kernel.ErrSourceUnavailable`. |
| `TTCTRANS-SC-022` | `TTCTRANS-FR-019` | unit | `Places` with `geocode.json` and a copy with one feature at 40, 40. The answer leaves that feature out. A feature with no name gets the street and the house number. |

## The use cases

| ID | Verifies | Layer | Given, When, Then |
| -- | -------- | ----- | ----------------- |
| `TTCTRANS-SC-023` | `TTCTRANS-FR-001`, `TTCTRANS-FR-003` | unit | A fake list. `1000` gives `1:1000` first. `godziashvili` gives `1:1000`. `ვაკის პარკი` in `en` gives each "Vake Park" in English. `isani` with `metro` gives the metro station only. |
| `TTCTRANS-SC-024` | `TTCTRANS-FR-002` | unit | 41.7251, 44.7554 and 300 m: `1:1000` at 3 m, `1:1055` at 232 m, `1:1054` at 278 m. `1:1001` at 305 m is out. No radius uses 500 m. 40, 40 gives `ErrOutsideTbilisi`. |
| `TTCTRANS-SC-025` | `TTCTRANS-FR-004` | unit | `1000` and `1:1000` resolve to one stop. `1:gondola_1` resolves. `999999` gives `ErrUnknownStop`, and the fake arrival source gets no call. |
| `TTCTRANS-SC-026` | `TTCTRANS-FR-007`, `TTCTRANS-FR-009` | unit | Stop `1000`, route `472`, limit 4. The source gets `1:minibusR25521` and 4. Each arrival has the kind `minibus`. An empty answer gives an empty list. |
| `TTCTRANS-SC-027` | `TTCTRANS-FR-005`, `TTCTRANS-INV-003` | unit | 12 arrivals and the limit 5. The answer holds 5, soonest first, a tie sorted by route number. |
| `TTCTRANS-SC-028` | `TTCTRANS-FR-008` | unit | `385` gives "Vake-Saburtalo" first. `1` gives the metro line and the cable car. No text and `metro` gives lines `1` and `2`. No text gives buses before minibuses, `310` before `385`. |
| `TTCTRANS-SC-029` | `TTCTRANS-FR-010` | unit | `1` with no kind gives `*AmbiguousRouteError` with 2 routes. `1` and `metro` gives two directions of 16 stations each. `999` gives `ErrUnknownRoute`. |
| `TTCTRANS-SC-030` | `TTCTRANS-FR-011` | unit | `385`, stop `1:1001`, 2026-10-19: the times start `06:45`, `06:55`, from position 1, not position 22. No date on 2026-10-20 at 08:00 in Tbilisi uses 2026-10-20. |
| `TTCTRANS-SC-031` | `TTCTRANS-FR-011` | unit | Metro line `1` with no direction gives `*DirectionError` with 2 directions. `385` with no direction uses its one direction. `385` and stop `1:1000` gives `ErrStopNotOnDirection`. |
| `TTCTRANS-SC-032` | `TTCTRANS-FR-012` | unit | 2026-10-25 answers. 2026-10-26 gives `*DateOutsideTimetableError` with 2026-10-19 and 2026-10-25. |
| `TTCTRANS-SC-033` | `TTCTRANS-FR-013`, `TTCTRANS-FR-015`, `TTCTRANS-INV-001` | unit | `plan.json` through a fake planner. The ride `310` runs from `1:1000` to `1:2372`. The ride `532` is a `minibus`. The first walk starts at a point with no stop id. A point at 40, 40 gives `ErrOutsideTbilisi`, and the planner gets no call. |
| `TTCTRANS-SC-034` | `TTCTRANS-FR-016`, `TTCTRANS-FR-018` | unit | `arrive_by`, no date, `09:00`, on 2026-10-19 at 22:00 in Tbilisi: the request holds `2026-10-19T09:00+04:00`. `depart_at` with no time gives `ErrInvalidClockTime`. No preference gives `fastest`. |
| `TTCTRANS-SC-035` | `TTCTRANS-FR-019` | unit | 8 places and the limit 5 give 5. An empty text gives a refusal before the call. |

## The tools and the module

| ID | Verifies | Layer | Given, When, Then |
| -- | -------- | ----- | ----------------- |
| `TTCTRANS-SC-036` | `TTCTRANS-FR-021` | integration | The TTC is on. `tools/list` holds the 8 tools, each with the annotations of `MCP-004`. With the TTC off, it holds no tool that starts with `transit_`, and the log holds `TTC is off`. |
| `TTCTRANS-SC-037` | `TTCTRANS-FR-005`, `TTCTRANS-FR-006` | integration | `transit_list_arrivals` with `1000`. `source` is `TTC`. `as_of` and each `expected` end with `+04:00`. The first arrival is `531`, `minibus`, live, in 1 minute. `expected` is `as_of` plus `minutes`. |
| `TTCTRANS-SC-038` | `TTCTRANS-FR-012`, `TTCTRANS-FR-017` | integration | Each row of the errors table of `spec-tools.md` gives a result with `IsError` and the exact message. No message holds a vendor body or a status code. |
| `TTCTRANS-SC-039` | `TTCTRANS-FR-013` to `TTCTRANS-FR-015` | integration | `transit_plan_trip` from 41.7251, 44.7554 to 41.7151, 44.8271. 3 itineraries. The first has `duration_minutes` 61, `transfers` 1, and a leg `bus` `310` from 21:42 to 21:59 over 4,913 m. |
| `TTCTRANS-SC-040` | `TTCTRANS-NFR-001` | unit | `testing/synctest`. The fake source serves stops at 10:00. At 09:59 the next day the cache answers with no call. At 10:00 it calls again. If that call fails, the answer holds the old list with `Stale`. A timetable read on Sunday is not served on Monday. |
| `TTCTRANS-SC-041` | `TTCTRANS-NFR-002` | unit | `testing/synctest`. Two calls for one stop 14 seconds apart cause one source call. A call at 16 seconds causes a second. If it fails, the answer is the error, with no stale value. |
| `TTCTRANS-SC-042` | `TTCTRANS-NFR-003` | integration | 30 calls of `transit_search_stops`, `transit_list_nearby_stops`, and `transit_search_routes`. `httptest` counts 4 calls in total: 2 of stops, 2 of routes. |
| `TTCTRANS-SC-043` | `TTCTRANS-NFR-004` | integration | 20 calls of `transit_list_arrivals` for 20 stops at once, with the defaults of the configuration. `httptest` gets at most 5 calls in any second. |
| `TTCTRANS-SC-044` | `TTCTRANS-FR-022` | unit | `LoadConfig` with no key fails, and the error names `GEOMCP_TRANSIT_TTC_API_KEY`. With `GEOMCP_TRANSIT_TTC_ENABLED=false` and no key it passes. The defaults match 4.3 of `spec.md`. |

## Retired identifiers

This file has no retired identifier.
