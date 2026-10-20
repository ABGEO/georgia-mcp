# bogoffers

A Go client of the offers API of Bank of Georgia. It covers the list of card offers.

The API is unofficial. Bank of Georgia publishes no documentation of it: the page
[bankofgeorgia.ge/ka/offers-hub/catalog](https://bankofgeorgia.ge/ka/offers-hub/catalog)
reads it. The API can change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/bogoffers
```

```go
client := bogoffers.New()

page, err := client.Offers(ctx, bogoffers.OffersQuery{Language: "en", PageSize: 100})
if err != nil {
    return err
}

for _, offer := range page.Result.Offers {
    fmt.Println(offer.Title, offer.ShortDesc)
}
```

## The API

| Method   | Request                                                   | Answer           |
| -------- | --------------------------------------------------------- | ---------------- |
| `Offers` | `POST /get-offers?pageInfo=true&pageNumber=N&pageSize=M`, body `{}` | One page of offers |

The API needs no key. This client mirrors what the API answered on 2026-10-20:

- A page holds at most 100 offers. A larger page fails with the status 500.
- `pageNumber` counts from 0. `totalPageCount` gives the count of pages.
- The header `Accept-Language: en` gives the texts in English. With no header, they
  are in Georgian.
- `SegmentTypes` holds `RETAIL`, `SOLO`, and `WM` (Wealth Management), separated by `;`.
- `ProductCodes` holds the codes of the cards and the packages: `SOLO`, `AMEX`.
- `StartDate` and `EndDate` are milliseconds of the Unix epoch.
- `LongDesc` holds the terms in HTML.
- A `Code` other than 0 tells a failure of the vendor.

The client does not retry and does not cache. Pass an `http.Client` with
`WithHTTPClient` to set a timeout or a rate limit.
