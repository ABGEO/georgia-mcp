---
id: PLATFORM
title: The platform of the server, the scenarios
type: spec
status: approved
created: 2026-10-17
updated: 2026-10-22
context: platform
approved_by: Temuri
approved_on: 2026-10-17
constrained_by: [TST]
requirements: features/platform/requirements.md
---

# Specification: The platform of the server, the scenarios

A test module is a `mcpserver.Module` in a `_test.go` file. It registers the tool
`test_get_echo`, which returns its input. `slowModule` registers `test_get_slow`,
which waits 200 ms.

## The transports

### `PLATFORM-SC-001`

**Verifies:** `PLATFORM-FR-001`
**Layer:** integration

**Given** `httpserver` serves `Registry.Handler()` on `127.0.0.1:0`, with the test module.
**When** a `StreamableClientTransport` client connects and lists the tools.
**Then** the list holds `test_get_echo`, and a call of it returns its input.

### `PLATFORM-SC-002`

**Verifies:** `PLATFORM-FR-002`
**Layer:** integration

**Given** `Registry.RunStdio` runs on an `IOTransport` over two pipes, with the test module.
**When** a client on the other ends of the pipes lists the tools.
**Then** the list holds `test_get_echo`.

### `PLATFORM-SC-003`

**Verifies:** `PLATFORM-FR-003`
**Layer:** integration

**Given** one registry with the test module.
**When** a client lists the tools over HTTP, and a second client over stdio.
**Then** `cmp.Diff` of the two lists is empty.

### `PLATFORM-SC-004`

**Verifies:** `PLATFORM-FR-003`
**Layer:** integration

**Given** a registry with no module.
**When** a client lists the tools over HTTP, and a second client over stdio.
**Then** each list is empty, and each call returns no error.

### `PLATFORM-SC-005`

**Verifies:** `PLATFORM-FR-004`
**Layer:** integration

**Given** a registry built with `buildinfo.Info{Version: "v0.1.0"}`.
**When** a client connects.
**Then** the server info holds the name `georgia-mcp` and the version `v0.1.0`.

## The registry

### `PLATFORM-SC-006`

**Verifies:** `PLATFORM-FR-005`
**Layer:** unit

**Given** a registry, and the test module that imports only `mcpserver` and the SDK.
**When** `Register` receives the test module.
**Then** it returns nil, and the tool has the four annotations of `MCP-004`.

### `PLATFORM-SC-007`

**Verifies:** `PLATFORM-FR-005`
**Layer:** unit

**Given** two modules that both register `test_get_echo`.
**When** `Register` receives both.
**Then** the error matches `ErrDuplicateTool` with `errors.Is`, and its text holds `test_get_echo`.

## The probes

### `PLATFORM-SC-008`

**Verifies:** `PLATFORM-FR-006`
**Layer:** unit

**Given** a server that is not ready.
**When** a client requests `GET /livez`.
**Then** the answer is 200 with `ok`.

### `PLATFORM-SC-009`

**Verifies:** `PLATFORM-FR-007`
**Layer:** integration

| Case                     | State                            | `GET /readyz` |
| ------------------------ | -------------------------------- | ------------- |
| Before the registration  | `SetReady` not called            | 503           |
| After the registration   | `SetReady(true)`                 | 200           |
| At the shutdown          | The context of `Run` is cancelled, a slow request runs | 503 |

## The HTTP limits

### `PLATFORM-SC-010`

**Verifies:** `PLATFORM-FR-008`
**Layer:** unit

| Allowed list             | `Origin` header          | Status of `POST /mcp` |
| ------------------------ | ------------------------ | --------------------- |
| `https://a.example`      | Absent                   | Not 403               |
| `https://a.example`      | `https://a.example`      | Not 403               |
| `https://a.example`      | `https://b.example`      | 403                   |
| Empty                    | `https://a.example`      | 403                   |
| Empty                    | Absent                   | Not 403               |

### `PLATFORM-SC-011`

**Verifies:** `PLATFORM-FR-009`
**Layer:** unit

**Given** `RateLimit` 2 and `RateWindow` 1 minute.
**When** one address sends three requests to `POST /mcp`.
**Then** the first two are not 429. The third is 429 with a `Retry-After` header.

### `PLATFORM-SC-034`

**Verifies:** `PLATFORM-FR-009`
**Layer:** unit

**Given** `RateLimit` 1, and no trusted proxy range.
**When** one address sends two requests to `POST /mcp`, each with another `X-Forwarded-For`.
**Then** the second request is 429.

## The configuration

### `PLATFORM-SC-012`

**Verifies:** `PLATFORM-FR-010`
**Layer:** unit

**Given** `t.Setenv("GEOMCP_HTTP_RATE_LIMIT", "30")` and no `.env`.
**When** `bootstrap.LoadConfig` runs.
**Then** `HTTP.RateLimit` is 30 and `HTTP.Address` is `:8080`.

### `PLATFORM-SC-013`

**Verifies:** `PLATFORM-FR-011`
**Layer:** unit

**Given** `t.Chdir` to a directory whose `.env` holds `GEOMCP_HTTP_RATE_LIMIT=10`
and `GEOMCP_LOG_FORMAT=text`, and `t.Setenv("GEOMCP_HTTP_RATE_LIMIT", "20")`.
**When** `bootstrap.LoadConfig` runs.
**Then** `HTTP.RateLimit` is 20 and `Log.Format` is `text`.

### `PLATFORM-SC-014`

**Verifies:** `PLATFORM-FR-012`
**Layer:** unit

| Variable                      | Value   | Result                                           |
| ----------------------------- | ------- | ------------------------------------------------ |
| None                          |         | No error                                         |
| `GEOMCP_HTTP_ADDRESS`         | `abc`   | An error that holds `GEOMCP_HTTP_ADDRESS`        |
| `GEOMCP_HTTP_RATE_LIMIT`      | `zero`  | An error that holds `GEOMCP_HTTP_RATE_LIMIT`     |
| `GEOMCP_HTTP_RATE_LIMIT`      | `0`     | An error that holds `GEOMCP_HTTP_RATE_LIMIT`     |
| `GEOMCP_LOG_FORMAT`           | `xml`   | An error that holds `GEOMCP_LOG_FORMAT`          |
| `GEOMCP_LOG_LEVEL`            | `loud`  | An error that holds `GEOMCP_LOG_LEVEL`           |

### `PLATFORM-SC-036`

**Verifies:** `PLATFORM-FR-012`
**Layer:** unit

**Given** a struct under the prefix `GEOMCP_`, with `envPrefix:"VENDOR_"` on a struct
that holds each field below.

| Field and tag                          | Value                    | Error                                                       |
| -------------------------------------- | ------------------------ | ----------------------------------------------------------- |
| `BaseURL url.URL`, `http_url`          | `ftp://example.com`      | `GEOMCP_VENDOR_BASE_URL: must be an http or https URL`      |
| `Rate float64`, `gt=0`                 | `0`                      | `GEOMCP_VENDOR_RATE: must be more than 0`                   |
| `APIKey string`, `required_if=Enabled true` | empty, `ENABLED=true` | `GEOMCP_VENDOR_API_KEY: must be set while GEOMCP_VENDOR_ENABLED is true. Set it, or set GEOMCP_VENDOR_ENABLED=false` |
| `APIKey string`, `required_if=Enabled true` | empty, `ENABLED=false` | No error                                                  |
| `Ranges []string`, `dive,cidr`         | `10.0.0.0/8,nope`        | `GEOMCP_VENDOR_RANGES: nope must be a CIDR range`           |

Two invalid values give two errors, and each one is a `*FieldError`.

## The commands and the lifecycle

### `PLATFORM-SC-015`

**Verifies:** `PLATFORM-FR-013`
**Layer:** unit

**Given** `GEOMCP_HTTP_ADDRESS=abc`, which fails the validation.
**When** the root command runs `version` into a buffer.
**Then** it returns nil, and the buffer has three lines that start with `version `,
`commit `, and `go `.

### `PLATFORM-SC-016`

**Verifies:** `PLATFORM-FR-014`, `PLATFORM-NFR-002`
**Layer:** integration

**Given** `httpserver.Run` with `slowModule`, and a call of `test_get_slow` in progress.
**When** the test cancels the context of `Run`.
**Then** the call returns its result, and `Run` returns nil less than 10 seconds after the cancel.

### `PLATFORM-SC-029`

**Verifies:** `PLATFORM-NFR-001`
**Layer:** integration

**Given** a built binary and a free port.
**When** the test starts `georgia-mcp serve` and polls `/readyz` each 10 ms.
**Then** the first 200 arrives less than 1 second after the start of the process.

## The log

### `PLATFORM-SC-017`

**Verifies:** `PLATFORM-FR-015`
**Layer:** unit

| Call                                         | Records | Attributes                                                         |
| -------------------------------------------- | ------- | ------------------------------------------------------------------ |
| `test_get_echo` succeeds                     | 1       | `mcp.method.name`, `gen_ai.tool.name`, `event.duration`            |
| A tool returns a `ToolError`                 | 1       | The above, and `error.type` = `tool_error`                         |
| `tools/list`                                 | 0       |                                                                    |

### `PLATFORM-SC-018`

**Verifies:** `PLATFORM-FR-016`
**Layer:** unit

| Request                         | Status | Records |
| ------------------------------- | ------ | ------- |
| `POST /mcp`                     | 200    | 1       |
| `GET /livez`                    | 200    | 0       |
| `GET /readyz`, not ready        | 503    | 1       |

### `PLATFORM-SC-033`

**Verifies:** `PLATFORM-INV-003`
**Layer:** unit

**Given** a logger at info level, and a second logger at debug level.
**When** each one logs a call of `test_get_echo` with the input `secret-value`.
**Then** no record of the info logger holds `secret-value`. The debug record holds
`mcp.tool.arguments`.

## The cache

Each test of the cache runs in `synctest.Test` and uses `Cache[string]` and `Cache[int]`.

### `PLATFORM-SC-019`

**Verifies:** `PLATFORM-FR-017`
**Layer:** unit

**Given** `TTL` 1 hour, and a load that counts its calls.
**When** `Get` runs at 0, at 59 minutes, and at 61 minutes.
**Then** the load runs two times, at 0 and at 61 minutes.

### `PLATFORM-SC-020`

**Verifies:** `PLATFORM-FR-018`
**Layer:** unit

**Given** an empty cache, and a load that blocks until the test releases it.
**When** 10 goroutines call `Get` with one key, and the test releases the load.
**Then** the load runs one time, and the 10 calls return the same value.

### `PLATFORM-SC-021`

**Verifies:** `PLATFORM-FR-019`
**Layer:** unit

**Given** `TTL` 1 hour, `MaxStale` 24 hours, and a value fetched at 0.
**When** at 2 hours the load fails with an error that wraps `kernel.ErrSourceUnavailable`.
**Then** `Get` returns the value with `Stale: true` and the fetch time 0, and no error.

### `PLATFORM-SC-022`

**Verifies:** `PLATFORM-FR-019`
**Layer:** unit

**Given** the cache of `PLATFORM-SC-021`.
**When** at 2 hours the load fails with an error that does not wrap `kernel.ErrSourceUnavailable`.
**Then** `Get` returns that error and no value.

### `PLATFORM-SC-023`

**Verifies:** `PLATFORM-FR-019`
**Layer:** unit

**Given** the cache of `PLATFORM-SC-021`.
**When** at 26 hours the load fails with `kernel.ErrSourceUnavailable`.
**Then** `Get` returns the error and no value.

### `PLATFORM-SC-024`

**Verifies:** `PLATFORM-FR-020`
**Layer:** unit

**Given** `MaxEntries` 10.
**When** `Get` stores 100 keys.
**Then** after `CleanUp`, the store holds 10 entries or fewer.

## The vendor limit

### `PLATFORM-SC-025`

**Verifies:** `PLATFORM-FR-021`
**Layer:** unit

**Given** an `httptest` server that counts requests, and `httpclient.New` with
`Rate` 1 and `Burst` 1.

| Case                                     | Result                                                    |
| ---------------------------------------- | --------------------------------------------------------- |
| Two requests, no deadline                | Both reach the server. The second waits about 1 second.   |
| A second request with a 100 ms deadline  | The error wraps `kernel.ErrSourceUnavailable`. The server counts 1. |

## The vendors

### `PLATFORM-SC-035`

**Verifies:** `PLATFORM-FR-025`
**Layer:** unit

**Given** a registry with a logger that records, and entries whose sources record a close.

| Entries                                          | Result                                                     |
| ------------------------------------------------ | ---------------------------------------------------------- |
| A on, B off, C on                                | A and C, in this order. One info record: `B is off`, with the variable of B. |
| A on, B on with a failing `Build`                | An error that wraps the failure and names B. A is closed.  |
| A off, B off                                     | No source, no error. Two info records.                     |
| `CloseAll` of two sources, the first fails       | Both are closed. The error is the failure of the first.    |

## The vendor clients

### `PLATFORM-SC-037`

**Verifies:** `PLATFORM-FR-026`
**Layer:** unit

**Given** an `httptest` server that records the request, and `vendorhttp.New` with the
module `example.com/pkg/acme`, a sentinel, the header `Accept: application/json`, and a
limit of 8 bytes.

| Request and answer                                  | Result                                                     |
| --------------------------------------------------- | ---------------------------------------------------------- |
| `Fetch` of the path `rates`, the query `a=1`, 200 `{"x":1}` | The path `/rates?a=1`, `Accept`, a User-Agent that starts with `georgia-mcp-acme/`, the body. |
| `Fetch`, 200 with a body of 20 bytes                | The first 8 bytes.                                         |
| `Fetch`, 503 `down`                                 | A `*StatusError` with 503 and `down`. `errors.Is` finds the sentinel. |
| `Do`, 204                                           | The answer with 204, and no error.                         |
| `FetchJSON`, 200 `{"x":1}`                          | The target holds 1.                                        |
| `Do` with the method `POST`, a body, and `Accept: text/html` | The server gets `POST`, the body, and `text/html`.        |


### `PLATFORM-SC-026`

**Verifies:** `PLATFORM-FR-022`
**Layer:** manual

**Given** `task build-image`.
**When** the owner runs the image with port 8080 published.
**Then** `GET /readyz` answers 200, the process runs as the user `nonroot`, and the image
labels hold the version and the commit.

### `PLATFORM-SC-027`

**Verifies:** `PLATFORM-FR-023`
**Layer:** manual

**Given** a `.env` copied from `configs/.env.example`.
**When** the owner runs `task compose-up`.
**Then** `GET http://localhost:8080/readyz` answers 200.

### `PLATFORM-SC-028`

**Verifies:** `PLATFORM-FR-024`
**Layer:** manual

**Given** a clean checkout.
**When** the owner runs `task lint`, `task test`, and `task build`.
**Then** each command exits with 0.

### `PLATFORM-SC-030`

**Verifies:** `PLATFORM-NFR-003`
**Layer:** manual

**Given** `task build-image`.
**When** the owner reads the size with `docker image inspect`.
**Then** the size is less than 40 MB.

## The invariants

### `PLATFORM-SC-031`

**Verifies:** `PLATFORM-INV-001`
**Layer:** integration

**Given** an empty working directory.
**When** the binary runs `serve`, answers one tool list, and stops on a signal.
**Then** the working directory is still empty.

### `PLATFORM-SC-032`

**Verifies:** `PLATFORM-INV-002`
**Layer:** integration

**Given** the binary runs `stdio` with `GEOMCP_LOG_LEVEL=debug`.
**When** a client lists the tools and closes the input.
**Then** each line on stdout parses as a JSON-RPC message, and stderr holds the log records.

## Retired identifiers

This file has no retired identifier.
