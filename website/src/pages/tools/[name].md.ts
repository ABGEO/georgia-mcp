import type { APIRoute, GetStaticPaths } from "astro"

import { toolPaths, type CatalogContext, type CatalogTool } from "../../lib/catalog"
import { toolMarkdown } from "../../lib/catalog-files"

export const getStaticPaths = toolPaths satisfies GetStaticPaths

export const GET: APIRoute = ({ props }) => {
  const { tool, context } = props as { tool: CatalogTool; context: CatalogContext }

  return new Response(toolMarkdown(tool, context), { headers: { "Content-Type": "text/markdown; charset=utf-8" } })
}
