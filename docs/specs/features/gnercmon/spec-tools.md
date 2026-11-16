---
id: GNERCMON
title: The monthly electricity balance and prices of GNERC, the tools
type: spec
status: approved
created: 2026-11-17
updated: 2026-11-17
context: energy
approved_by: Temuri
approved_on: 2026-11-17
constrained_by: [MCP]
requirements: features/gnercmon/requirements.md
---

# Specification: The monthly electricity balance and prices of GNERC, the tools

## 1. Tools

`spec.md` 4.5 gives the errors of both tools.

`energy_get_balance`. Title: `Get the electricity balance of Georgia`. Description:

> Returns the monthly electricity balance of Georgia from the Georgian National Energy and
> Water Supply Regulatory Commission (GNERC): the generation by type of power plant, the
> import and the export by country, the supply, the consumption by group, the losses, the
> domestic consumption, and the transit, in million kWh, for one month from 2022-01, or the
> total of one year from 2022. GNERC publishes a month about three weeks after it ends.
> With no input, it returns the newest published month. A year that is not over holds only
> its published months: see months. Data from GNERC.

`energy_list_prices`. Title: `List the electricity prices of Georgia`. Description:

> Returns three monthly electricity prices of Georgia from the Georgian National Energy
> and Water Supply Regulatory Commission (GNERC), in tetri per kWh: the weighted average
> price of balancing electricity, the guaranteed capacity fee, and the price of the
> deregulated power plants. Give a period of at most 60 months from 2015-01. With no
> input, it returns the last 12 published months. Data from GNERC.

```go
type getBalanceInput struct {
    Month string `json:"month,omitempty" jsonschema:"A month from 2022-01, as YYYY-MM, for example 2026-03. Leave month and year empty for the newest published month."`
    Year  int    `json:"year,omitempty"  jsonschema:"A year from 2022, for the total of the year. Do not give it with month."`
}

type balanceOutput struct {
    Source              string            `json:"source"               jsonschema:"The source of the data: Georgian National Energy and Water Supply Regulatory Commission (GNERC)."`
    AsOf                string            `json:"as_of"                jsonschema:"The instant at which GNERC last changed the workbook, or at which the server read it, RFC 3339."`
    Stale               bool              `json:"stale"                jsonschema:"True when GNERC did not answer and the server returns data it read before."`
    Period              string            `json:"period"               jsonschema:"The month, YYYY-MM, or the year, YYYY, of the balance."`
    Months              int               `json:"months"               jsonschema:"The count of published months that the period holds: 1 for a month, up to 12 for a year."`
    NewestMonth         string            `json:"newest_month"         jsonschema:"The newest month that GNERC published, YYYY-MM."`
    Unit                string            `json:"unit"                 jsonschema:"The unit of each value: million kWh."`
    Generation          generationBalance `json:"generation"           jsonschema:"The generation of the power plants."`
    StationOwnUse       string            `json:"station_own_use,omitempty" jsonschema:"The electricity that the power plants use or lose before the grid."`
    Import              countryBalance    `json:"import"               jsonschema:"The import, by country of origin."`
    Supply              string            `json:"supply"               jsonschema:"The generation less the station own use, plus the import."`
    Consumption         consumptionBalance `json:"consumption"         jsonschema:"The consumption, which GNERC counts with the export and the losses. It equals the supply."`
    Export              countryBalance    `json:"export"               jsonschema:"The export, by country of destination."`
    Losses              *lossesBalance    `json:"losses,omitempty"     jsonschema:"The electricity that the grid operators bought to cover the losses of their grids."`
    DomesticConsumption string            `json:"domestic_consumption,omitempty" jsonschema:"The generation plus the import, less the export."`
    Transit             *transitBalance   `json:"transit,omitempty"    jsonschema:"The electricity that crossed Georgia to Turkey."`
}

type generationBalance struct {
    Total   string `json:"total"             jsonschema:"The total generation."`
    Thermal string `json:"thermal,omitempty" jsonschema:"The thermal power plants."`
    Wind    string `json:"wind,omitempty"    jsonschema:"The wind power plants."`
    Solar   string `json:"solar,omitempty"   jsonschema:"The solar power plants. Absent before 2025."`
    Hydro   string `json:"hydro,omitempty"   jsonschema:"The hydro power plants."`
}

type countryBalance struct {
    Total        string `json:"total"                    jsonschema:"The total."`
    Russia       string `json:"russia,omitempty"         jsonschema:"Russia."`
    Turkey       string `json:"turkey,omitempty"         jsonschema:"Turkey."`
    Azerbaijan   string `json:"azerbaijan,omitempty"     jsonschema:"Azerbaijan."`
    Armenia      string `json:"armenia,omitempty"        jsonschema:"Armenia."`
    IntoAbkhazia string `json:"into_abkhazia,omitempty"  jsonschema:"The part of the import that went into Abkhazia. Import only."`
}

type consumptionBalance struct {
    Total           string            `json:"total"                      jsonschema:"The total consumption."`
    Abkhazia        string            `json:"abkhazia,omitempty"         jsonschema:"Abkhazia."`
    Suppliers       *suppliersBalance `json:"suppliers,omitempty"        jsonschema:"The suppliers of electricity."`
    Direct          string            `json:"direct,omitempty"           jsonschema:"The consumers connected to the transmission grid."`
    StoppedStations string            `json:"stopped_stations,omitempty" jsonschema:"The own use of the power plants while they stood still."`
}

type suppliersBalance struct {
    Total           string `json:"total,omitempty"            jsonschema:"Every supplier."`
    UniversalService string `json:"universal_service,omitempty" jsonschema:"The suppliers of the universal service."`
    PublicService   string `json:"public_service,omitempty"   jsonschema:"The suppliers of the public service."`
    FreeMarket      string `json:"free_market,omitempty"      jsonschema:"The suppliers of the free market. Absent before 2024."`
}

type lossesBalance struct {
    Total        string `json:"total,omitempty"        jsonschema:"Every grid."`
    Transmission string `json:"transmission,omitempty" jsonschema:"The transmission grid."`
    Distribution string `json:"distribution,omitempty" jsonschema:"The distribution grids."`
}

type transitBalance struct {
    Total              string `json:"total,omitempty"                jsonschema:"Every route."`
    AzerbaijanToTurkey string `json:"azerbaijan_to_turkey,omitempty" jsonschema:"From Azerbaijan to Turkey."`
    RussiaToTurkey     string `json:"russia_to_turkey,omitempty"     jsonschema:"From Russia to Turkey."`
    ArmeniaToTurkey    string `json:"armenia_to_turkey,omitempty"    jsonschema:"From Armenia to Turkey."`
}

type listPricesInput struct {
    From string `json:"from,omitempty" jsonschema:"The first month, YYYY-MM, from 2015-01. Leave it empty for 11 months before to."`
    To   string `json:"to,omitempty"   jsonschema:"The last month, YYYY-MM. Leave it empty for the newest published month."`
}

type pricesOutput struct {
    Source      string              `json:"source"       jsonschema:"The source of the data: Georgian National Energy and Water Supply Regulatory Commission (GNERC)."`
    AsOf        string              `json:"as_of"        jsonschema:"The instant at which GNERC last changed a price workbook, or at which the server read them, RFC 3339."`
    Stale       bool                `json:"stale"        jsonschema:"True when GNERC did not answer and the server returns data it read before."`
    NewestMonth string              `json:"newest_month" jsonschema:"The newest month that GNERC published, YYYY-MM."`
    Unit        string              `json:"unit"         jsonschema:"The unit of each price: tetri per kWh. 100 tetri are 1 lari."`
    Months      []monthPricesOutput `json:"months"       jsonschema:"Each published month of the period, the oldest first."`
}

type monthPricesOutput struct {
    Month       string `json:"month"                  jsonschema:"The month, YYYY-MM."`
    Balancing   string `json:"balancing,omitempty"    jsonschema:"The weighted average price of the balancing electricity sold in the month."`
    CapacityFee string `json:"capacity_fee,omitempty" jsonschema:"The guaranteed capacity fee for each kWh."`
    Deregulated string `json:"deregulated,omitempty"  jsonschema:"The price of the electricity bought from deregulated power plants to set the balancing price."`
}
```

Each value is written with `StringFixed(Precision)`. A group pointer is nil when the
figures hold none of its items. `source` is
`Georgian National Energy and Water Supply Regulatory Commission (GNERC)`. `as_of` is
`Modified`, or `ReadAt` when `Modified` is zero, in `kernel.TbilisiZone()`, written with
`time.RFC3339`. `unit` is `million kWh` or `tetri per kWh`.

## Retired identifiers

This file has no retired identifier.
