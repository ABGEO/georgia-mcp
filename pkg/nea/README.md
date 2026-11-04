# nea

A Go client of the air quality data of the National Environmental Agency of Georgia
(NEA), as the map at `air.gov.ge` reads it. It returns the hourly values of the
automatic stations, and the municipalities of Georgia.

The methods are unofficial. The NEA publishes no documentation of them.

```
go get github.com/abgeo/georgia-mcp/pkg/nea
```

```go
client := nea.New()

stations, err := client.Hourly(ctx, nea.HourlyQuery{
    From: "2026-11-04T00:00:00", To: "2026-11-05T13:00:00", Station: "TSRT",
})
if err != nil {
    return err
}

fmt.Println(stations[0].Equipment[0].Substance.Name) // PM10
```

## The API

| Method           | Request                       | Answer                                           |
| ---------------- | ----------------------------- | ------------------------------------------------ |
| `Hourly`         | `GET /api/get_data_1hour/`    | The stations, their substances, and the values of each hour |
| `Municipalities` | `GET /api/municipalities/`    | The municipalities, in Georgian and in English   |

The client mirrors what the API answers today:

- **A path without its final slash answers 301.** The client ends each path with a slash.
- A time is a local time of Tbilisi with no offset: `2026-11-04T23:00:00`.
- The time of some substances runs ahead of the clock. On 2026-11-05, PM10 and PM2.5
  ran 6 hours ahead, O3 and CO 2 hours.
- An unknown station code, and a time in another form, answer `[]`.
- A value can be negative, and has up to 13 decimals. The client keeps its text.
- Each substance carries the upper limit of each level: good, fair, moderate, poor.
- A unit holds markup: `μg/m<sup>3</sup>`.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
