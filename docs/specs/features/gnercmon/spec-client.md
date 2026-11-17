---
id: GNERCMON
title: The monthly electricity balance and prices of GNERC, the client
type: spec
status: approved
created: 2026-11-17
updated: 2026-11-17
context: energy
approved_by: Temuri
approved_on: 2026-11-17
constrained_by: [CLT, DOM, RSL]
requirements: features/gnercmon/requirements.md
---

# Specification: The monthly electricity balance and prices of GNERC, the client

## 1. `pkg/gnerc`

```go
const DefaultBaseURL = "https://data.gnerc.org"

var (
    ErrUnexpectedStatus = errors.New("gnerc: unexpected status")
    ErrNotWorkbook      = errors.New("gnerc: not a workbook")
)

type APIError = vendorhttp.StatusError

// NewTransport returns a transport that trusts the roots of the system and the
// intermediate certificate that GNERC does not send. The default client of New uses it.
func NewTransport() *http.Transport

// Links returns each link of the home page, in the order of the page.
func (c *Client) Links(ctx context.Context) ([]Link, error)
// Workbook returns the workbook of the file name in the folder excel.
func (c *Client) Workbook(ctx context.Context, name string) (Workbook, error)

type Link struct {
    Href string // "https://data.gnerc.org/excel/2015 - 2026 ფაქტიური ბალანსი.xlsx", as the page writes it
    Text string // "დეტალური მონაცემების გადმოწერა", with its spaces collapsed
}

type Workbook struct {
    LastModified string  // the header Last-Modified, or empty
    Sheets       []Sheet // in the order of the workbook
}

type Sheet struct {
    Name string // "2026 GEO"
    Rows []Row  // the rows that hold a cell, in order
}

type Row struct {
    Number int    // 1 for the first row
    Cells  []Cell // in order
}

type Cell struct {
    Column string // "C"
    Type   string // the attribute t: "s", "n", "str", "b", "e", or empty for a number
    Text   string // the text of a shared string, or the text of <v>: "388.95957600000003"
}
```

`Links` sends `GET /` and reads each `<a href>` with `golang.org/x/net/html`. `Workbook`
sends `GET /excel/<name>` with `Path: []string{"excel", name}`, so the client escapes the
spaces and the Georgian letters. The body limit is 4 MiB: the balance has 454 KB, and grows
by about 25 KB a year. `GNERCMON-DD-003` gives the decoding of a workbook.
`NewTransport` clones `http.DefaultTransport` and sets `RootCAs` to `x509.SystemCertPool()`
with the embedded `godaddy-g2.pem` appended. `GNERCMON-DD-002`.

## 2. `adapter/gnerc`

`NewAdapter(client *gnercclient.Client, now func() time.Time)`.
`Balance` reads `Links`, then the workbook whose link matches the table below, then maps it.
`Prices` reads `Links`, then the three workbooks, one at a time. `GNERCMON-DD-001`.

| Workbook       | The base name of its link, with each run of spaces as one space, holds |
| -------------- | ---------------------------------------------------------------------- |
| Balance        | `ფაქტიური ბალანსი`                                                     |
| Balancing      | `საბალანსო ელექტროენერგიის ფასები`                                     |
| Capacity fee   | `გარანტირებული სიმძლავრის საფასური`                                    |
| Deregulated    | `დერეგულირებული სადგურების ფასები`                                     |

The adapter takes the base name from the path of `Href` with `url.Parse` and `path.Base`,
and passes it to `Workbook`. It ignores the host of `Href`. No link, or more than one
link, for a workbook is a failure.

A balance sheet is a sheet named `<year> GEO` with a year from 2022. The adapter ignores
every other sheet. In a sheet, the header row is the row whose cell `B` reads `დასახელება`.
Its cells name the columns: `იანვარი` to `დეკემბერი` for the months, `სულ` for the year
total. Each later row with a label in `B` that the table below holds gives one item. A
label is the text with each run of spaces as one space, trimmed, and with a trailing
footnote mark `(*)` to `(*****)` removed. The labels keep the spelling of GNERC.

| Label in column `B`                                                    | `Item`                    |
| ---------------------------------------------------------------------- | ------------------------- |
| `ელექტროენერგიის ჯამური გამომუშავება`                                  | `generation_total`        |
| `თბოელექტროსადგურების ჯამური გამომუშავება`                             | `generation_thermal`      |
| `ქარის ელექტროსადგურების ჯამური გამომუშავება`                          | `generation_wind`         |
| `მზის ელექტროსადგურების ჯამური გამომუშავება`                           | `generation_solar`        |
| `ჰიდროელექტროსადგურების ჯამური გამომუშავება`                           | `generation_hydro`        |
| `ელექტროენერგიის საერთო სასადგურო დანახარჯი`                           | `station_own_use`         |
| `ელექტროენერგიის ჯამური იმპორტი`                                       | `import_total`            |
| `რუსეთიდან`, `თურქეთიდან`, `აზერბაიჯანიდან`, `სომხეთიდან`              | `import_russia`, `import_turkey`, `import_azerbaijan`, `import_armenia` |
| `მ.შ. აფხაზეთში`                                                       | `import_into_abkhazia`    |
| `ელექტროენერგიის ჯამური მიწოდება`                                      | `supply`                  |
| `ელექტროენერგიის ჯამური მოხმარება`                                     | `consumption_total`       |
| `აფხაზეთი`                                                             | `consumption_abkhazia`    |
| `ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება`                       | `consumption_suppliers`   |
| `უნივერსალური მომსახურების მიმწოდებლების ჯამური მოხმარება`             | `consumption_universal_service` |
| `საჯარო მომსახურების სახით ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება` | `consumption_public_service` |
| `თავისუფალი მომხმარბლის სახით ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება` | `consumption_free_market` |
| `პირდაპირი მომხმარებლების ჯამური მოხმარება`                            | `consumption_direct`      |
| `გაჩერებული ელექტროსადგურების საკუთარი მოხამარება`                     | `consumption_stopped_stations` |
| `ელექტროენერგიის ჯამური ექსპორტი`                                      | `export_total`            |
| `რუსეთში`, `თურქეთში`, `აზერბაიჯანში`, `სომხეთში`                      | `export_russia`, `export_turkey`, `export_azerbaijan`, `export_armenia` |
| `ელექტროენერგიის გადამცემი და გამანაწილებელი ქსელის ელექტროენერგიის ჯამური დანაკარგი` | `losses_total` |
| `გადამცემი სისტემის ოპერატორის მიერ ქსელში დანაკარგების დაფარვის მიზნით შესყიდული ჯამური ელექტროენერგია` | `losses_transmission` |
| `გამანაწილებელი სისტემის ოპერატორების მიერ ქსელში დანაკარგების დაფარვის მიზნით შესყიდული ჯამური ელექტროენერგია` | `losses_distribution` |
| `საქართველოს შიდა მოხმარება`                                           | `domestic_consumption`    |
| `ელექტროენეგიის ჯამური ტრანზიტი`                                       | `transit_total`           |
| `აზერბაიჯანიდან თურქეთში`, `რუსეთიდან თურქეთში`, `სომხეთიდან თურქეთში` | `transit_azerbaijan_turkey`, `transit_russia_turkey`, `transit_armenia_turkey` |

A price sheet is the first sheet of its workbook. The header row is the row whose cell
`A` reads `თვე`. Its cells from `B` hold the years. Each later row whose cell `A` names a
month, `იანვარი` to `დეკემბერი`, gives the price of that month in each year.

| Cell                                   | Domain value                                          |
| -------------------------------------- | ----------------------------------------------------- |
| A value of a balance or a price        | The text with each space and each non-breaking space removed, then `decimal.NewFromString`, then `Round`. An empty cell in a row that exists is zero in a balance and absent in a price. Any other text is a failure. |
| A year of the header of a price sheet  | `strconv.Atoi` of the text. Any other text is a failure. |
| `Workbook.LastModified`                | `http.ParseTime`, in `kernel.TbilisiZone()`. Empty or invalid gives the zero time. |

`ReadAt` is `now()` in `kernel.TbilisiZone()` after the last workbook. The `Modified` of the
prices is the latest of the three workbooks.

## 3. Vendor endpoint

Every row: `pkg/gnerc`, no key, no: files that the official
regulator links from its page, with no documented format.

| Method and path                       | Realizes                                             |
| ------------------------------------- | ---------------------------------------------------- |
| `GET /`                               | `GNERCMON-FR-001` to `GNERCMON-FR-012` (`GNERCMON-DD-001`) |
| `GET /excel/<name of the balance>`    | `GNERCMON-FR-001` to `GNERCMON-FR-006`, `GNERCMON-FR-011`, `GNERCMON-FR-012` |
| `GET /excel/<name of each price>`     | `GNERCMON-FR-007` to `GNERCMON-FR-012`               |

Real answers, recorded on 2026-11-17, cut:

```
GET https://data.gnerc.org/
200 text/html; charset=UTF-8 (126506 bytes)
<a href="https://data.gnerc.org/excel/2015 - 2026 ფაქტიური ბალანსი.xlsx" ...>დეტალური მონაცემების გადმოწერა</a>
<a href="https://data.gnerc.org/excel/საბალანსო ელექტროენერგიის ფასები 2015 - 2026 წლები.xlsx" ...>
<a href="https://data.gnerc.org/excel/დერეგულირებული სადგურების ფასები 2015 - 2026 წლები.xlsx" ...>
<a href="https://data.gnerc.org/excel/გარანტირებული სიმძლავრის საფასური  2015 - 2026 წლები.xlsx" ...>

GET https://data.gnerc.org/excel/2015%20-%202026%20%E1%83%A4...%E1%83%98.xlsx
200 application/vnd.openxmlformats-officedocument.spreadsheetml.sheet (453839 bytes)
last-modified: Wed, 21 Oct 2026 09:45:18 GMT
xl/workbook.xml:     <sheet name="2026 GEO" sheetId="19" r:id="rId12"/>
xl/worksheets/sheet12.xml, row 4:
<c r="A4" s="482"><v>1</v></c><c r="B4" s="483" t="s"><v>320</v></c>
<c r="C4" s="484"><f t="shared" ref="C4:N4" si="0">C5+C11+C22+C15</f><v>910.16262400000005</v></c>
...
<c r="L4" s="485"><f t="shared" si="0"/><v>0</v></c>
<c r="O4" s="486"><f t="shared" ref="O4:O21" si="1">SUM(C4:N4)</f><v>10308.3816561</v></c>

GET https://data.gnerc.org/excel/%E1%83%93...xlsx (deregulated)
200 (10900 bytes); row 7, the year 2021: <c ... t="s"><v>..</v></c> -> " 12.304 "
```

The server sends the certificate `*.gnerc.org`, issued by `Go Daddy Secure Certificate
Authority - G2`, with no intermediate. `openssl s_client` gives the code 21. A Go client
with a pool that holds only that intermediate answered 200. An empty pool failed with
`x509: certificate signed by unknown authority`.

## Retired identifiers

This file has no retired identifier.
