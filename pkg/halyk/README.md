# halyk

A Go client of the exchange rates of Halyk Bank Georgia, a commercial bank of Georgia.
It reads the table of the current rates of the Georgian rate page.

The source is unofficial. Halyk Bank publishes no API of its rates: the client reads
[halykbank.ge/ka/individuals/currency](https://halykbank.ge/ka/individuals/currency), a
page for people. The page can change with no notice, so this package can change in any
release.

```go
client := halyk.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Currency+": buy", row.Buy, "sell", row.Sell, "GEL")
}
```

## The page

| Method  | Request                        | Answer                                          |
| ------- | ------------------------------ | ----------------------------------------------- |
| `Rates` | `GET /ka/individuals/currency` | Each row of the tab "მიმდინარე ვალუტის კურსი" |

The client mirrors what the page shows today:

- **One answer holds every currency**: 7 rows on 2026-10-25.
- A row has the code, the buy price, and the sell price. The client finds each column
  by its title, not by its place.
- **Each price is for one unit**: `KZT` buys for `0.0054`.
- Each price is the text of the page, with four decimal places: `2.5870`.
- The line "თარიღი" above the table gives the minute of the request, not of the
  publication. The client does not read it.
- The page also holds the prices of its converter as JSON, with the trailing zeros cut.
  The client does not read them.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
