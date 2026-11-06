# Security assessment

Assessed on 2026-11-07, for v1.1.2. The maintainer reviews it at each minor release
and after each security report.

## The system

Georgia MCP is a read-only MCP server. A client (an AI assistant) calls a tool. The
server calls the API or the web page of a source, converts the answer, and returns
structured JSON with the source and the time of the data. The server writes nothing to
a source. It keeps no state between two requests, apart from a cache of the answers of
the sources.

| Actor            | Trust                                                         |
| ---------------- | ------------------------------------------------------------- |
| An MCP client    | Untrusted. Anyone can call the hosted server without an account. |
| A source         | Untrusted. Its answer can be wrong, late, hostile, or very large. |
| The operator     | Trusted. Holds the API keys and the deployment.               |
| The maintainer   | Trusted. See [`GOVERNANCE.md`](../.github/GOVERNANCE.md).      |

The assets are the API keys of the sources, the availability of the hosted server, the
integrity of the released files and images, and the privacy of the users.

## Attack surface

| Entry point                       | Reached by                     | Input                                    | Protection                                                    |
| --------------------------------- | ------------------------------ | ---------------------------------------- | ------------------------------------------------------------- |
| `POST, GET, DELETE /mcp`          | Anyone, without an account     | MCP messages, the arguments of a tool    | The rate limit, the `Origin` check, the header limits (`API-002` to `API-007`). The server checks the arguments against the input schema of each tool, which rejects an unknown field. |
| `GET /livez`, `GET /readyz`       | Anyone                         | None                                     | No call to a source, no data (`API-005`).                     |
| stdio                             | The local user who starts it   | MCP messages                             | Runs with the rights of that user. Nothing but the protocol on stdout (`LOG-002`). |
| The calls to the sources          | The server, out to each source | The answer of a source                   | A fixed host for each client, a body limit, a timeout (`CLT-004`, `RSL-001`). |
| The configuration                 | The operator                   | Environment variables                    | Validated at the start. A wrong value stops the server (`CFG`). |
| The workflows                     | A pull request, a push, a tag  | The code and the metadata of the event   | No permission by default. Untrusted values pass through `env`. zizmor checks each workflow. A pull request reads no secret. |

The critical code paths are the tool handlers, the vendor clients that parse the
answers, and the release workflow that signs the files. Threats 1, 4, and 5 cover them.

## Threats

Each threat has a likelihood and an impact: low, medium, or high.

### 1. A hostile or broken answer from a source

Likelihood medium, impact medium. A source changes its format, sends a huge body, or
places text that the server parses.

- A client reads at most a fixed count of bytes from a body (`CLT-004`).
- Each call has a timeout of 10 seconds by default (`RSL-001`), and the server does
  not retry (`RSL-002`).
- An adapter converts the answer into the types of the domain. A tool never returns
  the body of a source (`DOM-005`, `MCP-006`).
- A tool input sets query values only. It never chooses the host or the path of a call.
- The scheduled live tests detect a source that changes its format.

### 2. Prompt injection through the data of a source

Likelihood medium, impact low. The text of a source (an offer, a movie description)
tells the model to do something.

- The text arrives as a field of the structured output, with the source named.
- Every tool is read-only, with `readOnlyHint` true and `destructiveHint` false (`MCP-004`).
  An injected instruction cannot make this server write, pay, or send anything.

The residual risk sits in the client: a model can still act on the text with another
tool. The client decides that.

### 3. A leaked API key

Likelihood low, impact medium. A key leaks in a commit, a log, or an error.

- A secret never enters the repository. `.env` stays out of git (`CFG-007`), and
  push protection of secret scanning blocks a known key format.
- The server never logs a key, a secret, or the arguments of a tool call above `debug`
  (`LOG-008`).
- A tool failure is a short error result, never the body or the headers of a source
  (`MCP-006`).
- Each key reads public data. A leaked key costs the quota of the source, not the data
  of a user. The maintainer rotates it at the source.

### 4. Abuse or overload of the hosted server

Likelihood medium, impact medium. A client floods `/mcp` and exhausts the server or
the quota of a source.

- A rate limit applies to each client address on `/mcp` and answers 429 with
  `Retry-After` (`API-002`, `API-007`). A client cannot choose its own address through
  `X-Forwarded-For`: the server trusts only the proxy ranges of its configuration.
- The `http.Server` limits the header time and size (`API-006`).
- The cache answers a repeated call without a call to the source, and joins two equal
  calls into one (`RSL-005`). When a source fails, it returns a stale value with its
  age (`RSL-006`).
- The server rejects a request whose `Origin` is not allowed (`API-004`), against DNS
  rebinding.

A large distributed attack stays out of scope, as [`SECURITY.md`](../.github/SECURITY.md)
says.

### 5. A compromised dependency or build

Likelihood low, impact high. A dependency, an action, or an image carries malicious
code.

- Each action and each base image is pinned by digest. Renovate proposes each update
  as a pull request, which the maintainer reviews (`BLD`).
- CI runs dependency review, govulncheck, CodeQL, and zizmor on the workflows.
- A vendor client imports the standard library and named exceptions only (`CLT-002`).
- Each workflow starts with no permission. Each job asks for the permissions it needs.
- Each release has `checksums.txt` signed with cosign, SLSA provenance, and SBOMs. The
  images are signed. A user can check each one, as the README says.

### 6. A compromised maintainer account

Likelihood low, impact high. An attacker pushes code or publishes a release.

- The ruleset on `main` requires a pull request, signed commits, a linear history, and
  the checks `CI` and `Images`.
- The signatures and the provenance name the workflow and the tag that built a file.
  A file that someone builds elsewhere fails the check.

### 7. Exposure of the data of a user

Likelihood low, impact medium. The server logs what a user asks.

- The hosted server keeps an access log and a record of each tool call. It never keeps
  the input of a call. See the [privacy policy](https://geo-mcp.abgeo.cloud/privacy/).
- The server needs no account and asks for no personal data.

## Out of scope

- The correctness and the security of each source. Report a problem to its publisher.
- The behavior of an MCP client and of its model.
