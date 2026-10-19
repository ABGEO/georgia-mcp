---
id: GLO
title: Glossary
type: glossary
status: active
created: 2026-10-17
updated: 2026-10-19
---

# Glossary

One term has one meaning. A document uses the term, not a synonym. `LNG-002`.
A domain term enters this file when the owner approves the requirements that
define it. The identifier of a term is `GLO-<term>`.

## The system

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| server             | The Georgia MCP process. It serves the tools.                                                          |
| client             | A program that connects to the server over MCP, for example a desktop chat application. Not a vendor client. |
| model              | The language model that reads the tools through a client and calls them.                               |
| tool               | One MCP tool that the server serves. It has a name, an input, and an output.                           |
| transport          | The channel that carries MCP: streamable HTTP or stdio.                                                |
| vendor             | An organization that publishes data that a tool returns, for example the NBG or the TTC.               |
| source             | The system of a vendor that the server reads.                                                          |
| vendor client      | A Go module in `pkg/` that calls the API of one vendor. See `CLT`.                                     |
| unofficial API     | An API that a vendor uses in its own application and does not document for others.                     |

## The architecture

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| context            | A bounded context. One subject of Georgian public life, with its own model and language. See `DOM-001`. |
| domain             | The layer of a context that holds its value objects, read models, and domain errors.                   |
| value object       | An immutable domain value that a constructor validates. See `DOM-003`.                                |
| read model         | A plain struct of value objects that a tool returns. It holds no rule.                                 |
| use case           | One operation of a context in `app`. A tool calls it. See `DOM-010`.                                  |
| port               | An interface in `app` that a use case needs from outside the context. See `DOM-009`.                  |
| adapter            | Code that connects a port or a use case to the outside: a tool, or a vendor client.                    |
| driving adapter    | An adapter that calls a use case. In this server, the tools of a context.                              |
| driven adapter     | An adapter that implements a port. In this server, the code that calls a vendor client.                |
| anti-corruption layer | The conversion of a vendor shape to a domain value, inside a driven adapter. See `DOM-005`.        |
| shared kernel      | `internal/pkg/kernel`. The types that two or more contexts share. See `DOM-008`.                           |
| composition root   | `internal/app/georgia-mcp/bootstrap`. The one package that builds every context. See `ARC-009`.                        |
| stale value        | A cached value past its time to live that the server still returns while the source fails. `RSL-006`. |

## Georgia

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| NBG                | The National Bank of Georgia. It publishes the official exchange rate of the lari.                     |
| TTC                | The Tbilisi Transport Company. It runs the buses, the metro, and the cable cars of Tbilisi.           |
| lari               | The currency of Georgia. Code `GEL`.                                                                   |
| language           | The language of a text that a source publishes: `ka` (Georgian) or `en` (English). See `MCP-007`.      |

## Fuel

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| fuel company       | A company that sells fuel at its own network of stations in Georgia, for example Wissol.              |
| company code       | The short lowercase name of a fuel company in a request: `wissol`.                                     |
| product            | One fuel that a fuel company sells under its own name: "Eko Premium".                                  |
| grade              | The class of a product that every fuel company shares. `WISFUEL` 5.1 lists the grades and their units. |
| standard price     | The price in lari of one unit of a product at a station with an attendant.                             |
| self-service price | The price in lari of one unit of a product at a self-service station of the company.                   |
| effective date     | The date from which a fuel company applies a price.                                                    |
