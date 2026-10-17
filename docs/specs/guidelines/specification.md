---
id: SPC
title: The specification artifacts
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [docs/specs/features/]
related: [PRC, TRC, LNG, MCP, CLT, RSL, CFG]
---

# The specification artifacts

## SPC-001

A feature directory holds these files and no other:

| File                        | Holds                            | Present when                                     |
| --------------------------- | -------------------------------- | ------------------------------------------------ |
| `requirements.md`           | Stage 1                          | Always                                           |
| `requirements-<subject>.md` | One subject of the requirements  | `requirements.md` passes the target of `LNG-013` |
| `spec.md`                   | Stage 2                          | Always                                           |
| `spec-<subject>.md`         | One subject of the specification | `spec.md` passes the target of `LNG-013`         |

## SPC-002

A feature belongs to one bounded context. The frontmatter of each document names
it in the `context` field. A feature that needs two contexts needs an ADR.
A feature that builds no context, only the shared parts of the server, names
`platform` in the field.

## SPC-003

A specification declares each contract surface that the feature adds.
Section 4.2 holds the domain model. Section 4.3 holds every other surface.
One table for each surface. These columns are the minimum:

| Surface              | Columns                                              | Governed by          |
| -------------------- | ---------------------------------------------------- | -------------------- |
| MCP tool             | Tool, Input, Output, Errors, Realizes                | `MCP-002` to `MCP-009` |
| Vendor endpoint      | Client, Method and path, Key, Official, Realizes     | `CLT-003`, `CLT-009` |
| Cache                | Port method, Time to live, Maximum stale, Realizes   | `RSL-004`, `RSL-006` |
| Configuration        | Variable, Type, Default, Secret, Realizes            | `CFG-003`, `CFG-005` |
| CLI command          | Command, Does, Realizes                              | `CLI-001`            |
| HTTP route           | Method, Path, Access, Realizes                       | `API-001`            |

Delete a table that the feature does not need. Keep no empty heading. See `LNG-013`.

## SPC-004

Every row of a declaration table names a requirement identifier in its `Realizes`
column. The coverage table in section 2 names the section that holds the table.

**Why:** A surface that no requirement demands is scope that nobody approved.

## SPC-005

The specification of a tool gives the input and the output as the Go structs of
`adapter/tool`, with every tag. The structs are the schema. See `MCP-003`.

## SPC-006

A diagram is a mermaid block inside the specification. Draw one when the design
holds an order that a table cannot carry: a sequence across two or more components,
a state machine, or a model with three or more related types.

## SPC-007

The owner approves a change to a declaration table as a change to `spec.md`.

## Retired identifiers

This file has no retired identifier.
