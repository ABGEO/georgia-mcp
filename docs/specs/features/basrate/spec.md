---
id: BASRATE
title: The commercial exchange rates of Basisbank
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: rates
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [ARC, DOM, PKG, MCP, CLT, RSL, CFG, LOG, GO, TST]
requirements: features/basrate/requirements.md
---

# Specification: The commercial exchange rates of Basisbank

## 1. Summary

This feature adds the vendor client `pkg/basisbank`, a client of the unofficial rate method of
the site of Basisbank, and Basisbank as a source of `rates_get_commercial_rates`. The tool,
the use case, the domain, and the cache of `TBCRATE` do not change. One call reads every
currency.

## 2. Coverage

| Requirement       | Where this specification realizes it                       |
| ----------------- | ---------------------------------------------------------- |
| `BASRATE-FR-001`  | 4.1 `adapter/basisbank`. 4.3 vendor endpoint. `BASRATE-DD-001` |
| `BASRATE-FR-002`  | `BASRATE-DD-002`                                           |
| `BASRATE-FR-003`  | 4.1 `adapter/basisbank`. `BOGRATE-DD-008`                      |
| `BASRATE-FR-004`  | 4.3 configuration. `BOGRATE-DD-006`, `BOGRATE-DD-007`      |
| `BASRATE-NFR-001` | 4.3 configuration. `LIBRATE-DD-005`                        |
| `BASRATE-INV-001` | `TERARATE-DD-004`                                          |

## 3. Guideline compliance

| Rule                   | How this specification obeys it                                              |
| ---------------------- | ---------------------------------------------------------------------------- |
| `CLT-001`              | `pkg/basisbank` is a package of the server module: `github.com/abgeo/georgia-mcp/pkg/basisbank`. |
| `CLT-002`, `CLT-004`   | The standard library and `pkg/vendorhttp` only.                              |
| `CLT-003`              | The types carry the field names of the vendor in their JSON tags. `BASRATE-DD-003`. |
| `CLT-005`              | `Accept: application/json`. No `WithAPIKey`.                                 |
| `CLT-007`              | A status other than 200 is an `*APIError`.                                   |
| `CLT-009`              | The package comment and the README say that the API is unofficial.           |
| `DOM-005`, `DOM-011`   | `adapter/basisbank` converts the vendor shape and wraps `kernel.ErrSourceUnavailable`. |
| `DOM-009`              | Basisbank is an adapter of `CommercialRateSource`.                           |
| `MCP-002`              | No new tool. The description lists `basisbank` through `GetCommercialRates.Banks`. |
| `RSL-003` to `RSL-006` | Basisbank sits behind a `CommercialRateCache`, as TBC does.                  |
| `RSL-008`              | The configuration gives the limit of the calls.                              |
| `CFG-003`, `CFG-005`   | Each variable has the prefix `GEOMCP_RATES_BASISBANK_` and a default in `rates.Defaults`. |

## 4. Design

### 4.1 Components

Every path is new unless the table says "change". `R` is `internal/app/georgia-mcp/rates`.

| Layer          | Path                                  | Types and functions                                          |
| -------------- | ------------------------------------- | ------------------------------------------------------------ |
| Guidelines     | `docs/specs/guidelines/glossary.md` (change) | The term of `BASRATE` 5.                              |
| Vendor client  | `pkg/basisbank/`                          | `Client`, `New`, `Option`, `Day`, `XRates`, `APIError`, `ErrUnexpectedStatus`, `ErrMalformedRates`. |
| Driven adapter | `R/adapter/basisbank/`                    | `Adapter`, `NewAdapter`.                                     |
| Module         | `R/module.go`, `R/config.go` (change) | `Config.Basisbank`, `basisbankSource`.                               |
| Configuration  | `configs/.env.example` (change)       | The variables of 4.3.                                        |

**`pkg/basisbank`.** The shape of `pkg/terabank`, with one method.

```go
const DefaultBaseURL = "https://static.bb.ge/api/view"

var (
    ErrUnexpectedStatus = errors.New("basisbank: unexpected status")
    ErrMalformedRates   = errors.New("basisbank: malformed rates") // xrates is not a JSON object
)

func New(opts ...Option) *Client // the options of vendorhttp
func (c *Client) Rates(ctx context.Context) ([]Day, error)

// Day is the rates of Basisbank for one date.
type Day struct {
    Date   string // "2026-10-25"
    XRates XRates
}

// XRates is the text of the field xrates of a day, decoded.
type XRates struct {
    Kurs     map[string]string `json:"kurs"`     // the official rate, per the unit of the NBG
    KursBuy  map[string]string `json:"kursBuy"`  // "USD": "2.5600"
    KursSell map[string]string `json:"kursSell"`
}

type APIError = vendorhttp.StatusError
```

`Rates` sends `POST /bb/websiteXrates` with the body `http.NoBody` and no token header.
It decodes `[{"date": string, "xrates": string}]`, then decodes each `xrates` text into
`XRates`. A text that does not decode returns an error that wraps `ErrMalformedRates`.
Any status other than 200 returns an `*APIError` that wraps `ErrUnexpectedStatus`. The
body limit is 64 KiB: the answer has 721 bytes.

**`adapter/basisbank`.** `Bank` returns
`{Code: "basisbank", Name: "Basisbank", Kind: domain.KindBank}`. `CommercialRates` sends one
call, and takes the day of `BASRATE-DD-002`. For each requested code it reads the key of
that code in `KursBuy` and `KursSell`:

| Field of the day  | Domain value                       |
| ----------------- | ---------------------------------- |
| `KursBuy[code]`   | `Buy`. Parsed as `TERARATE-DD-004` gives. |
| `KursSell[code]`  | `Sell`.                            |
| `Kurs`            | Not read.                          |

A requested code with no key in one of the two sets, or with a price that is empty or
zero, goes to `Missing`. A price that is not a number or is below zero fails the bank.
`ReadAt` is the instant of the answer.

**`module.go` (change).** `commercialSources` builds the source of Basisbank when
`cfg.Basisbank.Enabled` is true. Else it logs at info level `Basisbank is off` with the
attribute `variable=GEOMCP_RATES_BASISBANK_ENABLED`.

### 4.2 Domain model

No change. `TBCRATE` 4.2 holds the model.

### 4.3 Declarations

**Vendor endpoint.**

| Client       | Method and path           | Key  | Official              | Realizes         |
| ------------ | ------------------------- | ---- | --------------------- | ---------------- |
| `pkg/basisbank`  | `POST /bb/websiteXrates`  | None | No, an unofficial API | `BASRATE-FR-001` |

A real answer, recorded on 2026-10-25, with the key of one currency cut from each set:

```
POST https://static.bb.ge/api/view/bb/websiteXrates
Content-Length: 0
200 application/json
X-Ratelimit-Limit: 1600
Access-Control-Allow-Origin: *
[{"date":"2026-10-25","xrates":"{\"kurs\":{\"CNY\":\"3.8939\",\"EUR\":\"2.9090\",\"GBP\":\"3.4749\",\"USD\":\"2.5971\"},\"kursBuy\":{\"CNY\":\"0.3700\",\"EUR\":\"2.8160\",\"GBP\":\"3.3460\",\"USD\":\"2.5600\"},\"kursSell\":{\"CNY\":\"0.4100\",\"EUR\":\"2.9980\",\"GBP\":\"3.5960\",\"USD\":\"2.6300\"}}"},
 {"date":"2026-10-24","xrates":"{\"kurs\":{...},\"kursBuy\":{...},\"kursSell\":{...}}"}]
```

**Configuration.**

| Variable                       | Type     | Default                         | Secret | Realizes          |
| ------------------------------ | -------- | ------------------------------- | ------ | ----------------- |
| `GEOMCP_RATES_BASISBANK_ENABLED`   | bool     | `true`                          | No     | `BASRATE-FR-004`  |
| `GEOMCP_RATES_BASISBANK_BASE_URL`  | URL      | `https://static.bb.ge/api/view` | No     | `BASRATE-FR-001`  |
| `GEOMCP_RATES_BASISBANK_TIMEOUT`   | duration | `10s`                           | No     | `BASRATE-FR-001`  |
| `GEOMCP_RATES_BASISBANK_RATE`      | float    | `1`                             | No     | `BASRATE-NFR-001` |
| `GEOMCP_RATES_BASISBANK_BURST`     | int      | `1`                             | No     | `BASRATE-NFR-001` |

`rates.Config` gains `Basisbank vendors.Config` with `envPrefix:"BASISBANK_"`. `rates.Defaults`
gives the defaults of this table (`PLATFORM-DD-014`). The cache is a
`CommercialRateCache`: 5 minutes to live, 55 minutes of maximum stale, 256 entries.

## 5. Design decisions

### `BASRATE-DD-001`

**Realizes:** `BASRATE-FR-001`
**Decision:** The client reads `websiteXrates`, with a POST and no token.
**Rationale:** The page reads this method, so its answer is the rate that a person sees.
The answer has 721 bytes. A POST is the call of the page, so a filter that serves the
page serves the client. The method answers without the token of the page, so the client
needs no first call to fetch one.
**Alternatives:** `Rate.ashx`. 3.4 MB of history in each call, over HTTP with no TLS.
The page `bb.ge/ge/currencies`. It holds the same text in a page of 3.4 MB.

### `BASRATE-DD-002`

**Realizes:** `BASRATE-FR-002`
**Decision:** The adapter parses each `Date` as `2006-01-02` and takes the day with the
greatest date. An empty list, a date that does not parse, or two days of one date fail
the bank.
**Rationale:** The answer of 2026-10-25 holds the newest date first, but the order is not
a contract. The greatest date is the current rate under any order.
**Alternatives:** The first day. A change of the order would give the rate of the day
before, with no error.

### `BASRATE-DD-003`

**Realizes:** `CLT-003`
**Decision:** The client decodes the text of `xrates`, and returns it as `XRates`. The
prices stay text.
**Rationale:** The text is the data of the vendor, written twice. A caller that receives
the raw text decodes it again in each project. The decode interprets no value.
**Alternatives:** `Day.XRates` as a `string`. The adapter would then decode a vendor
format, the work of a client (`CLT-004`).

## 6. Scenarios

`pkg/basisbank/testdata/` holds `website-xrates.json`, the real answer of 2026-10-25.

### `BASRATE-SC-001`

**Verifies:** `BASRATE-FR-001`, `BASRATE-FR-004`
**Layer:** unit

**Given** an `httptest` server that records the request.
**When** `Rates` runs.
**Then** the method is `POST`, the path is `/bb/websiteXrates`, `Content-Length` is `0`,
`Accept` is `application/json`, the User-Agent starts with `georgia-mcp/`, and the
request carries no `X-CSRF-Token` and no key header.

### `BASRATE-SC-002`

**Verifies:** `BASRATE-FR-001`
**Layer:** unit

**Given** the vendor serves `website-xrates.json`.
**When** `Rates` runs.
**Then** the result holds 2 days. The first has `Date` `2026-10-25`, `KursBuy["USD"]`
`2.5600`, `KursSell["USD"]` `2.6300`, and `Kurs["CNY"]` `3.8939`.

### `BASRATE-SC-003`

**Verifies:** `BASRATE-FR-001`
**Layer:** unit

| Vendor answer                                   | Client result                                |
| ----------------------------------------------- | -------------------------------------------- |
| `[{"date":"2026-10-25","xrates":"not json"}]`   | `errors.Is(err, ErrMalformedRates)`          |
| 429 with a body of 2 KiB                        | `*APIError` with 429 and the first 1 KiB     |
| 200 with `[` only                               | An error that is not an `*APIError`          |

### `BASRATE-SC-004`

**Verifies:** `BASRATE-FR-001`
**Layer:** live

**Given** the real method of Basisbank.
**When** `Rates` runs.
**Then** a day holds a buy price and a sell price of `USD` above zero.

### `BASRATE-SC-005`

**Verifies:** `BASRATE-FR-001`, `BASRATE-INV-001`
**Layer:** unit

**Given** the vendor serves `website-xrates.json`.
**When** the adapter asks for `CNY`, `EUR`, and `USD`.

| Currency | Buy formats as | Sell formats as |
| -------- | -------------- | --------------- |
| `CNY`    | `0.37`         | `0.41`          |
| `EUR`    | `2.816`        | `2.998`         |
| `USD`    | `2.56`         | `2.63`          |

### `BASRATE-SC-006`

**Verifies:** `BASRATE-FR-002`
**Layer:** unit

| Days of the answer, in order                                    | The adapter reads |
| --------------------------------------------------------------- | ----------------- |
| `2026-10-25` with `USD` `2.5600`, `2026-10-24` with `USD` `2.5500` | `2.56`         |
| `2026-10-24` with `USD` `2.5500`, `2026-10-25` with `USD` `2.5600` | `2.56`         |
| none                                                            | An error that wraps `kernel.ErrSourceUnavailable` |
| `25.10.2026`                                                    | An error that wraps `kernel.ErrSourceUnavailable` |
| `2026-10-25` twice                                              | An error that wraps `kernel.ErrSourceUnavailable` |

### `BASRATE-SC-007`

**Verifies:** `BASRATE-FR-003`
**Layer:** unit

**Given** a day with the prices of `USD` of `website-xrates.json`, and the prices of
`CHF` of the table.
**When** the adapter asks for `CHF`, `JPY`, and `USD`.
**Then** the result holds the rate of `USD`, and `Missing` is `[CHF JPY]`.

| `KursBuy["CHF"]` | `KursSell["CHF"]` |
| ---------------- | ----------------- |
| `0.0000`         | `3.2000`          |
| `3.1000`         | absent            |

### `BASRATE-SC-008`

**Verifies:** `BASRATE-FR-001`
**Layer:** unit

| Vendor answer                         | Adapter error                        |
| ------------------------------------- | ------------------------------------ |
| 500                                   | Wraps `kernel.ErrSourceUnavailable`  |
| `KursBuy["USD"]` is `n/a`             | Wraps `kernel.ErrSourceUnavailable`  |
| `KursBuy["USD"]` is `-2.5600`         | Wraps `kernel.ErrSourceUnavailable`  |

### `BASRATE-SC-009`

**Verifies:** `BASRATE-FR-004`, `BASRATE-NFR-001`
**Layer:** unit

**Given** an environment with no variable of Basisbank.
**When** the configuration loads.
**Then** `Basisbank.Enabled` is true, `Basisbank.BaseURL` is `https://static.bb.ge/api/view`,
`Basisbank.HTTP.Rate` is 1, and `Basisbank.HTTP.Burst` is 1.

### `BASRATE-SC-010`

**Verifies:** `BASRATE-FR-001`
**Layer:** integration

**Given** Basisbank on, every other bank off, the vendor serves `website-xrates.json`, and a fixed clock.
**When** a client calls `rates_get_commercial_rates` with `USD` and `CHF`.
**Then** the bank `basisbank` has `source` `Basisbank`, `as_of` is the instant of the clock,
`buy` is `2.56`, `sell` is `2.63`, and `missing` is `["CHF"]`.

### `BASRATE-SC-011`

**Verifies:** `BASRATE-FR-004`
**Layer:** integration

**Given** Basisbank off and `bog` on.
**When** a client calls the tool with the bank `basisbank`.
**Then** the tool result reads `banks: "basisbank" is not a bank of this server. Use one of: bog.`
The log holds one info record that names `GEOMCP_RATES_BASISBANK_ENABLED`.

## 7. Build plan

| #   | Step                                                                   | Realizes                                   | Done |
| --- | ---------------------------------------------------------------------- | ------------------------------------------ | ---- |
| 1   | `pkg/basisbank`, `testdata/website-xrates.json`, README, example, live test. | `BASRATE-FR-001`                         | [ ]  |
| 2   | `adapter/basisbank`.                                                       | `BASRATE-FR-001` to `BASRATE-FR-003`, `BASRATE-INV-001` | [ ]  |
| 3   | `config.go`: `Config.Basisbank`, `Defaults`. `configs/.env.example`.       | `BASRATE-FR-004`, `BASRATE-NFR-001`        | [ ]  |
| 4   | `module.go`: the source of Basisbank, and the info record. The glossary. | `BASRATE-FR-001`, `BASRATE-FR-004`       | [ ]  |

## 8. Out of scope for this specification

Nothing that the requirements demand.

## Retired identifiers

This file has no retired identifier.
