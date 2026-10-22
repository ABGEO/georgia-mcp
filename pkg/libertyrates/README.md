# libertyrates

A Go client of the exchange rates of Liberty Bank. It reads the rate block of the
Georgian home page.

The source is unofficial. Liberty Bank publishes no API of its rates: the client reads
[www.libertybank.ge/ka/](https://www.libertybank.ge/ka/), a page for people. The page
can change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/libertyrates
```

```go
client := libertyrates.New()

rows, err := client.Rates(ctx)
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Unit+": buy", row.Commercial.Buy, "sell", row.Commercial.Sell, "GEL")
}
```

## The page

| Method  | Request    | Answer                                     |
| ------- | ---------- | ------------------------------------------ |
| `Rates` | `GET /ka/` | Each currency of the block "ვალუტის კურსი" |

The client mirrors what the page shows today:

- **One answer holds every currency**: 8 of them on 2026-10-21.
- A currency has four columns. The client finds each column by its title:
  `Commercial` (the rate at a branch), `InternetBank` (the internet bank, for a person
  up to 100 000 GEL), and `Transfer` (the special rate for money transfers). `Official`
  is the rate of the NBG.
- `InternetBank` and `Transfer` are empty for a currency that has none.
- **Each price is for `Unit`, not for one unit**: `1000 AMD` buys for `6.2220`.
- Each price is the text of the page, with four decimal places: `2.5470`.
- The page holds the block twice. The client reads the first one.
- The page holds no instant.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
