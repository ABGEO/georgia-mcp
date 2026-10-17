---
id: CLI
title: The command line
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [cmd/, internal/app/georgia-mcp/cli/]
related: [LIF, ARC, LOG]
---

# The command line

The library is cobra, `github.com/spf13/cobra`.

## CLI-001

The root command is `georgia-mcp`. The tree is:

```
georgia-mcp serve      The streamable HTTP transport. See API.
georgia-mcp stdio      The stdio transport.
georgia-mcp version    The version, the commit, and the Go version.
```

A new command needs a requirement.

## CLI-002

`main` builds a context that `SIGINT` and `SIGTERM` cancel, with
`signal.NotifyContext`. It runs the root command with `ExecuteContext`.

## CLI-003

`PreRunE` validates and prepares. `RunE` performs the work.
A command returns an error. Only `main` calls `os.Exit`.

**Why:** The root command owns the exit code and the cleanup.

## CLI-004

A command takes the context with `cmd.Context()`. It passes that context to
everything that it starts.

## CLI-005

The commands take no flag that the configuration also holds. The configuration is
the one source of a value. See `CFG`.

## Retired identifiers

This file has no retired identifier.
