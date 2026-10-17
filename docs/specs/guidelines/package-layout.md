---
id: PKG
title: Package layout
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [., cmd/, internal/, pkg/, build/, deployments/, configs/, test/]
related: [ARC, DOM, CLT, GO]
---

# Package layout

## PKG-001

The repository follows https://github.com/golang-standards/project-layout.
Create a directory only when the code needs it.

```
cmd/georgia-mcp/main.go          The entry point. It calls internal/app/georgia-mcp/cli.
internal/
  app/georgia-mcp/               The application code.
    cli/                         The command tree. See CLI.
    bootstrap/                   The composition root. See ARC-009.
    <context>/                   One bounded context. See PKG-002.
  pkg/                           The private libraries that the application shares.
    kernel/                      The shared kernel. See DOM-008.
    buildinfo/                   The version from the build. See BLD-004.
    config/                      The loader of the configuration. See CFG.
    logger/                      The logger. See LOG.
    httpserver/                  The router and the middleware. See API.
    httpclient/                  The client of a vendor call. See RSL.
    mcpserver/                   The MCP server and its middleware. See MCP.
    cache/                       The cache. See RSL.
pkg/<vendor>/                    One vendor client module. See CLT.
build/package/                   The Dockerfile and its ignore file. See BLD-005.
deployments/                     The compose file of a local run.
configs/                         The example of the configuration. See CFG-007.
test/e2e/                        The tests that start the built binary.
docs/specs/                      The specifications. See PRC.
```

A package of `internal/pkg` holds no domain term, except `kernel`. It never imports
`internal/app`.

**Why:** A Go developer finds each kind of file where the common layout puts it.

## PKG-002

A bounded context uses this layout.

| Path                    | Holds                                                         |
| ----------------------- | ------------------------------------------------------------- |
| `module.go`             | `Module`, `New`, and `Register`. See `PKG-003`.               |
| `config.go`             | The configuration of the context. See `CFG-004`.              |
| `domain/`               | The value objects, the read models, the domain errors.        |
| `app/`                  | The use cases, `port.go`, `errors.go`. See `DOM-009`.         |
| `adapter/tool/`         | The MCP tools of the context. The driving adapter.            |
| `adapter/<vendor>/`     | The driven adapter that implements a port with one client.    |
| `adapter/cache/`        | The decorator of each port that the cache wraps. See `RSL-003`. |

## PKG-003

`module.go` declares the type `Module` and two functions:

```go
func New(cfg Config, logger *slog.Logger) (*Module, error)
func (m *Module) Register(registry *mcpserver.Registry) error
```

`New` builds the adapters, wraps each driven adapter in the cache, and builds the
use cases. `Register` adds every tool of the context with `mcpserver.AddTool`, and
returns the error of a duplicate name. `module.go` holds no domain
logic.

**Why:** A reader sees the whole wiring of a context in one screen.

## PKG-004

A driven adapter imports its client with the alias `<vendor>client`.

```go
import nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
```

**Why:** The adapter package and the client package share the vendor name.

## PKG-005

A package name is one lowercase noun. It uses no underscore and no abbreviation
that the glossary does not hold. The names `app`, `cli`, `mcpserver`,
`httpserver`, `httpclient`, and `buildinfo` are allowed.

A name does not repeat in the identifiers of its package: `rates.Module`, not
`rates.RatesModule`.

## PKG-006

A package with more than one file keeps its package comment in `doc.go`.
`doc.go` holds the comment and nothing else.

## PKG-007

Every package of the server lives under `internal`. The server module exports nothing.
A vendor client module exports its client. See `CLT-001`.

## Retired identifiers

This file has no retired identifier.
