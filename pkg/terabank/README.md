# terabank

A Go client of the exchange rates of Terabank.

The API is unofficial. Terabank publishes no documentation of it: the page
[terabank.ge/ka/retail/exchangerates](https://terabank.ge/ka/retail/exchangerates)
reads it. The API can change with no notice, so this package can change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/terabank
```

```go
client := terabank.New()

rates, err := client.CrossRates(ctx)
if err != nil {
    return err
}

for _, rate := range rates {
    fmt.Println("1", rate.ISO+": buy", rate.Buy, "sell", rate.Sell, "GEL")
}
```

## The API

| Method       | Request                                            | Answer                              |
| ------------ | -------------------------------------------------- | ----------------------------------- |
| `CrossRates` | `POST /_mvcapi/CurrencyRatesApi/GetTeraCrossRates` | The rate of each currency in GEL    |

The client mirrors what the API answers today:

- **The method takes a POST with an empty body.** A POST with no `Content-Length`
  answers 411, and a GET answers 404. The client sends `Content-Length: 0`.
- **One answer holds every currency**: `USD`, `EUR`, and `GBP` on 2026-10-21.
- Every price is the price in GEL of one unit. The bank calls them cross rates.
- `Buy` and `Sell` are `json.Number`, so they keep the digits of the answer: `2.5960`.
- An item also holds the prices of the day before, and the flag that tells the page
  to show them.
- An answer has the field `succeeded`. False gives `ErrNotSucceeded`.
- The answer holds no instant. Terabank publishes no limit of the calls.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
