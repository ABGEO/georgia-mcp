import { describe, expect, test } from "vitest"

import { SITE_NAME, SITE_URL } from "../../src/site"
import { contexts, page, pagePaths } from "./helpers"

type Node = Record<string, unknown> & { "@type": string }

const homes = [
  { path: "/", lang: "en" as const },
  { path: "/ka/", lang: "ka" as const },
]

const contextNames = Object.values(contexts as unknown as Record<string, { name: { en: string; ka: string } }>)

function blocks(path: string): string[] {
  return page(path)
    .querySelectorAll('script[type="application/ld+json"]')
    .map((script) => script.textContent)
}

function graph(path: string): Node[] {
  return (JSON.parse(blocks(path)[0]) as { "@graph": Node[] })["@graph"]
}

describe("the structured data", () => {
  // JSONLD-SC-003: Each home page holds one block that parses, with the site of the page.
  test.each(homes)("$path holds one block with its site", ({ path, lang }) => {
    expect(blocks(path)).toHaveLength(1)

    const site = graph(path).find((node) => node["@type"] === "WebSite")
    expect(site?.url).toBe(new URL(path, SITE_URL).href)
    expect(site?.inLanguage).toBe(lang)
  })

  // JSONLD-SC-003: No other page holds a block.
  test("no other page holds a block", () => {
    const others = [...pagePaths(), "/404.html"].filter((path) => !homes.some((home) => home.path === path))

    for (const path of others) expect(blocks(path), path).toEqual([])
  })

  // JSONLD-SC-004: The block agrees with the page.
  test.each(homes)("$path states what the page shows", ({ path, lang }) => {
    const html = page(path)
    const app = graph(path).find((node) => node["@type"] === "SoftwareApplication")!
    const description = html.querySelector('meta[name="description"]')?.getAttribute("content")

    expect(app.name).toBe(SITE_NAME)
    expect(html.querySelector("title")?.text).toContain(SITE_NAME)
    expect(app.description).toBe(description)
    for (const context of contextNames) {
      expect((app.description as string).toLowerCase()).not.toContain(context.name[lang].toLowerCase())
    }
    expect(app).not.toHaveProperty("softwareVersion")
    expect(html.querySelector("footer a.version")?.text).toBe("dev")
  })
})
