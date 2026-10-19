# rompetrol

A Go client of the fuel prices of Rompetrol Georgia. It covers the current prices.

Rompetrol publishes no API. The client reads the price table of the home page
[www.rompetrol.ge](https://www.rompetrol.ge/), a page for people. The page can change
with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/rompetrol
```

```go
client := rompetrol.New()

rows, err := client.Prices(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Product, row.Price, "GEL")
}
```

## The page

| Method   | Request | Answer                                     |
| -------- | ------- | ------------------------------------------ |
| `Prices` | `GET /` | Each row of the price table, in page order |

The page needs no key. This client mirrors what the page showed on 2026-10-19:

- The table follows the anchor `id="pricelist"`. Each row has a product and a price in
  GEL for one litre, as text: `4.02`.
- A row of a self-service price follows the row of its product. Its name starts with
  `- ` and ends with `(თვითმომსახურების სადგურებზე)`. The name does not always repeat
  the name of the product.
- The page shows no date, no instant, and no product code.
- A page with no table gives `ErrNoPriceTable`. A row with a count of cells other than
  two gives `ErrMalformedTable`.

The client parses the page with `golang.org/x/net/html`. It does not retry and does not
cache. Pass an `http.Client` with `WithHTTPClient` to set a timeout or a rate limit.
