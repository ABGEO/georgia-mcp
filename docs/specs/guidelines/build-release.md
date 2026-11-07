---
id: BLD
title: Build and release
type: guideline
status: active
created: 2026-10-17
updated: 2026-11-07
scope: [., pkg/, website/, build/package/, build/cursor-plugin/, .cursor-plugin/, deployments/, Taskfile.yml, website/Taskfile.yml, .goreleaser.yaml, .github/labels.yml, .github/release-drafter.yml, .github/workflows/]
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
`task website:build` run the commands of `website/`. `website/Taskfile.yml` holds the
tasks of the website, and `Taskfile.yml` includes it under the namespace `website`. A
task of the website stops when the version of Node is older than `website/.nvmrc`.

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
last two. `task image` passes `git describe` and the commit. The final stage also carries
the label `io.modelcontextprotocol.server.name`, with the name of `BLD-011`.

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
| A push to `main`      | `latest`, `sha-<short>`                   | Yes  |
| A tag `v<x.y.z>`      | `<x.y.z>`, `<x.y>`, `<x>`                 | Yes  |
| A pull request        | `pr-<n>`                                  | No   |

`latest` follows `main`. A release pins a version. A tag
`v0.<y>.<z>` gets no `<x>` tag, because a release before `1.0.0` keeps no compatibility.

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
| Go         | `cmd/`, `internal/`, `pkg/`, `test/`, `go.mod`, `go.sum`, `.golangci.yaml` | `Go lint`, `Go test` (`go test -race -coverprofile` through `gotestsum`, `go build`), `Coverage`, `Go vulnerabilities` (`BLD-007`) |
| `website/` | `website/`                                                                 | `Website lint`, `Website test` (`pnpm test` builds the site)                           |
| Workflows  | `.github/workflows/`, `.github/zizmor.yml`                                 | `Workflow audit` (`zizmor`)                                                            |
| Manifests  | Every pull request                                                         | `Dependency review`                                                                    |
| Commits    | Every pull request                                                         | `DCO` (`GIT-008`)                                                                      |

`Go test` writes the total coverage and the coverage of each package to the summary of
the job, and keeps the profile as an artifact. `gotestsum` (a tool in `go.mod`) writes a
JUnit report of each test, and the job keeps it as an artifact, also when a test fails.
`Coverage` uploads both to Codecov with GitHub OIDC, so the job that runs the code of a
pull request holds no `id-token`. It uploads the profile only when the tests pass, and
the report in each case: Codecov detects a flaky test from the failed runs. A pull
request from a fork uploads without a token. `.github/codecov.yml` makes each status of
Codecov informational: the coverage gates no merge. A failed upload fails no job.

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

## BLD-010

`task test:live` runs the `live` tests of `TST-004`:
`go test -tags live -run '^TestLive' -count=1 ./internal/... ./pkg/...`. The
task reads the keys of the sources from `.env`. A test whose key is empty skips.

`.github/workflows/live.yml` runs the same command each day at 03:17 UTC and on a
manual run, without the packages of this table. The tests of such a package stay in the
repository, and `task test:live` still runs them.

| Package      | Reason                                                                  |
| ------------ | ----------------------------------------------------------------------- |
| `pkg/wissol` | `api.wissol.ge` does not answer a GitHub runner. The hosted server reaches it. |

The keys come from the repository secrets
`GEOMCP_TRANSIT_TTC_API_KEY`, `GEOMCP_RATES_TBC_API_KEY`, and
`GEOMCP_RATES_CREDO_API_KEY`. The job summary lists each failed test and each skipped
test.

A run on `main` reports to one issue, titled `Live tests fail`:

| Result of the run | Open issue  | Action                                      |
| ----------------- | ----------- | ------------------------------------------- |
| A test failed     | None        | Open the issue with the failed tests.       |
| A test failed     | One         | Comment on the issue with the failed tests. |
| Passed            | One         | Close the issue with a link to the run.     |
| Passed            | None        | Nothing.                                    |

The job that runs the tests holds the secrets and reads the repository only. A
separate job, with no secret, writes to the issue.

**Why:** A source changes its page or its API without notice. `go test ./...` runs no
`live` test, so only a scheduled run finds the change before a user does.

## BLD-011

`build/mcp-registry/server.json` describes the server to the official MCP Registry, in
the schema `2025-12-11`. Its name is `io.github.ABGEO/georgia-mcp`. It lists two ways to
connect:

| Entry            | Value                                                                         |
| ---------------- | ----------------------------------------------------------------------------- |
| A remote         | `streamable-http` at `https://geo-mcp.abgeo.cloud/mcp`                        |
| An OCI package   | `ghcr.io/abgeo/georgia-mcp:<version>`, over stdio, with the argument `stdio` |

The package lists the key and the switch of each source that needs a key. Each such
source is off by default (`CFG-008`), so the package starts with no variable.

The job `MCP Registry` of `.github/workflows/images.yml` runs on a tag `v<x.y.z>`, after
each image is pushed. It verifies the release of `mcp-publisher` with `cosign`, writes
`<x.y.z>` into the version and the image tag of a copy of `server.json`, signs in with
GitHub OIDC, and publishes the copy. The version of `server.json` in the repository is
that of the last release. `MCP_PUBLISHER_VERSION` pins `mcp-publisher`, and Renovate
proposes a new release.

**Why:** The registry checks the label of `BLD-005` on the image of the tag, so the job
waits for the push. OIDC needs no secret. A version in the registry cannot change, so
only a release publishes.

## BLD-012

`.github/workflows/scorecard.yml` runs the OpenSSF Scorecard on a push to `main`, each
Monday at 04:15 UTC, and on a change to the branch rules. It publishes the result to
`api.scorecard.dev`, which serves the badge of the README, and uploads it to code
scanning.

The job holds only the steps that the workflow restrictions of the Scorecard allow:
`actions/checkout`, `ossf/scorecard-action`, `actions/upload-artifact`, and
`github/codeql-action/upload-sarif`. The job has no `run` step and no environment
variable, and only it holds `id-token: write`. The workflow blocks the cache.

**Why:** The API refuses a result from a workflow that breaks the restrictions. A check
such as `Maintained` changes with time, so the weekly run keeps the badge current.

## BLD-013

`.github/workflows/release.yml` runs GoReleaser (`.goreleaser.yaml`) on a tag `v<x.y.z>`.
It builds `georgia-mcp` for `linux`, `darwin`, and `windows` on `amd64` and `arm64`, with
`CGO_ENABLED=0`, `-trimpath`, and the commit time as the time of each file. The ldflags
are `-s -w` only, so the version comes from `BLD-004`. Each archive holds the binary,
`LICENSE`, and `README.md`.

| File                                   | Holds                                                    |
| -------------------------------------- | -------------------------------------------------------- |
| `georgia-mcp_<x.y.z>_<os>_<arch>`      | The archive: `.zip` for `windows`, `.tar.gz` otherwise   |
| `<archive>.sbom.json`                  | The SPDX SBOM of the archive, from Syft                  |
| `georgia-mcp_<x.y.z>.openvex.json`     | The OpenVEX statement of govulncheck at the tag: each known vulnerability of a dependency, and whether it affects the server |
| `checksums.txt`                        | The SHA-256 of each archive, SBOM, and the OpenVEX file  |
| `checksums.txt.sigstore.json`          | The keyless `cosign` signature of `checksums.txt`        |
| `georgia-mcp_<x.y.z>.intoto.jsonl`     | The build provenance of each file of `checksums.txt`     |

release-drafter writes the notes of the release. The owner publishes the draft, which
creates the tag. Before GoReleaser, the job runs `govulncheck -format openvex` and sets the
author and the product (`pkg:golang/github.com/abgeo/georgia-mcp@v<x.y.z>`) of the
statement. GoReleaser finds the release of the tag, keeps its notes, and attaches
the files. `actions/attest` records the provenance with GitHub, and the job attaches its
bundle. `GORELEASER_VERSION` pins GoReleaser, and Renovate proposes a new release.

**Why:** A user of stdio needs no Go and no Docker. The signature and the provenance let a
user check that a file comes from this workflow. The Scorecard finds the signature and
the provenance by the suffixes `.sigstore.json` and `.intoto.jsonl`. The OpenVEX file
tells a scanner which finding of a dependency does not reach the code of the server. GoReleaser adds files
to a published release, so the repository keeps immutable releases off.

## BLD-014

`.github/labels.yml` holds every label of the repository, with its color and its
description. `.github/workflows/labels.yml` syncs the labels on a push to `main` that
changes the file, and on a manual run. It deletes a label that the file does not hold. A
pull request that changes the file runs the sync without a change and shows its plan. To
rename a label, the file gives the old name in `from_name`.

The autolabeler of release-drafter labels a pull request from its title (`GIT-002`):

| Title                         | Label           |
| ----------------------------- | --------------- |
| `<type>(<scope>)!:`           | `breaking`      |
| `<type>(deps):`               | `dependencies`  |
| `feat:`                       | `enhancement`   |
| `fix:`                        | `bug`           |
| `docs:`                       | `documentation` |

It runs in `.github/workflows/release-drafter.yml` on each change to a pull request from a
branch of the repository, and removes such a label when the title no longer matches. A
fork gets a token that cannot add a label. `skip-changelog` keeps a pull request out of
the release notes. The owner adds it by hand.

**Why:** A label that lives only in the settings drifts, and a label that nobody adds
filters nothing. The title already follows `GIT-002`, so it gives the label.

## BLD-015

`go test ./...` runs only the seed corpus of each fuzz test of `TST-009`.
`.github/workflows/fuzz.yml` fuzzes each of them on Monday and Thursday at 02:41 UTC,
and on a manual run:

1. A job lists each fuzz test with `go test -list '^Fuzz'`. A new fuzz test joins the
   run with no change to the workflow.
2. A job for each fuzz test runs
   `go test -run '^$' -fuzz '^<name>$' -fuzztime 5m <package>`. A manual run sets the
   minutes. The timeout of the job, 30 minutes, caps them.
3. Each job restores the generated corpus of its fuzz test (`$GOCACHE/fuzz`) from the
   cache of the last run. A run on `main` saves the corpus again, also when the job
   fails.
4. A failed job keeps the failing input from `testdata/fuzz/<name>` as an artifact.

The run uses no `-race`. A fuzz test parses one input on one goroutine, and the race
detector slows each input.

A run on `main` reports to one issue, titled `Fuzz tests fail`, as `BLD-010` does for
the live tests. The issue lists each failed fuzz test. The job that writes to the issue
reads no code.

To fix a failure, commit the failing input under `testdata/fuzz/<name>/` of the
package. `go test ./...` then runs it as a seed, and the fix makes it pass.

**Why:** The seed corpus tests the inputs that the author thought of. Fuzzing finds the
input that nobody thought of. GitHub drops a cache that no run reads for 7 days, so two
runs a week keep the corpus.

## BLD-016

`build/cursor-plugin/` holds the plugin of the Cursor Marketplace, and
`.cursor-plugin/marketplace.json` at the root of the repository points to it.

| File                                             | Holds                                                                  |
| ------------------------------------------------ | ---------------------------------------------------------------------- |
| `build/cursor-plugin/.cursor-plugin/plugin.json` | The name `georgia-mcp`, the display name `Georgia MCP`, the version, the description of `BLD-011`, the author, the licence, and the logo. |
| `build/cursor-plugin/mcp.json`                   | The server `georgia-mcp` at `https://geo-mcp.abgeo.cloud/mcp`, as the install link of `WEBSITE-DD-026` gives it. |
| `build/cursor-plugin/assets/logo.svg`            | A copy of `website/public/favicon.svg`.                                |
| `.cursor-plugin/marketplace.json`                | The owner, the version, and the entry with the source `./build/cursor-plugin`. |

Before each release, the owner writes the version `<x.y.z>` of the tag into
`plugin.json` and `marketplace.json`, in the last pull request before the tag. No
workflow writes it.

**Why:** Cursor reads the plugin from the default branch of the repository. It has no
upload, and it needs no tag. Cursor reviews each change by hand before the listing
changes, so a change to the plugin reaches readers late. A workflow that prepares each
file of a release replaces the manual step later.

## Retired identifiers

This file has no retired identifier.
