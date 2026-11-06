---
id: BLD
title: Build and release
type: guideline
status: active
created: 2026-10-17
updated: 2026-11-06
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

`go.mod` declares the current Go release, as `go 1.27.3`.

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

Before a release, `go tool govulncheck ./...` passes. `BLD-009` runs it on each change
to the Go code.

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

A pull request and a push to `main` build only the image whose files changed:

| Image                 | Paths                                                                                  |
| --------------------- | -------------------------------------------------------------------------------------- |
| `georgia-mcp`         | `cmd/`, `internal/`, `pkg/`, `go.mod`, `go.sum`, `build/package/`                      |
| `georgia-mcp-website` | `website/`                                                                             |

A change to `images.yml`, a tag, and a manual run build both images. A release so
publishes each image at its version. A push to `main` that changes one image leaves the
other image with no `sha-<short>` tag for that commit. The job `Images` fails when an
image that ran has failed. The branch protection of `main` requires `Images`, because a
skipped image reports no check.

`VERSION` is `git describe --tags --always --dirty` and `REVISION` is the commit.
The label `org.opencontainers.image.source` of each image links the package to the repository.
Each action is pinned to the commit of a release, with the version in a comment.
Renovate (`.github/renovate.json`) proposes a new pin each week.

## BLD-009

`.github/workflows/ci.yml` runs the commands of `BLD-001` and the security scanners on
a push to `main`, on a pull request that is not a draft, and on a manual run. A job runs
only for the part that the change touches:

| Part       | Paths                                                                      | Jobs                                                                                   |
| ---------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Go         | `cmd/`, `internal/`, `pkg/`, `test/`, `go.mod`, `go.sum`, `.golangci.yaml` | `Go lint`, `Go test` (`go test -race`, `go build`), `Go vulnerabilities` (`BLD-007`)   |
| `website/` | `website/`                                                                 | `Website lint`, `Website test` (`pnpm test` builds the site)                           |
| Workflows  | `.github/workflows/`, `.github/zizmor.yml`                                 | `Workflow audit` (`zizmor`)                                                            |
| Manifests  | Every pull request                                                         | `Dependency review`                                                                    |

`Dependency review` fails on a new dependency with a known vulnerability, or with a
license that `.github/dependency-review.yml` does not allow. Every allowed license is
compatible with the GPL-3.0 license of the server. `.github/zizmor.yml` holds each
finding of `zizmor` that the workflow accepts, with the reason.

The repository also enables the dependency graph, the Dependabot alerts, secret
scanning with push protection, and the default setup of CodeQL. Renovate, not
Dependabot, proposes the updates.

A change to `ci.yml` and a manual run start every job. The job `CI` fails when a job
that ran has failed. The branch protection of `main` requires `CI` and no job of the
workflow, because a skipped job passes a required check.

The workflow takes the Go release from `go.mod` and the Node release from
`website/.nvmrc`. It pins `golangci-lint` to the version of the owner (`BLD-006`).
Each action is pinned to the commit of a release, as in `BLD-008`.

The workflow caches the Go build and module caches of each Go job, the Playwright
browser, and the fonts that Astro downloads. Only a run on `main` saves a cache. A pull
request restores the cache of `main`.

## Retired identifiers

This file has no retired identifier.
