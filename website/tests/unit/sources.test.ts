import { describe, expect, test } from "vitest"

import { buildCatalog, CatalogError } from "../../src/lib/catalog"
import { contextContentSchema } from "../../src/lib/schemas"
import { sourceGroups } from "../../src/lib/sources"
import { content, contextContent, definition, recording, valid } from "./fixtures"

describe("sourceGroups", () => {
  // SOURCES-SC-001: A group for each context, a source for each vendor, with the tools that read it.
  test("groups the sources by context with their tools", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.contexts.fuel = contextContent(2, 2)
    data.tools.fuel_get_prices.vendors = ["vendor0"]
    data.definitions.fuel_list_stations = definition("fuel_list_stations")
    data.tools.fuel_list_stations = { ...content("fuel"), vendors: ["vendor1"] }
    data.recordings["fuel_list_stations.today"] = recording("fuel_list_stations")
    data.definitions.rates_get_official_rates = definition("rates_get_official_rates")
    data.tools.rates_get_official_rates = content("rates")
    data.recordings["rates_get_official_rates.today"] = recording("rates_get_official_rates")
    data.contexts.rates = contextContent(1, 1)
    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)

    const groups = sourceGroups(catalog)

    expect(groups.map((group) => group.context.key)).toEqual(["rates", "fuel"])
    expect(groups[0].sources.map((source) => source.vendor.code)).toEqual(["vendor0"])
    expect(groups[1].sources.map((source) => [source.vendor.code, source.tools.map((tool) => tool.name)])).toEqual([
      ["vendor0", ["fuel_get_prices"]],
      ["vendor1", ["fuel_list_stations"]],
    ])
  })
})

describe("the sources data", () => {
  // SOURCES-SC-002: A vendor that no tool reads fails the build.
  test("refuses a vendor that no tool reads", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.contexts.fuel = contextContent(1, 2)

    let problems: string[] = []
    try {
      buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)
    } catch (error) {
      if (error instanceof CatalogError) problems = error.problems
    }

    expect(problems).toContain("fuel.vendor1: no tool reads the vendor")
  })

  // SOURCES-SC-002: A vendor with the address of an API, or with no kind, fails the schema.
  test.each([
    ["the address of an API", { url: "https://example.ge/api/rates" }],
    ["an address with no TLS", { url: "http://example.ge/rates" }],
    ["no kind", { kind: undefined }],
  ])("refuses a vendor with %s", (_, change) => {
    const context = contextContent(1, 1)
    context.vendors[0] = { ...context.vendors[0], ...change } as (typeof context.vendors)[number]

    expect(contextContentSchema.safeParse(context).success).toBe(false)
  })
})
