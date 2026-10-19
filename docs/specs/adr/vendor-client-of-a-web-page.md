---
id: ADR-0003
title: A vendor client of a web page
type: adr
status: accepted
created: 2026-10-19
updated: 2026-10-19
decided: 2026-10-19
changes: [CLT-002, CLT-003, CLT-004, CLT-005]
supersedes:
superseded_by:
---

# A vendor client of a web page

## Context

`ROMFUEL` adds Rompetrol. Rompetrol publishes no API. Its home page holds the prices in
an HTML table, rendered by the server. No call of the site returns the prices as JSON.
The owner accepted the page as the source (`ROMFUEL` requirements, question 1).

The rules of `CLT` assume an API:

- `CLT-002` allows the standard library only, unless the API demands more.
- `CLT-003` mirrors the API in types with JSON tags.
- `CLT-004` decodes a body with `encoding/json` or `encoding/xml`.
- `CLT-005` sets `Accept: application/json`.

The standard library has no HTML parser. `encoding/xml` refuses most real HTML: the
page of Rompetrol has unclosed elements and `&nbsp;`.

## Decision

A vendor client of a web page parses the page with `golang.org/x/net/html`. It returns
the text of the cells that hold the data, and sets `Accept: text/html`.

## Rationale

- The Go team maintains `golang.org/x/net`. It follows the release policy of Go, and
  needs no other module.
- The parser follows the HTML5 algorithm of a browser. It reads the page as the
  browser of a person reads it.
- A regular expression over the page breaks on a new attribute, a new line, or an
  entity. Its failure is silent.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A regular expression with the standard library | It breaks on a small change of the markup, with no error. |
| `encoding/xml` with `Strict: false` | It does not build the tree of a browser. Its result on real HTML is not defined. |
| `github.com/PuerkitoBio/goquery` | A selector library on top of `x/net/html`, with a second dependency. One table needs no selector. |

## Consequences

The rules change as follows:

- `CLT-002`: a client of a web page imports `golang.org/x/net/html`. No other client
  imports it.
- `CLT-003`: a client of a web page returns the text of each cell of the data, as the
  page shows it. Its types carry no JSON tag.
- `CLT-004`: a client of a web page parses the body with `golang.org/x/net/html`.
- `CLT-005`: a client of a web page sets `Accept: text/html`.
- The glossary term "vendor client" reads "calls the API of one vendor, or reads its web
  page". A new term, "web page source", names a source that is a page for people.

- Positive: a vendor with no API can join the server.
- Negative: a client of a web page breaks when the vendor changes its markup. Its
  failure is an error, and the server reports the company as unavailable.
- Work that follows: `pkg/rompetrol` (`ROMFUEL`).
