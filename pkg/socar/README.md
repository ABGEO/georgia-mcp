# socar

A Go client of the fuel price API of SOCAR Georgia Petroleum. It covers the current prices.

The API is unofficial. SOCAR publishes no documentation of it: the page
[sgp.ge](https://sgp.ge/) reads it. The API can change with no notice, so the module
stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/socar
```

```go
client := socar.New()

prices, err := client.CurrentPrices(ctx)
if err != nil {
    return err
}

for _, price := range prices.Results {
    fmt.Println(price.FuelNameEng, price.FuelUnitPrice, "GEL")
}
```

## The API

| Method          | Request                                | Answer                                |
| --------------- | -------------------------------------- | ------------------------------------- |
| `CurrentPrices` | `GET /integration/info/current-prices` | Every product with its current price  |

The API needs no key. This client mirrors what the API answered on 2026-10-19:

- **One answer holds every product**, in `GetCurrentPrices.Results`, with `Status` `"1"`
  and `Message` `"Success"`.
- Each product has a code, a name in Georgian and in English, and a price in GEL for one
  unit: a litre, or a cubic metre for CNG.
- `FuelUnitPrice` is a `json.Number`, so it keeps the digits of the answer: `4.9`.
- `FuelNameEng` keeps the spaces of the answer: `"LPG "`.
- `ActionDate` is not the date of the last change of the price. On 2026-10-19 Nano Super
  had `2023-09-23`, and the price archive of the site showed a change on `2026-10-09`.
- No product has a price at a self-service station.
- An error of the backend is a 400 with a JSON body that holds the text of an exception.
  The client returns it as an `*APIError`.
- The price archive of the site gave LPG `1.75` from 2026-10-13. This API gave `2.09` from
  the same day, and the home page showed `2.09`.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
