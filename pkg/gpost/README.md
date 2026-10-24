# gpost

A Go client of the postal codes of Georgian Post. It searches the postal code of an
address, a street, a town, or a village of Georgia.

Georgian Post publishes no API. The client reads the search of the page
[www.gpost.ge/help/postal-codes](https://www.gpost.ge/help/postal-codes), a page for
people. The page can change with no notice, so the module stays at `v0`.

```
go get github.com/abgeo/georgia-mcp/pkg/gpost
```

```go
client := gpost.New()

rows, err := client.PostalCodes(ctx, "ამბერკი", "ka")
if err != nil {
    return err
}

for _, row := range rows {
    fmt.Println(row.Code, row.Address, row.District)
}
```

## The page

| Method        | Requests                                             | Answer                                   |
| ------------- | ---------------------------------------------------- | ---------------------------------------- |
| `PostalCodes` | `GET /help/postal-codes`, then `POST /Help/FindPostalCode` | Each address of the search, in page order |

The page needs no key. This client mirrors what the page showed on 2026-10-24:

- The search needs the anti-forgery token of a visit to the page: the value of the form
  `__AjaxAntiForgeryForm` and the cookie of the same answer. Each call of `PostalCodes`
  visits the page again. The client keeps no token.
- The cookie `lang` selects the language of the answer: `ka` or `en`. The search
  matches a text in Georgian or in English in both languages.
- The answer groups the addresses under a district: a city or a municipality. Each
  address has a postal code of 4 digits as text: `0186`.
- Georgian Post gives a postal code for each street in Tbilisi, Batumi, and Kutaisi only.
  For another town or a village, search its name.
- A search with no match gives an empty list. A page with no token gives `ErrNoToken`.
  An answer that the client cannot read gives `ErrMalformedResult`.

The client parses the page with `golang.org/x/net/html`. It does not check a postal code,
does not retry, and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit. Search only on the demand
of a person: Georgian Post publishes no limit.
