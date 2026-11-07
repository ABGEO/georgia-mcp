import { existsSync, readFileSync } from "node:fs"
import { join } from "node:path"

import { describe, expect, test } from "vitest"
import { parse as parseYaml } from "yaml"

import { localize, t } from "../../src/i18n"
import { REPO_URL, SITE_URL } from "../../src/site"
import { contexts, definitions, page, pagePaths, root, text, toolContent } from "./helpers"

type ContextYaml = { order: number; vendors: Array<{ code: string }> }

const langs = ["en", "ka"] as const
const ordered = Object.entries(contexts as unknown as Record<string, ContextYaml>).toSorted(
  ([, a], [, b]) => a.order - b.order,
)

function sourcesPage(lang: (typeof langs)[number]) {
  return page(localize("/sources/", lang))
}

function readers(context: string, code: string): string[] {
  return Object.keys(definitions)
    .filter((name) => {
      const tool = toolContent(name) as ReturnType<typeof toolContent> & { context: string }
      return tool.context === context && (tool.vendors ?? []).includes(code)
    })
    .toSorted()
}

describe.each(langs)("the sources page in %s", (lang) => {
  const html = sourcesPage(lang)
  const copy = t(lang)

  // SOURCES-SC-003: The groups follow the contexts, and each holds the vendors of its context.
  test("lists the vendors of each context in order", () => {
    const groups = html.querySelectorAll("section.group")

    expect(groups.map((group) => group.id)).toEqual(ordered.map(([key]) => key))
    for (const [key, context] of ordered) {
      const ids = html.querySelectorAll(`section#${key} article.source`).map((article) => article.id)
      expect(ids).toEqual(context.vendors.map((vendor) => `${key}-${vendor.code}`))
    }
    expect(text(html.querySelector("p.lead"))).not.toMatch(/\d/)
  })

  // SOURCES-SC-004: Each source shows its kind, says when it is not official, and links to a page for people.
  test("shows the kind and the link of each source", () => {
    const labels = Object.values(copy.sources.kinds)
    const notes = copy.sources.notes as Record<string, string>

    for (const article of html.querySelectorAll("article.source")) {
      const kind = article.getAttribute("data-kind") ?? ""
      expect(labels, article.id).toContain(text(article.querySelector(".kind")))
      expect(text(article.querySelector(".kind"))).toBe((copy.sources.kinds as Record<string, string>)[kind])
      if (kind in notes) expect(text(article.querySelector(".note")), article.id).toBe(notes[kind])
      else expect(article.querySelector(".note"), article.id).toBeNull()

      const url = new URL(article.querySelector("a.visit")?.getAttribute("href") ?? "")
      expect(url.protocol, article.id).toBe("https:")
      expect(url.host, article.id).not.toBe(new URL(SITE_URL).host)
      expect(url.pathname, article.id).not.toContain("/api")
    }
  })

  // SOURCES-SC-005: Each source names the tools that read it, linked in the language of the page.
  test("names the tools of each source", () => {
    for (const article of html.querySelectorAll("article.source")) {
      const [context, code] = article.id.split("-")
      const links = article.querySelectorAll("ul.tools a")

      expect(links.map((link) => text(link)).toSorted(), article.id).toEqual(readers(context, code))
      for (const link of links) expect(link.getAttribute("href")).toBe(localize(`/tools/${text(link)}/`, lang))
    }
  })

  // SOURCES-SC-006: The page says how the server reads a source, and how a vendor asks for a change.
  test("says how the server reads a source and how a vendor asks", () => {
    expect(text(html.querySelector("#how"))).toContain(copy.sources.how.paragraphs[0])

    const hrefs = html.querySelectorAll("#vendors a").map((link) => link.getAttribute("href"))
    expect(hrefs).toEqual([`${REPO_URL}/issues/new?template=source_request.yml`])
  })
})

describe("the links to the sources page", () => {
  // SOURCES-SC-007: The footer of each page links to the sources page of its language.
  test.each([...pagePaths(), "/404.html"])("%s links to the sources page in its footer", (path) => {
    const want = path.startsWith("/ka/") ? "/ka/sources/" : "/sources/"
    const links = page(path)
      .querySelectorAll("footer a")
      .map((link) => link.getAttribute("href"))

    expect(links).toContain(want)
  })

  // SOURCES-SC-007: A tool page links to the group of its context.
  test("each tool page links to the group of its context", () => {
    for (const name of Object.keys(definitions)) {
      const { context } = toolContent(name) as ReturnType<typeof toolContent> & { context: string }
      for (const lang of langs) {
        const link = page(localize(`/tools/${name}/`, lang)).querySelector("#vendors a.sources-link")
        expect(link?.getAttribute("href"), name).toBe(`${localize("/sources/", lang)}#${context}`)
      }
    }
  })
})

// SOURCES-SC-006: The form of a source request exists, with its label.
test("the repository has the form of a source request", () => {
  const github = join(root, "..", ".github")
  const form = join(github, "ISSUE_TEMPLATE", "source_request.yml")

  expect(existsSync(form)).toBe(true)
  expect((parseYaml(readFileSync(form, "utf8")) as { labels: string[] }).labels).toEqual(["source"])
  const labels = parseYaml(readFileSync(join(github, "labels.yml"), "utf8")) as Array<{ name: string }>
  expect(labels.map((label) => label.name)).toContain("source")
})
