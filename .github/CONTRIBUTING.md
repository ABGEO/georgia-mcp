# Contributing

Bug reports and proposals go in [issues](https://github.com/abgeo/georgia-mcp/issues).
Open an issue before a pull request for a new source or a new tool.

## The process

The code comes from the documents in [`docs/specs/`](../docs/specs/README.md). Read that
guide first.

| Lane      | For                                                                     | Start with                                       |
| --------- | ----------------------------------------------------------------------- | ------------------------------------------------ |
| Full lane | A new context, a new vendor client, a new tool, or a change that a client can see | `requirements.md`, then `spec.md`, in `docs/specs/features/<key>/` |
| Fast lane | A defect, a refactor that keeps the behavior, a dependency update        | The identifier that the change restores          |

If no statement in `docs/specs/` covers the change, the fast lane is not correct.

## Setup

You need Go (the version in [`go.mod`](../go.mod)) and [Task](https://taskfile.dev).
For the website, you need Node (the version in `website/.nvmrc`) and pnpm.

```sh
cp configs/.env.example .env
task run
```

## Checks

Run the commands of the part that you changed. Each must pass.

| You changed                      | Run                                       |
| -------------------------------- | ----------------------------------------- |
| Go (`internal/`, `pkg/`, `cmd/`) | `task lint test build`                    |
| `website/`                       | `task website:lint website:test website:build` |

## Pull request

- The title is the commit message: `<type>(<scope>): <description>`.
  [`git.md`](../docs/specs/guidelines/git.md) holds the types and the scopes. A check
  fails a title with an unknown type.
- The pull request merges as one squashed commit.
- A test cites the identifier of the scenario that it verifies.
- A document, a comment, and a tool description follow
  [`language.md`](../docs/specs/guidelines/language.md).

By contributing, you agree to the [code of conduct](CODE_OF_CONDUCT.md) and to license
your work under the [GPL-3.0](../LICENSE).
