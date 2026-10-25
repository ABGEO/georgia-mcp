# credo

A Go client of the exchange rates of Credo Bank, a commercial bank of Georgia.

The source is unofficial. Credo Bank publishes no API: the client calls the rate method
that the site [credobank.ge](https://credobank.ge) reads. Noxtton hosts it. The method
can change with no notice, so this package can change in any release.

The method needs a key in the header `x-api-key`. The client holds no key. Pass one with
`WithAPIKey`. With no key, or a wrong key, the method answers 401.

```go
client := credo.New(credo.WithAPIKey(os.Getenv("GEOMCP_RATES_CREDO_API_KEY")))

rates, err := client.CurrentRates(ctx, credo.MethodCommercial, time.Now())
if err != nil {
    return err
}

for _, rate := range rates {
    fmt.Println(rate.FromCcy+": buy", rate.Buy, "sell", rate.Sell, rate.ToCcy)
}
```

## The method

| Method         | Request                                              | Answer                     |
| -------------- | ---------------------------------------------------- | -------------------------- |
| `CurrentRates` | `GET /v2/credo/current-rates?method=<kind>&date=<at>` | Each pair of currencies    |

The client mirrors what the method answers today:

- **One answer holds every currency**: 3 pairs on 2026-10-25.
- `date` is the instant in UTC with milliseconds: `2026-10-25T08:42:36.000Z`.
- `MethodCommercial` gives the rate of the branches. The site also reads
  `FxRatesForCard`, `FXRatesCardTransactions`, and `FxRatesForTransfer`.
- A pair names both currencies and the count of units. **Each price is for
  `Multiplier` units**: 1 for each pair on 2026-10-25.
- Each price is a JSON number: `2.593`. The client keeps it as a `json.Number`.
- The answer holds `success`. A false `success` is `ErrNotSucceeded`.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
