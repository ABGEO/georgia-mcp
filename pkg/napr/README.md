# napr

A Go client of the cadastral map of NAPR, the National Agency of Public Registry of
Georgia, at `maps.gov.ge`. It finds a parcel by its cadastral code, the parcels near a
point, and addresses by a text.

The methods serve the map page of NAPR and have no documentation.

```
go get github.com/abgeo/georgia-mcp/pkg/napr
```

```go
client := napr.New()

results, err := client.Search(ctx, "01.15.05.066.020")
if err != nil {
    return err
}

label, _ := napr.Label(results[0].Details.InfoLink)

infos, err := client.Parcel(ctx, label)
if err != nil {
    return err
}

fmt.Println(infos[0].Detail.AreaMeasured, infos[0].Detail.Unit) // 618 კვ.მ.
```

## The API

| Method         | Request                                               | Answer                              |
| -------------- | ----------------------------------------------------- | ----------------------------------- |
| `Search`       | `POST /map/portal/search`                             | A parcel, or addresses              |
| `ParcelsAt`    | `GET /lr/bo/mg/getinfo2`                              | The parcels near a point            |
| `Parcel`       | `GET /lr/bo/mg/getinfo.alpha?lbl=...&fmt=json`        | The detail, owners, and documents   |
| `ParcelShape`  | `GET /lr/bo/mg/getinfo.alpha?lbl=...&res=shp`         | The shape of a parcel, WKT          |
| `AddressShape` | `GET /ar/mg/getinfo?lbl=...&res=shp`                  | The shape of an address, WKT        |

The client mirrors what the map answers today:

- Every text is Georgian, with `lang=en` too.
- A label must go with no encoding. An encoded `:` or `/` gives no data, with the status 200.
- The answer of `ParcelsAt` is not valid JSON when an address holds a quotation mark.
  The client escapes it before the decode.
- A search for an address gives addresses, not parcels. The order is poor.
- An empty keyword gives an "Access Denied" page. The client refuses it.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
