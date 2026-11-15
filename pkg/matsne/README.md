# matsne

A Go client of the Legislative Herald of Georgia
([matsne.gov.ge](https://matsne.gov.ge)), the official publisher of the legal acts of
Georgia.

The Legislative Herald publishes no API. The client reads its web pages for people. The
pages can change with no notice, so this package can change in any release.

```
go get github.com/abgeo/georgia-mcp/pkg/matsne
```

```go
client := matsne.New()

doc, err := client.Document(ctx, "ka", "1043717", -1)
if err != nil {
    return err
}

fmt.Println(doc.Title)    // საქართველოს საგადასახადო კოდექსი
fmt.Println(doc.Selected) // 248
```

## The pages

| Method     | Request                                   | Answer                                                 |
| ---------- | ----------------------------------------- | ------------------------------------------------------ |
| `Search`   | `GET /{ka,en}/document/search`            | The total and each result of one page of 20            |
| `Suggest`  | `POST /document/suggest`, form `text`     | The titles that match a Georgian text, at most 50      |
| `Document` | `GET /{ka,en}/document/view/{id}`         | The metadata, the versions, and the paragraphs of an act |

The pages need no key. This client mirrors what the pages showed on 2026-11-15:

- A bot filter guards the site. It answers with an "Access Denied" page of the status
  200 to the TLS handshake of Go. The default client dials with the handshake of
  Chrome 131 over HTTP/1.1, with the User-Agent of this package. Use `NewTransport` if
  you pass your own `http.Client`. The client returns `ErrBlocked` for that page.
- The site gives the same page for an act or a version that does not exist.
- A Georgian page marks each part with a class: `karixml` a section, `tavixml` a chapter,
  `muxlixml` an article, `abzacixml` a paragraph. An English page is Word markup:
  `MsoNormal` paragraphs, with each article heading in bold.
- A sup becomes `^` and its text: `მუხლი 133^3` is article 133 with the superscript 3.
- A translation trails the Georgian versions. The English page with no version is the
  newest translated version. The English page of a version with no translation holds no
  paragraph.
- The client returns the text of the page. It does not map a status or parse a date: the
  meaning of the page belongs to the caller.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
