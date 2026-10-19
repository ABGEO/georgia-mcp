---
title: Architecture decision records
type: index
status: active
created: 2026-10-17
updated: 2026-10-17
---

# Architecture decision records

An ADR records a decision that outlives one feature.
An accepted ADR is the only thing that changes a guideline.

Write an ADR when:

- A guideline rule is wrong or incomplete.
- A design decision changes more than one bounded context.
- The system needs a new technology.
- The guidelines need a new file.

A decision inside one feature is a `DD` item in its specification, not an ADR.

The identifier has the form `ADR-####`, from `0001`. The number is permanent.
The file name holds the slug only. See `TRC-008`.

| Status     | Meaning                                                |
| ---------- | ------------------------------------------------------ |
| proposed   | The ADR exists. The owner did not decide.              |
| accepted   | The owner accepted it. The guidelines change.          |
| rejected   | The owner rejected it. The file stays for the history. |
| superseded | A later ADR replaced it. Name the later ADR.           |

## The index

| ID         | Title                                        | File                                      | Status   | Date       | Rules that change |
| ---------- | -------------------------------------------- | ----------------------------------------- | -------- | ---------- | ----------------- |
| `ADR-0001` | Bounded contexts with ports and adapters     | `bounded-contexts-with-ports-and-adapters.md` | accepted | 2026-10-17 | None. It founds `ARC`, `DOM`, `PKG`. |
| `ADR-0002` | The civil date in the shared kernel          | `civil-date-in-the-shared-kernel.md`      | accepted | 2026-10-19 | `DOM-008` |
| `ADR-0003` | A vendor client of a web page                | `vendor-client-of-a-web-page.md`          | accepted | 2026-10-19 | `CLT-002` to `CLT-005` |
