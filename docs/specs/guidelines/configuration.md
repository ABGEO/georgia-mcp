---
id: CFG
title: Configuration and secrets
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-17
scope: [internal/pkg/config/, internal/app/georgia-mcp/bootstrap/, internal/app/georgia-mcp/*/config.go, configs/]
related: [ARC, LIF, LOG]
---

# Configuration and secrets

## CFG-001

The configuration comes from environment variables. `github.com/caarlos0/env/v11`
decodes them into structs.

## CFG-002

At the start, the process loads `.env` from the working directory when the file
exists. `github.com/joho/godotenv` loads it. A variable of the environment wins
over the same variable in `.env`.

**Why:** A developer keeps a local file. A container sets the environment.
The environment must win, so a deployment never depends on a stray file.

## CFG-003

Every variable has the prefix `GEOMCP_`.
A context adds its own prefix: `GEOMCP_RATES_`, `GEOMCP_TRANSIT_`.

## CFG-004

Each context declares its `Config` struct in `config.go`. The root struct in
`bootstrap` holds the struct of each context with an `envPrefix` tag.
A context receives its own struct. It reads no variable itself.

```go
type Config struct {
    Log   logger.Config     `envPrefix:"LOG_"`
    HTTP  httpserver.Config `envPrefix:"HTTP_"`
    Rates rates.Config      `envPrefix:"RATES_"`
}
```

## CFG-005

A field declares its default in an `envDefault` tag. A required field carries
`required` and `notEmpty`. A struct with a rule that a tag cannot express has a
`Validate() error` method. The process validates the configuration at the start
and stops on an error.

**Why:** A wrong value must fail at the start, not at the first tool call.

## CFG-006

A duration is a `time.Duration`, written as `10s`. A URL is a `url.URL`.
A listen address is a `host:port` string.

## CFG-007

A secret does not go into the repository. `.env` stays in `.gitignore`.
`configs/.env.example` lists every variable, with its default or an empty value.

## Retired identifiers

This file has no retired identifier.
