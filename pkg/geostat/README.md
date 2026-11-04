# geostat

A Go client of the PxWeb API of Geostat, the National Statistics Office of Georgia.
It lists the folders and the tables of the database, describes a table, and queries
its cells.

```
go get github.com/abgeo/georgia-mcp/pkg/geostat
```

```go
client := geostat.New()

dataset, err := client.Query(ctx, "en",
    "Price Statistics/CPI/National Consumer Price Index/CPI_in_Georgia_(previous_month=100)_orenovani2.px",
    []geostat.Selection{
        {Code: "Region", Values: []string{"0"}},
        {Code: "Groups", Values: []string{"0"}},
        {Code: "Year", Values: []string{"22"}},
        {Code: "month", Values: []string{"9"}},
    })
if err != nil {
    return err
}

fmt.Println(dataset.Data[0].Values[0]) // 100.9
```

## The API

| Method   | Request                              | Answer                                     |
| -------- | ------------------------------------ | ------------------------------------------ |
| `Folder` | `GET /{en\|ka}/Database/{folder}/`   | The folders and the tables of the folder   |
| `Table`  | `GET /{en\|ka}/Database/{table}/`    | The title and the dimensions of the table  |
| `Query`  | `POST /{en\|ka}/Database/{table}/`   | The cells of a selection                   |

The client mirrors what the API answers today:

- **The server answers 404 to a path whose last segment holds a dot, unless the path
  ends with a slash.** The client ends each path with a slash.
- The codes of a dimension and of its values are the same in `ka` and in `en`. The
  language `ru` answers 404.
- The values of a dimension go from the oldest to the newest. The filter `top` of
  Geostat returns the first values, so the client sends the filter `item` only.
- A figure that Geostat did not publish is `..`.
- The answer of a query starts with a byte order mark. The client removes it.
- An unknown code in a query answers 404 with `{"error":"Parameter error"}`.
- The search of the API uses an index of 2024. Its paths answer 404 today. The client
  does not cover it.
- A table has no unit field. The title often names the unit.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
