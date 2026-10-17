---
id: NBGRATE
title: The official exchange rates of the NBG
type: requirements
status: approved
created: 2026-10-18
updated: 2026-10-18
context: rates
approved_by: Temuri
approved_on: 2026-10-18
constrained_by: [DOM, MCP, RSL, CLT, ARC]
---

# Requirements: The official exchange rates of the NBG

## 1. Problem

A person in Georgia asks "what is the dollar today", "what was the euro on my contract
date", or "how did the pound move this month". The answer that a bank, a court, a tax
form, or an accountant accepts is the official rate of the National Bank of Georgia.

A model has no current rate and no history. It guesses, or it cites a commercial rate
that differs from the official one.

This feature is the first source of the `rates` context. Commercial rates of the banks
come in later features.

## 2. Users

| User                   | Need                                                                    |
| ---------------------- | ----------------------------------------------------------------------- |
| A person with a client | The official rate of one or more currencies, today or on a past date.   |
| A person with a client | The movement of one currency over a period of up to one month.          |

## 3. Source

| Vendor | Source                                                        | Official                   | Publication period                          | Terms of use               |
| ------ | ------------------------------------------------------------- | -------------------------- | ------------------------------------------- | -------------------------- |
| NBG    | `nbg.gov.ge/gw/api/ct/monetarypolicy/currencies/{en,ka}/json` | Yes. Linked from the page. Not documented. | Each business day, no later than 17:00, in force from the next day | Public. No key. Indicative rate. |

The NBG publishes no documentation of the API. Its page links the JSON and the RSS
address with the parameter `date` only. The parameter `currencies` has no official
description.

The "Rules Concerning the Determination of the Official Exchange Rate" of the NBG
(Decree N6 of 2009, last amended 2020) bind the meaning of the data:

- Article 2.1: the NBG calculates the rate against the dollar each business day.
- Article 2.9: the NBG sets the other rates as cross rates through the dollar.
- Article 3.1: the NBG publishes the rates no later than 17:00 of the calculation day.
  They are in force from the next day until the next set.
- Article 1.5: the official rate is indicative. Its use is not obligatory, except for
  the settlements of the NBG with the state budget and the cases that the law sets.

Observed behavior of the source, on 2026-10-18:

- One call returns the set of one date: 42 currencies, each with a quantity, a rate,
  the change, the name, the publication instant, and the valid date.
- The set reaches back to 1995.
- A date with no own set (a weekend) returns the set in force on it.
- A date after the newest set, and an invalid date, return the newest set with no error.
- An unknown code, or a code with no rate on the date, returns an empty list.
- A filter of more than one currency returns an empty list for most combinations.
- No call returns a period.

## 4. Out of scope

- A conversion of an amount between two currencies.
- A commercial rate of a bank. A later feature of the `rates` context adds it.
- A history longer than 31 days in one request.
- The rate of a central bank other than the NBG.

## 5. Definitions

| Term                 | Meaning                                                                              |
| -------------------- | ------------------------------------------------------------------------------------ |
| official rate        | The price in lari of a quantity of a foreign currency that the NBG sets for a date.  |
| rate set             | Every official rate that the NBG publishes for one valid date.                       |
| quantity             | The count of units of the currency that one official rate prices: 1, 10, 100, 1000, or 10000. |
| rate per unit        | The official rate divided by its quantity.                                           |
| valid date           | The date from which a rate set is in force. It stays in force until the next set.    |
| publication instant  | The instant at which the NBG publishes a rate set.                                   |
| change               | The difference between an official rate and the rate of the same currency in the previous set. |
| Tbilisi date         | The calendar date in the zone `Asia/Tbilisi`.                                        |

## 6. Functional requirements

### `NBGRATE-FR-001`

The server must return the official rate of each requested currency on a requested date.

**Examples:**

- Normal case: `USD` and `EUR` on 2026-10-17. The answer holds 2.5982 and 2.9274.
- Limit case: `usd` in lower case. The answer holds the rate of `USD`.
- Unwanted case: the code `US`. The server refuses it: a code has three letters.

### `NBGRATE-FR-002`

The server must return every official rate of the set when the request names no currency.

### `NBGRATE-FR-003`

The server must use the Tbilisi date of the request when the request names no date.

### `NBGRATE-FR-004`

The server must return the rate set in force on the requested date, and name its valid date.

**Why:** A person who asks for a Sunday wants the rate that applied on that Sunday.

**Examples:**

- Normal case: 2026-10-15. The valid date is 2026-10-15.
- Limit case: Sunday 2026-10-11. The valid date is 2026-10-10.

### `NBGRATE-FR-005`

The server must refuse a date after the valid date of the newest set. The message must
name the newest valid date.

**Why:** The source answers such a date with the newest set and no warning. A model
would cite it as the rate of the future date.

**Examples:**

- Normal case: on 2026-10-16 after 17:00, the date 2026-10-17. The server returns the set.
- Unwanted case: on 2026-10-18, the date 2026-10-25. The server refuses it and names 2026-10-17.

### `NBGRATE-FR-006`

The server must refuse a date that is not a valid calendar date.

### `NBGRATE-FR-007`

The server must name each requested currency that has no rate on the date, and return
the rates of the other currencies.

**Examples:**

- Normal case: `USD` and `XXX`. The answer holds the rate of `USD` and names `XXX` as missing.
- Limit case: `BYN` on 2010-06-01, before the NBG priced it. The answer names `BYN` as missing.
- Limit case: 1990-01-01, before the first set. Every requested code is missing.

### `NBGRATE-FR-008`

The server must give each official rate for its quantity, and per unit.

**Why:** "100 JPY = 1.6455" read as the price of one yen is wrong by a factor of 100.

### `NBGRATE-FR-009`

The server must give the change of each official rate.

### `NBGRATE-FR-010`

The server must give the name of each currency in the requested language, Georgian or
English. The default is English.

### `NBGRATE-FR-011`

The server must give the publication instant of each rate set.

### `NBGRATE-FR-012`

The server must return the official rates of one currency over a period of up to 31 days,
one rate for each valid date in the period.

**Examples:**

- Normal case: `USD` from 2026-10-01 to 2026-10-17. One rate for each set in the period.
- Limit case: a period of one day. One rate.
- Unwanted case: a period of 32 days. The server refuses it and names the limit.
- Unwanted case: an end before the start. The server refuses it.

### `NBGRATE-FR-013`

The server must return a rate set that it read before, with the instant that it read it,
when the NBG does not answer and the set is less than 7 days old in the server.

**Why:** A rate set of the past never changes. An outage of the NBG must not hide it.

### `NBGRATE-FR-014`

The server must report that the NBG did not answer when it holds no such rate set.

## 7. Non-functional requirements

### `NBGRATE-NFR-001`

A new rate set must appear in the answers less than 15 minutes after its publication instant.

### `NBGRATE-NFR-002`

The server must send at most one call to the NBG for one date and one language in each
15 minutes. For a valid date before the Tbilisi date of the request, the limit is one
call in each 30 days.

**Why:** A past set never changes. The NBG must not see the traffic of every client.

### `NBGRATE-NFR-003`

The server must answer a history of 31 days in less than 10 seconds when it holds none
of the sets. Measure from the receipt of the request to the last byte of the answer.

## 8. Invariants

### `NBGRATE-INV-001`

Every official rate in an answer carries the valid date of its set.

### `NBGRATE-INV-002`

The valid date of an answer is never after the requested date.

### `NBGRATE-INV-003`

An official rate in an answer has the exact digits that the NBG publishes. A rate per unit
is the exact quotient of the rate and the quantity.

## 9. Constraints from the guidelines

| Rule                 | Guideline  | Effect on this feature                                                  |
| -------------------- | ---------- | ----------------------------------------------------------------------- |
| `DOM-001`            | Domain     | The context is `rates`. The NBG is its first vendor.                    |
| `DOM-006`, `MCP-009` | Domain     | A rate is an exact decimal, a string in the answer.                     |
| `DOM-007`            | Domain     | A valid date is a civil date in `Asia/Tbilisi`.                         |
| `MCP-005`            | MCP tools  | Each answer names the source and the instant of its data.               |
| `MCP-007`            | MCP tools  | The language input takes `ka` and `en`.                                 |
| `MCP-008`            | MCP tools  | The history is bounded: 31 days.                                        |
| `RSL-004`, `RSL-006` | Resilience | The time to live and the stale age follow `NBGRATE-NFR-002` and `NBGRATE-FR-013`. |
| `CLT-001`, `CLT-009` | Clients    | The NBG client is a module in `pkg/nbg`. It needs no key.               |

## 10. Open questions

| #   | Question                                                                                                  | Owner  | Answer |
| --- | --------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | The source marks the publication instant with `Z`, as UTC, but gives about 17:01. How does the server read it? | Temuri | As Tbilisi time. Article 3.1 sets publication no later than 17:00 Tbilisi time. Read as UTC, 17:01 is 21:01 in Tbilisi, four hours after the limit. |

## Retired identifiers

This file has no retired identifier.
