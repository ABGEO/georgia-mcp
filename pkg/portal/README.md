# portal

A Go client of the fuel prices of Portal, a fuel company of Georgia.

Portal publishes no API. The page
[portal.com.ge/ka/fuel-prices](https://portal.com.ge/ka/fuel-prices) renders on the
server, and sends its data to the script of the page as a stream of JSON. The client asks
for that stream and reads the price list from it. The data of a web page can change with
no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/portal
```

```go
client := portal.New()

prices, err := client.FuelPrices(ctx)
if err != nil {
    return err
}

for _, price := range prices {
    fmt.Println(price.FuelType.Slug, price.Name, price.LatestPrice, "GEL")
}
```

## The page

| Method       | Request                                                     | Answer                     |
| ------------ | ----------------------------------------------------------- | -------------------------- |
| `FuelPrices` | `GET /ka/fuel-prices`, `RSC: 1`, `Accept: text/x-component` | The price list of the page |

The page needs no key. This client mirrors what the page sent on 2026-10-21:

- With the header `RSC: 1`, the page answers its data stream, about 112 KiB, instead of
  the HTML, about 193 KiB. The stream holds the price list as plain JSON under the key
  `fuelPricesData`. The client decodes the value after that key and ignores the rest.
- The list holds 5 entries. Each has a name, a type, an octane number, `latestPrice`,
  `latestOnlinePrice`, and a slug under `fuelType`.
- The prices are text in GEL for one litre: `"4.39"`.
- `latestOnlinePrice` is the price of a QR liter bought online. The page does not show it.
- The names are in English in each language of the page: `SUPER`, `EURO REGULAR`.
- The SUPER of Portal has the octane number 100.
- `robots.txt` disallows `/api/`. The client reads the page, not that path.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
