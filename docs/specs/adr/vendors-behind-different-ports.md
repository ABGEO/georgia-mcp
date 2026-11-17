---
id: ADR-0007
title: Vendors of one context behind different ports
type: adr
status: accepted
created: 2026-11-17
updated: 2026-11-17
decided: 2026-11-17
changes: [PKG-003]
supersedes:
superseded_by:
---

# Vendors of one context behind different ports

## Context

`PKG-003` makes a context with more than one vendor add each vendor to a
`vendors.Registry`. The registry is generic over one source type `T`, and `Build` returns
a `[]T`. The contexts `rates`, `fuel`, and `offers` use it: each vendor gives the same
port.

`GNERCMON` adds GNERC to `energy`, next to GSE. GSE gives `GridSource`. GNERC gives
`BalanceSource` and `PriceSource`. No one type `T` holds both, and each vendor has its own
cache and its own tools. The review of `GNERCMON` found `module.go` outside `PKG-003`.

## Decision

`PKG-003` binds the registry to the vendors of one port. A context whose vendors give
different ports builds each vendor in `module.go` with its own switch, and logs
`<vendor> is off` with the variable of the switch, as the registry does.

## Rationale

The registry exists to join the sources of one port into one list, for a use case that
asks each source. Vendors of different ports have no such list. Two `if` blocks of 15 lines
show the wiring in one screen, which is the reason of `PKG-003`.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A registry of `io.Closer`, and a type switch on each source. | A type switch on a vendor in `module.go` hides a wrong source until the start. The compiler checks the two `if` blocks. |
| A registry for each vendor in `energy`. | A registry of one entry joins nothing. |

## Consequences

- Positive: the rule names the case that the registry serves.
- Negative: a context with vendors of different ports repeats the log record of a switch
  that is off.
- Work that follows:
  - Change `PKG-003`.
  - Examine every specification: only `energy` has vendors of different ports. `GNERCMON`
    section 3 cites this ADR.
