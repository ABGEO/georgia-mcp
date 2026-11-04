---
id: GEOSTAT
title: The statistical tables and the inflation of Geostat, the tools
type: spec
status: approved
created: 2026-11-04
updated: 2026-11-04
context: stats
approved_by: Temuri
approved_on: 2026-11-04
constrained_by: [MCP]
requirements: features/geostat/requirements.md
---

# Specification: The statistical tables and the inflation of Geostat, the tools

## 1. The tools

| Tool                  | Input                                       | Output            | Errors | Realizes |
| --------------------- | ------------------------------------------- | ----------------- | ------ | -------- |
| `stats_search_tables` | `query` (required), `language`, `limit`     | `tablesOutput`    | 2      | `GEOSTAT-FR-001` to `GEOSTAT-FR-004`, `GEOSTAT-FR-017` |
| `stats_get_table`     | `code` (required), `language`               | `tableOutput`     | 2      | `GEOSTAT-FR-004`, `GEOSTAT-FR-005` |
| `stats_get_cells`     | `code` (required), `selection`, `language`  | `cellsOutput`     | 2      | `GEOSTAT-FR-006` to `GEOSTAT-FR-010` |
| `stats_get_inflation` | `month`, `region`, `group`, `language`      | `inflationOutput` | 2      | `GEOSTAT-FR-011` to `GEOSTAT-FR-014` |

| Tool                  | Title                                   | Description |
| --------------------- | --------------------------------------- | ----------- |
| `stats_search_tables` | Search the statistical tables of Georgia | Finds the tables of Geostat, the National Statistics Office of Georgia, whose title or folder holds every word of a text: for example prices, salaries, population, GDP, and trade. Returns the code, the title, the folder, and the update date of each table, the newest first. Then call stats_get_table with a code. Data from Geostat. |
| `stats_get_table`     | Describe a statistical table of Georgia | Returns the title and the dimensions of one table of Geostat, with the code and the text of each value. Read it before stats_get_cells. The values of a time dimension go from the oldest to the newest. A table has no unit field: the title names it. Data from Geostat. |
| `stats_get_cells`     | Get the figures of a statistical table  | Returns the figures of one table of Geostat for a selection: value codes, or the last values, for each dimension. A dimension with no selection takes every value. Selects at most 1000 cells. An unpublished figure is null. The title of the table names the unit. Data from Geostat. |
| `stats_get_inflation` | Get the inflation of Georgia            | Returns the inflation of Georgia for one month, in percent: annual (to the same month of the previous year), monthly (to the previous month), the 12 month average, and core inflation. For the whole country, a city, and a price group. The default is the newest published month. Data from the consumer price index of Geostat. |

```go
type searchTablesInput struct {
    Query    string `json:"query"              jsonschema:"The words to find in the title or the folder of a table, in the language of the request. 1 to 100 characters."`
    Language string `json:"language,omitempty" jsonschema:"The language of the titles: ka or en. The default is en."`
    Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of tables, 1 to 100. The default is 20."`
}

type tablesOutput struct {
    Source string        `json:"source" jsonschema:"The source of the data: Geostat."`
    AsOf   string        `json:"as_of"  jsonschema:"The instant at which the server read the catalog, RFC 3339."`
    Stale  bool          `json:"stale"  jsonschema:"True when Geostat did not answer the last read of the catalog."`
    Total  int           `json:"total"  jsonschema:"The count of tables that match. More than the count of tables when the limit cut the list."`
    Tables []tableOutput `json:"tables" jsonschema:"The matching tables, the newest update first."`
}

type getTableInput struct {
    Code     string `json:"code"               jsonschema:"The code of a table, from stats_search_tables: Social Statistics/Labour/Wages_Region.px."`
    Language string `json:"language,omitempty" jsonschema:"The language of the texts: ka or en. The default is en."`
}

type tableOutput struct {
    Source     string            `json:"source,omitempty"     jsonschema:"The source of the data: Geostat."`
    AsOf       string            `json:"as_of,omitempty"      jsonschema:"The instant at which the server read the table, RFC 3339."`
    Stale      bool              `json:"stale,omitempty"      jsonschema:"True when Geostat did not answer and the server returns a table it read before."`
    Code       string            `json:"code"                 jsonschema:"The code of the table."`
    Title      string            `json:"title"                jsonschema:"The title of the table. It often names the unit."`
    Folder     string            `json:"folder"               jsonschema:"The folder path of the table, from the subject down, as in the code. The same in both languages."`
    Updated    string            `json:"updated"              jsonschema:"The date on which Geostat last changed the table, YYYY-MM-DD."`
    Dimensions []dimensionOutput `json:"dimensions,omitempty" jsonschema:"The dimensions of the table, each with its values in the order of Geostat."`
}

type dimensionOutput struct {
    Code   string        `json:"code"             jsonschema:"The code of the dimension, for a selection."`
    Text   string        `json:"text"             jsonschema:"The name of the dimension."`
    Values []valueOutput `json:"values,omitempty" jsonschema:"The values of the dimension."`
}

type valueOutput struct {
    Code string `json:"code" jsonschema:"The code of the value, for a selection."`
    Text string `json:"text" jsonschema:"The text of the value."`
}

type getCellsInput struct {
    Code      string            `json:"code"                jsonschema:"The code of a table, from stats_search_tables."`
    Selection []selectionInput  `json:"selection,omitempty" jsonschema:"The values of each dimension. A dimension that this list omits takes every value."`
    Language  string            `json:"language,omitempty"  jsonschema:"The language of the texts: ka or en. The default is en."`
}

type selectionInput struct {
    Dimension string   `json:"dimension"        jsonschema:"The code of a dimension, from stats_get_table."`
    Values    []string `json:"values,omitempty" jsonschema:"The codes of the values. Give values or last, not both."`
    Last      int      `json:"last,omitempty"   jsonschema:"The count of the last values of the dimension, the newest for a time dimension."`
}

type cellsOutput struct {
    Source     string            `json:"source"     jsonschema:"The source of the data: Geostat."`
    AsOf       string            `json:"as_of"      jsonschema:"The instant at which the server read the figures, RFC 3339."`
    Stale      bool              `json:"stale"      jsonschema:"True when Geostat did not answer and the server returns figures it read before."`
    Table      tableOutput       `json:"table"      jsonschema:"The table, with no dimension values."`
    Dimensions []dimensionOutput `json:"dimensions" jsonschema:"The dimensions of each cell, in the order of the values of a cell."`
    Cells      []cellOutput      `json:"cells"      jsonschema:"The selected cells."`
}

type cellOutput struct {
    Values []string `json:"values" jsonschema:"The text of the value of the cell on each dimension, in the order of dimensions."`
    Figure *string  `json:"figure" jsonschema:"The figure as a decimal string, or null when Geostat did not publish it."`
}

type getInflationInput struct {
    Month    string `json:"month,omitempty"    jsonschema:"The month, YYYY-MM. The default is the newest published month."`
    Region   string `json:"region,omitempty"   jsonschema:"georgia, tbilisi, kutaisi, batumi, gori, telavi, or zugdidi. The default is georgia."`
    Group    string `json:"group,omitempty"    jsonschema:"The price group: total, food, alcohol_tobacco, clothing, housing, furnishings, health, transport, communication, recreation, education, restaurants, or miscellaneous. The default is total."`
    Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
}

type inflationOutput struct {
    Source  string        `json:"source"  jsonschema:"The source of the data: Geostat."`
    AsOf    string        `json:"as_of"   jsonschema:"The instant at which the server read the indices, RFC 3339."`
    Stale   bool          `json:"stale"   jsonschema:"True when Geostat did not answer and the server returns indices it read before."`
    Month   string        `json:"month"   jsonschema:"The month, YYYY-MM."`
    Region  namedOutput   `json:"region"  jsonschema:"The region."`
    Group   namedOutput   `json:"group"   jsonschema:"The price group."`
    Annual  *string       `json:"annual"  jsonschema:"The change of prices since the same month of the previous year, in percent. Null when not published."`
    Monthly *string       `json:"monthly" jsonschema:"The change of prices since the previous month, in percent. Null when not published."`
    Average *string       `json:"average" jsonschema:"The change of the average prices of the last 12 months over the 12 months before, in percent. Published for the group total only."`
    Core    *string       `json:"core"    jsonschema:"The core inflation, in percent. Published for georgia and the group total only."`
    Tables  []tableOutput `json:"tables"  jsonschema:"The tables of Geostat that the answer reads, with their update dates."`
}

type namedOutput struct {
    Code string `json:"code" jsonschema:"The code."`
    Name string `json:"name" jsonschema:"The name in the language of the request."`
}
```

`source` is `Geostat`. `as_of` is `ReadAt` in `kernel.TbilisiZone()`, written with
`time.RFC3339`. A `limit` of 0 asks the default. `updated` is the date of
`CatalogEntry.Updated`. Inside `cellsOutput` and `inflationOutput`, a `tableOutput`
leaves `source`, `as_of`, `stale`, and `dimensions` empty.

## 2. Errors

| Condition                                        | Sentinel                      | Tool message |
| ------------------------------------------------ | ----------------------------- | ------------ |
| A language other than `ka` and `en`              | `kernel.ErrUnknownLanguage`   | `language: "ru" is not a language of this tool. Use ka or en.` |
| An empty or long query                           | `app.ErrInvalidQuery`         | `query: give 1 to 100 characters.` |
| A limit out of range                             | none, the tool checks it      | `limit: 500 is out of range. Use 1 to 100.` |
| No catalog of the language yet                   | `app.ErrCatalogLoading`       | `The server is still reading the catalog of Geostat. Try again in a few minutes. stats_get_inflation answers now.` |
| A code that `ParseTableCode` refuses             | `domain.ErrInvalidTableCode`  | `code: "x" is not a table code. Call stats_search_tables for the codes.` |
| A code with no table                             | `app.ErrUnknownTable`         | `code: Geostat has no table "<code>". Call stats_search_tables for the codes.` |
| An unknown dimension                             | `app.ErrInvalidSelection`     | `selection: the table has no dimension "Town". Use Region, Year, Quarter.` |
| An unknown value                                 | `app.ErrInvalidSelection`     | `selection: the dimension "Quarter" has no value "9". Use 0, 1, 2, 3, 4.` With more than 20 values: `Call stats_get_table for its 1661 values.` |
| Values and last together, a last below 1, or a dimension twice | `app.ErrInvalidSelection` | `selection: give values or last for "Year", not both, and last from 1.` / `selection: the dimension "Year" appears twice.` |
| More than 1,000 cells                            | `app.ErrTooManyCells`         | `selection: the selection holds 25116 cells. Select 1000 or fewer: give values or last for more dimensions.` |
| A bad month                                      | `domain.ErrInvalidMonth`      | `month: give the month as YYYY-MM, for example 2026-10.` |
| A month with no annual inflation                 | `app.ErrMonthNotPublished`    | `month: Geostat published no consumer price index for 2026-11. The newest month is 2026-10.` |
| An unknown region or group                       | `domain.ErrUnknownRegion`, `domain.ErrUnknownPriceGroup` | `region: "rustavi" is not a region of the index. Use georgia, tbilisi, kutaisi, batumi, gori, telavi, or zugdidi.` The same form for `group`. |
| Geostat fails, with no stale value               | `kernel.ErrSourceUnavailable` | `Geostat did not answer. Try again in a few minutes.` |
| Any other error                                  | none                          | `The server failed to read the data of Geostat.` |

## Retired identifiers

This file has no retired identifier.
