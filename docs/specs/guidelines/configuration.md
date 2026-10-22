---
id: CFG
title: Configuration and secrets
type: guideline
status: active
created: 2026-10-17
updated: 2026-10-22
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

A field declares its default in an `envDefault` tag, and its rule in a `validate` tag of
`github.com/go-playground/validator/v10`. A struct that more than one vendor uses,
`vendors.Config` and `httpclient.Config`, has no `envDefault` tag: a tag holds one value
for every vendor. The context gives the value of each vendor in `Defaults() Config`,
`bootstrap.Defaults` composes them, and `config.Load` starts from that value. A struct
has no `Validate` method. `config.Load` validates the configuration at the start and
stops on an error. Each error names the variable.

| Value                                | Tag                        |
| ------------------------------------ | -------------------------- |
| The base address of a vendor         | `http_url`                 |
| A timeout, a rate                    | `gt=0`                     |
| A burst, a count of entries          | `min=1`                    |
| The key of a vendor with a switch    | `required_if=Enabled true` |

A rule that no tag of the validator expresses is a tag that `internal/pkg/config`
registers, with its message.

**Why:** A wrong value must fail at the start, not at the first tool call. A rule in a tag
needs no method that repeats it for each vendor. A default of a shared struct differs
for each vendor, and its tag cannot: a Go value holds each one, next to the others of its
context.

## CFG-006

A duration is a `time.Duration`, written as `10s`. A URL is a `url.URL`.
A listen address is a `host:port` string.

## CFG-007

A secret does not go into the repository. `.env` stays in `.gitignore`.
`configs/.env.example` lists every variable, with its default or an empty value.

## CFG-008

A vendor that its context can serve without has a switch `<PREFIX>_ENABLED`, on by
default. The operator turns the vendor off with `false`. A vendor with a key needs the
key while it is on: an empty key fails the start, and the error names the key and the
switch.

```
GEOMCP_RATES_TBC_ENABLED=true
GEOMCP_RATES_TBC_API_KEY=
```

The configuration of such a vendor is `vendors.Config`: the switch, the base address,
and the calls. `vendors.Defaults(address)` gives it on, at the address, with 1 call each
second. `WithLimit` sets another limit. A vendor with a value of its own embeds
`vendors.Config` and adds the field.

```go
type Config struct {
    Wissol vendors.Config `envPrefix:"WISSOL_"`
    BOG    vendors.Config `envPrefix:"BOG_"`
}

func Defaults() Config {
    return Config{
        Wissol: vendors.Defaults("https://api.wissol.ge"),
        BOG:    vendors.Defaults("https://api.businessonline.ge").WithLimit(0.8, 10),
    }
}
```

**Why:** One rule for every vendor. A vendor never goes off because a value is missing,
so a wrong deployment fails at the start. See `CFG-005`.

## Retired identifiers

This file has no retired identifier.
