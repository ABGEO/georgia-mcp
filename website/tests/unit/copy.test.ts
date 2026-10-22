import { readdirSync, readFileSync } from "node:fs"
import { join, resolve } from "node:path"

import { expect, test } from "vitest"

import { en } from "../../src/i18n/en"
import { ka } from "../../src/i18n/ka"

const root = resolve(import.meta.dirname, "../..")

function strings(value: unknown): string[] {
  if (typeof value === "string") return [value]
  if (typeof value === "function") return [String(value(1)), String(value("x"))]
  if (value && typeof value === "object") return Object.values(value).flatMap(strings)

  return []
}

// WEBSITE-SC-033: No text of the site holds an em dash.
test("the copy holds no em dash", () => {
  const dir = join(root, "src/content/tools")
  const yaml = readdirSync(dir).map((file) => readFileSync(join(dir, file), "utf8"))
  const all = [...strings(en), ...strings(ka), ...yaml, readFileSync(join(root, "src/content/contexts.yaml"), "utf8")]

  expect(all.filter((value) => value.includes("—"))).toEqual([])
})
