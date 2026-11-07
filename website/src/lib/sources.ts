import type { Catalog, CatalogContext, CatalogTool } from "./catalog"
import type { Vendor } from "./schemas"

/** The sources of one context, each with the tools that read it. */
export type SourceGroup = {
  context: CatalogContext
  sources: Array<{ vendor: Vendor; tools: CatalogTool[] }>
}

/** The sources of each context, with the tools that read each one, in the order of the home page. */
export function sourceGroups(catalog: Catalog): SourceGroup[] {
  return catalog.contexts.map((context) => ({
    context,
    sources: context.vendors.map((vendor) => ({
      vendor,
      tools: context.tools.filter((tool) => tool.vendors.some((v) => v.code === vendor.code)),
    })),
  }))
}
