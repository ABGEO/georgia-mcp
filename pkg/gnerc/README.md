# gnerc

A Go client of the workbooks that the Georgian National Energy and Water Supply
Regulatory Commission (GNERC) links from `data.gnerc.org`: the monthly balance of
electricity and the monthly prices. It returns the links of the home page and the cells
of a workbook as text.

The workbooks are unofficial as an API. GNERC documents no format.

```
go get github.com/abgeo/georgia-mcp/pkg/gnerc
```

```go
client := gnerc.New()

book, err := client.Workbook(ctx, "2015 - 2026 ფაქტიური ბალანსი.xlsx")
if err != nil {
    return err
}

fmt.Println(book.Sheets[11].Name) // 2026 GEO
```

## The API

| Method     | Request              | Answer                                         |
| ---------- | -------------------- | ---------------------------------------------- |
| `Links`    | `GET /`              | Each link of the home page, with its text      |
| `Workbook` | `GET /excel/<name>`  | Each sheet, and each cell with a value as text |

The client mirrors what the site holds today:

- The name of a file holds its range of years: `2015 - 2026 ფაქტიური ბალანსი.xlsx`. It
  changes when GNERC adds a year. The home page links the current name.
- The server sends no intermediate certificate. `NewTransport` trusts the roots of the
  system and the embedded `Go Daddy Secure Certificate Authority - G2`. The default
  client uses it.
- A cell holds a binary float as text: `388.95957600000003`. The client keeps the text.
- A shared string keeps its spaces, the non-breaking ones included: `" 12.304 "`.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit. Give it the transport of
`NewTransport`.
