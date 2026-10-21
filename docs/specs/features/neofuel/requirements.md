---
id: NEOFUEL
title: The price of compressed natural gas of Neogas
type: requirements
status: approved
created: 2026-10-21
updated: 2026-10-21
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [DOM, MCP, RSL, CLT, CFG]
---

# Requirements: The price of compressed natural gas of Neogas

## 1. Problem

A driver of a car on gas asks "where is CNG cheapest". The server answers for Wissol
and SOCAR. Neogas is a network of gas stations across Georgia. It sells compressed
natural gas only, at all but one of its stations.

## 2. Users

| User                   | Need                                                    |
| ---------------------- | ------------------------------------------------------- |
| A person with a client | The current price of CNG at Neogas, next to the others. |
| The operator           | Turns Neogas off with no change to the code.            |

## 3. Source

| Vendor | Source | Official | Publication period | Terms of use |
| ------ | ------ | -------- | ------------------ | ------------ |
| Neogas | The price calculator of the home page `neogas.ge/ka/` | No. A web page for people. | Not published. The page shows no date. | No key. No published terms and no published limit. `robots.txt` disallows `/admin_resources` and `/js`. |

Neogas publishes no API and no price list. The home page renders one price on the
server, in its calculator of the savings of a car on gas.

Observed behavior of the source on 2026-10-21:

- The calculator has two panes: a jeep and a car. Each pane shows "გაზის ფასი" (Gas
  Price) with the same value: `1.60`. 44 KB.
- The value is decimal text in lari, with two places. The page names no unit. The
  stations of Georgia sell CNG by the cubic metre.
- The page shows no date, no product code, and no other product. A news item of
  2024-11-12 says that the station at the airport added petrol and diesel. The site
  shows no price for them.
- The page `neogas.ge/en/` shows the same value.
- The page holds the text of PHP notices: `Notice: Undefined property`.

## 4. Out of scope

- The petrol and the diesel of the station at the airport. The site gives no price.
- A price of the card NeoCard, or of a campaign.

## 5. Definitions

| Term   | Meaning                                                      |
| ------ | ------------------------------------------------------------ |
| Neogas | A network of gas stations of Georgia. Company code `neogas`. |

## 6. Functional requirements

### `NEOFUEL-FR-001`

The server must return the standard price of the CNG of Neogas, with the grade `cng`.

**Examples:**

- Normal case: `neogas` on 2026-10-21. CNG has the standard price `1.60` for one cubic metre.
- Limit case: the grade `diesel` at `neogas`. The answer names `diesel` as a grade that
  Neogas does not sell.
- Unwanted case: a price that is not a number, or not above zero. The server treats the
  answer of Neogas as a failure.

### `NEOFUEL-FR-002`

The server must give the CNG of Neogas a name in each language: "ბუნებრივი აირი" in
`ka`, "CNG" in `en`.

**Why:** The page names the price, not the product: "გაზის ფასი".

### `NEOFUEL-FR-003`

The server must treat a page of Neogas with no gas price, or with two gas prices that
differ, as a failure of Neogas.

**Why:** Neogas can change its page with no notice. Two different values give no
single price.

**Examples:**

- Normal case: two panes with `1.60`. The server returns `1.60`.
- Limit case: one pane with `1.60`. The server returns `1.60`.
- Unwanted case: no pane, or `1.60` and `1.65`. The server reports that Neogas did not
  answer, and returns the prices of the other companies.

### `NEOFUEL-FR-004`

The server must treat Neogas as a fuel company that publishes no effective date, and no
self-service price.

**Why:** The page shows neither. `ROMFUEL-FR-004` and `ROMFUEL-FR-005` then apply.

### `NEOFUEL-FR-005`

The server must read Neogas with no key. The operator must be able to turn Neogas off.
It is on by default.

**Examples:**

- Normal case: no configuration. The server offers `neogas`.
- Unwanted case: a request names `neogas` while it is off. The server refuses it as a
  company that it cannot read.

## 7. Non-functional requirements

`WISFUEL-NFR-001` and `WISFUEL-NFR-002` apply to Neogas. This feature adds none.

## 8. Invariants

`WISFUEL-INV-001` and `WISFUEL-INV-003` apply to Neogas. This feature adds none.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                         |
| -------------------- | ------------- | -------------------------------------------------------------- |
| `DOM-005`            | Domain        | The grade, the name, and the check of the two panes belong to the server. |
| `DOM-006`, `MCP-009` | Domain        | A price is an exact decimal, a string in the answer.           |
| `MCP-005`            | MCP tools     | Each answer names Neogas and the instant at which the server read it. |
| `CLT-002`, `CLT-004` | Clients       | The client reads a web page, as `ADR-0003` allows.             |
| `CLT-009`, `CLT-011` | Clients       | The client says that the source is unofficial. It stays at `v0`. |
| `CFG-003`, `CFG-004` | Configuration | The switch is a field of the configuration of `fuel`.          |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 30 minutes, the stale age 12 hours.        |

## 10. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Is the price of Neogas for one cubic metre? The page names no unit. | Temuri | Yes. The grade `cng` gives the cubic metre (2026-10-21). |

## Retired identifiers

This file has no retired identifier.
