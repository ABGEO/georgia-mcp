import { expect, test } from "vitest"

import { cursorInstallUrl } from "../../src/lib/install"

const url = "https://geo-mcp.abgeo.cloud/mcp"

// WEBSITE-SC-054: The install link carries the name and the address of the server.
test("builds the install link of Cursor", () => {
  const link = cursorInstallUrl("georgia-mcp", url)
  const prefix = "cursor://anysphere.cursor-deeplink/mcp/install?name=georgia-mcp&config="

  expect(link.startsWith(prefix)).toBe(true)
  expect(JSON.parse(atob(decodeURIComponent(link.slice(prefix.length))))).toEqual({ url })
})
