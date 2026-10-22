# liberty

A Go client of the offers of Studio, the premium package of Liberty Bank. It covers the
offers API of studioliberty.ge and the page of each offer.

The API is unofficial. Liberty Bank publishes no documentation of it: the page
[studioliberty.ge/ka/studio-lifestyle/offers](https://studioliberty.ge/ka/studio-lifestyle/offers)
reads it. The page of an offer is a page for people. Both can change with no notice, so
the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/liberty
```

```go
client := liberty.New()

offers, err := client.Offers(ctx, liberty.OffersQuery{Page: 1, Language: "en"})
if err != nil {
    return err
}

for _, offer := range offers {
    fmt.Println(offer.Title, offer.Labels, offer.RemainingDays)
}
```

## The API

| Method       | Request                                                  | Answer                           |
| ------------ | -------------------------------------------------------- | -------------------------------- |
| `Categories` | `POST /_mvcapi/offersapi/loadcategories`                 | The categories of the offers     |
| `Types`      | `POST /_mvcapi/offersapi/loadtypes`                      | The types of the offers          |
| `Offers`     | `POST /_mvcapi/offersapi/loadOffers`                     | One page of 16 offers            |
| `OfferPage`  | `GET /<language>/studio-lifestyle/offers/details/<id>`   | The text of the page of an offer |

The API needs no key. This client mirrors what the site answered on 2026-10-20:

- A page holds 16 offers, from page 1. A page with less than 16 is the last.
- An offer has no category, no type, and no date. A list with one category or one type
  is the only way to know them.
- The labels are free text: "5% Cashback", "Up to 30% Discount".
- `RemainingDays` is a text: "72 Days left".
- Each answer wraps the list in `"succeeded"`. A false value is `ErrNoSuccess`.
- `OfferPage` parses the page with `golang.org/x/net/html` and returns the title, the
  days left, and the text of each paragraph of the terms.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
