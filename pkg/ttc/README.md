# ttc

A Go client of the passenger information API of the Tbilisi Transport Company (TTC):
the stops, the routes, the arrivals, the timetables, the trip plans, and the place
search of Tbilisi.

The API is unofficial. The TTC publishes no documentation of it: the web application
[transit.ttc.com.ge](https://transit.ttc.com.ge/) reads it. The API can change with no
notice, so this package can change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/ttc
```

```go
client := ttc.New(ttc.WithAPIKey(key))

arrivals, err := client.ArrivalTimes(ctx, "1:1000", ttc.ArrivalTimesQuery{Locale: "en"})
if err != nil {
    return err
}

for _, arrival := range arrivals {
    fmt.Println(arrival.ShortName, arrival.Headsign, arrival.RealtimeArrivalMinutes)
}
```

## The API

| Method         | Request                                            | Answer                                    |
| -------------- | -------------------------------------------------- | ----------------------------------------- |
| `Stops`        | `GET /v2/stops`                                    | Every stop                                |
| `Routes`       | `GET /v3/routes?modes=BUS,SUBWAY,GONDOLA`          | Every route                               |
| `Route`        | `GET /v3/routes/{id}`                              | A route with one pattern for each direction |
| `PatternStops` | `GET /v3/routes/{id}/stops?patternSuffix=`         | The stops of a pattern, in order          |
| `Schedule`     | `GET /v3/routes/{id}/schedule?patternSuffix=`      | The timetable of a pattern for this week  |
| `ArrivalTimes` | `GET /v2/stops/{id}/arrival-times`                 | The next arrivals at a stop               |
| `Plan`         | `GET /v2/plan`                                     | Up to 3 itineraries between two points    |
| `Geocode`      | `GET /v2/geocode`                                  | Places of OpenStreetMap                   |

Every call needs a key in the header `X-api-key`. The web page of the TTC publishes the
key, and the TTC can change it. Pass it with `WithAPIKey`. A wrong key gives 401. This
client mirrors what the API answered on 2026-10-19:

- Each call takes `locale`: `ka` or `en`. Some names of minibus routes stay in Georgian
  with `en`.
- A station of metro or cable car has no `code`.
- A minibus route has the mode `BUS`, and an identifier that starts with `1:minibus`.
- With no route, `ArrivalTimes` gives the next arrival of each route. With a route, it
  gives several arrivals of that route.
- `scheduledArrivalMinutes` can be negative while `realtime` is true.
- A schedule covers the current week. A time of `24:05` is after midnight.
- The instants of a plan are in UTC. A leg gives the coordinates of its ends, and no stop
  identifier. A point outside Tbilisi gives 400.
- An unknown stop gives an empty list of arrivals. An unknown route gives 500.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
