import { existsSync, readFileSync } from "node:fs"
import { join } from "node:path"

import { describe, expect, test } from "vitest"

import { AUTHOR_URL, CLAUDE_DIRECTORY_URL, REPO_URL, SERVER_URL, SITE_URL } from "../../src/site"
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

  // WEBSITE-SC-051: The guide of claude.ai links to the directory, keeps the address, and names no limit of the Free plan.
  test.each(homes)("%s links the guide of claude.ai to the directory", (path) => {
    const lang = path === "/" ? "en" : "ka"
    const panel = page(path).querySelector(`#guide-${lang}-claude`)

    expect(panel?.querySelectorAll("a").map((a) => a.getAttribute("href"))).toContain(CLAUDE_DIRECTORY_URL)
    expect(panel?.querySelector("input")?.getAttribute("value")).toBe(SERVER_URL)
    expect(text(panel)).not.toMatch(/Free plan|უფასო პაკეტ/)
  })
})

describe("the legal pages", () => {
  const legal = ["/privacy/", "/ka/privacy/", "/terms/", "/ka/terms/"]

  // WEBSITE-SC-047: Each legal page has a title, the effective date, sections, the name, the server, and the contact.
  test.each(legal)("%s holds the document", (path) => {
    const html = page(path)
    const body = text(html.querySelector("main"))

    expect(text(html.querySelector("main h1"))).not.toBe("")
    expect(html.querySelector("main time")?.getAttribute("datetime")).toBe("2026-10-24")
    expect(html.querySelectorAll("main h2").length).toBeGreaterThan(0)
    expect(body).toContain("Georgia MCP")
    expect(body).toContain(SERVER_URL)
    expect(body).toContain("me@abgeo.dev")
  })

  // WEBSITE-SC-048: The privacy policy names the records and says that the server keeps no input of a tool call.
  test("the privacy policy names the records", () => {
    const body = text(page("/privacy/").querySelector("main"))

    expect(body).toContain("IP address")
    expect(body).toContain("user agent")
    expect(body).toContain("name of the tool")
    expect(body).toContain("does not keep the input of a tool call")
  })

  // WEBSITE-SC-049: The footer links to the legal pages in the language of the page.
  test.each([...paths, "/404.html"])("%s links to the legal pages", (path) => {
    const links = page(path).querySelectorAll("footer a").map((a) => a.getAttribute("href"))
    const prefix = path.startsWith("/ka/") ? "/ka" : ""

    expect(links).toContain(`${prefix}/privacy/`)
    expect(links).toContain(`${prefix}/terms/`)
  })
})

// WEBSITE-SC-050: The token of the domain verification is one line of base64url.
test("serves the token of the domain verification", () => {
  const token = readFileSync(join(dist, ".well-known", "openai-apps-challenge"), "utf8")

  expect(token).toMatch(/^[A-Za-z0-9_-]{43}$/)
})
