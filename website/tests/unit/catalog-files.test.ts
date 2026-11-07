import { describe, expect, test } from "vitest"

import { buildCatalog } from "../../src/lib/catalog"
import { llmsText, toolMarkdown } from "../../src/lib/catalog-files"
import { REPO_URL, SERVER_URL, SITE_SUMMARY, SITE_URL } from "../../src/site"
import { content, definition, recording, valid } from "./fixtures"

describe("llmsText", () => {
  // LLMSTXT-SC-001: The file names the server, links the guides, and lists each tool on one line.
  test("writes the header, the links, and one line for each tool", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.definitions.fuel_get_prices.description = "Returns the prices.\nData from the companies."
    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)

    const text = llmsText(catalog)
    const lines = text.split("\n")

    expect(lines[0]).toBe("# Georgia MCP")
    expect(lines).toContain(`> ${SITE_SUMMARY}`)
    expect(lines.indexOf(`> ${SITE_SUMMARY}`)).toBeLessThan(lines.findIndex((l) => l.startsWith("## ")))
    expect(text).toContain(SERVER_URL)
    expect(text).toContain(`](${SITE_URL}/#install)`)
    expect(text).toContain(`](${REPO_URL})`)
    expect(lines).toContain(
      "- [Title of fuel_get_prices](https://geo-mcp.abgeo.cloud/tools/fuel_get_prices.md): `fuel_get_prices`. Returns the prices. Data from the companies.",
    )
  })
})

describe("toolMarkdown", () => {
  // LLMSTXT-SC-003: A copy holds the page address, the server, the inputs, and the output of the example.
  test("writes the inputs and the example of a tool", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.definitions.fuel_get_prices.inputSchema = {
      type: "object",
      properties: { fuel: { type: "string", description: "The grade of the fuel.", default: "petrol" } },
    }
    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)
    const [context] = catalog.contexts

    const text = toolMarkdown(catalog.tools[0], context)

    expect(text.startsWith("# fuel_get_prices\n")).toBe(true)
    expect(text).toContain(`${SITE_URL}/tools/fuel_get_prices/`)
    expect(text).toContain(SERVER_URL)
    expect(text).toContain("- `fuel` (string, optional, default `petrol`): The grade of the fuel.")
    expect(text).toContain('"source": "Wissol"')
    expect(text).toContain('"as_of": "2026-10-20T10:00:00+04:00"')
  })

  // LLMSTXT-SC-003: A tool with no input says so.
  test("says that a tool takes no input", () => {
    const data = valid("fuel_get_prices", "fuel")
    data.definitions.air_get_quality = definition("air_get_quality")
    data.tools.air_get_quality = content("fuel")
    data.recordings["air_get_quality.today"] = recording("air_get_quality")
    const catalog = buildCatalog(data.definitions, data.recordings, data.tools, data.contexts)
    const [context] = catalog.contexts
    const tool = catalog.tools.find((t) => t.name === "air_get_quality")!

    expect(toolMarkdown(tool, context)).toContain("The tool takes no input.")
  })
})
