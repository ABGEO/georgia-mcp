# Governance

Georgia MCP has one maintainer. The maintainer decides, reviews, merges, and releases.

## Roles

| Role        | Who                                          | Responsibilities                                                                                                                                                         |
| ----------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Maintainer  | Temuri Takalandze ([@ABGEO](https://github.com/ABGEO)) | Sets the direction and the specifications. Reviews and merges each pull request as the code owner. Publishes each release. Answers each vulnerability report, as [`SECURITY.md`](SECURITY.md) says. Runs the hosted server. Holds each resource of the next section. |
| Contributor | Anyone                                       | Opens an issue, a discussion, or a pull request, as [`CONTRIBUTING.md`](CONTRIBUTING.md) says. Signs off each commit. A contributor has no write access and no access to a sensitive resource. |
| Bot         | Renovate, release-drafter, GitHub Actions    | Renovate proposes the dependency updates. release-drafter writes the draft of the release notes. The workflows build, test, sign, and publish, with the permissions that each job declares. A bot merges nothing without the approval of the maintainer. |

A decision happens in an issue, a discussion, or a pull request. The maintainer has
the last word.

## Access to sensitive resources

The maintainer is the only member with access to each resource in this table.

| Resource                                                       | Access                                    |
| -------------------------------------------------------------- | ----------------------------------------- |
| The repository `ABGEO/georgia-mcp`, its settings, and its ruleset | Admin, the only one who can bypass the ruleset |
| The Actions secrets: the API keys of TTC, TBC Bank, and Credo Bank | Admin of the repository                   |
| The container images on `ghcr.io/abgeo`                         | Owner of the packages                     |
| The hosted server `geo-mcp.abgeo.cloud` and its deployment configuration, with the same API keys | Operator                                  |
| The domains `abgeo.cloud` and `abgeo.dev`                       | Owner                                     |
| The listings in the Claude directory and the OpenAI plugins     | Owner                                     |
| The project 16395 on bestpractices.dev                          | Owner                                     |
| The private vulnerability reports of the repository             | Admin of the repository                   |

These resources need no stored secret. A workflow proves its identity to each of them
with the OIDC token of GitHub Actions:

- The cosign signatures of the release files and of the images (keyless).
- The build provenance of each release.
- The entry `io.github.ABGEO/georgia-mcp` in the MCP Registry.
- The coverage upload to Codecov.

## Granting access

A contributor gets write access, or access to a sensitive resource, only after a record
of reviewed contributions. The maintainer proposes it in a public issue and decides
there. A new maintainer turns on two-factor authentication and signs each commit.
The access starts with the least that the role needs.

When a person joins as a maintainer, this file names them and the resources they get.
When a person leaves, the maintainer removes their access and rotates each secret
that they held.

## Secrets

| Secret                      | Stored in                                                      | Who reads it                                     |
| --------------------------- | -------------------------------------------------------------- | ------------------------------------------------ |
| The API keys of the sources | The Actions secrets of the repository, and the environment of the hosted server | The live tests, the hosted server, the maintainer |
| A key of a contributor      | The local `.env`, which git ignores                            | That contributor                                 |

- A secret never enters the repository, an issue, a log, or an error (`CFG-007`,
  `LOG-008`). Push protection of secret scanning blocks a known key format.
- Only the scheduled live tests read the API keys. No workflow that a pull request
  starts reads a secret.
- Signing and publishing use the OIDC token of a workflow, never a stored key.
- The maintainer rotates a secret at its source when it leaks, when a person who held
  it leaves, or when the source asks. Then the maintainer replaces it in each store.
