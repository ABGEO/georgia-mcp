# nager

A Go client of the public holidays of Nager.Date, an open source project at
`date.nager.at`. It returns the public holidays of one country in one year.

The project is not official. It computes the holidays from its own rules.

```
go get github.com/abgeo/georgia-mcp/pkg/nager
```

```go
client := nager.New()

holidays, err := client.PublicHolidays(ctx, 2026, "GE")
if err != nil {
    return err
}

fmt.Println(holidays[0].Date, holidays[0].LocalName) // 2026-01-01 ახალი წელი
```

## The API

| Method           | Request                                           | Answer                              |
| ---------------- | ------------------------------------------------- | ----------------------------------- |
| `PublicHolidays` | `GET /api/v3/PublicHolidays/{year}/{countryCode}` | The holidays of the year, by date   |

The client mirrors what the API answers today:

- A holiday has a date, a local name, an English name, and its types: `Public`, `Observance`.
- Two days can share a name: New Year is 2026-01-01 and 2026-01-02.
- A year before 1975 or after 2075 answers 400. An unknown country answers 404.
- The address `nagerholidays.com/api/v4` gives no local name.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
