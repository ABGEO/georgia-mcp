---
id: ADR-0006
title: A shared module of the vendor clients
type: adr
status: accepted
created: 2026-10-22
updated: 2026-10-22
decided: 2026-10-22
changes: [CLT-002, CLT-004, CLT-005, CLT-007]
supersedes:
superseded_by:
---

# A shared module of the vendor clients

## Context

The server has 19 vendor clients. `CLT-002` lets a client import the standard library
only, so each client wrote the same code: the options of `CLT-005`, the request, the
limit of the body, the `APIError` of `CLT-007`, and the version for the User-Agent. That
is about 130 lines in each client, 2,400 of the 5,600 lines of `pkg/`. A change to one
rule needs 19 changes.

## Decision

A module `pkg/vendorhttp`, with the standard library only, holds that code and the
options. Each client imports it. The option type of a client is `vendorhttp.Option`, so
the caller passes `vendorhttp.WithBaseURL` to any client. A client declares an option
only for a value of its own vendor, for example a key. It keeps `New`, `APIError`, and
`ErrUnexpectedStatus`.

## Rationale

- The repeated code is the contract of `CLT`, not the mechanics of HTTP. One module
  holds the contract one time.
- The module imports the standard library only, so a project that imports a client
  gets no third-party module.
- One option type lets a client extend the options with no copy of the shared ones:
  `tbc.New(vendorhttp.WithBaseURL(address), tbc.WithAPIKey(key))`.
- `errors.Is(err, gulf.ErrUnexpectedStatus)` does not change, and the message names the vendor.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| `github.com/go-resty/resty` | It replaces the request and the decode, about 30 of the 130 lines. The options, `APIError`, and the version stay in each client, and every project that imports a client gets a third-party module. |
| A wrapper of each shared option in each client | 60 functions that repeat the same 3 options, and a fourth for each new shared option. |
| No change | 19 copies of each rule. |

## Consequences

The rules change as follows:

- `CLT-002`: a client imports `pkg/vendorhttp`. It is the only module of this
  repository that a client imports.
- `CLT-004`: `vendorhttp.Client` sends the request, limits the body, and decodes JSON.
- `CLT-005`: the caller passes the options of `vendorhttp`. A client declares only the
  option of a value of its own vendor.
- `CLT-007`: a client with no field of its own in its error can declare
  `vendorhttp.StatusError` as its `APIError`. Another client keeps its own type.

- Positive: one place for the request, the limit, the error, and the User-Agent.
- Negative: a client release needs a release of `pkg/vendorhttp` first. A caller of a
  client imports `pkg/vendorhttp` for the shared options. The API of each client changes.
- Work that follows: `pkg/vendorhttp`, and each of the 19 clients (`PLATFORM-FR-026`).
