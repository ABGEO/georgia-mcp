import type { APIRoute } from "astro"

import { getCatalog } from "../lib/catalog"
import { toolsJson } from "../lib/catalog-files"

export const GET: APIRoute = async () =>
  new Response(`${JSON.stringify(toolsJson(await getCatalog()), null, 2)}\n`, {
    headers: { "Content-Type": "application/json; charset=utf-8" },
  })
