---
id: TST
title: Testing
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [cmd/, internal/, pkg/, test/]
related: [GO, PRC, TRC, CLT]
---

# Testing

## TST-001

A test cites the scenario identifier that it verifies, in a comment.

```go
// NBGRATE-SC-003: An unknown currency code returns a tool error.
func TestGetOfficialRatesRejectsUnknownCode(t *testing.T) {
```

## TST-002

Write the test before the code. Run the test. Confirm that it fails for the correct reason.

## TST-003

Do not change a test or a scenario to make a test pass. A test that looks wrong
means the specification is wrong or the code is wrong. Correct the specification
first. See `PRC-001`.

## TST-004

A scenario declares its layer:

| Layer         | Runs                                                           | Network |
| ------------- | -------------------------------------------------------------- | ------- |
| `unit`        | One package. A fake replaces each port.                        | None    |
| `integration` | A tool call through the in-memory MCP transport, to `httptest`. | None    |
| `live`        | A vendor client against the real vendor. Build tag `live`.     | Yes     |
| `manual`      | The owner, with a real MCP client.                             | Yes     |

`go test ./...` runs `unit` and `integration` only.

## TST-005

The library is the standard `testing` package with `github.com/google/go-cmp/cmp`
for a comparison. A test uses no assertion library and no generated mock.
A fake is a struct that the test writes.

## TST-006

A test of a vendor client or of a driven adapter reads a recorded response from
`testdata/`. The file holds a real answer of the vendor, with every personal value
replaced.

## TST-007

A test with more than two cases is a table with `t.Run`. A test takes its context
from `t.Context()`. A test of a time to live uses `testing/synctest`.

## TST-008

Every test run uses `-race`.

## TST-009

A constructor that parses a text from outside the process has a fuzz test.
A value object of a tool input is one example. See `DOM-003`.

**Why:** A model sends any text. A fuzz test finds the input that panics.

## Retired identifiers

This file has no retired identifier.
