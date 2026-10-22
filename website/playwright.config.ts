import { defineConfig, devices } from "@playwright/test"

const port = 4322

export default defineConfig({
  testDir: "tests/browser",
  fullyParallel: true,
  use: {
    baseURL: `http://localhost:${port}`,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `pnpm preview --port ${port} --ignore-lock`,
    port,
    reuseExistingServer: false,
  },
})
