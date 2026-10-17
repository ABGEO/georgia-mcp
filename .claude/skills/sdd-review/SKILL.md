---
name: sdd-review
description: Review the implementation of a feature against its specification, the guidelines, Effective Go, the Uber Go Style Guide, and the MCP tool rules. Use after stage 3, before the owner commits. Triggers - "review the feature", "check the implementation", "is this compliant", "review <KEY>".
---

# Review: the implementation of a feature

Usage: `/sdd-review <feature key> [path ...]`

You review the code that `docs/specs/features/<key>/` produced. You change no file unless
the owner asks for a correction. The review is a report.

## Read first

1. `docs/specs/features/<key>/requirements.md` and every `spec*.md`.
2. The `constrained_by` guidelines of each document, in full. Do not review from memory.
3. `go-style.md`, `domain.md`, `package-layout.md`, `mcp-tools.md`, `clients.md`,
   `resilience.md`, `logging.md`, `testing.md`, `language.md`.
4. Every file that the components table names, and every test. Read each file whole.

## Steps

1. **Run the checks** of `BLD-001` for each module of the feature, with `-count=1`.
   A failure is a finding.
2. **Check the trace.** Every requirement has a coverage row and code that realizes
   it. Every scenario has a test that cites it, at its layer. Every declaration
   table matches the code: tool names, input and output structs, annotations, cache
   periods, configuration variables, vendor endpoints.
3. **Check the guidelines.** Report only a rule that breaks. The rules that break most often:

   | Area      | Rules                                                                     |
   | --------- | ------------------------------------------------------------------------- |
   | Layers    | `ARC-007`, `ARC-008`, `DOM-005`, `DOM-009`, `PKG-002`, `PKG-003`          |
   | Domain    | `DOM-003` validating constructor, `DOM-006` no float, `DOM-007` zone      |
   | Tools     | `MCP-002` to `MCP-009`: name, schema tags, annotations, `as_of`, errors   |
   | Clients   | `CLT-002`, `CLT-005` to `CLT-009`                                         |
   | Resilience | `RSL-001` timeout, `RSL-003` cache as a decorator, `RSL-005` one call    |
   | Logging   | `LOG-002` stdout in stdio, `LOG-005`, `LOG-008`                           |
   | Comments  | `LNG-006`, `LNG-007`, `LNG-011`, `TRC-007`                                |

4. **Check the Go code** against Effective Go and the Uber guide (`GO-010`): a dropped
   error, a double log, a sentinel compared with `==`, an interface at the producer,
   a constructor that returns an interface, a goroutine with no owner, a dropped
   context, a mutable package variable, an `init`, a magic number.
5. **Check the tool from the side of the model.** Read each description and each
   field description as a model reads them. Is the unit given? Is the source named?
   Does each error message name the next action? Is the output bounded?
6. **Check the quality.** A `nolint` with no reason, a test that asserts less than its
   scenario, a fake that tests nothing, a call with no timeout, a secret in a log.

## Report

| #   | Severity | File:line | Rule | Finding | Correction |
| --- | -------- | --------- | ---- | ------- | ---------- |

- **Blocker**: a broken requirement, a security defect, a failing check.
- **Major**: a broken guideline rule, a declaration that differs from the code, a missing test.
- **Minor**: style, naming, a comment, a missing edge test.

After the table: the result of each check, and each finding that needs a decision of
the owner, as a question. Do not commit.
