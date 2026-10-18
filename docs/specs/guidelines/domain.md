---
id: DOM
title: The domain model
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/app/georgia-mcp/*/domain/, internal/app/georgia-mcp/*/app/, internal/pkg/kernel/]
related: [ARC, PKG, CLT, GLO]
---

# The domain model

Georgia MCP uses the strategic part of domain driven design in full: the bounded
context, the ubiquitous language, and the anti-corruption layer. It uses the
tactical part only where a context holds a rule. `ADR-0001` gives the reason.

## DOM-001

A bounded context covers one subject that a person in Georgia asks about.
Its name is a lowercase noun: `rates`, `transit`, `legislation`.
A context can read more than one vendor. A vendor belongs to one context.

**Why:** A person asks "what is the rate of the dollar", not "what does the NBG
API return". The context holds the question. The vendor is a detail.

## DOM-002

A type in `domain` carries the name of a term in `GLO`.
A new domain term enters the glossary when the owner approves its requirements.

## DOM-003

A value object is an immutable type with a constructor that validates it.
A constructor returns the value and an error. The zero value is never a valid value.

```go
code, err := domain.ParseCurrencyCode("usd")
```

**Why:** A value that passed the constructor is valid everywhere. No layer checks
it a second time.

## DOM-004

An entity or an aggregate exists only when the context enforces a rule across
more than one value. A read model with no rule is a plain struct of value objects.

**Why:** Most tools repeat what a source publishes. An aggregate with no rule is
code that no requirement asks for.

## DOM-005

A vendor shape never enters `domain` or `app`.
The driven adapter converts the vendor shape to a domain value. It is the
anti-corruption layer of the context.

**Why:** A vendor renames a field without notice. The change must stop at the adapter.

## DOM-006

A monetary amount, a price, or an exchange rate never uses a binary floating point
type. It uses `decimal.Decimal` of `github.com/shopspring/decimal`.

**Why:** 0.1 has no exact binary form. A rate of 2.5982 must stay 2.5982.

## DOM-007

A date that a Georgian source publishes is a civil date in the zone `Asia/Tbilisi`.
An instant carries a zone. A domain type never holds a time without a zone.

## DOM-008

`internal/pkg/kernel` is the shared kernel. It holds a type only when two contexts need it.
The kernel starts with two members: the language of a text (`ka` or `en`), and
the sentinel `ErrSourceUnavailable` of `DOM-011`.

**Why:** A large kernel couples every context. Each type in it is a cost that the
second user pays for.

## DOM-009

`app` declares each driven port as an interface, in `port.go`.
A port speaks in domain types. A port names a capability, not a vendor:
`RateSource`, not `NBGClient`.

## DOM-010

A use case is one exported type in `app` with one method. The method takes a
context and a query struct, and returns a domain value and an error.

## DOM-011

A domain error is a sentinel in `domain/errors.go`. A port error of one context is a
sentinel in `app/errors.go`. A driven adapter wraps every vendor failure with
`kernel.ErrSourceUnavailable`, and the cache of `RSL-006` reads that sentinel.

**Why:** A tool maps a sentinel to a message for the model. See `MCP-006`.
A vendor error type must not reach the tool.

## Retired identifiers

This file has no retired identifier.
