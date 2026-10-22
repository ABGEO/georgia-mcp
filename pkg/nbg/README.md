# nbg

A Go client of the official exchange rates of the National Bank of Georgia (NBG).
It covers every address of the API: the rates as JSON, the RSS feed, and the codes.

```
go get github.com/abgeo/georgia-mcp/pkg/nbg
```

```go
client := nbg.New()

sets, err := client.RateSets(ctx, nbg.LanguageEnglish, nbg.RateSetsQuery{
    Date:       time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC),
    Currencies: []string{"USD"},
})
if err != nil {
    return err
}

for _, currency := range sets[0].Currencies {
    fmt.Println(currency.Quantity, currency.Code, "=", currency.RateFormated, "GEL")
}
```

## The API

| Method     | Request                                                   | Answer                            |
| ---------- | --------------------------------------------------------- | --------------------------------- |
| `RateSets` | `GET /currencies/{en\|ka}/json/?date=…&currencies=…`      | The set in force on the date      |
| `RSS`      | `GET /currencies/{en\|ka}/rss/?date=…&currencies=…`       | The same set as an RSS 2.0 feed   |
| `Codes`    | `GET /currencies/codes`                                   | Every code, with a Georgian name  |

The NBG links the JSON and the RSS address from its page of rates, and publishes no
documentation of the API. This client mirrors what the API answers today:

- A zero `Date` asks for the newest set. A date after the newest set, and a date that
  the API cannot read, return the newest set.
- A date before the first set returns an empty list.
- `Currencies` goes to the NBG as a list joined with commas, unchanged. **The NBG answers
  most lists of two or more codes with an empty list.** One code works. To read several
  currencies, ask for the full set and filter it.
- The publication instant of a currency carries the suffix `Z`, but holds the wall time
  of Tbilisi.
- An item of the RSS feed holds the rates as an HTML table in `Description`. The client
  does not parse it.
- The codes hold Georgian names only. The list can name a code that no rate set holds.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
