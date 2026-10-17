---
id: ADR-0001
title: Bounded contexts with ports and adapters
type: adr
status: accepted
created: 2026-10-17
updated: 2026-10-17
decided: 2026-10-17
changes: []
supersedes:
superseded_by:
---

# Bounded contexts with ports and adapters

## Context

Georgia MCP serves tools about several Georgian public services in one process:
the exchange rates of the NBG and of commercial banks, the public transport of the
TTC, the Legislative Herald, and the Public Service Development Agency.

The facts that shape the design:

- Each subject has its own words. A rate has a currency and a validity date. A stop
  has a route and an arrival time. The two share almost nothing.
- Each vendor is outside the control of the owner. Four of the five sources have no
  published API. The NBG and TBC Bank publish one. The TTC answers an API that its
  own web application uses. The Legislative Herald and the agency publish HTML only.
- The server keeps no record. Every tool reads public data and repeats it.
- The owner wants a vendor client to be useful to another project.
- One owner builds every tool. No third party adds a tool.

## Decision

The server is one binary with one bounded context for each subject. Each context
uses ports and adapters. The server applies strategic domain driven design in full,
and tactical domain driven design only where a context holds a rule. A vendor client
is a separate Go module with no dependency on the server.

## Rationale

**One binary, no plugin.** One owner builds every tool, so a plugin contract buys
nothing. The compiler checks every boundary that a plugin loader checks at run time.

**A bounded context for each subject.** A context holds its own language, so a
change to the transit model never touches the rates model. A context is also the
unit of the work: one requirements document opens a context or adds to one.

**Ports and adapters inside a context.** Each source changes without notice and
four have no API contract. A driven port in the words of the domain (`RateSource`)
keeps the vendor change inside one adapter. The MCP tool is a driving adapter, so
the use case has no MCP type and a test calls it with no protocol.

**Strategic design in full, tactical design on demand.** Most tools repeat a
published value. An aggregate, a repository, or a domain event in such a context is
code that no requirement asks for. The bounded context, the glossary, and the
anti-corruption layer cost little and protect the model from every vendor. A value
object with a validating constructor is the one tactical pattern that every context
uses, because each tool validates the input of a model.

**A vendor client as a module.** A client with its own `go.mod` has its own version
and imports only resty. Another project imports it with no MCP dependency.
The client mirrors the vendor. The server interprets the data in its adapter.

## Alternatives

| Alternative                                      | Why we did not select it                                                                          |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------- |
| A flat layout: one package for each vendor       | The MCP types, the vendor shapes, and the rules mix in one package. A vendor change reaches the tool. |
| Full tactical DDD in every context               | Aggregates and repositories with no rule and no store. Three layers of code for one HTTP call.    |
| A Go module for each context                     | Ten `go.mod` files for code that no other project imports. `internal/` gives the same boundary.   |
| A plugin system, as in a hub and spoke design    | One owner, one build. A loader and a version contract cost more than a rebuild.                    |
| A vendor client inside the server module         | Another project must import the whole server module, with the MCP SDK and every dependency.       |

## Consequences

- Positive: a vendor change stays inside one adapter. A use case is testable with a
  fake port and no network. A client is reusable alone.
- Positive: `golangci-lint` with `depguard` checks the import rules of `ARC-007`.
- Negative: a small tool has four packages: domain, app, the driven adapter, and the tool.
- Negative: a change to a vendor client needs a tag before `go install` sees it. `BLD-003`.
- Work that follows: the first feature builds `internal/pkg`, `internal/app/georgia-mcp/bootstrap`,
  and the first context, and proves the layout.
