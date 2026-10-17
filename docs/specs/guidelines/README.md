---
title: The Georgia MCP guidelines
type: index
status: active
created: 2026-10-17
updated: 2026-10-17
---

# The Georgia MCP guidelines

The guidelines hold the rules of the whole system. A rule applies to every
feature, to every bounded context, and to every line of code.
A specification must not contradict a rule.

A rule is not about one feature. A statement about one feature belongs in
`docs/specs/features/<key>/requirements.md`.

## The guidelines

The frontmatter of a file holds the identifier. The identifier is the prefix of
every rule in that file.

| ID    | File                                       | Governs                                                    |
| ----- | ------------------------------------------ | ---------------------------------------------------------- |
| `ARC` | [architecture.md](./architecture.md)       | The shape of the system. The modules. The dependency rule. |
| `DOM` | [domain.md](./domain.md)                   | The bounded contexts and the domain model.                 |
| `PKG` | [package-layout.md](./package-layout.md)   | The directories and the package names.                     |
| `MCP` | [mcp-tools.md](./mcp-tools.md)             | The MCP server and the contract of a tool.                 |
| `CLT` | [clients.md](./clients.md)                 | The vendor client modules.                                 |
| `RSL` | [resilience.md](./resilience.md)           | The timeouts, the cache, and the limits of a vendor call.  |
| `API` | [http.md](./http.md)                       | The HTTP routes and the middleware.                        |
| `CFG` | [configuration.md](./configuration.md)     | The configuration and the secrets.                         |
| `CLI` | [cli.md](./cli.md)                         | The command tree.                                          |
| `LIF` | [lifecycle.md](./lifecycle.md)             | The start order and the shutdown order.                    |
| `LOG` | [logging.md](./logging.md)                 | The logger, the attributes, the redaction.                 |
| `GO`  | [go-style.md](./go-style.md)               | The Go code style.                                         |
| `TST` | [testing.md](./testing.md)                 | The tests.                                                 |
| `BLD` | [build-release.md](./build-release.md)     | The build and the release.                                 |
| `PRC` | [process.md](./process.md)                 | How a change moves from an idea to the code.               |
| `SPC` | [specification.md](./specification.md)     | The artifacts of a specification and each declaration.     |
| `GIT` | [git.md](./git.md)                         | The commits.                                               |
| `TRC` | [traceability.md](./traceability.md)       | The identifiers and the frontmatter.                       |
| `LNG` | [language.md](./language.md)               | How to write every document, comment, and commit message.  |
| `GLO` | [glossary.md](./glossary.md)               | The terms. One term has one meaning.                       |

## Where to put a new rule

| The rule is about                                          | Put it in |
| ---------------------------------------------------------- | --------- |
| The shape of the system, a module, or a technology choice  | `ARC`     |
| A bounded context, a value object, or a domain error       | `DOM`     |
| A directory or a package name                              | `PKG`     |
| An MCP tool, its input, its output, or its error           | `MCP`     |
| A vendor client module                                     | `CLT`     |
| A timeout, a cache, a rate limit, or a retry               | `RSL`     |
| An HTTP route or the middleware                            | `API`     |
| A configuration value or a secret                          | `CFG`     |
| A command or a flag                                        | `CLI`     |
| The start order or the shutdown                            | `LIF`     |
| A log message or a log attribute                           | `LOG`     |
| How to write Go                                            | `GO`      |
| A test                                                     | `TST`     |
| A build step or a release step                             | `BLD`     |
| How we work                                                | `PRC`     |
| An artifact or a declaration in a specification            | `SPC`     |
| A commit, a branch, or a tag                               | `GIT`     |
| An identifier or a frontmatter field                       | `TRC`     |
| A word, a sentence, or a document                          | `LNG`     |

A rule that fits no guideline needs a new guideline. A new guideline needs an ADR.

## How a rule changes

An accepted ADR in [`docs/specs/adr/`](../adr/) changes a rule. Nothing else changes a rule.

1. Write an ADR. Use [`docs/specs/adr/template.md`](../adr/template.md).
2. Name each rule identifier that changes.
3. After the owner accepts the ADR, correct the guideline file.
4. Set the `updated` field in the frontmatter of the guideline.
5. Find each specification that cites the rule. Examine each one.
6. Add a row to the `Retired identifiers` section for a rule that you remove.
