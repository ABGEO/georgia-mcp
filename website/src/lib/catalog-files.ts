import { REPO_URL, SERVER_URL, SITE_NAME, SITE_SUMMARY, SITE_URL } from "../site"
import type { Catalog, CatalogContext, CatalogTool } from "./catalog"
import { inputRows } from "./inputs"
import type { Definition } from "./schemas"

/** The object of /tools.json. */
export type ToolsJson = {
  name: "georgia-mcp"
  title: string
  url: string
  tools: Array<Definition & { page: string; markdown: string }>
}

/** The address of the tool page of a tool, in English. */
export function toolPageUrl(name: string): string {
  return `${SITE_URL}/tools/${name}/`
}

/** The address of the Markdown copy of a tool. */
export function toolMarkdownUrl(name: string): string {
  return `${SITE_URL}/tools/${name}.md`
}

function oneLine(text: string): string {
  return text.replace(/\s+/g, " ").trim()
}

function json(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

/** The text of /llms.txt. */
export function llmsText(catalog: Catalog): string {
  return [
    `# ${SITE_NAME}`,
    "",
    `> ${SITE_SUMMARY}`,
    "",
    `Connect a client to ${SERVER_URL}, over streamable HTTP, with no sign-in.`,
    "",
    "## Install",
    "",
    `- [Install guides](${SITE_URL}/#install): the steps for each client, and for a local server.`,
    `- [Repository](${REPO_URL}): the source, the releases, and the container image.`,
    "",
    "## Tools",
    "",
    ...catalog.tools.map(
      (tool) => `- [${tool.title}](${toolMarkdownUrl(tool.name)}): \`${tool.name}\`. ${oneLine(tool.description)}`,
    ),
    "",
  ].join("\n")
}

function inputLines(tool: CatalogTool): string[] {
  const rows = inputRows(tool.inputSchema)
  if (rows.length === 0) return ["The tool takes no input."]

  return rows.map((row) => {
    const parts = [row.type, row.required ? "required" : "optional"]
    if (row.default !== undefined) parts.push(`default \`${String(row.default)}\``)

    return `- \`${row.name}\` (${parts.join(", ")}): ${oneLine(row.description)}`
  })
}

/** The Markdown copy of one English tool page. */
export function toolMarkdown(tool: CatalogTool, context: CatalogContext): string {
  return [
    `# ${tool.name}`,
    "",
    `> ${oneLine(tool.summary.en)}`,
    "",
    `- Title: ${tool.title}`,
    `- Context: ${context.name.en}`,
    `- Sources: ${tool.vendors.map((vendor) => vendor.en).join(", ")}`,
    `- Page: ${toolPageUrl(tool.name)}`,
    `- Server: ${SERVER_URL}`,
    "",
    "## Description",
    "",
    oneLine(tool.description),
    "",
    "## Input",
    "",
    ...inputLines(tool),
    "",
    "## Examples",
    ...tool.examples.flatMap((example) => [
      "",
      `### ${oneLine(example.question.en)}`,
      "",
      "Input:",
      "",
      "```json",
      json(example.input),
      "```",
      "",
      `Output, recorded at ${example.recordedAt}:`,
      "",
      "```json",
      json(example.output),
      "```",
    ]),
    "",
  ].join("\n")
}

/** The object of /tools.json. */
export function toolsJson(catalog: Catalog): ToolsJson {
  return {
    name: "georgia-mcp",
    title: SITE_NAME,
    url: SERVER_URL,
    tools: catalog.tools.map((tool) => ({
      name: tool.name,
      title: tool.title,
      description: tool.description,
      inputSchema: tool.inputSchema,
      outputSchema: tool.outputSchema,
      annotations: tool.annotations,
      page: toolPageUrl(tool.name),
      markdown: toolMarkdownUrl(tool.name),
    })),
  }
}
