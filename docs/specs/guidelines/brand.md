---
id: BRD
title: Brand
type: guideline
status: active
created: 2026-10-20
updated: 2026-10-20
scope: [website/, README.md, the social preview image]
related: [LNG, GLO]
---

# Brand

## BRD-001

This guideline governs everything a person sees outside an MCP client: the website,
the README, the favicon, and the social preview image. `MCP` and `LNG` govern the
text that a model reads.

## BRD-002

The brand has four traits. Every decision in this file serves one of them.

| Trait   | It means                                                     | It does not mean                       |
| ------- | ------------------------------------------------------------ | -------------------------------------- |
| Civic   | A public notice. Each number names its source and its date.  | A government look: seals, flags, crests. |
| Precise | A strict grid. Tabular numerals. Real output of the tools.   | Cold or dense.                         |
| Warm    | Pastel colors on paper. Human copy. A visible author.        | Rustic nostalgia, wine and grapes.     |
| Playful | A stamp, a ticket edge, one joke on a page.                  | A mascot in each section. Bounce.      |

## BRD-003

Do not use these:

- A dark-first page, a terminal costume with window buttons, or a green of a terminal.
- A gradient, a glow, a blur behind glass, or a bento grid of feature cards.
- A stock photo or a generated illustration.
- The national flag, the Bolnisi cross, or another state or church symbol.
  Georgia MCP is not an official body.

**Why:** These mark the generic developer tool. The brand must not look like one.

## BRD-004

The name is **Georgia MCP**, in Latin script, in both languages.
Do not translate it. Do not write it as one word.
`georgia-mcp` is the identifier of the server. Use it in code and in commands only.

## BRD-005

The mark is the Asomtavruli letter mani, `Ⴋ` (U+10AB). It has four parts: a top bar,
a stem, a round bowl, and an arm across the stem.

| Property    | Value                                                     |
| ----------- | --------------------------------------------------------- |
| Grid        | 24 by 24 units. The letter fills 17 by 18.                |
| Stroke      | 3.5 units, square ends. Draw it as a path. Do not type it. |
| Clear space | 4 units on each side.                                     |
| Minimum     | 16 px.                                                    |
| Versions    | Ink on paper. Wine text on wine fill (the favicon). One color. |

Do not rotate, outline, stretch, or shade the mark.

**Why:** Mani is the M of MCP. Its bowl gives the warm trait, and its straight strokes
sit on the grid. The arm reads as a connector: an MCP server connects a model to a source.
Gani, `Ⴂ`, has right angles only, and it reads as an abstract bracket.

## BRD-006

The lockup is the mark, a gap of 8 units, and the name in Fraunces 600.
The name and the mark share one cap height. The lockup is the same in both languages.

## BRD-007

The palette is pastel. Each color is a CSS token. A component uses the token, never the value.

| Token     | Light     | Dark      | Use                                  |
| --------- | --------- | --------- | ------------------------------------ |
| `paper`   | `#FBF7F0` | `#1A1614` | The page.                            |
| `surface` | `#FFFDF9` | `#241F1C` | A card, a code block.                |
| `ink`     | `#2A2320` | `#F3ECE2` | The text.                            |
| `muted`   | `#6B6058` | `#ABA096` | Secondary text, a Georgian echo.     |
| `line`    | `#E8DFD2` | `#3B332D` | A border, a divider, the band.       |

Each hue has a pastel fill and a deep tone for text. Wine is the brand hue.

| Hue      | Light fill | Light text | Dark fill | Dark text | Use                         |
| -------- | ---------- | ---------- | --------- | --------- | --------------------------- |
| `wine`   | `#F4CFD3`  | `#7A1F2B`  | `#4A2229` | `#F2A9B2` | The mark, a link, a button. |
| `butter` | `#F6E3A8`  | `#7A5410`  | `#4A3B1A` | `#F2D58A` | The rates subject.          |
| `peach`  | `#F9CDB8`  | `#9A3D22`  | `#4E2C22` | `#F5B39A` | The fuel subject.           |
| `sky`    | `#C9DEEE`  | `#2C5B7A`  | `#1F3546` | `#A9CBE6` | The transit subject.        |
| `lilac`  | `#E2D2EC`  | `#6E3B6B`  | `#3A2A42` | `#D5BDE6` | The offers subject.         |

- A fill is a background. Never write text in a fill color.
- On a fill, write in `ink` or in the text tone of the same hue.
- A large area (a hero, a section band) uses a fill. A small accent (a link, an icon) uses a text tone.
- Each text pair passes WCAG AA (4.5:1) on `paper` and on its own fill, in both themes.

The page follows the theme of the system. A toggle overrides it.

## BRD-008

Each subject owns one hue from `BRD-007` and one Lucide icon.
A subject hue appears on the elements of that subject only.
A section uses `wine` and one subject hue at most.

## BRD-009

| Role             | Font               | Rule                                              |
| ---------------- | ------------------ | ------------------------------------------------- |
| Display          | Fraunces           | Headings in Latin. Weight 600. `SOFT` axis 50.    |
| Body             | Geist              | Text in Latin. 17 px, line height 1.6.            |
| Data and code    | JetBrains Mono     | A tool name, an input, an output, a command.      |
| Georgian         | Noto Sans Georgian | All Georgian text. Weight 700 for a heading.      |

- The type scale has a ratio of 1.25.
- Show every rate, price, and time with tabular numerals (`font-variant-numeric: tabular-nums`).
- Show a rate or a price as the tool returns it. Do not round it.
- Use Mtavruli (Georgian capitals) for a label of three words or fewer. Never for a sentence.

**Why:** Fraunces has no Georgian glyphs. One Georgian font keeps the Georgian page even.

## BRD-010

The Georgian motif is quiet. A screen holds one motif at most. These are the motifs:

| Motif | Form                                                                        |
| ----- | --------------------------------------------------------------------------- |
| Mark  | `Ⴋ`, per `BRD-005`.                                                         |
| Echo  | On the English site, a heading carries its Georgian word below it, in `muted`. |
| Band  | A divider of the top bar of mani with a repeated bowl below it, 8 px high, in `line`. |

## BRD-011

The playful devices:

| Device | Form                                                                                   | Limit               |
| ------ | -------------------------------------------------------------------------------------- | ------------------- |
| Stamp  | A short status ("Read-only", "Live", "No key") in mono capitals, a 2 px border, turned 2 to 4 degrees. | One on a screen.    |
| Ticket | The card of a tool example, with a perforated edge, like a transit ticket.             | Tool examples only. |
| Byline | "Made with :3 in Georgia by ABGEO" in the footer. "ABGEO" links to `https://www.abgeo.dev`. | The footer only.    |

## BRD-012

- Use a 12-column grid. The content is 1200 px wide at most.
- Use a spacing unit of 4 px.
- Use a radius of 4 px. A stamp has a radius of 2 px.
- Separate with a 1 px `line` border. Do not use a shadow.
- At a width of 360 px, the page has no horizontal scroll.

## BRD-013

- An animation lasts 200 ms at most and changes `opacity` or `transform` only.
- A stamp presses once when it enters the view.
- Under `prefers-reduced-motion: reduce`, nothing moves.
- Do not use parallax. Do not take control of the scroll.

## BRD-014

The copy of the website obeys `LNG-006` and `LNG-007`. These rules replace the
other `LNG` rules for the copy:

- Address the reader. Use "you" in English and "შენ" in Georgian.
- A contraction is correct in English.
- Each number names its source and its date.
- A section holds one playful line at most. An install step and an error hold none.
- A vendor keeps its official name in each language.
- A tool name stays as the server registers it, in both languages.

## BRD-015

Every page exists in English and in Georgian, with the same sections in the same order.
English is the default language. A page shows the output of a tool as the server
returned it. Do not draw a mockup of an answer.

## Retired identifiers

This file has no retired identifier.
