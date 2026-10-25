# crystal

A Go client of the exchange rates of Crystal, a microfinance organization of Georgia.
It calls the rate method of the rate app of Crystal.

The source is unofficial. Crystal publishes no documented API: the page
[crystal.ge/valutis-kursebi/](https://crystal.ge/valutis-kursebi/) shows the app
`currency.crystal.ge` in a frame, and the app reads this method. The method can change
with no notice, so this package can change in any release.

```go
client := crystal.New()

rates, err := client.CashRates(ctx)
if err != nil {
    return err
}

for _, rate := range rates {
    fmt.Println(rate.Multiplier, rate.SourceISO+": buy", rate.BuyRate, "sell", rate.SellRate, rate.TargetISO)
}
```

## The method

| Method      | Request                                                  | Answer                        |
| ----------- | -------------------------------------------------------- | ----------------------------- |
| `CashRates` | `GET /api/Conversion/GetWebRateByProductID?IsCash=true`  | The default cash rates        |

The client mirrors what the method answers today:

- **One answer holds every currency**: 11 items on 2026-10-25, one of them the cross
  rate of EUR against USD.
- The list sits in two envelopes. Each one can report a failure with the status 200.
  The client checks both.
- `Multiplier` gives the count of units of each price: 1, 10, or 1000.
- Each price is a JSON number with the trailing zeros cut: `6`, `2.595`. The client keeps
  its digits in a `json.Number`.
- The answer holds no instant. A branch can give a rate other than this default rate.
- The script of the app sends a token. The method answers without it, and the client
  sends none.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
