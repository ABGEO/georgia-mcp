---
title: Spec driven development in Georgia MCP
type: index
status: active
created: 2026-10-17
updated: 2026-10-17
---

# Spec driven development in Georgia MCP

This directory holds the specifications of Georgia MCP.
Code comes from these documents. The documents do not come from the code.

Read this file one time in full. It is the narrative guide to the process.
The binding rules live in the guidelines:

| Guideline | File                                                       | Holds                                |
| --------- | ---------------------------------------------------------- | ------------------------------------ |
| `PRC`     | [guidelines/process.md](./guidelines/process.md)           | The rules of the process.            |
| `TRC`     | [guidelines/traceability.md](./guidelines/traceability.md) | The identifiers and the frontmatter. |
| `LNG`     | [guidelines/language.md](./guidelines/language.md)         | How to write every document.         |

[guidelines/README.md](./guidelines/README.md) indexes every guideline.

## 1. The four stages

| Stage           | Input                             | Output                           | Approval           |
| --------------- | --------------------------------- | -------------------------------- | ------------------ |
| 0 Guidelines    | The system as it must be          | `guidelines/<slug>.md`           | Owner, per rule    |
| 1 Requirements  | An idea, a problem, a need        | `features/<key>/requirements.md` | Owner, per feature |
| 2 Specification | The requirements, the guidelines  | `features/<key>/spec.md`         | Owner, per feature |
| 3 Code          | The specification, the guidelines | Source and tests                 | Owner, per change  |

### Stage 0: Guidelines

The guidelines hold the rules of the whole system. One file holds one subject.
The file identifier is the prefix of each rule in it.
`MCP-004` is the fourth rule in `guidelines/mcp-tools.md`.

- The guidelines exist before the first requirements document.
- A guideline rule is not about one feature.
- A change to a rule needs an ADR in `adr/`.
- After a change to a rule, examine each specification that cites the rule.

### Stage 1: Requirements

The requirements document tells what the product must do. It does not tell how.

- One statement has one identifier and one behavior.
  Divide a statement that joins two behaviors with the word "and".
- A statement stays true after a full re-implementation in a different language.
- A statement is testable from the outside.
  A non-functional statement gives the quantity, the limit, and the measurement point.
- A statement gives the reason in a `Why` line, unless the reason is obvious.
- A statement names no technology: no library, no package, no data structure.
- A domain noun is a term in the [glossary](./guidelines/glossary.md).
- A statement that is not trivial has two to four examples: a normal case,
  a limit case, and an unwanted case. They become scenarios in stage 2.

A requirements document does more than repeat the words of the owner. Before the review:

1. Find the contradictions between the statements. Remove them.
2. Estimate the cost. Tell the owner when a statement is expensive.
3. Ask for the aim behind an expensive statement. Propose a cheaper alternative.
4. Close the gaps. Ask for each state, each transition, and each error case.
5. Ask a question when two readings of a statement are possible.

### Stage 2: Specification

The specification tells how the product does what the requirements demand.
It holds every technical decision that the code generation needs.

- The specification obeys the guidelines. It does not contradict a rule.
- Each section cites the requirement identifiers that it realizes.
- Each design decision has an identifier, a `Decision` line, and a `Rationale` line.
- Each non-functional requirement has a scenario that measures it.
- The specification declares each contract surface that the feature adds. See `SPC`.
- A design decision that changes more than one bounded context becomes an ADR.

### Stage 3: Code

- Write the tests first. Run them. Confirm that they fail.
- Write the code. Run the tests until they pass.
- Do not change a scenario or a test to make a test pass. Go to section 4.
- A test cites the scenario that it verifies. Production code cites none. See `TRC-007`.
- Run the linter, the tests, and the build of each part that the change touches.
  `BLD-001` gives the commands.
- A pass is an exit code. A visual check is not a pass.

## 2. The two lanes

### The full lane

Use the full lane for:

- A new bounded context, or a new vendor client.
- A new MCP tool, or a change to the input or the output of a tool.
- A change to the behavior that a client of the server can see.

The full lane runs stage 1, then stage 2, then stage 3.

### The fast lane

Use the fast lane for:

- A defect correction.
- A refactor that keeps the behavior.
- A dependency update.
- A build or tooling change.

The fast lane makes no new document. The change cites the identifier that it restores.

**The gate:** if no statement covers the work, the fast lane is not correct.
Stop. Open the full lane.

## 3. Directory layout

```
docs/specs/
  README.md                  This guide. Type: index.
  guidelines/                The rules. One file for each subject.
    README.md                The index. It names every guideline and its prefix.
  adr/                       The decision records. One file for each decision.
    README.md                The ADR index.
    template.md              The ADR template.
  templates/
    requirements.md          The template for stage 1.
    spec.md                  The template for stage 2.
  features/
    <key>/                   One directory for each feature. The key is lowercase.
      requirements.md        Stage 1.
      spec.md                Stage 2.
  work/                      Not in git. The working notes.
```

A file name holds the slug only. The frontmatter holds the identifier. See `TRC-008`.
The feature key `NBGRATE` uses the directory `features/nbgrate/`.

## 4. The change process

Find the earliest document that is wrong. Check in this order:

| Symptom                                        | Classification       | Action                                                            |
| ---------------------------------------------- | -------------------- | ----------------------------------------------------------------- |
| The code contradicts the specification         | Defect               | Correct the code. Fast lane.                                      |
| The specification contradicts the requirements | Specification change | Correct `spec.md`. Generate the code again.                       |
| The requirements do not express the need       | Requirement change   | Correct `requirements.md`, then the specification, then the code. |
| A guideline rule is wrong                      | Guideline change     | Write an ADR. Then examine every specification.                   |

- Correct the wrong document first. Do not correct the code first.
- After a change to an identifier, find each reference to it. Examine each one.
- A new capability starts with a requirement statement, not with a technology choice.
