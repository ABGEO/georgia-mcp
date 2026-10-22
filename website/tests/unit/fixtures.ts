import type { ContextContent, Definition, Recording, ToolContent } from "../../src/lib/schemas"

export const outputSchema = {
  type: "object",
  properties: {
    price: { type: "string" },
    source: { type: "string" },
    as_of: { type: "string" },
  },
  required: ["price", "source", "as_of"],
}

export function definition(name: string): Definition {
  return {
    name,
    title: `Title of ${name}`,
    description: `Description of ${name}`,
    inputSchema: { type: "object", properties: {} },
    outputSchema,
    annotations: { readOnlyHint: true },
  }
}

export function content(context: ToolContent["context"]): ToolContent {
  return {
    context,
    summary: { en: "An English summary.", ka: "ქართული შეჯამება." },
    vendors: ["vendor0"],
    examples: [
      {
        id: "today",
        question: { en: "What is it today?", ka: "რა არის დღეს?" },
        answer: { en: "It is 4.65 lari.", ka: "4.65 ლარია." },
        input: {},
      },
    ],
  }
}

export function recording(tool: string, output: unknown = { price: "4.65", source: "Wissol", as_of: "2026-10-20T10:00:00+04:00" }): Recording {
  return { tool, example: "today", recordedAt: "2026-10-20T10:00:00+04:00", output }
}

export function contextContent(order: number, vendors = 1): ContextContent {
  return {
    order,
    hue: "peach",
    icon: "fuel",
    name: { en: `Context ${order}`, ka: `კონტექსტი ${order}` },
    vendors: Array.from({ length: vendors }, (_, i) => ({
      code: `vendor${i}`,
      en: `Vendor ${i}`,
      ka: `მომწოდებელი ${i}`,
    })),
  }
}

/** One valid tool of one context, as the four inputs of buildCatalog. */
export function valid(name = "fuel_get_prices", ctx: ToolContent["context"] = "fuel") {
  return {
    definitions: { [name]: definition(name) },
    recordings: { [`${name}.today`]: recording(name) },
    tools: { [name]: content(ctx) },
    contexts: { [ctx]: contextContent(1) } as Record<string, ContextContent>,
  }
}
