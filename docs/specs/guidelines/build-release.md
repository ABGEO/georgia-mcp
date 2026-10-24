---
id: BLD
title: Build and release
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-25
scope: [., pkg/, website/, build/package/, deployments/, Taskfile.yml, .github/workflows/]
related: [ARC, CLT, GO, TST]
---

# Build and release

## BLD-001

Run the linter, the tests, and the build of each part that the change touches.
Every command must pass before the work is complete.

| You changed                  | Commands                                                           |
| ---------------------------- | ------------------------------------------------------------------ |
| Go (`internal/`, `pkg/`, `cmd/`) | `golangci-lint run ./...`, `go test -race ./...`, `go build ./...` |
| `website/`                   | `cd website && pnpm lint && pnpm test && pnpm build`               |

`Taskfile.yml` runs each row with one command: `task lint`, `task test`, and `task build`
run the Go commands, and `task website:lint`, `task website:test`, and
`task website:build` run the commands of `website/`. A task of the website stops when
the version of Node is older than `website/.nvmrc`.

## BLD-002

`go.mod` declares the current Go release, as `go 1.27.2`.

**Why:** The server is a binary, so it takes the newest release. A project that imports
a vendor client needs the same release.

## BLD-003

Before a release of the server, `task release-check` passes. It runs `go build ./...`
and `BLD-007`.

## BLD-004

The version of the binary comes from `runtime/debug.ReadBuildInfo`. The build
passes no version with `-ldflags`.

**Why:** Go writes the module version and the commit into the binary.

## BLD-005

The container image builds with `CGO_ENABLED=0` and `-trimpath` in a multi-stage
`build/package/Dockerfile`, from the repository root. The build stage copies the whole
repository, `.git` included, so Go writes the version of `BLD-004`. The build stage runs on `$BUILDPLATFORM` and compiles with
`GOOS=$TARGETOS GOARCH=$TARGETARCH`, so a second platform needs no emulation. Cache mounts keep the module cache and the build cache. The final stage is a distroless static image
that runs as a user that is not root. Its command is `georgia-mcp serve`.

The final stage carries the OpenContainers labels `title`, `description`, `source`,
`version`, and `revision`. The build arguments `VERSION` and `REVISION` give the
last two. `task image` passes `git describe` and the commit.

## BLD-006

A development tool that a module needs is a `tool` directive in its `go.mod`, and
runs with `go tool <name>`. `golangci-lint` is the one exception: it runs as a binary,
at the version that the owner installs.

**Why:** The `tool` directive pins the version in `go.mod`. The authors of
`golangci-lint` advise against a build from a module, because its dependencies
conflict with those of the project.

## BLD-007

Before a release, `go tool govulncheck ./...` passes.

## BLD-008

`.github/workflows/images.yml` builds two images for `linux/amd64` and `linux/arm64` and
publishes them to GitHub Packages:

| Image                                  | Dockerfile                    | Context    |
| -------------------------------------- | ----------------------------- | ---------- |
| `ghcr.io/abgeo/georgia-mcp`            | `build/package/Dockerfile`    | `.`        |
| `ghcr.io/abgeo/georgia-mcp-website`    | `website/.docker/Dockerfile`  | `website/` |

| Event                 | Tags                                      | Push |
| --------------------- | ----------------------------------------- | ---- |
| A push to `main`      | `main`, `sha-<short>`                     | Yes  |
| A tag `v<x.y.z>`      | `<x.y.z>`, `<x.y>`, `latest`, `sha-<short>` | Yes  |
| A pull request        | `pr-<n>`                                  | No   |

`VERSION` is `git describe --tags --always --dirty` and `REVISION` is the commit.
The label `org.opencontainers.image.source` of each image links the package to the repository.
Each action is pinned to the commit of a release, with the version in a comment.
`.github/dependabot.yml` proposes a new pin each week.

## Retired identifiers

This file has no retired identifier.
