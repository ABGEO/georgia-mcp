import { expect, test } from "vitest"

import { cursorInstallUrl, vscodeInstallUrl } from "../../src/lib/install"

const url = "https://geo-mcp.abgeo.cloud/mcp"

// WEBSITE-SC-054: Each install link carries the name and the address of the server.
test("builds the install link of Cursor", () => {
  const link = cursorInstallUrl("georgia-mcp", url)
  const prefix = "cursor://anysphere.cursor-deeplink/mcp/install?name=georgia-mcp&config="

  expect(link.startsWith(prefix)).toBe(true)
  expect(JSON.parse(atob(decodeURIComponent(link.slice(prefix.length))))).toEqual({ url })
})

// WEBSITE-SC-054: Each install link carries the name and the address of the server.
test("builds the install link of VS Code", () => {
  const link = vscodeInstallUrl("georgia-mcp", url)
  const prefix = "vscode:mcp/install?"

  expect(link.startsWith(prefix)).toBe(true)
  expect(JSON.parse(decodeURIComponent(link.slice(prefix.length)))).toEqual({ name: "georgia-mcp", type: "http", url })
})
