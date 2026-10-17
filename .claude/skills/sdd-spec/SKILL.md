---
name: sdd-spec
description: Stage 2 of the spec driven process. Write or change the technical specification for a feature from its approved requirements. Use after the requirements document is approved and before any code. Triggers - "write the spec", "design this feature", "technical design", "how do we build".
---

# Stage 2: Specification

Usage: `/sdd-spec <feature key>`

You write `docs/specs/features/<key>/spec.md`. You do **not** write production code.

## Read first

1. `docs/specs/features/<key>/requirements.md`. Its status must be `approved`.
2. `docs/specs/guidelines/README.md`, then each guideline that this feature touches.
   Always: `ARC`, `DOM`, `PKG`, `MCP`, `RSL`. For a new vendor: `CLT`.
3. `docs/specs/guidelines/specification.md` (`SPC`), `traceability.md`, `language.md`.
4. The code that this feature touches. Read the real files. Do not guess a signature.
   Read the real API of each library in the module cache, for example
   `go doc github.com/modelcontextprotocol/go-sdk/mcp AddTool`.

**Stop if the requirements are not `approved`, or if an open question has no answer.**

## Steps

1. **Coverage.** Every `FR`, `NFR`, and `INV` gets a row in section 2.
2. **Source.** Call the vendor one time. Record the real request and the real answer.
   The answer becomes the recorded response of `TST-006`.
3. **Domain model.** Name each value object, read model, domain error, and port,
   with its Go signature. Apply `DOM-003` to `DOM-011`. Add an aggregate only when
   the context enforces a rule across values (`DOM-004`).
4. **Declarations.** Fill section 4.3 with each surface that `SPC-003` lists. Every
   row names a requirement. Give the Go structs of each tool input and output (`SPC-005`).
5. **Options.** When a decision has more than one reasonable option, present the
   options with the AskUserQuestion tool. Record the answer as a `<KEY>-DD-###`.
6. **Scenarios.** Each `NFR` needs a scenario that measures it. Each non-trivial
   `FR` needs the normal, the limit, and the unwanted case. Give each a layer (`TST-004`).
7. **Compliance.** Fill section 3. A design that contradicts a rule needs an ADR.
   A decision that changes more than one context becomes an ADR.
8. **Build plan.** Order the work from the inside out: client, domain, app,
   driven adapter, tool, registration in `bootstrap`.
9. **Report** each decision in one line, each postponed requirement, and each ADR.
   Then stop.

## Self check before you report

- [ ] The frontmatter holds every field that `TRC-008` demands.
- [ ] Every requirement has a row in the coverage table.
- [ ] Every path and type exists, or the document says "create".
- [ ] Every `DD` has a Decision, a Rationale, and an Alternatives line.
- [ ] Every `NFR` has a scenario that measures it.
- [ ] Every tool has a name that obeys `MCP-002`, and its structs in the document.
- [ ] Every port method has a row in the cache table, or a reason for no cache.
- [ ] No section contradicts a guideline rule.
- [ ] The document holds no em dash and no filler word.
- [ ] You wrote no production code.
