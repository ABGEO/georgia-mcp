---
id: API
title: The HTTP server
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/pkg/httpserver/]
related: [ARC, MCP, LOG, RSL, CFG]
---

# The HTTP server

## API-001

The router is chi, `github.com/go-chi/chi/v5`. The server has these routes and no other:

| Method and path         | Serves                                         | Access              |
| ----------------------- | ---------------------------------------------- | ------------------- |
| `POST, GET, DELETE /mcp` | The streamable HTTP transport of MCP          | Public, rate limit  |
| `GET /livez`            | The liveness of the process                    | Public              |
| `GET /readyz`           | The readiness of the process                   | Public              |

A new route needs a requirement. The server has no REST API.

## API-002

The router applies this middleware, in this order:

1. The request identifier.
2. The client address. It is the TCP peer, or the first address of `X-Forwarded-For`
   from the right that is outside the trusted proxy ranges of the configuration.
3. The access log. See `LOG-010`.
4. The recoverer.
5. The check of the `Origin` header, on `/mcp`. See `API-004`.
6. The rate limit for each client address of step 2, on `/mcp`. `go-chi/httprate` applies it.

**Why:** A header that the server does not check lets a client choose its own address,
and so its own bucket of the rate limit.

The router applies no request timeout to `/mcp`. The tool call carries its own
deadline. See `RSL-001`.

## API-003

The MCP handler runs in stateless mode. It keeps no session between two requests.

**Why:** Any copy of the server answers any request. A load balancer needs no
sticky session. `ARC-005`.

## API-004

The server rejects a request to `/mcp` whose `Origin` header is present and is not
in the allowed list of the configuration. The answer is 403.

**Why:** The MCP specification asks a server to check `Origin`, against DNS rebinding.

## API-005

`GET /livez` answers 200 while the process runs.
`GET /readyz` answers 200 after `bootstrap` registers every tool, and 503 during
the shutdown. Neither route calls a vendor. See `RSL-009`.

## API-006

The `http.Server` sets `ReadHeaderTimeout` to 5 seconds, `IdleTimeout` to 120 seconds,
and `MaxHeaderBytes` to 64 KiB.

## API-007

The rate limit answers 429 with a `Retry-After` header.
The configuration gives the limit and the window.

## Retired identifiers

This file has no retired identifier.
