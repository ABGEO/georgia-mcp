# scapp

A Go client of the exchange rates of Scapp, a microfinance organization of Georgia,
formerly Swiss Capital. It reads the rate table of the Georgian rate page.

The source is unofficial. Scapp publishes no API of its rates: the client reads
[scapp.ge/ka/currency](https://scapp.ge/ka/currency), a page for people. The page can
change with no notice, so this package can change in any release.

```go
client := scapp.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Unit+": buy", row.Buy, "sell", row.Sell, "GEL")
}
```

## The page

| Method  | Request            | Answer                                  |
| ------- | ------------------ | --------------------------------------- |
| `Rates` | `GET /ka/currency` | Each row of the table "მიმდინარე კურსები" |

The client mirrors what the page shows today:

- **One answer holds every currency**: 6 rows on 2026-10-25.
- A row has the unit with the code in one cell, the buy price, and the sell price. The
  client finds each price by its title, not by its place.
- **Each price is for the unit of its row, not for one unit**: `1 USD`, and one row of
  10 units on 2026-10-25.
- Each price is the text of the page, with three decimal places: `2.900`.
- The converter of the page holds the same prices, with the trailing zeros cut. The
  client does not read it.
- The page holds no instant. A cache in front of it keeps it for up to 5 minutes.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
