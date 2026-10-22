# kursi

A Go client of the exchange rates of Kursi.ge, an online exchange bureau of Georgia.

The method is unofficial. Kursi.ge documents a
[merchant API](https://docs.kursi.ge/merchant/), whose method of the rates
`GET /api/currency` needs the token of a merchant account. This client reads the public
method that the page [kursi.ge](https://kursi.ge/ka/) reads. It needs no token and is
not in the documentation, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/kursi
```

```go
client := kursi.New()

pairs, err := client.Currencies(ctx)
if err != nil {
    return err
}

for _, pair := range pairs {
    if pair.BaseCurrencyCode == "GEL" {
        fmt.Println("1", pair.SecondaryCurrencyCode+": buy", pair.BuyRate, "sell", pair.SellRate, "GEL")
    }
}
```

## The API

| Method       | Request                       | Answer                            |
| ------------ | ----------------------------- | --------------------------------- |
| `Currencies` | `GET /api/public/currencies`  | Every pair, with its two rates    |

The client mirrors what the API answers today:

- **One answer holds every pair**: 8 on 2026-10-21. 4 of them have the base `GEL`.
  The others are between two foreign currencies.
- For the base `GEL`, `BuyRate` is the price in GEL that Kursi.ge pays for one unit of
  the secondary currency, and `SellRate` the price that it asks.
- A pair also holds the official rate of the NBG, its change, and the rates of TBC and
  Bank of Georgia that Kursi.ge repeats.
- Each price is `json.Number`, so it keeps the digits of the answer: `2.5940`.
- The answer forbids a cache, and holds no instant. Kursi.ge publishes no limit of the
  calls.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
