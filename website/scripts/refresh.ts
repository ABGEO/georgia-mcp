import { readdir, readFile, writeFile } from "node:fs/promises"
import { join, resolve } from "node:path"

import { Client } from "@modelcontextprotocol/sdk/client/index.js"
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js"
import { parse } from "yaml"

const website = resolve(import.meta.dirname, "..")
const repository = resolve(website, "..")
const toolsDir = join(website, "src/content/tools")
const definitionsFile = join(website, "src/data/definitions.json")
const recordingsFile = join(website, "src/data/recordings.json")

type Example = { id: string; input: Record<string, unknown> }

// The server reads the catalog of Geostat in the background after its start. A search
// before the first read ends gets this message. GEOSTAT-NFR-007 bounds the read at 15 minutes.
const catalogLoading = "The server is still reading the catalog"
const loadingRetry = 10_000
const loadingLimit = 15 * 60_000

async function readExamples(): Promise<Array<{ tool: string; example: Example }>> {
  const files = (await readdir(toolsDir)).filter((file) => file.endsWith(".yaml")).toSorted()
  const examples: Array<{ tool: string; example: Example }> = []
  for (const file of files) {
    const content = parse(await readFile(join(toolsDir, file), "utf8")) as { examples?: Example[] }
    for (const example of content.examples ?? []) {
      examples.push({ tool: file.replace(/\.yaml$/, ""), example })
    }
  }

  return examples
}

function sortedJSON(entries: Record<string, unknown>): string {
  const sorted = Object.fromEntries(Object.entries(entries).toSorted(([a], [b]) => a.localeCompare(b)))

  return `${JSON.stringify(sorted, null, 2)}\n`
}

function errorText(content: unknown): string {
  if (!Array.isArray(content)) return "no text"

  return content
    .filter((part): part is { type: "text"; text: string } => part?.type === "text")
    .map((part) => part.text)
    .join(" ")
}

type CallResult = Awaited<ReturnType<Client["callTool"]>>

// callWhenReady sends a call again while the server still reads a catalog.
async function callWhenReady(client: Client, name: string, input: Record<string, unknown>): Promise<CallResult> {
  const deadline = Date.now() + loadingLimit
  for (;;) {
    const result = await client.callTool({ name, arguments: input })
    if (!result.isError || !errorText(result.content).startsWith(catalogLoading) || Date.now() >= deadline) {
      return result
    }

    console.log(`waiting for the catalog: ${name}`)
    await new Promise((done) => setTimeout(done, loadingRetry))
  }
}

async function main(): Promise<void> {
  const examples = await readExamples()
  // The server reads the keys from the .env file of the repository root, so it starts there.
  const transport = new StdioClientTransport({
    command: "go",
    args: ["run", "./cmd/georgia-mcp", "stdio"],
    cwd: repository,
    env: process.env as Record<string, string>,
    stderr: "inherit",
  })
  const client = new Client({ name: "georgia-mcp-website-refresh", version: "1.0.0" })
  await client.connect(transport)

  try {
    const definitions: Record<string, unknown> = {}
    let cursor: string | undefined
    do {
      const page = await client.listTools({ cursor })
      for (const tool of page.tools) {
        definitions[tool.name] = {
          name: tool.name,
          title: tool.title ?? tool.annotations?.title ?? tool.name,
          description: tool.description ?? "",
          inputSchema: tool.inputSchema,
          outputSchema: tool.outputSchema ?? {},
          annotations: tool.annotations ?? {},
        }
      }
      cursor = page.nextCursor
    } while (cursor)

    const recordings: Record<string, unknown> = {}
    const failures: string[] = []
    for (const { tool, example } of examples) {
      const key = `${tool}.${example.id}`
      const result = await callWhenReady(client, tool, example.input)
      if (result.isError) {
        failures.push(`${key}: ${errorText(result.content)}`)
        continue
      }

      recordings[key] = {
        tool,
        example: example.id,
        recordedAt: new Date().toISOString(),
        output: result.structuredContent,
      }
      console.log(`recorded ${key}`)
    }

    if (failures.length > 0) {
      console.error(`The refresh changed no file. ${failures.length} call(s) failed:`)
      for (const failure of failures) console.error(`  ${failure}`)
      process.exitCode = 1
      return
    }

    await writeFile(definitionsFile, sortedJSON(definitions))
    await writeFile(recordingsFile, sortedJSON(recordings))
    console.log(`wrote ${Object.keys(definitions).length} definitions and ${Object.keys(recordings).length} recordings`)
  } finally {
    await client.close()
  }
}

await main()
