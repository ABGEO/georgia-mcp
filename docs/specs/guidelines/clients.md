---
id: CLT
title: Vendor clients
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-25
scope: [pkg/]
related: [ARC, DOM, PKG, RSL, BLD]
---

# Vendor clients

A vendor client is a Go package that calls the API of one vendor, or reads its web
page. Another project can import it with no part of the server.

## CLT-001

A vendor client lives in `pkg/<vendor>`, as a package of the server module. It has no
`go.mod` of its own. The import path is `github.com/abgeo/georgia-mcp/pkg/<vendor>`.
The package name is the vendor name: `nbg`, `ttc`. A second client of the same vendor
adds its subject to the name: `bogoffers`. `ADR-0004`.

## CLT-002

A client imports the standard library and `pkg/vendorhttp` only. It imports no package
under `internal/`, no other client, and no MCP package. `ADR-0006`. A third-party library needs a requirement
that the API of the vendor demands, for example a protocol that the standard library
does not offer. A client of a web page imports `golang.org/x/net/html`. No other client
imports it. `ADR-0003`. A client of a vendor whose filter refuses the TLS handshake of Go
imports `github.com/refraction-networking/utls`. No other client imports it.

**Why:** A dependency of a client is a dependency of every project that uses it.

## CLT-003

A client mirrors the API of the vendor. Its types carry the field names of the
vendor in their JSON tags. A client does not interpret the data. A client of a web
page returns the text of each cell of the data, as the page shows it. Its types carry
no JSON tag. A client of a page that sends its data as JSON in the body (a data stream
of a web framework) mirrors that JSON, with the field names of the page in its tags.

**Why:** The server interprets the data in its adapter. See `DOM-005`.

## CLT-004

A client sends each request with `vendorhttp.Client`. It builds the address, sets the
headers, limits the body, and decodes JSON. The HTTP client is `net/http`. A client
decodes another body with `encoding/json` or `encoding/xml`. A client of a web page parses the body
with `golang.org/x/net/html`. A client of a data stream finds the JSON value in the
body and decodes it with `encoding/json`. It reads at most a fixed count of bytes from
a body.

A client of a vendor whose filter refuses the TLS handshake of Go exports `NewTransport`.
Its transport dials each TLS connection with the handshake of one fixed browser, over
HTTP/1.1. The default `http.Client` of the client uses it.

**Why:** A GET with a query and two headers needs no library. The standard library has
no major version to migrate.

## CLT-005

The constructor is `New(opts ...Option) *Client`, and `Option` is `vendorhttp.Option`.
The caller sets a value that every client has with the option of `vendorhttp`. A client
does not declare that option again. An option sets one value:

| Option                       | Default                                      |
| ---------------------------- | -------------------------------------------- |
| `vendorhttp.WithBaseURL`     | `DefaultBaseURL` of the client.              |
| `vendorhttp.WithHTTPClient`  | An `http.Client` with a timeout of 10 seconds. |
| `vendorhttp.WithUserAgent`   | `georgia-mcp-<vendor>/<version of the server> (+https://github.com/abgeo/georgia-mcp)`. |
| `WithAPIKey` of the client   | None. Only a client whose vendor needs a key. |

A value of one vendor is an option of its client that returns a `vendorhttp.Option`.

```go
client := tbc.New(vendorhttp.WithBaseURL(address), tbc.WithAPIKey(key))
```

The client sets `Accept: application/json` once, at construction. A client of a web
page sets `Accept: text/html`. A client of a data stream sets the `Accept` of the
stream and the header that the framework reads to send it, for example `RSC: 1`.

## CLT-006

Every method takes a `context.Context` as the first parameter.
Every request carries that context.

## CLT-007

A status that is not a success is an error. It is never an empty result.
The client returns `*APIError`, with the status code and the first 1 KiB of the body.
`APIError` wraps the sentinel `ErrUnexpectedStatus`, so `errors.Is` finds it. A client
whose error has no field of its own can declare `type APIError = vendorhttp.StatusError`.

## CLT-008

A client does not retry and does not cache. The caller decides both. See `RSL`.

**Why:** A retry inside a library hides the latency from the caller.

## CLT-009

A client holds no key in its source. The caller passes the key with `WithAPIKey`.
A vendor that publishes no API is "unofficial". The package comment and the
`README.md` of such a client say so.

## CLT-010

A client has a `README.md` with one example, and an `example_test.go` with an
`Example` function. A test of the client uses `httptest` and a recorded response
in `testdata/`. See `TST-005`.

## Retired identifiers

| ID      | Retired    | Reason                                                              |
| ------- | ---------- | ------------------------------------------------------------------- |
| CLT-011 | 2026-10-25 | A client is a package of the server module. A release of the server covers it. |
