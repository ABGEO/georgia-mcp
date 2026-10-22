import { Ajv2020 } from "ajv/dist/2020.js"

import {
  langs,
  type ContextContent,
  type Definition,
  type Localized,
  type Recording,
  type ToolContent,
  type Vendor,
} from "./schemas"

/** A tool example with its answer in each language and its recorded output. */
export type CatalogExample = Omit<ToolContent["examples"][number], "answer"> & {
  answer: Localized
  recordedAt: string
  output: unknown
}

/** A tool as the pages show it: the definition of the server joined with the content of the site. */
export type CatalogTool = Definition &
  Omit<ToolContent, "examples" | "summary" | "vendors"> & {
    summary: Record<(typeof langs)[number], string>
    examples: CatalogExample[]
    /** The vendors that the tool reads, in the order of the tool file. */
    vendors: Vendor[]
    /** True when the tool takes the code of a vendor: its description lists the vendors of the server. */
    takesVendorCode: boolean
  }

/** A context as the pages show it, with its tools. */
export type CatalogContext = ContextContent & { key: string; tools: CatalogTool[] }

/** The joined and checked data of every page. */
export type Catalog = {
  contexts: CatalogContext[]
  tools: CatalogTool[]
}

/** The error of a catalog that fails a check. It holds every problem, one line each. */
export class CatalogError extends Error {
  readonly problems: string[]

  constructor(problems: string[]) {
    super(`The tool data has ${problems.length} problem(s):\n${problems.map((p) => `  ${p}`).join("\n")}`)
    this.name = "CatalogError"
    this.problems = problems
  }
}

/** Joins the four data sets of the site, and throws a CatalogError when a check fails. */
export function buildCatalog(
  definitions: Record<string, Definition>,
  recordings: Record<string, Recording>,
  tools: Record<string, ToolContent>,
  contexts: Record<string, ContextContent>,
): Catalog {
  const problems: string[] = []
  const ajv = new Ajv2020({ strict: false, allErrors: true })
  const joined: CatalogTool[] = []

  for (const name of Object.keys(tools).toSorted()) {
    if (!(name in definitions)) problems.push(`${name}: no definition`)
  }

  for (const name of Object.keys(definitions).toSorted()) {
    const definition = definitions[name]
    const content = tools[name]
    if (!content) {
      problems.push(`${name}: no tool content`)
      continue
    }

    if (!(content.context in contexts)) problems.push(`${name}: unknown context ${content.context}`)
    for (const lang of langs) {
      if (!content.summary[lang]) problems.push(`${name}: no summary in ${lang}`)
    }
    if (content.examples.length === 0) problems.push(`${name}: no example`)

    const validate = ajv.compile(definition.outputSchema)
    const examples: CatalogExample[] = []
    for (const example of content.examples) {
      const key = `${name}.${example.id}`
      for (const lang of langs) {
        if (!example.answer[lang]) problems.push(`${key}: no answer in ${lang}`)
      }
      const recorded = recordings[key]
      if (!recorded) {
        problems.push(`${key}: no recording`)
        continue
      }

      if (!validate(recorded.output)) {
        for (const error of validate.errors ?? []) {
          problems.push(`${key}: output does not match the output schema at ${error.instancePath || "/"}: ${error.message}`)
        }
      }
      for (const field of ["source", "as_of"]) {
        if (!hasKey(recorded.output, field)) problems.push(`${key}: output has no ${field}`)
      }

      examples.push({
        ...example,
        answer: { en: example.answer.en ?? "", ka: example.answer.ka ?? "" },
        recordedAt: recorded.recordedAt,
        output: recorded.output,
      })
    }

    const vendors = toolVendors(name, content, contexts[content.context], problems)
    const listed = serverVendors(definition.description)
    if (listed) {
      for (const code of listed) {
        if (!content.vendors.includes(code)) problems.push(`${name}: the server lists ${code}`)
      }
      for (const code of content.vendors) {
        if (!listed.includes(code)) problems.push(`${name}: the server does not list ${code}`)
      }
    }

    joined.push({
      ...definition,
      ...content,
      summary: { en: content.summary.en ?? "", ka: content.summary.ka ?? "" },
      examples,
      vendors,
      takesVendorCode: listed !== undefined,
    })
  }

  const catalogContexts = Object.entries(contexts)
    .map(([key, ctx]) => ({ ...ctx, key, tools: joined.filter((tool) => tool.context === key) }))
    .toSorted((a, b) => a.order - b.order)

  for (const ctx of catalogContexts) {
    if (ctx.tools.length === 0) problems.push(`${ctx.key}: no tool`)
  }

  if (problems.length > 0) throw new CatalogError(problems)

  return { contexts: catalogContexts, tools: catalogContexts.flatMap((ctx) => ctx.tools) }
}

/** Gives the vendors of a tool from its context, and a problem for each code that the context lacks. */
function toolVendors(name: string, content: ToolContent, context: ContextContent | undefined, problems: string[]): Vendor[] {
  if (content.vendors.length === 0) problems.push(`${name}: no vendor`)

  const vendors: Vendor[] = []
  for (const code of content.vendors) {
    const vendor = context?.vendors.find((candidate) => candidate.code === code)
    if (vendor) vendors.push(vendor)
    else problems.push(`${name}: unknown vendor ${code}`)
  }

  return vendors
}

/**
 * Reads the codes of the list "The banks of this server: bog (Bank of Georgia), tbc (TBC Bank)." in a description.
 * A description with no such list gives undefined.
 */
function serverVendors(description: string): string[] | undefined {
  const start = description.indexOf("of this server:")
  if (start < 0) return undefined

  const rest = description.slice(start + "of this server:".length)
  const end = rest.indexOf(". ")
  const list = end < 0 ? rest : rest.slice(0, end)

  return [...list.matchAll(/([a-z][a-z0-9_]*) \(/g)].map((match) => match[1])
}

function hasKey(value: unknown, key: string): boolean {
  if (Array.isArray(value)) return value.length > 0 && value.every((item) => hasKey(item, key))
  if (value === null || typeof value !== "object") return false
  if (key in value) return true

  const children = Object.values(value).filter((child) => child !== null && typeof child === "object")
  return children.some((child) => hasKey(child, key))
}

function byId<T>(entries: Array<{ id: string; data: T }>): Record<string, T> {
  return Object.fromEntries(entries.map((entry) => [entry.id, entry.data]))
}

let cached: Catalog | undefined

/** Reads the four collections and builds the catalog one time for the whole build. */
export async function getCatalog(): Promise<Catalog> {
  if (cached) return cached

  const { getCollection } = await import("astro:content")

  cached = buildCatalog(
    byId(await getCollection("definitions")),
    byId(await getCollection("recordings")),
    byId(await getCollection("tools")),
    byId(await getCollection("contexts")),
  )

  return cached
}

/** Gives the static paths of the tool pages: one for each tool, with the tool and its context. */
export async function toolPaths() {
  const catalog = await getCatalog()

  return catalog.contexts.flatMap((context) =>
    context.tools.map((tool) => ({ params: { name: tool.name }, props: { tool, context } })),
  )
}
