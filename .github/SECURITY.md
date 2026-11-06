# Security policy

## Supported versions

| Version                                   | Supported |
| ----------------------------------------- | --------- |
| The hosted server, `geo-mcp.abgeo.cloud`  | Yes       |
| The latest release                        | Yes       |
| An older release                          | No        |

A release gets security fixes until the next release ships. Then it gets no more fixes,
and a fix goes into a new release. Update to it. The hosted server always runs a
supported release.

## Dependency and code findings

CI checks each pull request and each push to `main`. A blocking finding fails the
required check `CI`, so nothing merges and nothing ships while it is open.

| Finding                                                                  | Tool              | Threshold                                             |
| ------------------------------------------------------------------------ | ----------------- | ----------------------------------------------------- |
| A known vulnerability that the Go code calls                             | govulncheck       | Blocks at any severity. Fix it before the next merge. |
| A known vulnerability in a new or changed dependency                     | Dependency review | Blocks at any severity.                               |
| A dependency with a license outside the allowed list                     | Dependency review | Blocks. The list holds the licenses compatible with the GPL-3.0. |
| A known vulnerability that the code does not call                        | govulncheck       | Does not block. Take the fixed version when it ships: Renovate proposes it. |
| A weakness in a workflow                                                 | zizmor            | Blocks. Fix it, or suppress it in `.github/zizmor.yml` with the reason why it cannot be exploited. |
| A weakness in the code                                                   | CodeQL            | Fix a high or a critical alert before the next release. Dismiss an alert only with the reason why it cannot be exploited. |

Before a release, the maintainer checks that `main` has no open blocking finding.

Each release carries `georgia-mcp_<version>.openvex.json`, an
[OpenVEX](https://openvex.dev) document from govulncheck. It states each known
vulnerability of a dependency and whether it affects the server, for example
`not_affected` when the code never reaches the vulnerable function. `checksums.txt` and
its signature cover it.

## Report a vulnerability

Do not open a public issue.

Report it in private on the
[security page](https://github.com/abgeo/georgia-mcp/security/advisories/new) of the
repository. If you cannot use GitHub, write through the
[contact page](https://www.abgeo.dev/contact/).

Give the affected version or the hosted server, the steps to reproduce, and the impact.

## Response and disclosure

| Step                             | Time                               |
| -------------------------------- | ---------------------------------- |
| First answer to the report       | 7 days                             |
| Assessment of the report         | 14 days                            |
| Fix of a confirmed vulnerability | 90 days, sooner for a critical one |

The disclosure is coordinated: the advisory and the fix are published together, on the
security page of the repository. The advisory names you, unless you ask otherwise.
Please keep the report private until then.

## Assessment

The [security assessment](../docs/security-assessment.md) names the likely threats and
the defenses against each.

## Scope

In scope:

- The server, in this repository and at `geo-mcp.abgeo.cloud`.
- The vendor clients under `pkg/`.
- The website.
- The container images on `ghcr.io/abgeo`.

Out of scope:

- A site or an API of a source. Report it to its publisher.
- A wrong or a late value from a source. Open a bug report.
- A test that loads the hosted server, or a denial of service.
