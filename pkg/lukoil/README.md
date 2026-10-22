# lukoil

A Go client of the price history of Lukoil Georgia, a fuel company of Georgia.

Lukoil publishes no API. The client reads the table of the page
[www.lukoil.ge/prices-history](https://www.lukoil.ge/prices-history), a web page for
people. The page can change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/lukoil
```

```go
client := lukoil.New()

table, err := client.PriceHistory(ctx)
if err != nil {
    return err
}

fmt.Println(table.Header)              // [# Super Ecto 100 ... Euro Diesel შეცვლის თარიღი]
fmt.Println(table.Rows[len(table.Rows)-1])
```

## The page

| Method         | Request               | Answer                                  |
| -------------- | --------------------- | --------------------------------------- |
| `PriceHistory` | `GET /prices-history` | The head and each row of the price table |

The page needs no key. This client mirrors what the page showed on 2026-10-21:

- One table. The head names a number, each product, and "შეცვლის თარიღი", the instant
  of the change: `2026-10-05 10:01:46`, with no zone.
- One row for each change: 106 rows from 2022-07-15, the oldest first, on one page. Two
  rows can share a day.
- The prices are text in GEL for one litre, with two places: `4.60`.
- The names are in English: "Super Ecto", "Euro Diesel".
- Super Ecto 100 has `0.00` in each row: a product that Lukoil does not sell.
- The newest row equals the prices of the home page, which shows no date.
- The client returns the text of each cell. It does not pick a row: the meaning of the
  rows belongs to the caller.
- The page sets the cookies `XSRF-TOKEN` and `laravel_session`. The client sends none.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
