---
id: ADR-0008
title: A period with a maximum in place of a limit
type: adr
status: accepted
created: 2026-11-17
updated: 2026-11-17
decided: 2026-11-17
changes: [MCP-008]
supersedes:
superseded_by:
---

# A period with a maximum in place of a limit

## Context

`MCP-008` makes a tool that returns a collection take the optional input `limit`, with a
maximum of 100 or less. `rates_list_official_history` takes a period of at most 31 days
and no `limit`. `GNERCMON` adds `energy_list_prices`, which takes a period of at most 60
months and no `limit`. The review of `GNERCMON` found both outside `MCP-008`.

## Decision

A tool whose collection holds one item for each step of a period takes no `limit` when the
specification gives the maximum length of the period, at 100 steps or less. The tool
refuses a longer period with a message that names the maximum.

## Rationale

The period bounds the answer as a `limit` does: 60 months give 60 items. A `limit` next to
a period gives two inputs for one bound, and a model that sets both gets a period cut at
an end that it did not choose.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A `limit` on each tool of a period. | Two inputs for one bound. A `limit` under the length of the period drops the newest or the oldest items with no sign. |
| No change, and an exception in each specification. | A rule with an exception in each feature is not a rule. |

## Consequences

- Positive: the rule matches `rates_list_official_history` and `energy_list_prices`.
- Negative: a tool of a period must refuse a long period, not cut it.
- Work that follows:
  - Change `MCP-008`.
  - Examine every specification: `NBGRATE` and `GNERCMON` take a period with a maximum.
    `GNERCMON` section 3 cites this ADR.
