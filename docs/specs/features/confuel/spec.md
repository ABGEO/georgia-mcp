---
id: CONFUEL
title: The fuel prices of Connect
type: spec
status: approved
created: 2026-10-21
updated: 2026-10-22
context: fuel
approved_by: Temuri
approved_on: 2026-10-21
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/confuel/requirements.md
---

# Specification: The fuel prices of Connect

## 1. Summary

This feature adds the vendor client module `pkg/connect`, a client of the REST route
of the price plugin of Connect, and a source of `fuel`. The tool, the use case, the
domain, and the cache do not change. The adapter turns each binary float of the
vendor into its shortest decimal, as `DOM-006` gives.

## 2. Coverage

| Requirement       | Where this specification realizes it                 |
| ----------------- | ---------------------------------------------------- |
| `CONFUEL-FR-001`  | 4.1 `adapter/connect`. `CONFUEL-DD-002`              |
| `CONFUEL-FR-002`  | 4.1 `adapter/connect`. `CONFUEL-DD-002`, `CONFUEL-DD-004` |
| `CONFUEL-FR-003`  | `CONFUEL-DD-003`                                     |
| `CONFUEL-FR-004`  | `CONFUEL-DD-003`                                     |
| `CONFUEL-FR-005`  | 4.1 `adapter/connect`, `Company`. `CONFUEL-DD-005`   |
| `CONFUEL-FR-006`  | 4.1 `adapter/connect`                                |
| `CONFUEL-FR-007`  | 4.3 configuration                                    |

`WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-INV-001`, and `WISFUEL-INV-003` apply
through the cache and the tool of `WISFUEL`. `CONFUEL-SC-008` checks them for Connect.

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                                 |
| ---------------------- | ------------------------------------------------------------------------------- |
| `ARC-004`, `CLT-001`   | `pkg/connect` is a module: `github.com/abgeo/georgia-mcp/pkg/connect`, `go 1.26.0`. |
| `CLT-002`, `CLT-004`   | `net/http`, `net/url`, and `encoding/json` only.                                |
| `CLT-003`              | The types carry the field names of Connect. The client keeps each number as `json.Number`. |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`.                                    |
| `CLT-007`, `CLT-008`   | A status that is not 200 is an `*APIError`. No retry, no cache.                 |
| `CLT-009`, `CLT-011`   | The package comment and the README say that the API is unofficial. `v0`.        |
| `DOM-005`, `DOM-011`   | `adapter/connect` converts the prices, names the grade, and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-006`, `MCP-009`   | The conversion of `DOM-006` for a binary float. A string in the output.         |
| `RSL-004`, `RSL-006`   | `adapter/cache.PriceCache`, unchanged, with the values of `WISFUEL` 4.3.        |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_FUEL_CONNECT_` and a default in its tag.   |
| `LOG-002`              | The adapter logs through the `slog` logger of the module only.                  |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `F` is `internal/app/georgia-mcp/fuel`.

| Layer          | Path                                  | Types and functions                                        |
| -------------- | ------------------------------------- | ---------------------------------------------------------- |
| Vendor client  | `pkg/connect/`                        | `Client`, `New`, `Option`, `Snapshot`, `FuelPrice`, `APIError`, `ErrUnexpectedStatus`. |
| Driven adapter | `F/adapter/connect/`                  | `Adapter`, `NewAdapter`.                                   |
| Module         | `F/module.go`, `F/config.go` (change) | `Config.Connect`, `ConnectConfig`, `ConnectHTTPConfig`, `connectSource`. |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                      |
| Workspace      | `go.work` (change)                    | `use ./pkg/connect`.                                       |
| Website        | `website/src/content/contexts.yaml` (change) | Connect, კონექტი, in the list of `fuel`.          |

**`pkg/connect`.** The shape of `pkg/socar`.

```go
const DefaultBaseURL = "https://connect.com.ge/wp-json/fuel-prices/v1"

var ErrUnexpectedStatus = errors.New("connect: unexpected status")

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Current(ctx context.Context) (Snapshot, error)

// Snapshot is the answer of GET /current.
type Snapshot struct {
    Date     string               `json:"date"`      // "2026-10-21", the day of the snapshot
    IsManual bool                 `json:"is_manual"`
    Updated  string               `json:"updated"`   // "2026-10-21 04:00:02"
    Prices   map[string]FuelPrice `json:"prices"`    // by key: "super_ron98"
}

type FuelPrice struct {
    Label       string      `json:"label"`        // "სუპერი RON 98"
    Standard    json.Number `json:"standard"`     // "4.3499999999999996447286321199499070644378662109375"
    SelfService json.Number `json:"self_service"` // "" when absent or null
}

type APIError struct {
    StatusCode int
    Body       string // the first 1 KiB
}
```

`Current` sends `GET /current` with `Accept: application/json`. Any status other than
200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The body limit is 1 MiB.
The default User-Agent is
`georgia-mcp-connect/<version> (+https://github.com/abgeo/georgia-mcp)`.

**`adapter/connect`.** `Company` returns
`{Code: "connect", Name: "Connect", PublishesEffectiveDate: false}`. `Prices` sends one
call and maps each entry of `Prices`:

| Field of the vendor | Domain value                                                     |
| ------------------- | ---------------------------------------------------------------- |
| The key             | `ProductPrice.Code`, the grade and the English name of `CONFUEL-DD-003`. |
| `Label`             | `ProductName.Georgian`, trimmed.                                 |
| `Standard`          | `CONFUEL-DD-002`. Not a number, or not above zero: the answer fails. |
| `SelfService`       | `CONFUEL-DD-002`. Empty or zero: none. Negative: the answer fails. |
| `Date`, `Updated`, `IsManual` | Ignored. `EffectiveDate` is the zero date. `CONFUEL-DD-005`. |

`ReadAt` is the instant of the answer. An empty map fails the answer. Each failure
returns an error that wraps `kernel.ErrSourceUnavailable`.

**`module.go` (change).** `newModule` builds the source of Connect after the last
source that it builds today, when `cfg.Connect.Enabled` is true. Else it logs at info
level `Connect is off` with the attribute `variable=GEOMCP_FUEL_CONNECT_ENABLED`.

### 4.2 Domain model

No change. `WISFUEL` 4.2 and `ROMFUEL` 4.2 hold the model.

### 4.3 Declarations

**MCP tool.** No change. The description lists `connect (Connect, publishes no
effective date)` through `companyList`.

**Vendor endpoint.**

| Client        | Method and path | Key  | Official              | Realizes         |
| ------------- | --------------- | ---- | --------------------- | ---------------- |
| `pkg/connect` | `GET /current`  | None | No, an unofficial API | `CONFUEL-FR-001` |

A real answer, recorded on 2026-10-21, cut to two products:

```
GET https://connect.com.ge/wp-json/fuel-prices/v1/current
200 application/json; charset=UTF-8 (994 bytes, no Cache-Control)
{"date":"2026-10-21","is_manual":false,"updated":"2026-10-21 04:00:02","prices":{
 "super_ron98":{"label":"სუპერი RON 98",
  "standard":4.3499999999999996447286321199499070644378662109375,"self_service":4.25},
 "diesel_10ppm":{"label":"დიზელი 10PPM",
  "standard":4.79999999999999982236431605997495353221893310546875,
  "self_service":4.6500000000000003552713678800500929355621337890625}}}
```

A day with no snapshot, recorded on 2026-10-21:

```
GET https://connect.com.ge/wp-json/fuel-prices/v1/date/2020-01-01
404 application/json
{"message":"No prices for 2020-01-01."}
```

**Cache.** `adapter/cache.PriceCache` wraps the adapter, as for Wissol.

| Port method         | Key      | Time to live | Maximum stale | Entries | Realizes |
| ------------------- | -------- | ------------ | ------------- | ------- | -------- |
| `Prices` of Connect | `prices` | 30 minutes   | 11.5 hours    | 1       | `WISFUEL-NFR-001`, `WISFUEL-NFR-002`, `WISFUEL-FR-015` |

**Configuration.**

| Variable                       | Type     | Default                                         | Secret | Realizes          |
| ------------------------------ | -------- | ----------------------------------------------- | ------ | ----------------- |
| `GEOMCP_FUEL_CONNECT_ENABLED`  | bool     | `true`                                          | No     | `CONFUEL-FR-007`  |
| `GEOMCP_FUEL_CONNECT_BASE_URL` | URL      | `https://connect.com.ge/wp-json/fuel-prices/v1` | No     | `CONFUEL-FR-001`  |
| `GEOMCP_FUEL_CONNECT_TIMEOUT`  | duration | `10s`                                           | No     | `WISFUEL-FR-014`  |
| `GEOMCP_FUEL_CONNECT_RATE`     | float    | `1`                                             | No     | `WISFUEL-NFR-002` |
| `GEOMCP_FUEL_CONNECT_BURST`    | int      | `1`                                             | No     | `WISFUEL-NFR-002` |

`fuel.Config` gains `Connect ConnectConfig` with `envPrefix:"CONNECT_"`, with the shape
of `WissolConfig` and the defaults of this table.

## 5. Design decisions

### `CONFUEL-DD-001`

**Realizes:** `CONFUEL-FR-001`, `CLT-001`
**Decision:** `pkg/connect` covers `GET /current` only.
**Rationale:** The requirements leave out the history.
**Alternatives:** `/history` and `/date/{date}` now. Two methods with no user.

### `CONFUEL-DD-002`

**Realizes:** `CONFUEL-FR-001`, `CONFUEL-FR-002`, `WISFUEL-INV-001`, `DOM-006`
**Decision:** The adapter converts each `json.Number` with
`strconv.ParseFloat(s, 64)`, then `strconv.FormatFloat(f, 'f', -1, 64)`, then
`decimal.NewFromString`. An error of a step fails the answer.
**Rationale:** The text of the vendor is the exact value of the `float64` of 4.35. The
shortest text that names that `float64` is `4.35`, the price that Connect entered. A
price with 3 places, `4.355`, keeps its 3 places.
**Alternatives:** `decimal.NewFromString` and `Round(2)`. It drops a third place. The
HTML page. It gives no key for the grade (requirements, question 1).

### `CONFUEL-DD-003`

**Realizes:** `CONFUEL-FR-003`, `CONFUEL-FR-004`
**Decision:** The adapter maps the key to a grade and an English name with the tables of
`CONFUEL-FR-003` and `CONFUEL-FR-004`. A key that is not in the table gets the grade
`other` and no English name. The adapter logs a warning with the key and the label, one
time for each key.
**Rationale:** `SOCFUEL-DD-003`. The key is stable. The label carries the octane, and
can change with a campaign.
**Alternatives:** A grade from the `RON` number of the label. It gives no grade to a diesel.

### `CONFUEL-DD-004`

**Realizes:** `CONFUEL-FR-002`
**Decision:** An empty `SelfService` or a zero gives no self-service price. A negative
one fails the answer.
**Rationale:** `WISFUEL-DD-005`. Go decodes `null` and an absent field to `""`.
**Alternatives:** A pointer. It adds a case with no other meaning.

### `CONFUEL-DD-005`

**Realizes:** `CONFUEL-FR-005`, `ROMFUEL-DD-006`
**Decision:** The adapter ignores `Date`, `Updated`, and `IsManual`. Every price has the
zero date, and `Company.PublishesEffectiveDate` is false.
**Rationale:** On 2026-10-21 `Date` is the day of the snapshot. The change was on
2026-10-13. The date of a change needs a second call to `/history`.
**Alternatives:** The newest day with `is_manual` true in `/history`. A manual entry
can repeat the prices of the day before, so it is not a change either.

## 6. Scenarios

`pkg/connect/testdata/` holds `current.json`, the real answer of 2026-10-21 with 5
products, and `current-edge.json`, a copy with the key `hydro` and a `diesel_10ppm` with
`"self_service":null`. An integration test serves the answers from `httptest` and calls
the tool through the in-memory MCP transport.

### `CONFUEL-SC-001`

**Verifies:** `CONFUEL-FR-001` **Layer:** unit

**Given** an `httptest` server that records the request and serves `current.json`.
**When** `Current` runs.
**Then** the path is `/current`, `Accept` is `application/json`, the User-Agent starts
with `georgia-mcp-connect/`, and `Prices` holds 5 keys. `super_ron98` has the
`Standard` text `4.3499999999999996447286321199499070644378662109375`.

### `CONFUEL-SC-002`

**Verifies:** `CONFUEL-FR-006` **Layer:** unit

**Given** the vendor answers 404 with `{"message":"No prices"}`.
**When** `Current` runs.
**Then** the error is an `*APIError` with 404. `errors.Is(err, ErrUnexpectedStatus)` holds.

### `CONFUEL-SC-003`

**Verifies:** `CONFUEL-FR-001` **Layer:** live

**Given** the real Connect. **When** `Current` runs.
**Then** `Prices` holds one key or more, and each has a label and a standard price.

### `CONFUEL-SC-004`

**Verifies:** `CONFUEL-FR-001` to `CONFUEL-FR-005` **Layer:** unit

**Given** the vendor serves `current.json`, and a fixed clock.
**When** the adapter reads the prices.
**Then** the list holds 5 prices with the zero date, and `ReadAt` is the instant of the clock.

| Code                | Grade     | Standard | Self-service | English name      |
| ------------------- | --------- | -------- | ------------ | ----------------- |
| `super_ron98`       | `super`   | `4.35`   | `4.25`       | Super RON 98      |
| `ron96`             | `premium` | `4.15`   | `3.95`       | Premium RON 96    |
| `regular_ron93`     | `regular` | `3.95`   | `3.75`       | Regular RON 93    |
| `euro_diesel_10ppm` | `diesel`  | `4.9`    | `4.75`       | Euro Diesel 10PPM |
| `diesel_10ppm`      | `diesel`  | `4.8`    | `4.65`       | Diesel 10PPM      |

`Company` is `{connect, Connect, false}`.

### `CONFUEL-SC-005`

**Verifies:** `CONFUEL-FR-002`, `CONFUEL-FR-003`, `CONFUEL-FR-004` **Layer:** unit

**Given** the vendor serves `current-edge.json`.
**When** the adapter reads the prices two times.
**Then** `diesel_10ppm` has no self-service price. `hydro` has the grade `other`, and
its name in `en` is its Georgian label. The log holds one warning with `hydro`.

### `CONFUEL-SC-006`

**Verifies:** `CONFUEL-FR-001`, `CONFUEL-FR-002`, `CONFUEL-FR-006` **Layer:** unit

Each case fails with an error that wraps `kernel.ErrSourceUnavailable`.

| Answer                                        | Reason                          |
| --------------------------------------------- | ------------------------------- |
| 200, `{"prices":{}}`                          | No product.                     |
| 200, `{}`                                     | No product.                     |
| 200, a product with `"standard":0`            | A price not above zero.         |
| 200, a product with `"standard":"abc"`        | A price that is not a number.   |
| 200, a product with `"self_service":-1`       | A negative self-service price.  |
| 404                                           | An error of the client.         |

### `CONFUEL-SC-007`

**Verifies:** `CONFUEL-FR-005`, `CONFUEL-FR-007`, `WISFUEL-FR-014` **Layer:** integration

**Given** Connect serves `current.json`, and every other company answers 500.
**When** a client lists the tools, then calls `fuel_get_prices` with the grade `premium`.
**Then** the description holds `connect (Connect, publishes no effective date)`. The
result is not an error. `connect` holds Premium RON 96 at `4.15`, self-service `3.95`,
and no `effective_date`.

### `CONFUEL-SC-008`

**Verifies:** `WISFUEL-NFR-001`, `WISFUEL-NFR-002` **Layer:** unit

**Given** a cache over the adapter of Connect, a transport that counts its calls, and
a fake clock.
**When** `Prices` runs 10 times in 29 minutes, then one time at 31 minutes.
**Then** the transport counts 1 call after the first 10, and 2 calls after the last.

### `CONFUEL-SC-009`

**Verifies:** `CONFUEL-FR-007` **Layer:** unit

**Given** no variable of fuel. **When** the configuration loads.
**Then** Connect is on, with the base address
`https://connect.com.ge/wp-json/fuel-prices/v1`, a timeout of 10 seconds, the rate 1,
and the burst 1.

### `CONFUEL-SC-010`

**Verifies:** `CONFUEL-FR-007` **Layer:** integration

**Given** `GEOMCP_FUEL_CONNECT_ENABLED` is false, and Wissol is on.
**When** the module starts, and a client calls `fuel_get_prices` with the company `connect`.
**Then** the log holds `Connect is off` with `variable=GEOMCP_FUEL_CONNECT_ENABLED`. The
tool result is an error: `connect` is not a fuel company of this server.

## 7. Build plan

| #   | Step                                                                   | Realizes                            | Done |
| --- | ---------------------------------------------------------------------- | ----------------------------------- | ---- |
| 1   | `pkg/connect` module, recorded answers, README, example. `go.work`.    | `CONFUEL-FR-001`, `CONFUEL-FR-006`  | [x]  |
| 2   | `adapter/connect`.                                                     | `CONFUEL-FR-001` to `CONFUEL-FR-006` | [x]  |
| 3   | `module.go`, `config.go`, `configs/.env.example`.                      | `CONFUEL-FR-007`                    | [x]  |
| 4   | Move the term "Connect" to the glossary. Add Connect to the website.   | `DOM-002`                           | [x]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
