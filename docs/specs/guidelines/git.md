---
id: GIT
title: Git conventions
type: guideline
status: active
created: 2026-10-17
updated: 2026-11-07
scope: [.]
related: [PRC, LNG, CLT]
---

# Git conventions

A commit message names no statement identifier. The specification holds the
traceability, not the history.

## GIT-001

The owner makes every commit. A commit happens only when the owner asks for it in
the same session. A push happens only on a direct instruction of the owner.

## GIT-002

A commit message is one line, followed by the sign-off of `GIT-008`. It has no other
body.

```
<type>(<scope>): <description>

Signed-off-by: <name> <email>
```

## GIT-003

The type comes from this table. Do not invent a type.

| Type       | Use                                                            |
| ---------- | -------------------------------------------------------------- |
| `feat`     | New behavior that a client of the server can see.              |
| `fix`      | A defect correction.                                           |
| `refactor` | A change that keeps the behavior.                              |
| `perf`     | A change that improves the speed or the memory use.            |
| `test`     | A change to a test only.                                       |
| `docs`     | A change to a document only, including a document in `docs/specs/`. |
| `build`    | A change to the build or the container image.                  |
| `chore`    | A change to the tooling or to a dependency.                    |

## GIT-004

The scope names the part of the repository that the change touches.

| Scope       | For                                                              |
| ----------- | ---------------------------------------------------------------- |
| `<context>` | `internal/app/georgia-mcp/<context>`, for example `rates`        |
| `<vendor>`  | `pkg/<vendor>`, for example `nbg` or `ttc`                       |
| `platform`  | `internal/pkg`                                                   |
| `server`    | `cmd`, `internal/app/georgia-mcp/cli`, `bootstrap`, `test`       |
| `specs`     | `docs/specs/`                                                    |
| `build`     | `build/`, `deployments/`, `configs/`, `Taskfile.yml`             |
| `website`   | `website/`                                                       |
| `deps`      | A dependency update                                              |

Write no scope when the change touches no single part.

## GIT-005

The description uses the imperative mood, starts with a lowercase letter, and has
no full stop. The full line has 72 characters maximum. It obeys `LNG`.

```
feat(rates): add the tool that returns the official rates
fix(nbg): return an error for an empty answer
docs(specs): add the requirements of the official rates
```

## GIT-006

The git user of the system is the sole author of a commit.
Do not add a `Co-Authored-By` trailer.
Do not add a trailer, a line, or a footer that names a tool or a model.

**Why:** A commit records who is responsible for the change. The owner is responsible.

## GIT-007

A release is a tag `v<semver>`. A vendor client has no tag of its own.

**Why:** The repository is one module. One tag versions the server and every client.

## GIT-008

Each commit of a pull request carries a `Signed-off-by` trailer with the name and the
email of its author: `git commit -s`. The sign-off certifies the
[Developer Certificate of Origin](https://developercertificate.org/). The job `DCO` of
`BLD-009` fails a pull request with a commit that has no sign-off. It skips a commit of
a bot and a merge commit. The repository also requires a sign-off on a commit made on
the web.

**Why:** The sign-off records that the author has the right to submit the change under
the GPL-3.0. A bot certifies nothing, and a merge commit adds no work of its own.

## Retired identifiers

This file has no retired identifier.
