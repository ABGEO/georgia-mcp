import { existsSync, readdirSync, readFileSync } from "node:fs"
import { join } from "node:path"

import { describe, expect, test } from "vitest"

import { SERVER_URL, SITE_SUMMARY, SITE_URL } from "../../src/site"
import { contexts, definitions, dist, page, pagePaths, root } from "./helpers"

const names = Object.keys(definitions).toSorted()

const fullDefinitions = JSON.parse(readFileSync(join(root, "src/data/definitions.json"), "utf8")) as Record<
  string,
  Record<string, unknown>
>

const contextNames = Object.values(
  contexts as unknown as Record<string, { name: { en: string } }>,
).map((context) => context.name.en)

describe("/llms.txt", () => {
  const text = readFileSync(join(dist, "llms.txt"), "utf8")

  // LLMSTXT-SC-002: The file lists the tools of the server, and its summary names no subject.
  test("lists each tool of the server once", () => {
    const listed = [...text.matchAll(/^- \[[^\]]+\]\([^)]+\/tools\/([a-z_]+)\.md\)/gm)].map((match) => match[1])

    expect(listed.toSorted()).toEqual(names)
  })

  // LLMSTXT-SC-002: The file lists the tools of the server, and its summary names no subject.
  test("says what the server is with no subject and no count", () => {
    const summary = text.split("\n").find((line) => line.startsWith("> "))

    expect(summary).toBe(`> ${SITE_SUMMARY}`)
    for (const name of contextNames) expect(summary?.toLowerCase()).not.toContain(name.toLowerCase())
    expect(summary).not.toMatch(/\d/)
  })

  // LLMSTXT-SC-002: The file exists in English only.
  test("has no Georgian copy", () => {
    expect(existsSync(join(dist, "ka/llms.txt"))).toBe(false)
  })
})

describe("the Markdown copies", () => {
  // LLMSTXT-SC-004: One copy for each tool, in English only, with the description of its definition.
  test("exist for each tool and hold its description", () => {
    const copies = readdirSync(join(dist, "tools"))
      .filter((name) => name.endsWith(".md"))
      .map((name) => name.slice(0, -3))

    expect(copies.toSorted()).toEqual(names)
    for (const name of names) {
      const text = readFileSync(join(dist, "tools", `${name}.md`), "utf8")
      expect(text).toContain(definitions[name].description.replace(/\s+/g, " ").trim())
    }
  })

  // LLMSTXT-SC-004: One copy for each tool, in English only.
  test("do not exist in Georgian", () => {
    expect(readdirSync(join(dist, "ka/tools")).filter((name) => name.endsWith(".md"))).toEqual([])
    expect(existsSync(join(dist, "ka/tools.json"))).toBe(false)
  })
})

describe("/tools.json", () => {
  const catalog = JSON.parse(readFileSync(join(dist, "tools.json"), "utf8")) as {
    name: string
    url: string
    tools: Array<Record<string, unknown> & { name: string; page: string; markdown: string }>
  }

  // LLMSTXT-SC-005: The file holds the server and each tool definition as the server lists it.
  test("names the server", () => {
    expect(catalog.name).toBe("georgia-mcp")
    expect(catalog.url).toBe(SERVER_URL)
  })

  // LLMSTXT-SC-005: The file holds the server and each tool definition as the server lists it.
  test("holds each definition and the addresses of the tool", () => {
    expect(catalog.tools.map((tool) => tool.name).toSorted()).toEqual(names)

    for (const { page: pageUrl, markdown, ...entry } of catalog.tools) {
      expect(entry).toEqual(fullDefinitions[entry.name as string])
      expect(pageUrl).toBe(`${SITE_URL}/tools/${entry.name}/`)
      expect(markdown).toBe(`${SITE_URL}/tools/${entry.name}.md`)
    }
  })
})

describe("the links to the catalog", () => {
  const paths = [...pagePaths(), "/404.html"]

  // LLMSTXT-SC-006: Each page links to the catalog files, and each English tool page to its copy.
  test("each page links to both catalog files", () => {
    for (const path of paths) {
      const head = page(path).querySelector("head")
      const hrefs = head?.querySelectorAll('link[rel="alternate"]').map((link) => link.getAttribute("href")) ?? []

      expect(hrefs, path).toContain("/llms.txt")
      expect(hrefs, path).toContain("/tools.json")
    }
  })

  // LLMSTXT-SC-006: Each page links to the catalog files, and each English tool page to its copy.
  test("an English tool page links to its copy, a Georgian page to none", () => {
    for (const name of names) {
      const hrefs = page(`/tools/${name}/`)
        .querySelectorAll('link[rel="alternate"]')
        .map((link) => link.getAttribute("href"))
      expect(hrefs, name).toContain(`/tools/${name}.md`)
    }
    for (const path of paths.filter((p) => p.startsWith("/ka/"))) {
      const hrefs = page(path)
        .querySelectorAll("link")
        .map((link) => link.getAttribute("href") ?? "")
      expect(hrefs.filter((href) => href.endsWith(".md")), path).toEqual([])
    }
  })

  // LLMSTXT-SC-006: The robots file names both catalog files.
  test("the robots file names both catalog files", () => {
    const robots = readFileSync(join(dist, "robots.txt"), "utf8")

    expect(robots).toContain(`${SITE_URL}/llms.txt`)
    expect(robots).toContain(`${SITE_URL}/tools.json`)
  })
})
