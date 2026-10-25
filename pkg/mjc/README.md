# mjc

A Go client of the exchange rates of MJC Credit, a microfinance organization of
Georgia. It reads the rate table of the rate page.

The source is unofficial. MJC publishes no API of its rates: the client reads
[mjc.ge/rates](https://mjc.ge/rates), a page for people. The page can change with no
notice, so this package can change in any release.

```go
client := mjc.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Unit+": buy", row.Buy, "sell", row.Sell, "GEL")
}
```

## The page

| Method  | Request      | Answer                                  |
| ------- | ------------ | --------------------------------------- |
| `Rates` | `GET /rates` | Each row of the table "ვალუტის კურსები" |

The client mirrors what the page shows today:

- **One answer holds every currency**: 17 rows on 2026-10-25.
- A row has the unit with the code in one cell, the buy price, the sell price, and the
  official rate of the NBG. The client finds each column by its title, not by its place,
  and skips the official rate.
- **Each price is for the unit of its row, not for one unit**: `1 USD`, `10 PLN`,
  `1000 AMD`.
- Each price is the text of the page, with five decimal places: `2.59500`.
- The page holds no instant.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
