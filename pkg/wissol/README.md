# wissol

A Go client of the fuel price API of Wissol. It covers the current prices.

The API is unofficial. Wissol publishes no documentation of it: the page
[wissol.ge/en/fuel-prices](https://wissol.ge/en/fuel-prices) reads it. The API can
change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/wissol
```

```go
client := wissol.New()

prices, err := client.FuelPrices(ctx)
if err != nil {
    return err
}

for _, price := range prices {
    fmt.Println(price.FuelTypeEng, price.Price, "GEL")
}
```

## The API

| Method       | Request          | Answer                                   |
| ------------ | ---------------- | ---------------------------------------- |
| `FuelPrices` | `GET /FuelPrice` | Every product with its current prices    |

The API needs no key. This client mirrors what the API answered on 2026-10-19:

- **One answer holds every product**, the products that are not active included.
- Each product has a code, a name in Georgian, English, and Russian, and a price in
  GEL for one unit: a litre, or a cubic metre for `Wissol Gas`.
- `PriceSelfService` is the price at a self-service station. Wissol sends no value for
  a product that those stations do not sell. The field is then empty.
- `Price` and `PriceSelfService` are `json.Number`, so they keep the digits of the answer.
- `Date` is the day of the last change of the price, with no zone: `2026-10-08T00:00:00`.
- The answer holds no instant and no unit.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
