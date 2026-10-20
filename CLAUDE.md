# Georgia MCP

An MCP server for Georgian public services. One Go binary (`cmd/georgia-mcp`)
serves the tools over streamable HTTP and stdio. Each subject is a bounded context
under `internal/app/georgia-mcp/<context>`. Each vendor API is a standalone Go module under
`pkg/<vendor>`.

## Writing rules that apply to everything you produce

`docs/specs/guidelines/language.md` (`LNG`) binds every document in `docs/specs/`, this file,
every Go comment, every tool description, and every commit message.

- **Never use an em dash.** Use a period, a comma, a colon, or parentheses. (`LNG-006`)
- **Never use a filler word.** No "leverage", "robust", "comprehensive", "crucial",
  "seamless", "modern", "furthermore", "delve". `LNG-007` holds the full list.
- Use the active voice. Write short sentences. One instruction in one sentence. (`LNG-004`, `LNG-005`)
- **Write a comment only when the code cannot explain itself.** (`LNG-011`, `LNG-012`)
- Use the same word for the same thing. The glossary holds each term. (`LNG-002`)
- **Keep a document minimal.** No preamble, no closing summary. (`LNG-013`)

## Spec driven development, read this first

**The code comes from the documents.** `docs/specs/` is the only source of truth.

| Read                                                                   | For                                         |
| ---------------------------------------------------------------------- | ------------------------------------------- |
| [`docs/specs/README.md`](docs/specs/README.md)                                   | The process. The four stages. The two lanes. |
| [`docs/specs/guidelines/`](docs/specs/guidelines/README.md)                      | The rules. One file for each subject.        |
| [`docs/specs/guidelines/traceability.md`](docs/specs/guidelines/traceability.md) | The identifiers and the frontmatter.         |
| [`docs/specs/adr/`](docs/specs/adr/README.md)                                    | The decisions behind the rules.              |

Before you write code:

1. Read the guideline that governs the work. The prefix of a rule names its file:
   `MCP-004` is in `guidelines/mcp-tools.md`.
2. Find the feature in `docs/specs/features/<key>/`. Read `requirements.md`, then `spec.md`.
3. Cite the scenario identifier in the test that verifies it. Production code
   carries no identifier. (`TRC-007`)

**Full lane:** a new context, a new vendor client, a new tool, or a change to a tool
that a client can see. Use `/sdd-requirements`, `/sdd-spec`, `/sdd-implement`, `/sdd-review`.

**Fast lane:** a defect, a refactor that keeps the behavior, a dependency update.
Cite the identifier that the change restores.

**The gate:** if no statement covers the work, the fast lane is not correct. Stop.

## Commands

`BLD-001` holds the table. Run the commands of the module you changed.

| You changed        | Run                                                                      |
| ------------------ | ------------------------------------------------------------------------ |
| Server (`.`)       | `golangci-lint run ./...` then `go test -race ./...` then `go build ./...` |
| `pkg/<vendor>` | `cd pkg/<vendor> && golangci-lint run ./... && go test -race ./...`, then the server |
| `website/`     | `cd website && npm run lint && npm test && npm run build`                 |

## Rules that break most often

- A dependency points inward: `adapter` to `app` to `domain`. `domain` imports the
  standard library and `internal/pkg/kernel` only. (`ARC-007`)
- A context never imports another context. (`ARC-008`)
- A vendor shape never enters `domain` or `app`. The driven adapter converts it. (`DOM-005`)
- A port lives in `app/port.go` and names a capability, not a vendor. (`DOM-009`)
- A vendor client imports the standard library and resty only. Never the server. (`CLT-002`)
- A vendor client does not retry and does not cache. (`CLT-008`)
- A rate or an amount is never a float. In a tool output it is a decimal string. (`DOM-006`, `MCP-009`)
- Every tool is read-only, with the annotations of `MCP-004`.
- Every tool output carries `source` and `as_of`. (`MCP-005`)
- A tool failure is a tool result with `IsError`, never a vendor body. (`MCP-006`)
- In the stdio transport, nothing writes to stdout except the protocol. (`LOG-002`)
- A context reads its configuration from its `Config` struct, never from the environment. (`CFG-004`)
- Every exported identifier has a documentation comment. (`GO-003`)
- Wrap an error with `fmt.Errorf("doing the thing: %w", err)`. (`GO-004`)
- A test uses `testing` and `go-cmp`. No assertion library, no generated mock. (`TST-005`)
- Never change a test to make it pass. Correct the specification. (`TST-003`)

## Git

`docs/specs/guidelines/git.md` (`GIT`) holds every rule about a commit.

- **Commit only when the owner asks in that session.** Never push without a direct instruction. (`GIT-001`)
- **One line. No body.** `<type>(<scope>): <description>` (`GIT-002`)
- Scope: a context (`rates`), a vendor (`nbg`), `platform`, `server`, `specs`, `deps`. (`GIT-004`)
- **The system git user is the sole author. Never add a `Co-Authored-By` trailer,
  and never add any line that names a tool or a model.** (`GIT-006`)
