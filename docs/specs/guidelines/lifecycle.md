---
id: LIF
title: Process lifecycle
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/app/georgia-mcp/bootstrap/, internal/app/georgia-mcp/cli/]
related: [ARC, CLI, CFG, LOG]
---

# Process lifecycle

## LIF-001

The start order is fixed:

1. Load `.env`, then read and validate the configuration. See `CFG`.
2. Build the logger.
3. Build each context module.
4. Build the MCP server and register the tools of each context.
5. Start the transport of the selected command.

## LIF-002

A process that runs more than one long-lived service runs them in an `errgroup`.
The first error cancels the group.

## LIF-003

A cancelled context starts the shutdown. The process stops the services in the
reverse order of the start.

## LIF-004

A shutdown step runs with a context that the cancellation does not reach, and
with a timeout of 10 seconds.

```go
ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
```

**Why:** The context that triggered the shutdown is already cancelled. A step that
uses it cannot finish its work.

## LIF-005

`http.ErrServerClosed`, `context.Canceled`, and the end of the input of the stdio
transport mean a clean stop. Do not report them as a failure.

## LIF-006

A shutdown step logs its own error and does not stop the next step.
`Close` joins the errors with `errors.Join`.

## Retired identifiers

This file has no retired identifier.
