---
id: PLATFORM
title: The platform of the server
type: requirements
status: approved
created: 2026-10-17
updated: 2026-10-17
context: platform
approved_by: Temuri
approved_on: 2026-10-17
constrained_by: [ARC, DOM, PKG, MCP, RSL, API, CFG, CLI, LIF, LOG, BLD]
---

# Requirements: The platform of the server

## 1. Problem

Each bounded context needs the same parts before its first tool can run: two
transports, the configuration, the log, the health reports, the limits, the cache,
and the start and the stop of the process. The first context must not build them.

If the first context builds them, the parts take the shape of that context. The
second context then changes the parts, and the first context with them.

## 2. Users

| User                    | Need                                                                  |
| ----------------------- | --------------------------------------------------------------------- |
| A person with a client  | Connects a remote client or a desktop client to the server.           |
| The operator            | Deploys the server, configures it, reads its health and its log.      |
| The owner               | Adds a context with no change to the platform, and checks each module with one command. |

## 3. Out of scope

- Any tool and any bounded context. `NBGRATE` adds the first one.
- The authentication of a client. `API-001` makes `/mcp` public.
- TLS. The ingress in front of the server terminates it.
- The export of a metric or a trace.
- A pipeline of continuous integration.
- The publication of a release or of an image.

## 4. Definitions

| Term       | Meaning                                                                                 |
| ---------- | --------------------------------------------------------------------------------------- |
| liveness   | The report that the process runs.                                                       |
| readiness  | The report that the process accepts tool calls.                                         |
| probe      | A request for the liveness or the readiness, from an orchestrator.                      |

## 5. Functional requirements

### `PLATFORM-FR-001`

The server must accept an MCP session over streamable HTTP when the operator starts
it in HTTP mode. It must listen on port 8080, unless the configuration gives another address.

### `PLATFORM-FR-002`

The server must accept an MCP session over stdio when a client starts it in stdio mode.

### `PLATFORM-FR-003`

The server must list the same tools on both transports.

**Why:** A person gets the same answer from a remote client and from a desktop client.

**Examples:**

- Normal case: one test context registers one tool. Both transports list it.
- Limit case: no context exists. Both transports list zero tools, with no error.

### `PLATFORM-FR-004`

The server must give the name `georgia-mcp` and its version when a client starts a session.

### `PLATFORM-FR-005`

A new context must join the server through one registration in the composition root,
with no change to a shared part.

**Why:** Each later feature adds a context. A change to the platform for each one
means the platform has the wrong shape.

**Examples:**

- Normal case: a test context with one tool joins. The test changes no platform package.
- Unwanted case: two contexts register the same tool name. The server refuses to start
  and names the tool.

### `PLATFORM-FR-006`

The server must report its liveness at a fixed address while the process runs.

### `PLATFORM-FR-007`

The server must report that it is ready only after it registers every tool, and
must report that it is not ready from the start of the shutdown.

**Examples:**

- Normal case: the process runs and its tools exist. The report is "ready".
- Limit case: the shutdown starts. The report is "not ready" before the last request ends.

### `PLATFORM-FR-008`

The server must reject an HTTP request for MCP whose origin is present and is not in
the allowed list. The allowed list is empty unless the configuration gives one.

**Why:** A web page must not reach a server on the network of its visitor.

**Examples:**

- Normal case: a request with no origin. The server accepts it.
- Normal case: a request from an allowed origin. The server accepts it.
- Unwanted case: a request from another origin. The server answers "forbidden".
- Limit case: the allowed list is empty. Every request with an origin fails. Every
  request with no origin passes.

### `PLATFORM-FR-009`

The server must limit each client address to 60 HTTP requests for MCP in each minute,
unless the configuration gives another limit.

**Examples:**

- Normal case: the 60th request in a minute. The server accepts it.
- Unwanted case: the 61st request. The server answers "too many requests", with the
  number of seconds to wait.

### `PLATFORM-FR-010`

The server must read its configuration from the environment of the process.

### `PLATFORM-FR-011`

The server must read a local configuration file when one exists in the working
directory. A value in the environment must win over the same value in the file.

### `PLATFORM-FR-012`

The server must refuse to start when a configuration value is missing or invalid.
The message must name the variable.

**Examples:**

- Normal case: every value is valid. The server starts.
- Unwanted case: the HTTP address is `abc`. The server stops and names the variable.

### `PLATFORM-FR-013`

The server must print its version, its commit, and its Go version on request, with
no configuration.

### `PLATFORM-FR-014`

The server must stop when it receives an interrupt or a termination signal. It must
let each request in progress end, within 10 seconds.

### `PLATFORM-FR-015`

The server must write one log record for each tool call.

### `PLATFORM-FR-016`

The server must write one log record for each HTTP request, except a probe that succeeds.

### `PLATFORM-FR-017`

The server must return a cached value of a source until its time to live ends.

### `PLATFORM-FR-018`

The server must send one call to a source when two or more tool calls need the same
value and the value is not in the cache.

**Why:** Ten clients that ask at the same second must not send ten calls to a vendor.

### `PLATFORM-FR-019`

The server must return an expired value when the source is unavailable and the value
is younger than its maximum stale age. Any other failure of the source must reach the tool.

**Examples:**

- Normal case: the source times out, and the value expired 10 minutes ago with a
  maximum stale age of 1 day. The server returns the value with its age.
- Unwanted case: the source answers "unknown currency". The server returns the error.

### `PLATFORM-FR-020`

The server must keep no more than the configured count of values in the cache.

### `PLATFORM-FR-021`

The server must limit the rate of its calls to each vendor to the configured limit.
When the limit is full, a call must wait for a free slot until its timeout. A call that
reaches its timeout must fail as an unavailable source.

**Examples:**

- Normal case: a slot frees before the timeout. The call goes to the vendor.
- Unwanted case: no slot frees before the timeout. The tool reports that the source
  did not answer, and the server returns a stale value when `PLATFORM-FR-019` allows one.

### `PLATFORM-FR-022`

The server must run in HTTP mode from a container image.

### `PLATFORM-FR-023`

The owner must start the server with the local configuration with one command.

### `PLATFORM-FR-024`

The owner must run the linter, the tests, or the build of a module with one command each.

## 6. Non-functional requirements

### `PLATFORM-NFR-001`

The server must report "ready" less than 1 second after the process starts.
Measure from the start of the process to the first ready report, on the machine of the owner.

### `PLATFORM-NFR-002`

The process must end less than 10 seconds after it receives a termination signal.
Measure from the signal to the exit of the process.

### `PLATFORM-NFR-003`

The container image must be smaller than 40 MB.

**Why:** A small image starts in seconds on a new node.

## 7. Invariants

### `PLATFORM-INV-001`

The server writes no file and keeps no value past the end of the process.

### `PLATFORM-INV-002`

In stdio mode, the standard output carries MCP messages and nothing else.

**Why:** One other line on the standard output breaks the client.

### `PLATFORM-INV-003`

No log record holds a secret. No log record above the debug level holds the
arguments of a tool call.

## 8. Constraints from the guidelines

| Rule                   | Guideline      | Effect on this feature                                       |
| ---------------------- | -------------- | ------------------------------------------------------------ |
| `ARC-003`, `MCP-010`   | Architecture   | One set of tools on two transports.                          |
| `ARC-005`              | Architecture   | No database. The cache lives in memory.                      |
| `ARC-009`, `PKG-003`   | Architecture   | The composition root registers each context.                 |
| `DOM-008`, `DOM-011`   | Domain         | The kernel holds the language and the unavailable source.    |
| `MCP-001`, `MCP-011`   | MCP tools      | One server. One middleware logs each tool call.              |
| `API-001` to `API-007` | HTTP           | The routes, the middleware order, the origin, the limit.     |
| `CFG-001` to `CFG-007` | Configuration  | The environment, the local file, the prefix, the validation. |
| `CLI-001` to `CLI-005` | Command line   | `serve`, `stdio`, `version`.                                 |
| `LIF-001` to `LIF-006` | Lifecycle      | The start order and the shutdown.                            |
| `LOG-002`, `LOG-009`, `LOG-010` | Logging | The output of each mode. The records.                      |
| `RSL-003` to `RSL-008` | Resilience     | The cache, its stale values, its size, the vendor limit.     |
| `BLD-004`, `BLD-005`   | Build          | The version from the build. The image.                       |
| `GO-012`               | Go style       | The generic cache needs two instantiations. The tests give them. |

## 9. Open questions

| #   | Question                                                                                     | Owner  | Answer |
| --- | -------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | When a vendor limit is full, does a call wait for a free slot until its timeout, or fail at once? | Temuri | Wait until the timeout, then fail as an unavailable source. `PLATFORM-FR-021`. |
| 2   | What is the default list of allowed origins? | Temuri | Empty. A request with an origin fails. A request with no origin passes. `PLATFORM-FR-008`. |
| 3   | What is the default HTTP address? | Temuri | Port 8080. `PLATFORM-FR-001`. |

## Retired identifiers

This file has no retired identifier.
