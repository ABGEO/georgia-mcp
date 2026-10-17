---
id: TRC
title: Traceability
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [docs/specs/, tests]
related: [LNG, PRC, GIT, SPC]
---

# Traceability

Every item in a specification document has an identifier.
An identifier never changes and never repeats.

## TRC-001

The count of the segments tells you the level of an identifier.

| Level     | Form                 | Segments | Example          |
| --------- | -------------------- | -------- | ---------------- |
| Guideline | `<GID>-<NNN>`        | 2        | `MCP-004`        |
| Feature   | `<KEY>-<TYPE>-<NNN>` | 3        | `NBGRATE-FR-001` |

Two identifier kinds do not use this grammar:

| Identifier      | Form         | Example       |
| --------------- | ------------ | ------------- |
| Glossary term   | `GLO-<term>` | `GLO-context` |
| Decision record | `ADR-####`   | `ADR-0001`    |

## TRC-002

| Prefix kind          | Characters | Owner                  |
| -------------------- | ---------- | ---------------------- |
| Guideline ID (`GID`) | 2 to 3     | One guideline file.    |
| Feature key (`KEY`)  | 4 to 8     | One feature directory. |

The length rule guarantees that the two sets cannot collide.
A prefix uses uppercase letters and digits. It names the capability, not the
technology. A prefix becomes permanent when the owner approves the first document
that uses it. A retired prefix never repeats.

## TRC-003

The `TYPE` segment exists at the feature level only.

| Type  | Name                       | Document          | Answers                               |
| ----- | -------------------------- | ----------------- | ------------------------------------- |
| `FR`  | Functional requirement     | `requirements.md` | What must the product do?             |
| `NFR` | Non-functional requirement | `requirements.md` | How well must the product do it?      |
| `INV` | Invariant                  | `requirements.md` | What is always true?                  |
| `DD`  | Design decision            | `spec.md`         | Which technical option did we select? |
| `SC`  | Scenario                   | `spec.md`         | How do we prove the behavior?         |

## TRC-004

A counter belongs to one guideline file, or to one pair of `KEY` and `TYPE`.
`NBGRATE-FR-001` and `NBGRATE-DD-001` are two different items.
Allocate the next free number. Do not fill a gap. A draft renumbers freely.

## TRC-005

An approved identifier never repeats. To remove an item:

1. Delete the item from the body of the document.
2. Add one row to the `Retired identifiers` section at the end of the same document.
3. Give the date and the reason.

```markdown
| ID             | Retired    | Reason                               |
| -------------- | ---------- | ------------------------------------ |
| NBGRATE-FR-004 | 2026-11-02 | The behavior moved to BANKRATE-FR-002. |
```

A document with no retired identifier holds the sentence
"This file has no retired identifier." in place of the table.

## TRC-006

- A specification section cites the requirement identifiers that it realizes.
- A requirement cites a guideline rule only when the rule limits the statement.
- Write a reference as the bare identifier: `MCP-004`, `NBGRATE-FR-001`.
- A broken reference is a defect.

A document refers upward. A guideline does not name a feature. A requirements
document does not name a design decision.

```
<GID>-###  <---  <KEY>-FR-###  <---  <KEY>-DD-###
                      ^                     ^
                <KEY>-INV-###         <KEY>-SC-###  <---  test
```

## TRC-007

A test cites the identifier that it verifies, in a comment above the test. `TST-001`.

**Production code carries no identifier.** A comment there gives the reason that
the code cannot give.

**Why:** An identifier in production code goes stale when a document moves, and
nothing fails when it does. The test fails when it and its statement disagree.

## TRC-008

Every markdown document in `docs/specs/` starts with a YAML frontmatter block.
The file name holds the slug only. The frontmatter holds the identifier.

| Field     | Rule                                                                     |
| --------- | ------------------------------------------------------------------------ |
| `id`      | The prefix that this file owns. An index file has no `id`.               |
| `title`   | The name of the document. The first heading repeats it.                  |
| `type`    | One of: `guideline`, `glossary`, `adr`, `requirements`, `spec`, `index`. |
| `status`  | See the table below.                                                     |
| `created` | The date of the first version, as `YYYY-MM-DD`.                          |
| `updated` | The date of the last change, as `YYYY-MM-DD`.                            |

| Type           | Additional fields                                                                  |
| -------------- | ---------------------------------------------------------------------------------- |
| `guideline`    | `scope` (list of paths), `related` (list of guideline identifiers)                 |
| `requirements` | `context`, `approved_by`, `approved_on`, `constrained_by` (list of guideline IDs)  |
| `spec`         | `context`, `approved_by`, `approved_on`, `constrained_by`, `requirements` (path)   |
| `adr`          | `changes` (list of rule identifiers), `supersedes`, `superseded_by`, `decided`     |

| Type                    | Values of `status`                               |
| ----------------------- | ------------------------------------------------ |
| `guideline`, `glossary` | `active`, `retired`                              |
| `requirements`, `spec`  | `draft`, `in-review`, `approved`                 |
| `adr`                   | `proposed`, `accepted`, `rejected`, `superseded` |
| `index`                 | `active`                                         |

## The prefix register

Add a row before you create the file or the directory.

### Guideline identifiers

| ID    | File                           | Status |
| ----- | ------------------------------ | ------ |
| `ARC` | `guidelines/architecture.md`   | Active |
| `DOM` | `guidelines/domain.md`         | Active |
| `PKG` | `guidelines/package-layout.md` | Active |
| `MCP` | `guidelines/mcp-tools.md`      | Active |
| `CLT` | `guidelines/clients.md`        | Active |
| `RSL` | `guidelines/resilience.md`     | Active |
| `API` | `guidelines/http.md`           | Active |
| `CFG` | `guidelines/configuration.md`  | Active |
| `CLI` | `guidelines/cli.md`            | Active |
| `LIF` | `guidelines/lifecycle.md`      | Active |
| `LOG` | `guidelines/logging.md`        | Active |
| `GO`  | `guidelines/go-style.md`       | Active |
| `TST` | `guidelines/testing.md`        | Active |
| `BLD` | `guidelines/build-release.md`  | Active |
| `PRC` | `guidelines/process.md`        | Active |
| `SPC` | `guidelines/specification.md`  | Active |
| `GIT` | `guidelines/git.md`            | Active |
| `TRC` | `guidelines/traceability.md`   | Active |
| `LNG` | `guidelines/language.md`       | Active |
| `GLO` | `guidelines/glossary.md`       | Active |

### Decision records

| ID    | Directory | Status |
| ----- | --------- | ------ |
| `ADR` | `adr/`    | Active |

### Feature keys

| Key        | Directory            | Context    | Feature                                                        | Status |
| ---------- | -------------------- | ---------- | -------------------------------------------------------------- | ------ |
| `PLATFORM` | `features/platform/` | `platform` | The layout of the server, its shared parts, and their wiring.  | Approved |

## Retired identifiers

This file has no retired identifier.
