---
id: TBCRATE
title: The commercial exchange rates of TBC Bank
type: requirements
status: approved
created: 2026-10-18
updated: 2026-10-22
context: rates
approved_by: Temuri
approved_on: 2026-10-18
constrained_by: [DOM, MCP, RSL, CLT, CFG, LOG]
---

# Requirements: The commercial exchange rates of TBC Bank

## 1. Problem

A person who changes money at a bank pays the commercial rate of that bank, not the
official rate of the NBG. They ask "how much does TBC sell the dollar for today" or "how
many lari do I get for 100 euro at TBC". A model has no current commercial rate.

The commercial rate has two prices: the bank buys a currency at one price and sells it
at another. A model that gives one price, or swaps the two, gives a wrong answer.

This feature adds the first bank. Other banks join the same answer in later features.

## 2. Users

| User                   | Need                                                                       |
| ---------------------- | -------------------------------------------------------------------------- |
| A person with a client | The buy rate and the sell rate of a bank for one or more currencies, today. |
| The operator           | Runs the server with or without a key of TBC.                              |

## 3. Source

| Vendor   | Source                                              | Official                    | Publication period          | Terms of use                     |
| -------- | --------------------------------------------------- | --------------------------- | --------------------------- | -------------------------------- |
| TBC Bank | `api.tbcbank.ge/v1/exchange-rates/commercial`       | Yes. A documented developer API. | Not documented. The bank changes its rates during the day. | A developer account and a key. 60 calls in a minute for each key. No quota. |

Observed behavior of the source, from its documentation and its Postman collection on
2026-10-18:

- One call returns the current commercial rates: each currency with a buy rate and a
  sell rate, against the lari. The answer holds no instant.
- An optional filter takes a list of codes, separated by commas. An unknown code is
  left out with no error.
- An answer holds at most 10 currencies. With no filter, or a filter of more than 10
  codes, the answer holds 10 of them, and which 10 changes over time. On 2026-10-18 the
  bank quoted 21 currencies.
- Every rate is the price of one unit, the yen included. The JSON numbers drop a
  trailing zero: `1.35`.
- The API offers no date. It has no history of commercial rates.
- A missing or wrong key gives 401 in the problem format.
- The API also converts an amount, and repeats the official rates of the NBG.

## 4. Out of scope

- A conversion of an amount.
- The official rates through TBC. `NBGRATE` reads them from the NBG.
- A history of commercial rates. The source has none.
- A bank other than TBC. A later feature adds each one.
- Every rate of a bank with no currency named. The source has no reliable answer to it.
- The rates of a card, a branch, or an online channel, when they differ.

## 5. Definitions

| Term            | Meaning                                                                                  |
| --------------- | ---------------------------------------------------------------------------------------- |
| bank            | A commercial bank of Georgia that publishes commercial rates.                            |
| commercial rate | The two prices in lari at which a bank buys and sells one unit of a currency.            |
| buy rate        | The price in lari that the bank pays for one unit of a currency. A person who sells the currency gets it. |
| sell rate       | The price in lari that the bank asks for one unit of a currency. A person who buys the currency pays it. |
| bank code       | The short lowercase name of a bank in a request: `tbc`.                                  |

## 6. Functional requirements

### `TBCRATE-FR-001`

The server must return the buy rate and the sell rate of a bank for each requested currency.

**Examples:**

- Normal case: `USD` at `tbc`. The answer holds a buy rate and a sell rate.
- Limit case: `usd` in lower case. The answer holds the rates of `USD`.
- Unwanted case: the code `US`. The server refuses it: a code has three letters.

### `TBCRATE-FR-002`

The server must answer for every bank that it can read when the request names no bank.

**Why:** A person who asks "where is the dollar cheapest" names no bank.

### `TBCRATE-FR-003`

The server must refuse a bank code that it does not know. The message must name the
codes that it accepts.

### `TBCRATE-FR-004`

The server must name each requested currency that a bank does not quote, and return the
rates of the other currencies.

### `TBCRATE-FR-005`

The server must give each commercial rate as the price in lari of one unit of the currency.

**Why:** The official rate of the NBG prices some currencies per 100 units. One unit
everywhere keeps a comparison between banks correct.

### `TBCRATE-FR-006`

The server must state in each answer that the buy rate is what the bank pays, and the
sell rate is what the bank asks.

**Why:** "Buy" read from the side of the person is the opposite price.

### `TBCRATE-FR-007`

The server must give, for each bank, the oldest instant at which it read a rate of the answer.

**Why:** The source publishes no instant. A model must still tell how fresh a rate is.
The server reads each currency on its own, so the rates of one bank can have different
read instants. The oldest one bounds them all.

### `TBCRATE-FR-008`

The server must report a bank that does not answer, and return the rates of the other
banks.

**Examples:**

- Normal case: one bank fails, a second answers. The answer holds the second, and names the first.
- Limit case: the only requested bank fails, and the server holds no rates of it. The
  server reports that the bank did not answer.

### `TBCRATE-FR-009`

The server must return rates that it read less than 1 hour before, marked as stale, when
a bank does not answer.

**Why:** A bank changes its rates during the day. A rate older than 1 hour misleads more
than it helps.

### `TBCRATE-FR-011`

The server must refuse a request that names no currency, or more than 10 currencies. The
message must give the limit.

**Why:** TBC answers at most 10 currencies in one call, and has no reliable list of all.

**Examples:**

- Normal case: `USD` and `EUR`. The server answers.
- Limit case: 10 codes, or `USD` twice and nothing else. The server answers: a code named
  twice counts once.
- Unwanted case: no code, or 11 codes. The server refuses the request.

### `TBCRATE-FR-012`

The operator must be able to turn TBC off. It is on by default. When TBC is off, the
server must not offer TBC in an answer.

**Examples:**

- Normal case: TBC is on with a key. The server offers TBC.
- Limit case: TBC is off, and TBC is the only bank. The server offers no commercial rate.
- Unwanted case: a request names `tbc` while TBC is off. The server refuses it as a bank
  that it cannot read.

### `TBCRATE-FR-013`

If TBC is on and its key is empty, the server must not start. The error must name the
setting of the key and the setting of the switch.

**Why:** `CFG-008`. A missing key is a mistake of the deployment, not a choice. The TTC
behaves the same way (`TTCTRANS-FR-022`).

## 7. Non-functional requirements

### `TBCRATE-NFR-001`

A commercial rate in an answer must be less than 5 minutes old, unless the answer marks
it as stale. Measure from the read instant to the receipt of the request.

### `TBCRATE-NFR-002`

The server must send at most one call to a bank for the same currency in each 5 minutes,
and at most 60 calls to TBC in a minute.

**Why:** TBC allows 60 calls in a minute for each key.

## 8. Invariants

### `TBCRATE-INV-001`

The key of TBC never appears in a log record, an answer, or an error message.

### `TBCRATE-INV-002`

Every commercial rate in an answer carries its bank. The server read the rate at or
after the read instant that its bank gives.

### `TBCRATE-INV-003`

A commercial rate in an answer has the exact digits that the bank publishes, for one unit.

## 9. Constraints from the guidelines

| Rule                 | Guideline     | Effect on this feature                                              |
| -------------------- | ------------- | ------------------------------------------------------------------- |
| `DOM-001`            | Domain        | The context is `rates`. TBC is its second vendor.                   |
| `DOM-006`, `MCP-009` | Domain        | A rate is an exact decimal, a string in the answer.                 |
| `MCP-002`            | MCP tools     | The tool is `rates_get_commercial_rates`.                           |
| `MCP-005`            | MCP tools     | Each answer names the source and the instant of its data.           |
| `MCP-006`            | MCP tools     | A bank that fails gives a message for the model, not a vendor body. |
| `CLT-001`, `CLT-009` | Clients       | The client is a module in `pkg/tbc`. The caller passes the key.     |
| `CFG-007`            | Configuration | The key is a secret. It stays out of the repository.                |
| `LOG-008`            | Logging       | The key is never logged.                                            |
| `RSL-004`, `RSL-006` | Resilience    | The time to live is 5 minutes, the stale age 1 hour, for each currency. |

## 10. Open questions

| #   | Question                                                                                         | Owner  | Answer |
| --- | ------------------------------------------------------------------------------------------------ | ------ | ------ |
| 1   | Does TBC quote a currency per 100 units, as the NBG does for the yen? The documentation shows no quantity. A live call with the key answers it. | Temuri | No. Every rate is per unit. JPY 0.0157 against the NBG 1.6455 for 100, AMD 0.0065 against 7.1823 for 1000 (2026-10-18). |
| 2   | Does the TBC plan of the owner have a quota of calls? `TBCRATE-NFR-002` keeps the calls low in either case. | Temuri | No quota. A limit of 60 calls in a minute for each key (2026-10-18). |

## Retired identifiers

| ID             | Retired    | Reason                                                              |
| -------------- | ---------- | ------------------------------------------------------------------- |
| TBCRATE-FR-010 | 2026-10-22 | An empty key no longer turns TBC off. `TBCRATE-FR-012` and `TBCRATE-FR-013` replace it (`CFG-008`). |
