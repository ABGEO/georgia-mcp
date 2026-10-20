---
id: BLD
title: Build and release
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-20
scope: [., pkg/, website/, build/package/, deployments/, Taskfile.yml]
related: [ARC, CLT, GO, TST]
---

# Build and release

## BLD-001

Run the linter, the tests, and the build of each module that the change touches.
Every command must pass before the work is complete.

| Module you changed | Commands                                                                          |
| ------------------ | --------------------------------------------------------------------------------- |
| Server (`.`)       | `golangci-lint run ./...`, `go test -race ./...`, `go build ./...`                |
| `pkg/<vendor>`     | `cd pkg/<vendor> && golangci-lint run ./... && go test -race ./...`               |
| `website/`         | `cd website && npm run lint && npm test && npm run build`                         |

A change in a vendor client touches the server. Run the server commands too.

## BLD-002

`go.work` and the `go.mod` of the server declare the current Go release, as
`go 1.27.2`. The `go.mod` of a vendor client declares the oldest Go release that the
Go team still supports, as `go 1.26.0`.

**Why:** Since Go 1.21, the `go` line is a minimum that the toolchain enforces. A
client that declares the newest release forces every project that imports it to
upgrade. The server is a binary, so it takes the newest release.

## BLD-003

`go.work` gives the server the local copy of each vendor client during development.
Before the first tag of a client, the `go.mod` of the server holds no requirement of it:
Go checks a required pseudo-version against the repository, even in the workspace. A
release tags the client first (`GIT-007`), then requires that tag in the `go.mod` of the
server.
Before a release of the server, `task release-check` passes. It runs
`GOWORK=off go build ./...` in the server module, and `BLD-007`. `task build` keeps the
workspace, so it passes before a tag of a client exists.

**Why:** `go install github.com/abgeo/georgia-mcp/cmd/georgia-mcp@latest` reads
`go.mod`, not `go.work`. Only a build with the workspace off proves that `go.mod` is
complete.

## BLD-004

The version of the binary comes from `runtime/debug.ReadBuildInfo`. The build
passes no version with `-ldflags`.

**Why:** Go writes the module version and the commit into the binary.

## BLD-005

The container image builds with `CGO_ENABLED=0` and `-trimpath` in a multi-stage
`build/package/Dockerfile`, from the repository root. The build stage copies the whole
repository, `.git` and `go.work` included, so the workspace resolves every module and Go
writes the version of `BLD-004`. Cache mounts keep the module cache and the build cache. The final stage is a distroless static image
that runs as a user that is not root. Its command is `georgia-mcp serve`.

The final stage carries the OpenContainers labels `title`, `description`, `source`,
`version`, and `revision`. The build arguments `VERSION` and `REVISION` give the
last two. `task build-image` passes `git describe` and the commit.

## BLD-006

A development tool that a module needs is a `tool` directive in its `go.mod`, and
runs with `go tool <name>`. `golangci-lint` is the one exception: it runs as a binary,
at the version that the owner installs.

**Why:** The `tool` directive pins the version in `go.mod`. The authors of
`golangci-lint` advise against a build from a module, because its dependencies
conflict with those of the project.

## BLD-007

Before a release, `go tool govulncheck ./...` passes in each module.

## Retired identifiers

This file has no retired identifier.
