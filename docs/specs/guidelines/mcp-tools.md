---
id: MCP
title: The MCP server and its tools
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-18
scope: [internal/pkg/mcpserver/, internal/app/georgia-mcp/*/adapter/tool/]
related: [ARC, DOM, API, LOG, RSL]
---

# The MCP server and its tools

## MCP-001

The MCP library is the official Go SDK, `github.com/modelcontextprotocol/go-sdk`.
The process builds one `*mcp.Server`. Its name is `georgia-mcp`. Its version comes
from the build information. See `BLD-004`.

## MCP-002

A tool name has the form `<context>_<verb>_<object>`, in snake case, with 64
characters maximum. The verb comes from this table.

| Verb      | Use                                        |
| --------- | ------------------------------------------ |
| `get`     | One item, or one set that has one key.     |
| `list`    | A collection with a bound.                 |
| `search`  | A collection that a text query filters.    |
| `convert` | A calculation over published values.       |
| `plan`    | A composed answer, for example a trip.     |

Example: `rates_get_official_rates`, `transit_list_arrivals`.

**Why:** A client shows tools from many servers in one list. The prefix groups them.

## MCP-003

A tool registers with the generic `mcp.AddTool`. The input and the output are
structs in `adapter/tool`. The SDK derives both schemas from them.

Every field carries a `json` tag in snake case and a `jsonschema` tag with its
description. A field that is not required carries `omitempty`.

**Why:** The model reads the schema. A field with no description is a field that
the model guesses.

## MCP-004

A tool declares a title, a description, and these annotations:

| Annotation        | Value   |
| ----------------- | ------- |
| `ReadOnlyHint`    | `true`  |
| `IdempotentHint`  | `true`  |
| `DestructiveHint` | `false` |
| `OpenWorldHint`   | `true`  |

A tool that writes to a vendor needs an ADR.

The description is in English. It states what the tool returns, when to use it,
the unit of each number, and the source of the data.

## MCP-005

A tool returns structured content that matches its output schema.
The output carries `source` (the name of the vendor) and `as_of` (the instant that
the source published the data, or that the server read it). An output that joins
several sources carries `source` and `as_of` for each source.

**Why:** A model that cites a rate must cite its source and its date.

## MCP-006

A failure that the model can report or correct returns a tool result with
`IsError` set. The message is in English. It names the cause and the next action.
A protocol error is for a defect of the server only.

The specification maps each sentinel of `DOM-011` to its message.
A message never holds a vendor body, a vendor status code, or a Go error text.

| Sentinel               | Message form                                                  |
| ---------------------- | ------------------------------------------------------------- |
| A validation error     | `"<field>: <problem>. <the accepted values>."`                |
| `kernel.ErrSourceUnavailable` | `"<source> did not answer. Try again in a few minutes."` |

## MCP-007

A tool whose output holds a text from a source takes the optional input
`language`, with the values `ka` and `en`. The default is `en`.

## MCP-008

A tool that returns a collection takes the optional input `limit`.
The specification gives the default and the maximum. The maximum is 100 or less.

**Why:** A large answer fills the context window of the model.

## MCP-009

A monetary amount or a rate is a JSON string that holds the decimal value.
A date is `YYYY-MM-DD`. An instant is RFC 3339 with the offset.

**Why:** A JSON number passes through a binary float in most clients. See `DOM-006`.

## MCP-010

`internal/pkg/mcpserver` builds the server and its registry. `bootstrap` passes
each context module to `Registry.Register`. The HTTP transport and the stdio transport serve that same server.

## MCP-011

The MCP server applies one middleware to every tool call. It writes the log record
of `LOG-009`.

## Retired identifiers

This file has no retired identifier.
