---
id: GO
title: Go code style
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [cmd/, internal/, pkg/, test/]
related: [TST, BLD, LNG, PKG, DOM]
---

# Go code style

## GO-001

`golangci-lint` v2 checks every Go module. The configuration is `.golangci.yaml`
at the repository root. Run it in the module that the change touches. `BLD-001`.

## GO-002

The import order is: the standard library, then the external package, then the
package with the prefix `github.com/abgeo/georgia-mcp`. `gci` enforces the order.

## GO-003

Every exported identifier has a documentation comment.
Every package has a package comment.

## GO-004

An error gets a context with `fmt.Errorf` and the verb `%w`.
The context text is lowercase and describes the action:

```go
return fmt.Errorf("reading the official rates: %w", err)
```

## GO-005

A sentinel error is a package-level variable named `ErrXxx`. The message is
lowercase with the form `"<subject>: <problem>"`. `DOM-011` gives its place.

```go
ErrUnknownCurrency = errors.New("currency: unknown code")
```

## GO-006

A comment obeys `LNG`. Write one only when the code cannot explain itself (`LNG-011`).

## GO-007

Assert at compile time that a type implements the interface it claims.

```go
var _ app.RateSource = (*Adapter)(nil)
```

## GO-008

A constructor is `New` when the package builds one type, and `NewXxx` otherwise.
It returns the concrete type, not an interface. The `ireturn` linter enforces this.

## GO-009

The consumer declares the interface that it needs. `adapter/tool` declares the
interface of the use case that it calls. `app` declares the port. See `DOM-009`.

**Why:** A small interface at the consumer gives a test its seam with no mock tool.

## GO-010

Two external documents are the standard of Go code in this repository:

| Standard            | Source                                                |
| ------------------- | ----------------------------------------------------- |
| Effective Go        | https://go.dev/doc/effective_go                       |
| Uber Go Style Guide | https://github.com/uber-go/guide/blob/master/style.md |

A change that contradicts one of them is a defect. The order when two disagree:
a `GO` rule, then the Uber guide, then Effective Go.

## GO-011

Use the packages `slices`, `maps`, `cmp`, and `iter` of the standard library before
a loop of your own or a helper library.

## GO-012

A generic type or function needs two or more instantiations in the code.

**Why:** A type parameter with one instantiation is an abstraction that nobody needs.

## GO-013

Handle an error one time. A function returns the error or logs it, never both.
The function that ends the error path logs it.

**Why:** An error that each layer logs appears in the log one time for each layer.

## Retired identifiers

This file has no retired identifier.
