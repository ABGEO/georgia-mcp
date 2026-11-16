# gse

A Go client of the real time exports of Georgian State Electrosystem (GSE), the
operator of the transmission grid of Georgia, as its site `gse.com.ge` reads them. It
returns the grid of each 3 minutes and the frequency of each minute.

The exports are unofficial. GSE publishes no documentation of their format.

```
go get github.com/abgeo/georgia-mcp/pkg/gse
```

```go
zone, _ := time.LoadLocation("Asia/Tbilisi")
client := gse.New()

grid, err := client.Grid(ctx, time.Date(2026, time.November, 16, 12, 0, 0, 0, zone))
if err != nil {
    return err
}

fmt.Println(grid.Header.Time) // 16.11.2026 12:00:00
```

## The API

| Method      | Request                                                        | Answer                          |
| ----------- | -------------------------------------------------------------- | ------------------------------- |
| `Grid`      | `GET /storage/uploads/fact/GSE_EXPORT_NEW_PERIOD_03M_<stamp>.XML` | The generation, the consumption, and the flows, in MW |
| `Frequency` | `GET /storage/uploads/fact/GSE_EXPORT_PERIOD_01M_<stamp>.XML`  | The frequency, in Hz            |

The client mirrors what the files hold today:

- The stamp is a local time of Tbilisi, `DDMMYYYY_HHMM`. The minute of a grid file is a
  multiple of 3. The client formats the time as it is, in its own location.
- A file appears 1 to 2 minutes after its time. GSE keeps it for 31 days. A file that
  does not exist answers 404.
- The file declares ISO-8859-1, a stylesheet, and a DTD. The stylesheet and the DTD
  answer 404.
- GSE pads the name of a block with spaces. The client keeps them.
- A value has 6 decimals: `18.299999`, `-0.000000`. The client keeps its text.
- Each value carries a quality code: `ACT`, `SUB`, `NRE`, `BLS`. GSE documents none.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
