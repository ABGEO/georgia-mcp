import { describe, expect, test } from "vitest"

import { inputRows } from "../../src/lib/inputs"

describe("inputRows", () => {
  // WEBSITE-SC-013: One row for each property, and none for an empty schema.
  test("gives one row for each property of the input schema", () => {
    const schema = {
      type: "object",
      properties: {
        currencies: {
          type: "array",
          items: { type: "string" },
          description: "The ISO 4217 codes of the currencies.",
        },
        date: { type: "string", description: "The date of the rate.", default: "today" },
      },
      required: ["currencies"],
    }

    expect(inputRows(schema)).toEqual([
      {
        name: "currencies",
        type: "string[]",
        description: "The ISO 4217 codes of the currencies.",
        required: true,
        default: undefined,
      },
      { name: "date", type: "string", description: "The date of the rate.", required: false, default: "today" },
    ])
    expect(inputRows({ type: "object", properties: {} })).toEqual([])
    expect(inputRows({ type: "object" })).toEqual([])
  })

  // WEBSITE-SC-014: An array shows its item type, and an enum shows its values.
  test("names an array by its item type and an enum by its values", () => {
    const schema = {
      type: "object",
      properties: {
        codes: { type: "array", items: { type: "string" } },
        company: { type: "string", enum: ["wissol", "socar"] },
        grades: { type: ["null", "array"], items: { type: "string" } },
        direction: { type: ["null", "integer"] },
      },
    }

    expect(inputRows(schema).map((row) => row.type)).toEqual(["string[]", "wissol | socar", "string[]", "integer"])
  })
})
