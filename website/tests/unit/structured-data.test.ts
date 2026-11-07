import { describe, expect, test } from "vitest"

import { releaseVersion } from "../../src/lib/release"
import { structuredData } from "../../src/lib/structured-data"
import {
  AUTHOR_GITHUB_URL,
  AUTHOR_NAME,
  AUTHOR_URL,
  LICENSE_URL,
  REPO_URL,
  SITE_NAME,
  SITE_URL,
} from "../../src/site"

type Node = Record<string, unknown> & { "@id": string; "@type": string }

function nodes(lang: "en" | "ka", version: string) {
  const data = structuredData(lang, version, "A description.")
  const graph = data["@graph"] as Node[]
  const byType = (type: string) => graph.find((node) => node["@type"] === type)!

  return { data, graph, byType }
}

function references(value: unknown): string[] {
  if (Array.isArray(value)) return value.flatMap(references)
  if (value && typeof value === "object") {
    const entries = Object.entries(value)
    if (entries.length === 1 && entries[0][0] === "@id") return [entries[0][1] as string]

    return entries.filter(([key]) => key !== "@id").flatMap(([, v]) => references(v))
  }

  return []
}

describe("structuredData", () => {
  // JSONLD-SC-001: The graph holds the site, the application, the source, and the author.
  test("holds one node of each type, linked by @id", () => {
    const { data, graph, byType } = nodes("en", "v1.2.3")

    expect(data["@context"]).toBe("https://schema.org")
    expect(graph.map((node) => node["@type"]).toSorted()).toEqual([
      "Person",
      "SoftwareApplication",
      "SoftwareSourceCode",
      "WebSite",
    ])

    const ids = new Set(graph.map((node) => node["@id"]))
    for (const ref of references(graph)) expect(ids).toContain(ref)

    const author = byType("Person")
    expect(author).toMatchObject({ name: AUTHOR_NAME, url: AUTHOR_URL, sameAs: [AUTHOR_GITHUB_URL] })

    expect(byType("WebSite")).toMatchObject({ name: SITE_NAME, url: `${SITE_URL}/`, inLanguage: "en" })

    const app = byType("SoftwareApplication")
    expect(app).toMatchObject({
      name: SITE_NAME,
      description: "A description.",
      softwareVersion: "1.2.3",
      license: LICENSE_URL,
      offers: { "@type": "Offer", price: "0", priceCurrency: "GEL" },
      author: { "@id": author["@id"] },
    })

    expect(byType("SoftwareSourceCode")).toMatchObject({
      codeRepository: REPO_URL,
      programmingLanguage: "Go",
      license: LICENSE_URL,
      author: { "@id": author["@id"] },
      targetProduct: { "@id": app["@id"] },
    })
  })

  // JSONLD-SC-001: The Georgian home page states its own address and language.
  test("gives the Georgian site its address and language", () => {
    expect(nodes("ka", "v1.2.3").byType("WebSite")).toMatchObject({ url: `${SITE_URL}/ka/`, inLanguage: "ka" })
  })

  // JSONLD-SC-002: A site version that names no release states no version.
  test.each(["v1.1.2-8-g8def960", "dev"])("states no version for %s", (version) => {
    expect(nodes("en", version).byType("SoftwareApplication")).not.toHaveProperty("softwareVersion")
    expect(releaseVersion(version)).toBeUndefined()
  })

  // JSONLD-SC-002: A release gives its version without the "v".
  test("gives the version of a release", () => {
    expect(releaseVersion("v1.2.3")).toBe("1.2.3")
  })
})
