---
id: ADR-0004
title: A vendor in two contexts
type: adr
status: accepted
created: 2026-10-20
updated: 2026-10-20
decided: 2026-10-20
changes: [DOM-001, CLT-001]
supersedes:
superseded_by:
---

# A vendor in two contexts

## Context

`BOGOFFER` and `TBCOFFER` add the card offers of Bank of Georgia and TBC Bank to the
context `offers`. Both banks are vendors of `rates` already, through `pkg/bog` and
`pkg/tbc`.

- `DOM-001` says: "A vendor belongs to one context."
- `CLT-001` says: "The package name is the vendor name."

The offers come from other systems of the same banks:

| Vendor | `rates` reads                 | `offers` reads                      |
| ------ | ----------------------------- | ----------------------------------- |
| BOG    | `api.businessonline.ge`       | `bankofgeorgia.ge/api/bog-b/offers-hub` |
| TBC    | `api.tbcbank.ge`, with a key  | `apigw.tbcbank.ge/api/v1/marketing`, no key |

The two systems of each bank share no address, no key, no shape, and no owner team.

## Decision

A vendor can feed two contexts. Each context reads it through its own vendor client.
A vendor client belongs to one context. Its name is the vendor name, or the vendor name
and the subject when the vendor has a second client: `bogoffers`.

## Rationale

A person asks about rates and about offers as two subjects. A bank publishes them from
two systems. One client for both systems joins two base addresses, two sets of options,
and two release cycles in one module. `CLT-005` gives a client one base address.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| Add the offers to `pkg/bog` and `pkg/tbc`. | Two base addresses in one client. `CLT-005` gives one. A change of the offers releases the client of the rates. |
| Put the offers in the context `rates`. | `DOM-001`: a context covers one subject. A rate is not an offer. |

## Consequences

- Positive: each context stays independent. The clients of the rates do not change.
- Negative: a bank has two modules in `pkg/`.
- Work that follows:
  - `DOM-001`: "A vendor belongs to one context" becomes "A vendor client belongs to one context. A vendor can feed two contexts through two clients."
  - `CLT-001`: the package name is the vendor name, and the subject when the vendor has a second client.
  - The glossary entry of "vendor client" stays as it is.
