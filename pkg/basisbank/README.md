# basisbank

A Go client of the exchange rates of Basisbank, a commercial bank of Georgia.

The source is unofficial. Basisbank publishes no API: the client calls the rate method
that the page [bb.ge/ge/currencies](https://bb.ge/ge/currencies) reads. The method can
change with no notice, so this package can change in any release.

```go
client := basisbank.New()

days, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, day := range days {
    fmt.Println(day.Date, "USD: buy", day.XRates.KursBuy["USD"], "sell", day.XRates.KursSell["USD"])
}
```

## The method

| Method  | Request                       | Answer                         |
| ------- | ----------------------------- | ------------------------------ |
| `Rates` | `POST /bb/websiteXrates`      | The rates of two dates         |

The client mirrors what the method answers today:

- **One answer holds two dates**, the newest first. A date of a weekend repeats the
  rates of the last working day.
- The answer holds the rates of a date as a JSON text. The client decodes it into
  `XRates`: the official rate (`Kurs`), the buy price (`KursBuy`), and the sell price
  (`KursSell`).
- **Each buy and sell price is for one unit**. The official rate keeps the unit of the
  NBG: `CNY` per 10.
- Each price is the text of the answer, with four decimal places: `"2.5600"`.
- The page sends a token header. The method answers without it.

The site names a second source, `rates.basisbank.ge/Rate.ashx`: 3.4 MB of history in
each call, over HTTP only. The client does not read it.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
