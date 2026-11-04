---
id: GEOSTAT
title: The statistical tables and the inflation of Geostat, the scenarios
type: spec
status: approved
created: 2026-11-04
updated: 2026-11-04
context: stats
approved_by: Temuri
approved_on: 2026-11-04
constrained_by: [TST]
requirements: features/geostat/requirements.md
---

# Specification: The statistical tables and the inflation of Geostat, the scenarios

`pkg/geostat/testdata/` holds the answers of 2026-11-04: the root, the folders of the path
`Price Statistics/CPI/National Consumer Price Index` and `Social Statistics/Labour`, the
metadata and the cells of the four CPI tables for 2026, `Wages_Region.px`, and the 404
pages of IIS and of a parameter error. The tests of `adapter/tool` call each tool through
the in-memory MCP transport, with the real use cases and fake ports. The test of
`module.go` serves the files from `httptest`.

| ID               | Verifies | Layer | Given, when, then |
| ---------------- | -------- | ----- | ----------------- |
| `GEOSTAT-SC-001` | `GEOSTAT-NFR-006` | unit | A server records each request. `Folder`, `Table`, and `Query` each carry a User-Agent that starts with `georgia-mcp/` and holds `https://github.com/abgeo/georgia-mcp`, and `Accept: application/json`. |
| `GEOSTAT-SC-002` | `GEOSTAT-FR-005` | unit | `Table(ctx, "en", "Price Statistics/CPI/National Consumer Price Index/CPI_in_Georgia_(previous_month=100)_orenovani2.px")` requests the path that ends in `_orenovani2.px/`, with `%28` and `%20`. `Folder(ctx, "en", "")` requests `/en/Database/`. |
| `GEOSTAT-SC-003` | `GEOSTAT-FR-006` | unit | `Query` sends the body of 4.1 with `filter` `item`, and decodes an answer that starts with a byte order mark: 2 data rows, the figure `"100.9"`. |
| `GEOSTAT-SC-004` | `GEOSTAT-FR-015` | unit | A 404 page, a 500, and a parameter error each give an `*APIError` with its status. `errors.Is(err, ErrUnexpectedStatus)` holds. |
| `GEOSTAT-SC-005` | `GEOSTAT-INV-001`, `GEOSTAT-INV-002` | unit | `ParseFigure`: `105.9` keeps `"105.9"`; `..`, `.`, `-`, and `""` are unpublished; `1,5` and `abc` give `ErrInvalidFigure`. `Minus100`: `105.9` gives `5.9`, `100.0` gives `0.0`, `99.5` gives `-0.5`. A fuzz test holds that a published figure parses back to its text. |
| `GEOSTAT-SC-006` | `GEOSTAT-FR-005` | unit | `ParseTableCode`: `Social Statistics/Labour/Wages_Region.px` and `Population Census 2014/X/02TOTA~1.PX` are valid. `""`, `a//b.px`, `../x.px`, `Labour/Wages`, and 301 runes give `ErrInvalidTableCode`. A fuzz test. `ParseMonth`, `ParseRegion`, and `ParsePriceGroup` have their own table and fuzz test. |
| `GEOSTAT-SC-007` | `GEOSTAT-FR-001`, `GEOSTAT-FR-002`, `GEOSTAT-FR-003` | unit | A fake catalog of 3 entries. `salary region` gives `Wages_Region.px`. `CONSUMER price` gives the two CPI entries, the newest first. `zzqx` gives none and no error. The limit 1 gives 1 entry and `Total` 2. `""` and 101 runes give `ErrInvalidQuery`. |
| `GEOSTAT-SC-008` | `GEOSTAT-FR-005` | unit | The adapter with the recorded files: `Wages_Region.px` gives 3 dimensions with 12, 17, and 5 values, the title, and `Updated` 2026-10-08 19:17:47 in Tbilisi. `Labour/Nope.px` gives `app.ErrUnknownTable`. `Nope/Wages_Region.px` with a 404 folder gives it too. In `ka`, the catalog entry and `Table` of `Wages_Region.px` both give the folder `Social Statistics / Labour`. |
| `GEOSTAT-SC-009` | `GEOSTAT-FR-006`, `GEOSTAT-FR-007`, `GEOSTAT-FR-009` | unit | A fake table of `Wages_Region.px`. `Region` values `["1"]`, `Year` last 1, `Quarter` omitted: `Cells` gets `Region ["1"]`, `Year ["16"]`, `Quarter ["0".."4"]`, and the 5 rows are `["Tbilisi","2026","I"]` to `["Tbilisi","2026","Annual"]`. An unpublished figure stays unpublished. `Year` last 50 gives 17 codes. |
| `GEOSTAT-SC-010` | `GEOSTAT-FR-008`, `GEOSTAT-FR-010` | unit | A table of 7 × 13 × 23 × 12 values. No selection: `ErrTooManyCells` with 25116 cells, and no `Cells` call. `Town`, a value `99`, values and last together, last 0 with no values, last -1, and `Year` twice each give `ErrInvalidSelection` with the accepted codes. |
| `GEOSTAT-SC-011` | `GEOSTAT-FR-011`, `GEOSTAT-FR-012`, `GEOSTAT-FR-014` | unit | The adapter with the recorded CPI files gives for 2026, `georgia`, `total`, month 10: `Annual` `105.9`, `Monthly` `100.9`, and `Core` `4.0`. Month 11 is unpublished. `Average` holds `total` only. |
| `GEOSTAT-SC-012` | `GEOSTAT-FR-015` | unit | A metadata with a renamed group, a dimension with 12 values and 11 texts, a figure `abc`, and an update date `yesterday` each give an error that wraps `kernel.ErrSourceUnavailable`. A year that a table does not list gives an empty map and no error. |
| `GEOSTAT-SC-013` | `GEOSTAT-FR-011`, `GEOSTAT-FR-013` | unit | A fake source and today 2026-11-04. No month gives 2026-10, annual `5.9`, monthly `0.9`. The month 2026-11 gives a `MonthError` with the newest 2026-10. Today 2027-01-03 with no index for 2027 gives 2026-12 from the year before. `tbilisi` and `food` give an empty `Average` and `Core`. |
| `GEOSTAT-SC-014` | `GEOSTAT-FR-001`, `GEOSTAT-FR-003`, `GEOSTAT-FR-004` | integration | `stats_search_tables` with `salary` and no language: `source` `Geostat`, `as_of` with `+04:00`, `total` and `tables`. With `ru`: the message of `spec-tools.md` 2. With `limit` 500: the message of `spec-tools.md` 2. |
| `GEOSTAT-SC-015` | `GEOSTAT-FR-005`, `GEOSTAT-FR-006`, `GEOSTAT-FR-009` | integration | `stats_get_table` gives each dimension with its values and `updated` `2026-10-08`. `stats_get_cells` gives `figure` `"105.9"`, and `null` for a month with `..`. An unknown code gives the message of `spec-tools.md` 2. |
| `GEOSTAT-SC-016` | `GEOSTAT-FR-011`, `GEOSTAT-FR-004` | integration | `stats_get_inflation` with no input gives `month` `2026-10`, `annual` `"5.9"`, `region` `{"georgia","Georgia"}`. With `ka`, the region name is `საქართველო`. With `rustavi`, the message of `spec-tools.md` 2. The output has 4 `tables`, each with `updated`. |
| `GEOSTAT-SC-017` | `GEOSTAT-NFR-008` | unit | A server that hangs on the first call to one folder, then answers. The adapter with a recording `pause` sends that folder 2 times, with one pause of 2 seconds, and returns the catalog. A folder that fails 3 times gives `kernel.ErrSourceUnavailable` after 3 calls. A table query that fails is sent 1 time. |
| `GEOSTAT-SC-018` | `GEOSTAT-NFR-007`, `GEOSTAT-FR-017` | unit | In `synctest`, a `Holder` over a fake source that takes 5 minutes for each language. Before 5 minutes, `Catalog(en)` gives `ErrCatalogLoading`. At 5 minutes, it gives the catalog of `en`. At 10 minutes, `ka` is ready. The first read of `en` starts at 0. |
| `GEOSTAT-SC-019` | `GEOSTAT-NFR-001` | unit | In `synctest`, after a read at 0, the source counts a second read of `en` at 24 hours, not before. |
| `GEOSTAT-SC-020` | `GEOSTAT-NFR-002` | unit | In `synctest`, one read, then each read fails. At 25 hours, `Catalog` gives the catalog with `Stale`. At 7 days, it gives `kernel.ErrSourceUnavailable`. |
| `GEOSTAT-SC-021` | `GEOSTAT-NFR-009` | unit | In `synctest`, a read fails at 0. Before 15 minutes, `Catalog` of that language gives `kernel.ErrSourceUnavailable`. The next read starts at 15 minutes. A second `Start` sends no second read. `Close` returns within 10 seconds and no read starts after it. |
| `GEOSTAT-SC-022` | `GEOSTAT-NFR-004` | unit | In `synctest`, two `Cells` with the same selection in another order within 1 hour send 1 load. At 1 hour, a second. A failed load at 23 hours gives the stale value. At 24 hours, the error. The same for `Table` and `PriceIndices`. |
| `GEOSTAT-SC-023` | `GEOSTAT-NFR-003` | unit | A benchmark of `SearchTables.Search` over the recorded catalog of 1,046 entries: each search takes less than 500 ms. |
| `GEOSTAT-SC-024` | `GEOSTAT-NFR-005` | unit | In `bootstrap`, `LoadConfig` with no variable of stats gives Geostat on, at `https://pc-axis.geostat.ge/PXWeb/api/v1`, with a timeout of 10 seconds, the rate 9, and the burst 1. |
| `GEOSTAT-SC-025` | `GEOSTAT-FR-016` | unit | `stats.New` with `Enabled` false registers no tool and starts no read. With `Enabled` true, it registers the four tools, and `Close` stops the job. |
| `GEOSTAT-SC-026` | `GEOSTAT-FR-005`, `GEOSTAT-FR-011` | live | Against Geostat: `Table` of `Wages_Region.px` has the dimension `Region`. `PriceIndices(2026)` has a published `Annual` for `georgia`, `total`, and month 1. |

## Retired identifiers

This file has no retired identifier.
