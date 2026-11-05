---
id: LOG
title: Logging
type: guideline
status: active
created: 2026-10-17
updated: 2026-11-06
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
| `error`           | Only when the failure wraps `kernel.ErrSourceUnavailable`: the `Cause` of the `ToolError`, never its message. |

The level is `warn` when the cause wraps `kernel.ErrSourceUnavailable`, else `info`.

**Why:** The conventions give a collector one name for each attribute across servers.
A failed vendor is a fault of the server to watch. A wrong argument is not.

## LOG-010

The process writes one record for each HTTP request, except a probe of `/livez` or
`/readyz` that answers 200. `go-chi/httplog/v3` writes it, with the schema of the
OpenTelemetry semantic conventions. It logs no request body and no response body.

## LOG-011

The logger adds two attributes to every record. The attribute names come from the
OpenTelemetry semantic conventions for a resource:

| Attribute         | Value                          |
| ----------------- | ------------------------------ |
| `service.name`    | `georgia-mcp`                  |
| `service.version` | The version of the build.      |

**Why:** A collector holds the records of many services and many versions. The two
attributes tell them apart.

## LOG-012

The process writes one record for each request to a vendor, with the message
`vendor request`. The HTTP client of `internal/pkg/httpclient` writes it, so a vendor
client stays silent (`LOG-001`). The attribute names come from the OpenTelemetry
semantic conventions for an HTTP client:

| Attribute                   | Value                                                          |
| --------------------------- | -------------------------------------------------------------- |
| `vendor`                    | The name of the vendor: `Lukoil`.                              |
| `http.request.method`       | The method.                                                    |
| `server.address`            | The host of the vendor.                                        |
| `url.path`                  | The path. Only at the `debug` level (`LOG-008`).               |
| `http.response.status_code` | The status, when an answer came.                               |
| `event.duration`            | The duration in nanoseconds, until the head of the answer.     |
| `error.type`                | Only on a failure: the status as text, else the type of the Go error. |
| `error`                     | Only when no answer came: the error.                           |

| Outcome                                              | Level   |
| ---------------------------------------------------- | ------- |
| A status below 400                                   | `debug` |
| A status of 400 to 499, except 401, 403, and 429     | `info`  |
| A status of 401, 403, 429, or 500 and above          | `warn`  |
| No answer: a refused connection, a timeout           | `warn`  |
| No answer, because the caller cancelled the request  | `debug` |

**Why:** A 404 or a 400 often answers a wrong argument of the model. A 5xx, a refused
key, a limit, or no answer at all tells that the vendor or its contract fails.

## LOG-013

A use case that serves the answer of some vendors and leaves out the others writes one
record for each vendor that it leaves out. The level is `warn`, the message is
`source unavailable`, and the attributes are `vendor` and `error`.

**Why:** The tool call succeeds, so its record (`LOG-009`) holds no failure. Without this
record, a vendor that changes its format disappears from the answers with no trace.

## Retired identifiers

This file has no retired identifier.
