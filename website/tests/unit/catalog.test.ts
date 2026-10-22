import { describe, expect, test } from "vitest"

import { buildCatalog, CatalogError } from "../../src/lib/catalog"
import { content, contextContent, definition, recording, valid } from "./fixtures"

function problems(run: () => unknown): string[] {
  try {
    run()
  } catch (error) {
    if (error instanceof CatalogError) return error.problems
    throw error
  }
  throw new Error("buildCatalog did not throw")
}

describe("buildCatalog", () => {
  // WEBSITE-SC-005: A context holds its vendors and its tools.
  test("joins each context with its vendors and its tools", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.contexts.fuel = contextContent(2, 3)
    data.definitions.rates_get_official_rates = definition("rates_get_official_rates")
    data.tools.rates_get_official_rates = content("rates")
    data.recordings["rates_get_official_rates.today"] = recording("rates_get_official_rates")
    data.contexts.rates = contextContent(1, 1)

    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)

    expect(catalog.contexts.map((c) => c.key)).toEqual(["rates", "fuel"])
    const fuel = catalog.contexts[1]
    expect(fuel.vendors).toHaveLength(3)
    expect(fuel.tools.map((t) => t.name)).toEqual(["fuel_get_prices"])
    expect(catalog.contexts[0].vendors).toHaveLength(1)
    expect(catalog.tools.map((t) => t.name)).toEqual(["rates_get_official_rates", "fuel_get_prices"])
    expect(fuel.tools[0].examples[0]).toMatchObject({ id: "today", recordedAt: "2026-10-20T10:00:00+04:00" })
  })

  // WEBSITE-SC-017: A definition with no tool content fails the build.
  test("refuses a definition with no tool content", () => {
    const data = valid()
    data.definitions.transit_list_vehicles = definition("transit_list_vehicles")

    expect(problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))).toContain(
      "transit_list_vehicles: no tool content",
    )
  })

  // WEBSITE-SC-018: A tool content file with no definition fails the build.
  test("refuses a tool content file with no definition", () => {
    const data = valid()
    data.tools.fuel_get_price = content("fuel")

    expect(problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))).toContain(
      "fuel_get_price: no definition",
    )
  })

  // WEBSITE-SC-019: A missing summary and a missing recording each give a problem.
  test("refuses a tool with no Georgian summary, and an example with no recording", () => {
    const data = valid()
    data.tools.fuel_get_prices.summary = { en: "An English summary." }
    data.definitions.rates_get_official_rates = definition("rates_get_official_rates")
    data.tools.rates_get_official_rates = content("rates")
    data.contexts.rates = contextContent(2, 1)

    const found = problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))

    expect(found).toContain("fuel_get_prices: no summary in ka")
    expect(found).toContain("rates_get_official_rates.today: no recording")
  })

  // WEBSITE-SC-019: A tool with no example fails the build.
  test("refuses a tool with no example", () => {
    const data = valid()
    data.tools.fuel_get_prices.examples = []

    expect(problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))).toContain(
      "fuel_get_prices: no example",
    )
  })

  // WEBSITE-SC-020: An output that breaks the output schema names the path of the value.
  test("refuses an output that does not match the output schema", () => {
    const data = valid()
    data.recordings["fuel_get_prices.today"] = recording("fuel_get_prices", {
      price: 4.65,
      source: "Wissol",
      as_of: "2026-10-20T10:00:00+04:00",
    })

    const found = problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))

    expect(found).toHaveLength(1)
    expect(found[0]).toMatch(/^fuel_get_prices\.today: output does not match the output schema at \/price: /)
  })

  // WEBSITE-SC-021: source and as_of may sit at any depth, and both must exist.
  test("accepts source and as_of inside each item, and refuses an output with no as_of", () => {
    const listSchema = {
      type: "object",
      properties: {
        items: {
          type: "array",
          items: { type: "object", properties: { source: { type: "string" }, as_of: { type: "string" } } },
        },
      },
    }
    const data = valid()
    data.definitions.fuel_get_prices.outputSchema = listSchema
    data.recordings["fuel_get_prices.today"] = recording("fuel_get_prices", {
      items: [{ source: "Wissol", as_of: "2026-10-20T10:00:00+04:00" }],
    })
    expect(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)).not.toThrow()

    data.recordings["fuel_get_prices.today"] = recording("fuel_get_prices", { items: [{ source: "Wissol" }] })
    expect(problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))).toContain(
      "fuel_get_prices.today: output has no as_of",
    )
  })

  // WEBSITE-SC-034: Each example needs an answer in each language, and each context needs a tool.
  test("refuses an example with no answer in a language, and a context with no tool", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.tools.fuel_get_prices.examples[0].answer = { en: "It is 4.65 lari." }
    data.definitions.rates_get_official_rates = definition("rates_get_official_rates")
    data.tools.rates_get_official_rates = content("rates")
    data.tools.rates_get_official_rates.examples[0].answer = { ka: "2.59 ლარია." }
    data.recordings["rates_get_official_rates.today"] = recording("rates_get_official_rates")
    data.contexts.rates = contextContent(2)
    data.contexts.transit = contextContent(3)

    const found = problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))

    expect(found).toContain("fuel_get_prices.today: no answer in ka")
    expect(found).toContain("rates_get_official_rates.today: no answer in en")
    expect(found).toContain("transit: no tool")
  })

  // WEBSITE-SC-017: A tool of an unknown context fails the build.
  test("refuses a tool whose context is not in contexts.yaml", () => {
    const data = valid()
    data.contexts = {}

    expect(problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))).toContain(
      "fuel_get_prices: unknown context fuel",
    )
  })

  // WEBSITE-SC-043: A tool holds the vendors of its file, in its order, and takes their codes when the server lists them.
  test("gives each tool its vendors, and whether it takes their codes", () => {
    const data = valid("rates_get_commercial_rates", "rates")
    data.contexts.rates.vendors = [
      { code: "tbc", en: "TBC Bank", ka: "თიბისი ბანკი" },
      { code: "bog", en: "Bank of Georgia", ka: "საქართველოს ბანკი" },
    ]
    data.tools.rates_get_commercial_rates.vendors = ["bog", "tbc"]
    data.definitions.rates_get_commercial_rates.description =
      "Returns the rates. The banks of this server: bog (Bank of Georgia), tbc (TBC Bank). The rates are of today."
    data.definitions.rates_get_official_rates = definition("rates_get_official_rates")
    data.tools.rates_get_official_rates = { ...content("rates"), vendors: ["bog", "tbc"] }
    data.recordings["rates_get_official_rates.today"] = recording("rates_get_official_rates")

    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)
    const byName = Object.fromEntries(catalog.tools.map((tool) => [tool.name, tool]))

    expect(byName.rates_get_commercial_rates.vendors).toEqual([
      { code: "bog", en: "Bank of Georgia", ka: "საქართველოს ბანკი" },
      { code: "tbc", en: "TBC Bank", ka: "თიბისი ბანკი" },
    ])
    expect(byName.rates_get_commercial_rates.takesVendorCode).toBe(true)
    expect(byName.rates_get_official_rates.takesVendorCode).toBe(false)
  })

  // WEBSITE-SC-044: The vendors of a tool must exist in its context, and equal the vendors that the server lists.
  test("refuses vendors that do not match the context or the server", () => {
    const data = valid("rates_get_commercial_rates", "rates")
    data.contexts.rates.vendors = [
      { code: "tbc", en: "TBC Bank", ka: "თიბისი ბანკი" },
      { code: "kursi", en: "Kursi.ge", ka: "Kursi.ge" },
    ]
    data.tools.rates_get_commercial_rates.vendors = ["tbc", "kursi", "hydro"]
    data.definitions.rates_get_commercial_rates.description =
      "The banks of this server: tbc (TBC Bank), terabank (Terabank). The rates are of today."
    data.definitions.offers_search_offers = definition("offers_search_offers")
    data.tools.offers_search_offers = { ...content("rates"), vendors: [] }
    data.recordings["offers_search_offers.today"] = recording("offers_search_offers")

    const found = problems(() => buildCatalog(data.definitions, data.recordings, data.tools, data.contexts))

    expect(found).toContain("rates_get_commercial_rates: the server lists terabank")
    expect(found).toContain("rates_get_commercial_rates: the server does not list kursi")
    expect(found).toContain("rates_get_commercial_rates: unknown vendor hydro")
    expect(found).toContain("offers_search_offers: no vendor")
  })
})
