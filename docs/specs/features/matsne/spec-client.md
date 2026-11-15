---
id: MATSNE
title: The legal acts of the Legislative Herald, the client
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: law
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [CLT, DOM]
requirements: features/matsne/requirements.md
---

# Specification: The legal acts of the Legislative Herald, the client

## The client

The shape of `pkg/gulf`. `New` sets `Accept: text/html`, the transport of
`NewTransport`, and `vendorhttp.WithMaxBody(16 << 20)`: the page of the Tax Code was 4.9 MB
on 2026-11-09. `NewTransport` copies the transport of `pkg/gulf` (`CLT-002` forbids the import).

```go
const DefaultBaseURL = "https://matsne.gov.ge"

var (
    ErrUnexpectedStatus = errors.New("matsne: unexpected status")
    ErrBlocked          = errors.New("matsne: the bot filter refused the request")
    ErrMalformedPage    = errors.New("matsne: the page has an unknown shape")
)

type APIError = vendorhttp.StatusError

// Search returns one page of results of GET /{language}/document/search.
func (c *Client) Search(ctx context.Context, language string, query SearchQuery) (SearchPage, error)
// Suggest returns the titles of POST /document/suggest with the form field text.
func (c *Client) Suggest(ctx context.Context, text string) ([]Suggestion, error)
// Document returns GET /{language}/document/view/{id}, with ?publication= when publication >= 0.
func (c *Client) Document(ctx context.Context, language, id string, publication int) (Document, error)

// SearchQuery holds the parameters of the page. An empty field is not sent.
type SearchQuery struct {
    Query, QueryTarget, Type, Group, Status, Issuer, Number string
    SigningFrom, SigningTo string // "dd/mm/yyyy"
    RegistrationCode, Sort      string
    Page                        int // from 1; the page sends page=Page-1 when Page > 1
}

type SearchPage struct {
    Total string       // the digits of "სულ მოიძებნა: 29 ( ... )" or "Search count: 1 ( ... )"
    Items []SearchItem
}

type SearchItem struct {
    ID     string   // from the link /document/view/{id}
    Title  string
    Class  string   // "panel panel-success document-search-result-item"
    Fields []string // the texts of the small elements of the panel body, with no "●" and no link
}

type Suggestion struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    State string `json:"state"`
}

type Document struct {
    Title        string        // the head cell of table.table-info, the metadata table
    Meta         []MetaRow     // each row with two cells
    Publications []Publication // the options of #publication-switcher, in the order of the page
    Selected     string        // the value of the selected option
    Paragraphs   []Paragraph   // each p of div#maindoc, in order; no style, no script
}

type MetaRow struct{ Label, Value string }
type Publication struct{ Value, Text string } // "248", "28/10/2026"

type Paragraph struct {
    Class string // "muxlixml", "MsoNormal"
    Text  string // white space folded; a sup gives "^" and its digits, an invisible sup nothing, and a space at the end of a sup stays
    Bold  bool   // every letter of the text sits inside b or strong
}
```

Each method reads the title of the answer first. The title `Access Denied` gives
`ErrBlocked`. A page with no metadata table, or a search page with no total, gives
`ErrMalformedPage`.

## The adapter

 `NewAdapter(client *matsneclient.Client, now func() time.Time)`.

| Vendor field                          | Domain value                                                  |
| ------------------------------------- | ------------------------------------------------------------- |
| `SearchItem.ID`, `Suggestion.ID`      | `domain.ParseActID`.                                          |
| `SearchItem.Class`                    | `panel-success` active, `panel-danger` expired, `panel-warning` pending, other the zero status. |
| `SearchItem.Fields`                   | Type, issuer, number, signing date, in this order. `MATSNE-DD-011`. |
| `Suggestion.State`                    | `active`, `expired`, `toBeActivated` to the three statuses.   |
| `MetaRow` by its label               | `დოკუმენტის ნომერი` or `Document number`: the number. `დოკუმენტის მიმღები` or `Document issuer`: the issuer. `მიღების თარიღი` or `Date of issuing`: the signing date. `დოკუმენტის ტიპი` or `Document type`: the type. `გამოქვეყნების წყარო, თარიღი` or `Source and date of publishing`: the date of version 0 when the page has no `Publication` (`MATSNE-DD-017`). `სარეგისტრაციო კოდი` or `Registration code`: the registration code. The adapter ignores any other row. |
| `Paragraph` with an empty text        | Left out.                                                     |
| `Publication`                         | `domain.Version`. The number from `Value`, the date from `Text`. No `Publication` gives one version 0 (`MATSNE-DD-017`). |
| `Paragraphs`                          | `domain.Body`. `MATSNE-DD-002`, `MATSNE-DD-003`.               |
| A date `dd/mm/yyyy`                   | `kernel.Date`.                                                |

`DocumentType` maps to `Group` with this table: `law` 1000003, `parliament_resolution` 1333937,
`presidential_decree` 4829717, `government_resolution` 458, `government_decree` 1065776,
`edict` 1000005, `order` 1000006, `international_treaty` 1319499,
`constitutional_court_decision` 1093052, `court_act` 1176355, `self_government_ordinance`
1000008, `self_government_decision` 1000007, `revenue_service_order` 1365174,
`individual_legal_act` 1151616, `eu_law` 3591321, `draft` 1776650.
`ActStatus` maps to `Status`: `active`, `expired`, `toBeActivated`.

## Real answers

Read through a browser on 2026-11-15, cut:

```
POST https://matsne.gov.ge/document/suggest   text=საგადასახადო კოდექსი
200 application/json (45 items)
[{"id":"32842","title":"საქართველოს საგადასახადო კოდექსი","state":"expired"},
 {"id":"31690","title":"საქართველოს საგადასახადო კოდექსი","state":"expired"},
 {"id":"1043717","title":"საქართველოს საგადასახადო კოდექსი","state":"active"}, ...]

GET https://matsne.gov.ge/ka/document/search?query=შრომის კოდექსი&type=main&status=active&group=1000003
200 text/html. <small>სულ მოიძებნა: N ( 0.01 წამში )</small>. Each result:
<div class="panel panel-success document-search-result-item">
  <div class="panel-heading"><p><a href="/ka/document/view/{id}">... <b>შრომის კოდექსი</b> ...</a></p></div>
  <div class="panel-body"><ul class="list-inline"><li><small>საქართველოს ორგანული კანონი</small></li>
    <li><small>●</small></li><li><small>საქართველოს პარლამენტი</small></li><li><small>●</small></li>
    <li class="active"><small>N101-IIIრს-Xმპ</small></li><li><small>●</small></li>
    <li class="active"><small>28/01/2021</small></li></ul></div></div>

GET https://matsne.gov.ge/ka/document/search?registration_code=200000000.05.001.016012&type=main
200 text/html. One result, panel-success, 1043717.

GET https://matsne.gov.ge/ka/document/view/1043717
200 text/html (4.9 MB). Metadata table: th "საქართველოს საგადასახადო კოდექსი", then rows
"დოკუმენტის ნომერი | 3591", "დოკუმენტის მიმღები | საქართველოს პარლამენტი",
"მიღების თარიღი | 17/09/2010", "დოკუმენტის ტიპი | საქართველოს კანონი",
"გამოქვეყნების წყარო, თარიღი | სსმ, 54, 12/10/2010",
"სარეგისტრაციო კოდი | 200000000.05.001.016012". 249 options in #publication-switcher,
"248 28/10/2026" to "0 12/10/2010". Body: p.karixml "კარი I", p.karisataurixml
"ზოგადი დებულებანი", p.tavixml "თავი I", p.tavisataurixml "საქართველოს საგადასახადო სისტემა",
p.muxlixml "მუხლი 1. საქართველოს საგადასახადო კოდექსის რეგულირების სფერო", p.abzacixml.
Counts: 15 karixml, 45 tavixml, 367 muxlixml, 4,378 abzacixml.

GET https://matsne.gov.ge/en/document/view/1043717
200 text/html (3.0 MB). Title "TAX CODE OF GEORGIA". div.document > div.WordSection1 > p.MsoNormal.
"SECTION I" then "GENERAL PROVISIONS"; "Chapter I – Georgian Tax System";
bold "Article 1 – Scope of regulation of the Tax Code of Georgia"; "Article 133<sup>3</sup> – ...".
366 bold article headings.

GET https://matsne.gov.ge/en/document/view/5441237
200 text/html. A Georgian title and text under the English page. No error.

Any path from a client with the handshake of Go, on 2026-11-15:
200 text/html (840 bytes). <title>Access Denied</title>
```

## Retired identifiers

This file has no retired identifier.
