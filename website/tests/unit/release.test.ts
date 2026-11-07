import { expect, test } from "vitest"

import { releaseUrl } from "../../src/lib/release"

const repo = "https://github.com/abgeo/georgia-mcp"

// WEBSITE-SC-053: A release links to its page. Any other site version links to the list of releases.
test.each([
  ["v1.2.3", `${repo}/releases/tag/v1.2.3`],
  ["v1.1.2-8-g8def960", `${repo}/releases`],
  ["dev", `${repo}/releases`],
])("%s links to %s", (version, want) => {
  expect(releaseUrl(repo, version)).toBe(want)
})
