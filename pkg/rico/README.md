# rico

A Go client of the exchange rates of Rico Credit, a microfinance organization of
Georgia. It reads the rate table of the Georgian home page.

The source is unofficial. Rico Credit publishes no API of its rates: the client reads
[www.rico.ge/ka/](https://www.rico.ge/ka/), a page for people. The page can change
with no notice, so this package can change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/rico
```

```go
client := rico.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Unit, row.Currency+": buy", row.Buy, "sell", row.Sell, "GEL")
}
```

## The page

| Method  | Request    | Answer                                      |
| ------- | ---------- | ------------------------------------------- |
| `Rates` | `GET /ka/` | Each row of the table "ვალუტის კურსები"     |

The client mirrors what the page shows today:

- **One answer holds every currency**: 13 rows on 2026-10-21.
- A row has the code, the unit, the buy price, and the sell price. The client finds
  each price by its label, not by its place.
- **Each price is for `Unit` units, not for one unit**: `10 PLN` buys for `6.1000`.
- The table holds one cross rate, `USD-EUR`. Its prices are in dollars, not in GEL.
- Each price is the text of the page, with four decimal places: `2.6000`.
- The page also holds the rates of its calculator, divided by the unit and rounded.
  The client does not read them.
- The page holds no instant.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
