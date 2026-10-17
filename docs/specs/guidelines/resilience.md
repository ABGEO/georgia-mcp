---
id: RSL
title: Resilience of a vendor call
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/pkg/cache/, internal/app/georgia-mcp/*/module.go, internal/app/georgia-mcp/*/adapter/]
related: [ARC, CLT, MCP, CFG]
---

# Resilience of a vendor call

Every tool calls a vendor that the owner does not control. This guideline gives
the behavior of the server when a vendor is slow, fails, or limits the caller.

## RSL-001

Every call to a vendor has a timeout. The configuration of the context gives it.
The default is 10 seconds. The context of the tool call also limits it.

## RSL-002

A driven adapter does not retry a call.
A retry needs a requirement that names the vendor and the count.

**Why:** A model waits for the answer. Two retries of a 10 second timeout are 30 seconds.

## RSL-003

The cache sits in front of each driven port, as a decorator of the port.
`internal/pkg/cache` gives the generic type `cache.Cache[K, V]`.
`adapter/cache` of the context implements the port on it, one method at a time.
`module.go` wraps the driven adapter in the decorator. The driven adapter and the
use case do not know the cache. The cache stores the domain value, not the vendor shape.

```go
var source app.RateSource = ratecache.New(nbg.NewAdapter(client), cfg.Cache)
```

**Why:** Go cannot implement an interface with a generic type. The generic part is
the store. The decorator of each port is a few lines of code.

## RSL-004

The specification gives the time to live of each port method. It derives the
value from the publication period of the source.

| Data                        | Publication period | Typical time to live |
| --------------------------- | ------------------ | -------------------- |
| An official exchange rate   | One time each day  | 1 hour               |
| A list of stops             | Rarely             | 24 hours             |
| An arrival time             | Each few seconds   | None                 |

## RSL-005

The cache key is the port method and its normalized arguments.
Two calls with the same key while the value is missing cause one vendor call.

**Why:** Ten clients that ask at 17:01 must not send ten calls to the source.

## RSL-006

The cache keeps an expired value for the maximum stale period that the
specification gives. When the driven adapter returns `kernel.ErrSourceUnavailable`
inside that period, the decorator returns the stale value. Any other error passes through. `as_of` tells the model its age. See `MCP-005`.

## RSL-007

The cache lives in `internal/pkg/cache`. A context does not import a cache
library. The cache has a size limit in entries.

## RSL-008

The server limits the rate of its calls to each vendor. The configuration of the
context gives the limit.

**Why:** A vendor that blocks the server blocks every user of the server.

## RSL-009

A vendor failure never changes the readiness of the server. See `API-005`.

**Why:** One vendor outage must not remove the tools of every other vendor.

## Retired identifiers

This file has no retired identifier.
