/** One input of a tool, as the input table shows it. */
export type InputRow = {
  name: string
  type: string
  description: string
  required: boolean
  default: unknown
}

type Property = {
  type?: string | string[]
  items?: Property
  enum?: unknown[]
  description?: string
  default?: unknown
}

/** Gives one row for each property of the input schema of a tool, in the order of the schema. */
export function inputRows(schema: Record<string, unknown>): InputRow[] {
  const properties = (schema.properties ?? {}) as Record<string, Property>
  const required = new Set((schema.required ?? []) as string[])

  return Object.entries(properties).map(([name, property]) => ({
    name,
    type: typeName(property),
    description: property.description ?? "",
    required: required.has(name),
    default: property.default,
  }))
}

function typeName(property: Property): string {
  if (property.enum) return property.enum.map(String).join(" | ")

  // The server types an optional list as ["null", "array"]. A reader passes no null, so the table drops it.
  const types = (Array.isArray(property.type) ? property.type : [property.type ?? "unknown"]).filter((t) => t !== "null")

  return types.map((type) => (type === "array" ? `${property.items ? typeName(property.items) : "unknown"}[]` : type)).join(" | ")
}
