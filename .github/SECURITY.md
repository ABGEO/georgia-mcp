# Security policy

## Supported versions

| Version                                   | Supported |
| ----------------------------------------- | --------- |
| The hosted server, `geo-mcp.abgeo.cloud`  | Yes       |
| The latest release                        | Yes       |
| An older release                          | No        |

A fix goes into a new release. Update to it.

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
