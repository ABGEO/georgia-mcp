# bog

A Go client of the public rates API of Bank of Georgia. It covers the commercial
exchange rate.

```
go get github.com/abgeo/georgia-mcp/pkg/bog
```

```go
client := bog.New()

rate, err := client.CommercialRate(ctx, "USD")
if errors.Is(err, bog.ErrNoRate) {
    return nil // the bank has no rate of the code
}
if err != nil {
    return err
}

fmt.Println("1 USD: buy", rate.Buy, "sell", rate.Sell, "GEL")
```

## The API

| Method           | Request                                 | Answer                               |
| ---------------- | --------------------------------------- | ------------------------------------ |
| `CommercialRate` | `GET /api/rates/commercial/{currency}`  | The buy and sell rate of one currency |

[The documentation](https://api.bog.ge/docs/en/bonline/rates/commercial-currency.md)
describes the method. It needs no key. This client mirrors what the API answers today:

- **An answer holds one currency.** To read 10 currencies, send 10 calls.
- Every rate is the price in GEL of one unit, the yen included.
- `Buy` and `Sell` are `json.Number`, so they keep the digits of the answer. The bank
  pads each price to eight decimal places: `2.56500000`.
- The answer holds no instant.
- `GEL` and an unknown code give 204 with no body. The client returns `ErrNoRate`.
- A code that is not three letters gives 404, an `*APIError`.
- The bank allows 60 calls in a minute. Past the limit it answers 429 with the header
  `Retry-After`, in `APIError.RetryAfter`.

The client does not retry and does not cache. Pass an `http.Client` with
`WithHTTPClient` to set a timeout or a rate limit.
