import type { APIRoute } from "astro"

import { getCatalog } from "../lib/catalog"
import { llmsText } from "../lib/catalog-files"

export const GET: APIRoute = async () =>
  new Response(llmsText(await getCatalog()), { headers: { "Content-Type": "text/plain; charset=utf-8" } })
