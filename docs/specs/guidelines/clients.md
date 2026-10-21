---
id: CLT
title: Vendor clients
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-21
scope: [pkg/]
related: [ARC, DOM, PKG, RSL, BLD]
---

# Vendor clients

A vendor client is a Go module that calls the API of one vendor, or reads its web
page. Another project can import it with no part of the server.

## CLT-001

A vendor client lives in `pkg/<vendor>`, with its own `go.mod`.
The module path is `github.com/abgeo/georgia-mcp/pkg/<vendor>`.
The package name is the vendor name: `nbg`, `ttc`. A second client of the same vendor
adds its subject to the name: `bogoffers`. `ADR-0004`.

## CLT-002

A client imports the standard library only. It imports no package of the server
module, no other client, and no MCP package. A third-party library needs a requirement
that the API of the vendor demands, for example a protocol that the standard library
does not offer. A client of a web page imports `golang.org/x/net/html`. No other client
imports it. `ADR-0003`.

**Why:** A dependency of a client is a dependency of every project that uses it.

## CLT-003

A client mirrors the API of the vendor. Its types carry the field names of the
vendor in their JSON tags. A client does not interpret the data. A client of a web
page returns the text of each cell of the data, as the page shows it. Its types carry
no JSON tag. A client of a page that sends its data as JSON in the body (a data stream
of a web framework) mirrors that JSON, with the field names of the page in its tags.

**Why:** The server interprets the data in its adapter. See `DOM-005`.

## CLT-004

The HTTP client is `net/http`. The client builds the address with `net/url`, and decodes
the body with `encoding/json` or `encoding/xml`. A client of a web page parses the body
with `golang.org/x/net/html`. A client of a data stream finds the JSON value in the
body and decodes it with `encoding/json`. It reads at most a fixed count of bytes from
a body.

**Why:** A GET with a query and two headers needs no library. The standard library has
no major version to migrate.

## CLT-005

The constructor is `New(opts ...Option) *Client`. An option sets one value:

| Option            | Default                                      |
| ----------------- | -------------------------------------------- |
| `WithBaseURL`     | The production address of the vendor.        |
| `WithHTTPClient`  | An `http.Client` with a timeout of 10 seconds. |
| `WithUserAgent`   | `georgia-mcp-<vendor>/<version> (+https://github.com/abgeo/georgia-mcp)`. |
| `WithAPIKey`      | None. Only a client whose vendor needs a key. |

The client sets `Accept: application/json` once, at construction. A client of a web
page sets `Accept: text/html`. A client of a data stream sets the `Accept` of the
stream and the header that the framework reads to send it, for example `RSC: 1`.

## CLT-006

Every method takes a `context.Context` as the first parameter.
Every request carries that context.

## CLT-007

A status that is not a success is an error. It is never an empty result.
The client returns `*APIError`, with the status code and the first 1 KiB of the body.
`APIError` wraps the sentinel `ErrUnexpectedStatus`, so `errors.Is` finds it.

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

## CLT-011

A client release is a tag `pkg/<vendor>/v<semver>`. A client stays at `v0`
while its vendor publishes no API documentation.

## Retired identifiers

This file has no retired identifier.
