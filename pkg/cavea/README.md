# cavea

A Go client of the cinemas, movies, sessions, and seats of Cavea, the largest chain of
cinemas of Georgia.

Cavea publishes no API. The client reads the JSON methods under `/api/` that the pages of
[cavea.ge](https://cavea.ge) call, and the data stream of the pages `/movies` and
`/sessions`. The source is unofficial and can change with no notice, so this package can
change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/cavea
```

```go
client := cavea.New()

sessions, err := client.MovieSessions(ctx, movieID)
if err != nil {
    return err
}

for _, session := range sessions {
    fmt.Println(session.ID, session.SessionLanguage, session.Prices["std"], "GEL")
}
```

## The methods

| Method          | Request                                                    | Answer                                   |
| --------------- | ---------------------------------------------------------- | ---------------------------------------- |
| `Theaters`      | `GET /api/theaters`                                        | Every cinema, open or closed             |
| `HallMap`       | `GET /api/theaters/hall-map`                               | The label of each hall: `cep_1` is `IMAX` |
| `Movies`        | `GET /movies`, `RSC: 1`, and `?filter=coming-soon`         | The movies now showing, or coming soon   |
| `SearchMovies`  | `GET /api/movies/search?q=<text>`                          | The movies whose English name matches    |
| `Sessions`      | `GET /sessions`, `RSC: 1`                                  | Every session of the next days, by date  |
| `MovieSessions` | `GET /api/booking/sessions/all/<movie ID>`                 | The sessions of one movie, with each class price |
| `Availability`  | `GET /api/sessions/availability?ids=<codes>`               | The level of each session                |
| `SeatMap`       | `POST /api/booking/seat-ticket/<code>`, then `GET /api/booking/seats/<code>` | The seats of the hall of a session |

The site needs no key. This client mirrors what Cavea sent on 2026-10-25:

- Each call carries `Origin: https://cavea.ge` and `X-Requested-With: cavea-web`.
  `/api/booking/sessions/all` refuses a call with no `Origin`.
- With the header `RSC: 1`, a page answers its data stream. The stream holds the list as
  plain JSON under the key `movies` or `dateGroups`.
- An answer of the API has the field `status`: `success`, or `OK!` on
  `/api/booking/sessions/all`, or `error`. The client returns `ErrNotSuccess` for `error`.
- `/api/booking/sessions/all` sends its list as a JSON text in `data`, and the prices of
  each session as a JSON text in `prices`. The client decodes both.
- `Availability` takes at most 30 codes in one call.
- The seat map needs a ticket of one session, valid for 15 minutes, and the cookie
  `cavea_vid` of the answer of the ticket. `SeatMap` asks for both at each call.
- `robots.txt` disallows `/api/`.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
