# tbcoffers

A Go client of the marketing API of TBC Bank. It covers the card offers, their filters,
and the page of one offer.

The API is unofficial. TBC Bank publishes no documentation of it: the pages
[tbcbank.ge/ka/offers](https://tbcbank.ge/ka/offers) and
[tbcconcept.ge/ge/offers](https://tbcconcept.ge/ge/offers) read it. The API can change
with no notice, so this package can change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/tbcoffers
```

```go
client := tbcoffers.New()

page, err := client.Offers(ctx, tbcoffers.OffersRequest{Locale: "en-US", PageSize: 1000})
if err != nil {
    return err
}

for _, entry := range page.List {
    fmt.Println(entry.Title, entry.Description)
}
```

## The API

| Method    | Request                                               | Answer                         |
| --------- | ----------------------------------------------------- | ------------------------------ |
| `Filters` | `GET /filters?type=offer&locale=<l>&segment=<s>`      | The filters of one segment     |
| `Offers`  | `POST /entries/offer` with a JSON body                | One page of offers             |
| `Offer`   | `GET /entries/offer/<slug>/<locale>`                  | One offer with its terms       |

The API needs no key. This client mirrors what the API answered on 2026-10-20:

- A list with no segment holds every offer: 518. A page size of 1000 holds them all.
- A list holds no category, no kind of benefit, and no card. A list with one filter,
  `"Category:Leisure"`, gives the offers of that filter.
- The filters need a segment: `Concept`, `ForEveryone`, or `ForYouth`. With none, the
  API answers 400.
- A segment has an id that stays the same in each locale, and a label that does not.
- A date has no zone: `2026-10-20T00:00:00`. Many offers have no start date or no end date.
- The page of one offer holds its terms as a tree of nodes.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
