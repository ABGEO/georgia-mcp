import { readFile } from "node:fs/promises"
import { extname, join, resolve } from "node:path"

import { chromium } from "@playwright/test"

const website = resolve(import.meta.dirname, "..")
const dist = join(website, "dist")
const origin = "http://og.local"
const types: Record<string, string> = { ".html": "text/html", ".css": "text/css", ".woff2": "font/woff2", ".svg": "image/svg+xml" }

const browser = await chromium.launch()
try {
  const page = await browser.newPage({ viewport: { width: 1200, height: 630 } })
  const card = await readFile(join(website, "scripts/og-card.html"), "utf8")
  const mark = (await readFile(join(website, "src/assets/mark.svg"), "utf8")).replace("<svg ", '<svg width="300" height="300" ')
  let current = ""

  await page.route(`${origin}/**`, async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === "/card/") return route.fulfill({ contentType: "text/html", body: current })

    const file = join(dist, path.endsWith("/") ? `${path}index.html` : path)
    return route.fulfill({ contentType: types[extname(file)] ?? "application/octet-stream", body: await readFile(file) })
  })

  for (const lang of ["en", "ka"]) {
    await page.goto(`${origin}${lang === "en" ? "/" : "/ka/"}`)
    const hero = await page.evaluate(() => ({
      eyebrow: document.querySelector("#intro .eyebrow")?.textContent ?? "",
      title: document.querySelector("#intro h1")?.innerHTML ?? "",
      fonts: [...document.querySelectorAll("head style")]
        .map((style) => style.outerHTML)
        .filter((html) => html.includes("@font-face") || html.includes("--font-"))
        .join("\n"),
    }))

    current = card
      .replace("{{lang}}", lang)
      .replace("{{fonts}}", hero.fonts)
      .replace("{{eyebrow}}", hero.eyebrow)
      .replace("{{title}}", hero.title)
      .replace("{{mark}}", mark)
    await page.goto(`${origin}/card/`)
    await page.evaluate(() => document.fonts.ready)
    await page.screenshot({ path: join(website, `public/og/${lang}.png`) })
    console.log(`wrote public/og/${lang}.png`)
  }
} finally {
  await browser.close()
}
