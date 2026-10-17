---
id: LOG
title: Logging
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/pkg/logger/, internal/]
related: [MCP, API, LIF, LNG]
---

# Logging

## LOG-001

The logger is `log/slog`. The process builds one logger at the start and passes it
down. A package creates no logger of its own. A vendor client logs nothing.

## LOG-002

The `serve` command writes the log to stdout. The `stdio` command writes the log
to stderr.

**Why:** In the stdio transport, stdout carries the protocol. One log line there
breaks the client.

## LOG-003

The format is JSON, or text when the configuration says so.
The handler adds the source position only when the level is `debug`.

## LOG-004

A component adds two attributes with `With`: `component`, and a second attribute
that names the instance.

```go
logger.With(slog.String("component", "context"), slog.String("context", "rates"))
```

## LOG-005

An error is an attribute, `slog.Any("error", err)`. Never put an error inside the
message text.

## LOG-006

Use `InfoContext`, `ErrorContext`, or the matching variant when a context exists.

## LOG-007

A message is lowercase. It names the event, not the value. It obeys `LNG`.

## LOG-008

Never log a key, a secret, or the arguments of a tool call at a level above `debug`.

**Why:** A future tool can take a personal number. The rule must hold before it exists.

## LOG-009

The process writes one record for each tool call. The attribute names come from
the OpenTelemetry semantic conventions for MCP:

| Attribute         | Value                                                              |
| ----------------- | ------------------------------------------------------------------ |
| `mcp.method.name` | `tools/call`                                                       |
| `gen_ai.tool.name` | The name of the tool.                                             |
| `event.duration`  | The duration in nanoseconds.                                       |
| `error.type`      | Only on a failure: `tool_error` for a result with `IsError`, else the type of the Go error. |

**Why:** The conventions give a collector one name for each attribute across servers.

## LOG-010

The process writes one record for each HTTP request, except a probe of `/livez` or
`/readyz` that answers 200. `go-chi/httplog/v3` writes it, with the schema of the
OpenTelemetry semantic conventions. It logs no request body and no response body.

## Retired identifiers

This file has no retired identifier.
