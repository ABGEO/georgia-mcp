# inteli

A Go client of the exchange rates of Inteli Express, a microfinance organization of
Georgia. It reads the rate table of the Georgian home page.

The source is unofficial. Inteli Express publishes no API of its rates: the client reads
[inteliexpress.com/ka/main-page-geo/](https://inteliexpress.com/ka/main-page-geo/), a
page for people. The page can change with no notice, so this package can change in any
release.

```go
client := inteli.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Currency+": buy", row.Buy, "sell", row.Sell, row.Base)
}
```

## The page

| Method  | Request                  | Answer                                              |
| ------- | ------------------------ | --------------------------------------------------- |
| `Rates` | `GET /ka/main-page-geo/` | Each row of the table "ვალუტის სტანდარტული კურსები" |

The client mirrors what the page shows today:

- **One answer holds every currency**: 6 rows on 2026-10-25, one of them the cross rate
  of EUR against USD.
- **The table names no column.** A row has four cells: the currency of the prices, the
  currency that the row prices, and two prices, the lower first. The client finds the
  table by its title and returns the cells in that order.
- Each price is for one unit, as the text of the page, with three decimal places: `0.050`.
- The page holds no instant. A branch can give a rate other than this standard rate.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
