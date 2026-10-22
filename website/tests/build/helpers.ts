import { existsSync, readdirSync, readFileSync, statSync } from "node:fs"
import { join, relative, resolve } from "node:path"
import { gzipSync } from "node:zlib"

import { parse, type HTMLElement } from "node-html-parser"
import { parse as parseYaml } from "yaml"

export const root = resolve(import.meta.dirname, "../..")
export const dist = join(root, "dist")

if (!existsSync(join(dist, "index.html"))) {
  throw new Error("The build tests read dist/. Run pnpm build first.")
}

/** Lists the address of each built page, as "/", "/ka/", "/tools/x/". */
export function pagePaths(): string[] {
  const paths: string[] = []
  const walk = (dir: string) => {
    for (const name of readdirSync(dir)) {
      const full = join(dir, name)
      if (statSync(full).isDirectory()) walk(full)
      else if (name === "index.html") paths.push(`/${relative(dist, dir)}/`.replace(/\/+/g, "/"))
    }
  }
  walk(dist)

  return paths.toSorted()
}

/** Reads one built page, before a browser runs it. */
export function page(path: string): HTMLElement {
  const file = path.endsWith(".html") ? join(dist, path) : join(dist, path, "index.html")

  return parse(readFileSync(file, "utf8"), { blockTextElements: { script: true, style: true } })
}

/** The pair of a page in the other language. */
export function pair(path: string): string {
  return path.startsWith("/ka/") ? path.slice(3) : `/ka${path}`
}

/** The gzip size in bytes of every script that a built page loads. */
export function scriptBytes(path: string): number {
  return page(path)
    .querySelectorAll("script")
    .reduce((sum, script) => {
      const src = script.getAttribute("src")
      const body = src ? readFileSync(join(dist, src)) : Buffer.from(script.textContent)

      return sum + gzipSync(body).length
    }, 0)
}

/** The data files of the site. */
export const definitions = JSON.parse(readFileSync(join(root, "src/data/definitions.json"), "utf8")) as Record<
  string,
  { description: string }
>
export const recordings = JSON.parse(readFileSync(join(root, "src/data/recordings.json"), "utf8")) as Record<
  string,
  { recordedAt: string; output: unknown }
>

/** The contexts of the site, with their vendors, in the order of contexts.yaml. */
export const contexts = parseYaml(readFileSync(join(root, "src/content/contexts.yaml"), "utf8")) as Record<
  string,
  { vendors: Array<{ code: string; en: string; ka: string }> }
>

/** The hand-written content of one tool. */
export function toolContent(name: string) {
  return parseYaml(readFileSync(join(root, "src/content/tools", `${name}.yaml`), "utf8")) as {
    summary: { en: string; ka: string }
    vendors?: string[]
    examples: Array<{ id: string; question: { en: string; ka: string }; answer: { en: string; ka: string } }>
  }
}

/** Collapses the white space of a text, as a browser shows it. */
export function text(element: HTMLElement | null): string {
  return (element?.text ?? "").replace(/\s+/g, " ").trim()
}
