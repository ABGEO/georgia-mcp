---
id: GLO
title: Glossary
type: glossary
status: active
created: 2026-10-17
updated: 2026-10-24
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
| vendor client      | A Go module in `pkg/` that calls the API of one vendor, or reads its web page. See `CLT`.              |
| web page source    | A source that is a page for people, not an API. See `ADR-0003`.                                        |
| browser handshake  | The TLS handshake that a browser sends. A bot filter can refuse any other. See `CLT-002`.              |
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

## Rates

| Term                      | Meaning                                                                                        |
| ------------------------- | ---------------------------------------------------------------------------------------------- |
| bank                      | In `rates`, a company of Georgia that publishes commercial rates: a commercial bank, a microfinance organization, or an online exchange bureau. `RICORATE` 5. |
| bank kind                 | The kind of a bank in `rates`: `bank`, `microfinance organization`, or `online exchange bureau`. |
| commercial rate           | The two prices in lari at which a bank buys and sells one unit of a currency.                  |
| buy rate                  | The price in lari that the bank pays for one unit of a currency.                               |
| sell rate                 | The price in lari that the bank asks for one unit of a currency.                               |
| microfinance organization | A company of Georgia that the NBG registers to lend money. It is not a bank.                   |
| online exchange bureau    | A company of Georgia that the NBG registers to change money, by bank transfer only.            |
| Terabank                  | Terabank, a commercial bank. Bank code `terabank`.                                             |
| Rico Credit               | Rico Credit, a microfinance organization. Bank code `rico`.                                    |
| Kursi.ge                  | Kursi.ge, an online exchange bureau. Bank code `kursi`.                                        |
| Scapp                     | Scapp, a microfinance organization, formerly Swiss Capital. Bank code `scapp`.                 |
| Halyk Bank                | Halyk Bank Georgia, a commercial bank. Bank code `halyk`.                                      |
| Basisbank                 | Basisbank, a commercial bank. Bank code `basisbank`.                                           |
| Credo Bank                | Credo Bank, a commercial bank. Bank code `credo`.                                              |

## Fuel

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| fuel company       | A company that sells fuel at its own network of stations in Georgia, for example Wissol.              |
| company code       | The short lowercase name of a fuel company in a request: `wissol`.                                     |
| product            | One fuel that a fuel company sells under its own name: "Eko Premium".                                  |
| grade              | The class of a product that every fuel company shares. `WISFUEL` 5.1 lists the grades and their units. `SOCFUEL` 5 adds `lpg`. |
| standard price     | The price in lari of one unit of a product at a station with an attendant.                             |
| self-service price | The price in lari of one unit of a product at a self-service station of the company.                   |
| effective date     | The date from which a fuel company applies a price.                                                    |
| Rompetrol          | A fuel company of Georgia. Company code `rompetrol`.                                                   |
| SOCAR              | SOCAR Georgia Petroleum, a fuel company of Georgia. Company code `socar`.                              |
| `lpg`              | The grade of liquefied petroleum gas. Its unit is the litre.                                           |
| Connect            | A fuel company of Georgia. Company code `connect`.                                                     |
| Lukoil             | Lukoil Georgia, a fuel company of Georgia. Company code `lukoil`.                                      |
| Neogas             | A network of gas stations of Georgia. Company code `neogas`.                                           |
| Portal             | A fuel company of Georgia. Company code `portal`.                                                      |
| Gulf               | Gulf Georgia, a fuel company of Georgia. Company code `gulf`.                                          |

## Offers

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| bank               | A commercial bank of Georgia that publishes card offers. In `rates`, see the section Rates.            |
| bank code          | The short lowercase name of a bank in a request: `bog`.                                               |
| offer              | A benefit that a bank gives to its customers at a partner, for a period.                              |
| partner            | The company where a customer gets the benefit of an offer: a shop, a hotel, a spa.                    |
| benefit text       | The text of the bank that states the benefit of an offer: "20% Discount, unlimited".                  |
| benefit kind       | The class of a benefit that the bank states in a field, not in a text: `discount`, `cashback`, `points`, or `other`. `BOGOFFER` 5.3. |
| tier               | A group of customers of a bank that an offer is for: `standard`, `premium`, `private`, or `youth`. `BOGOFFER` 5.1. |
| card               | A card or a package of a bank that an offer names as a condition: "Visa Infinite", "PLUS".            |
| card code          | The short lowercase name of a card in a request: `visa_infinite`. Each bank has its own codes.        |
| offer category     | The class of the partner of an offer that every bank shares. `BOGOFFER` 5.2 lists the 14 categories.  |
| bank category      | The name that a bank gives to the category of an offer: "Health and Care".                            |
| offer period       | The dates from the start date to the end date of an offer, both included.                             |
| terms              | The full text of the conditions of an offer.                                                          |
| BOG                | Bank of Georgia, a bank. Bank code `bog`.                                                             |
| TBC                | TBC Bank, a bank. Bank code `tbc`.                                                                    |
| Concept            | The premium package of TBC.                                                                           |
| Liberty            | Liberty Bank, a bank. Bank code `liberty`.                                                            |
| Studio             | The premium package of Liberty.                                                                       |

## Transit

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| stop               | A place where a vehicle of the TTC takes passengers: a bus stop, a metro station, or a cable car station. |
| stop code          | The number on the sign of a bus stop: `1000`. A station of metro or cable car has none.               |
| route              | One line of the TTC with one number, for example the bus `385` or the metro line `1`.                 |
| route number       | The short name of a route that a vehicle shows: `385`.                                                |
| kind               | The type of vehicle of a route: `bus`, `minibus`, `metro`, or `cable_car`. A stop has the kind `bus`, `metro`, or `cable_car`. |
| direction          | One of the two ways of a route, with a headsign and an ordered list of stops.                         |
| headsign           | The destination that a vehicle shows: "Varketili".                                                    |
| arrival            | The expected arrival of a vehicle of one route at one stop.                                           |
| live arrival       | An arrival that the TTC computes from the position of the vehicle.                                    |
| timetable          | The planned times at which the vehicles of one direction leave each of its stops.                     |
| trip plan          | The itineraries between two points at one time.                                                       |
| itinerary          | One way to travel between two points: a sequence of legs.                                             |
| leg                | One part of an itinerary: a walk, or a ride on one route.                                             |
| place              | A named point in Tbilisi that a person can search: an address, a street, or a building.               |

## Postal

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| Georgian Post      | The national postal operator of Georgia. Vendor code `gpost`. |
| postal code        | The 4-digit code of Georgian Post for an address: `0186`. |
| address            | In `postal`, the text of Georgian Post that names a settlement and a street: `ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა`. |
| district           | In `postal`, the city or the municipality under which Georgian Post groups an address. |

## The website

| Term               | Meaning                                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------------------- |
| site               | The Georgia MCP website.                                                                              |
| page               | One address of the site in one language.                                                              |
| home page          | The page at the root of a language.                                                                   |
| tool page          | The page of one tool.                                                                                 |
| tool definition    | The name, the title, the description, and the input schema of a tool, as the server lists it.         |
| tool example       | A question in the words of a person, the input of one tool call, and its output, recorded from the server. |
| chat example       | A question in the words of a person and the answer of a model, written for the site from a recorded output. |
| tool summary       | One or two sentences, written for the site, that say what a tool answers.                             |
| install guide      | The steps that connect one client to the server.                                                      |
| legal page         | The privacy policy page or the terms of service page of the site.                                     |
| access log         | The record of one HTTP request that the server or the site writes.                                    |
