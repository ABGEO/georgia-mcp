import { existsSync } from "node:fs"
import { join } from "node:path"

import { describe, expect, test } from "vitest"

import { AUTHOR_URL, REPO_URL, SITE_URL } from "../../src/site"
import { definitions, dist, page, pagePaths, pair, text } from "./helpers"

const paths = pagePaths()
const homes = ["/", "/ka/"]
const tools = (prefix: string) =>
  paths.filter((path) => path.startsWith(`${prefix}/tools/`)).map((path) => path.split("/").at(-2))

describe("the page set", () => {
  // WEBSITE-SC-001: Each page has its pair in the other language, and no /en/ exists.
  test("pairs each English page with a Georgian page", () => {
    const english = paths.filter((path) => !path.startsWith("/ka/"))
    const georgian = paths.filter((path) => path.startsWith("/ka/"))

    expect(english.length).toBeGreaterThan(0)
    expect(georgian.map(pair).toSorted()).toEqual(english)
    expect(english.map(pair).toSorted()).toEqual(georgian)
    expect(existsSync(join(dist, "en"))).toBe(false)
  })

  // WEBSITE-SC-010: The site has no page under /mcp.
  test("has no page at /mcp", () => {
    expect(existsSync(join(dist, "mcp"))).toBe(false)
    expect(existsSync(join(dist, "ka", "mcp"))).toBe(false)
  })

  // WEBSITE-SC-012: One tool page in each language for each definition.
  test("has one tool page for each definition", () => {
    const names = Object.keys(definitions).toSorted()
    expect(names.length).toBeGreaterThan(0)
    expect(tools("")).toEqual(names)
    expect(tools("/ka")).toEqual(names)
  })
})

describe("each page", () => {
  // WEBSITE-SC-002: The language switch and the hreflang links point to the pair.
  test.each(paths)("%s links to its pair", (path) => {
    const html = page(path)
    const target = pair(path)
    const lang = path.startsWith("/ka/") ? "ka" : "en"
    const other = lang === "en" ? "ka" : "en"

    expect(html.querySelector("a.lang-switch")?.getAttribute("href")).toBe(target)
    expect(html.querySelector(`link[rel=alternate][hreflang=${other}]`)?.getAttribute("href")).toBe(
      new URL(target, SITE_URL).href,
    )
  })

  // WEBSITE-SC-023: The footer holds the byline, the author link, and the repository link.
  test.each([...paths, "/404.html"])("%s ends with the byline", (path) => {
    const footer = page(path).querySelector("footer")
    const links = footer?.querySelectorAll("a").map((a) => a.getAttribute("href")) ?? []

    expect(text(footer)).toContain("Made with ❤️ in Georgia by ABGEO")
    expect(links).toContain(AUTHOR_URL)
    expect(links).toContain(REPO_URL)
  })

  // WEBSITE-SC-027: The head holds a title, a description, and the preview image of the language.
  test.each(paths)("%s carries its title, description, and preview image", (path) => {
    const html = page(path)
    const lang = path.startsWith("/ka/") ? "ka" : "en"

    expect(text(html.querySelector("title"))).not.toBe("")
    expect(html.querySelector("meta[name=description]")?.getAttribute("content")).toBeTruthy()
    expect(html.querySelector("meta[property='og:image']")?.getAttribute("content")).toBe(`${SITE_URL}/og/${lang}.png`)
    expect(existsSync(join(dist, "og", `${lang}.png`))).toBe(true)
  })
})

describe("the home page", () => {
  // WEBSITE-SC-004: The sections come in the order of WEBSITE-FR-030.
  test.each(homes)("%s holds the sections in order", (path) => {
    const ids = page(path)
      .querySelectorAll("main > section")
      .map((section) => section.id)

    expect(ids).toEqual(["intro", "examples", "contexts", "install", "tools"])
  })

  // WEBSITE-SC-007: Eight install guides in the order of the specification.
  test.each(homes)("%s holds eight install guides", (path) => {
    const panels = page(path).querySelectorAll("#install [data-panel]")
    const lang = path === "/" ? "en" : "ka"

    expect(panels.map((panel) => panel.id)).toEqual(
      ["claude", "claudeApps", "claudeCode", "chatgpt", "cursor", "vscode", "binary", "docker"].map((key) => `guide-${lang}-${key}`),
    )
    expect(text(panels[7])).toContain("GEOMCP_TRANSIT_TTC_ENABLED")
    expect(text(panels[7])).toContain("GEOMCP_RATES_TBC_ENABLED")
  })
})
