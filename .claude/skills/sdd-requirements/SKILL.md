---
name: sdd-requirements
description: Stage 1 of the spec driven process. Write or change a requirements document for a feature, a new bounded context, a new vendor, or a new MCP tool. Use when the user starts a feature or a change to the behavior a client can see. Triggers - "new feature", "new tool", "new context", "add a vendor", "write requirements", "what should this do".
---

# Stage 1: Requirements

Usage: `/sdd-requirements <feature name or description>`

You write `docs/specs/features/<key>/requirements.md`.
You do **not** write code in this stage. You do **not** choose a technology.

## Read first

1. `docs/specs/README.md`, section 1, stage 1.
2. `docs/specs/guidelines/language.md` (`LNG`). No em dash. No filler word.
3. `docs/specs/guidelines/traceability.md` (`TRC`).
4. `docs/specs/guidelines/domain.md` (`DOM`) and `docs/specs/guidelines/mcp-tools.md` (`MCP`).
5. `docs/specs/guidelines/glossary.md`. Use the term from the glossary.

## Steps

### 1. Select the key and the context

Propose a feature key of four to eight uppercase characters (`TRC-002`). It names
the capability, not the technology. Check the register in `traceability.md`.
Name the bounded context that owns the feature (`DOM-001`, `SPC-002`).
Ask the owner to confirm both. Then add the row to the register.

### 2. Examine the source

Find the source of the data before you write a statement. Fill section 3 of the
template: the vendor, the address, official or unofficial, the publication period,
and the terms of use. Call the source one time to see a real answer. Tell the owner
when the source has no published API, a key, or terms that forbid reuse.

### 3. Interrogate the request

Do not write down the words of the owner. Examine them. Ask about:

- The question that a person asks the model, in their own words.
- Each input of the tool, its valid values, and its default.
- Each error case. What does the model tell the person when the vendor fails?
- Each limit: how many items, how old a value can be, which languages.
- The reason behind an expensive statement. Propose a cheaper alternative.

Ask the questions in one batch with the AskUserQuestion tool.

### 4. Write the statements

Copy `docs/specs/templates/requirements.md`. Set every field in the frontmatter (`TRC-008`).

- One identifier. One behavior.
- The statement stays true after a re-implementation in a different language.
- The statement is testable from the outside, through an MCP client.
- A `Why` line gives the reason.
- No technology. No package. No library. No tool name.
- Two to four examples for a statement that is not trivial.

A non-functional statement gives the quantity, the limit, the condition, and the
measurement point. See `LNG-008`.

### 5. Report

Tell the owner the count of the statements by type, each open question, and each
expensive statement with the cheaper alternative. Then stop.

## Self check before you report

- [ ] The frontmatter holds every field that `TRC-008` demands, with `context`.
- [ ] Section 3 names the source and says whether it is official.
- [ ] No statement names a technology, a package, or a library.
- [ ] Every non-functional statement has a number and a measurement point.
- [ ] Every domain noun is in the glossary or in section 5.
- [ ] The document holds no em dash and no filler word. (`LNG-006`, `LNG-007`)
- [ ] The document is within the size that `LNG-013` gives.
- [ ] You wrote no code.
