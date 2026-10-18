# tbc

A Go client of the developer API of TBC Bank. It covers the commercial exchange rates.

```
go get github.com/abgeo/georgia-mcp/pkg/tbc
```

```go
client := tbc.New(tbc.WithAPIKey(os.Getenv("TBC_API_KEY")))

rates, err := client.CommercialRates(ctx, []string{"USD", "EUR"})
if err != nil {
    return err
}

for _, rate := range rates.CommercialRatesList {
    fmt.Println("1", rate.Currency, "buy", rate.Buy, "sell", rate.Sell, rates.Base)
}
```

## The API

| Method            | Request                                                 | Answer                                 |
| ----------------- | ------------------------------------------------------- | -------------------------------------- |
| `CommercialRates` | `GET /v1/exchange-rates/commercial?currency=USD,EUR`    | The buy and sell rate of each currency |

A key of a developer account at [developers.tbcbank.ge](https://developers.tbcbank.ge)
goes in the header `apikey`. This client mirrors what the API answers today:

- **An answer holds at most 10 currencies.** With no `currency`, or more than 10 codes,
  TBC answers 10 of its currencies, and which 10 changes over time. To read a currency,
  name it.
- TBC leaves out a code that it does not quote, with no error.
- Every rate is the price in GEL of one unit, the yen included.
- `Buy` and `Sell` are `json.Number`, so they keep the digits of the answer. TBC writes
  JSON numbers, with no trailing zero: `2.88`.
- The answer holds no instant.
- A failure is an `*APIError` with the fields of the problem format: `Title`,
  `Detail`, `SystemCode`, `TraceID`. A missing or wrong key gives 401.
- TBC allows 60 calls in a minute for each key.

The client does not retry and does not cache. Pass an `http.Client` with
`WithHTTPClient` to set a timeout or a rate limit.
