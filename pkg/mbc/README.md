# mbc

A Go client of the exchange rates of MBC, a microfinance organization of Georgia. It
calls the rate method that the page of MBC reads.

The source is unofficial. MBC publishes no documented API: the page
[fx.mbc.com.ge](https://fx.mbc.com.ge) reads the method
`fxrates.mbc.com.ge:8022/api/fxrates/mbc/commercial`. The method can change with no
notice, so this package can change in any release.

```go
client := mbc.New()

rates, err := client.CommercialRates(ctx)
if err != nil {
    return err
}

for _, rate := range rates {
    fmt.Println(rate.Multiplier, rate.FromCcy+": buy", rate.Buy, "sell", rate.Sell, rate.ToCcy)
}
```

## The method

| Method            | Request                           | Answer                              |
| ----------------- | --------------------------------- | ----------------------------------- |
| `CommercialRates` | `GET /api/fxrates/mbc/commercial` | The list `FXRates`, and a timestamp |

The client mirrors what the method answers today:

- **One answer holds every currency**: 5 items on 2026-10-25, each against GEL.
- The page shows 3 of them and hides the others. The client returns all of them.
- Each price is a JSON number with four decimal places: `2.5950`. The client keeps its
  digits in a `json.Number`.
- `Multiplier` gives the count of units of each price. It was 1 for each item.
- `timestamp` is the instant of the answer, not of a change. The client does not read it.
- The method listens on port 8022, not on the port of the web.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
