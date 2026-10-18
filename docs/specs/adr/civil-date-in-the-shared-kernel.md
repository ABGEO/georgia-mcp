---
id: ADR-0002
title: The civil date in the shared kernel
type: adr
status: accepted
created: 2026-10-19
updated: 2026-10-19
decided: 2026-10-19
changes: [DOM-008]
supersedes:
superseded_by:
---

# The civil date in the shared kernel

## Context

`rates/domain` declares `Date`, a civil date in Tbilisi, and `TbilisiZone`. `NBGRATE`
uses them for the date of an official rate.

`WISFUEL` adds the context `fuel`. A fuel company gives each price the date from which
it applies the price (`WISFUEL-FR-005`). `DOM-007` makes that date a civil date in
`Asia/Tbilisi`. `ARC-008` forbids `fuel` to import `rates`.

`DOM-008` puts a type in the shared kernel when two contexts need it. The kernel holds
two members: the language and `ErrSourceUnavailable`.

## Decision

`Date`, `ParseDate`, `Today`, `TbilisiZone`, and `ErrInvalidDate` move from
`rates/domain` to `internal/pkg/kernel`. `rates` and `fuel` import them from there.

## Rationale

Two contexts need the same civil date, with the same zone and the same validation.
`DOM-008` names this case. The type has 7 methods and no rule of a context. The move
changes the imports of 16 files of `rates` and no behavior.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A second `Date` in `fuel/domain`. | Two copies of one type and its zone. A fix to one copy misses the other. |
| `time.Time` at midnight in Tbilisi for the date of a price. | `DOM-007` and `NBGRATE-DD-008` keep a civil date apart from an instant. A zone conversion can move the day. |
| An alias `type Date = kernel.Date` in `rates/domain`. | Two names for one thing. `LNG-002`. |

## Consequences

- Positive: `fuel` and any later context use one civil date.
- Negative: the kernel grows from two members to three. Each change to `Date` now
  touches two contexts.
- Work that follows:
  - Change `DOM-008`: the kernel holds the language, the civil date in Tbilisi, and the
    sentinel `ErrSourceUnavailable`.
  - Move the type and its tests in the fast lane, before `WISFUEL` step 2. The tests of
    `NBGRATE` must pass with no change other than the import.
  - Examine `NBGRATE` 4.1 and 4.2. They name `domain.Date`.
