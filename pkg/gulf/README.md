# gulf

A Go client of the fuel prices of Gulf Georgia, a fuel company of Georgia.

Gulf publishes no API. The client reads the table of the page
[gulf.ge/en/fuel_prices](https://gulf.ge/en/fuel_prices), a web page for people. The
page can change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/gulf
```

```go
client := gulf.New()

table, err := client.PriceArchive(ctx)
if err != nil {
    return err
}

fmt.Println(table.Columns[1]) // {G-Force Super g_col}
fmt.Println(table.Rows[0])    // [2026-10-19 4.69 4.39 4.09 5.05 4.99 4.59 4.29 3.99 4.95 4.89]
```

## The page

| Method         | Request               | Answer                                      |
| -------------- | --------------------- | ------------------------------------------- |
| `PriceArchive` | `GET /en/fuel_prices` | Each column and each row of the price table |

The page needs no key. This client mirrors what the page showed on 2026-10-22:

- A bot filter guards the site. It answers 403 to the TLS handshake of Go. The default
  client dials with the handshake of Chrome 131 over HTTP/1.1, with the User-Agent of
  this module. Use `NewTransport` if you pass your own `http.Client`.
- One table, `#price_archive_tbl`. The head holds the date (class `first`), then 5
  products of the tier Gulf (class `g_col`), then the same 5 of the tier Gulf+ (class
  `gp_col`). Gulf+ applies at the stations named "Gulf + ...".
- One row for each change, the newest first, 100 rows on each page. The date has no time.
- The prices are text in GEL for one litre, with two places: `4.69`. A price that Gulf
  does not apply is `0.00`.
- The client returns the text of each cell. It does not pick a row or a tier: the
  meaning of the table belongs to the caller.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
