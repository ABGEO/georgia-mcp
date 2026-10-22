import AxeBuilder from "@axe-core/playwright"
import { expect, test } from "@playwright/test"

test.describe("language", () => {
  test.use({ locale: "ka-GE" })

  // WEBSITE-SC-003: A Georgian browser stays on the English page at /.
  test("does not redirect by the language of the browser", async ({ page }) => {
    await page.goto("/")

    expect(new URL(page.url()).pathname).toBe("/")
    await expect(page.locator("html")).toHaveAttribute("lang", "en")
  })
})

test.describe("copy field", () => {
  // WEBSITE-SC-008: The copy button puts the server address on the clipboard.
  test("copies the address of the server", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"])
    await page.goto("/")
    await page.getByRole("tab", { name: "Claude Code" }).click()
    await page.locator("#guide-en-claudeCode").getByRole("button", { name: "Copy" }).click()

    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe("https://geo-mcp.abgeo.cloud/mcp")
    await expect(page.locator("#guide-en-claudeCode .status")).toHaveText("Copied")
  })

  // WEBSITE-SC-009: A refused clipboard selects the address and shows the fallback.
  test("selects the address when the clipboard refuses", async ({ page }) => {
    await page.addInitScript(() => {
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: () => Promise.reject(new Error("refused")) },
      })
    })
    await page.goto("/")
    await page.locator("#guide-en-claude").getByRole("button", { name: "Copy" }).click()

    const input = page.locator("#address-en-claude")
    const selection = await input.evaluate((el: HTMLInputElement) => [el.selectionStart, el.selectionEnd, el.value.length])
    expect(selection).toEqual([0, selection[2], selection[2]])
    await expect(page.locator("#guide-en-claude .status")).toHaveText("Press Ctrl+C to copy")
  })
})

test.describe("theme", () => {
  test.use({ colorScheme: "light" })

  // WEBSITE-SC-024: The selected theme follows the reader to the next page.
  test("keeps the selected theme on the next page", async ({ page }) => {
    await page.goto("/")
    const toggle = page.locator("[data-theme-toggle]")
    await toggle.click()
    await toggle.click()
    await expect(page.locator("html")).toHaveAttribute("data-theme", "dark")

    await page.goto("/tools/fuel_get_prices/", { waitUntil: "commit" })
    await page.waitForSelector("html[data-theme]", { state: "attached" })
    expect(await page.evaluate(() => document.documentElement.dataset.theme)).toBe("dark")
  })
})

test.describe("theme without storage", () => {
  test.use({ colorScheme: "dark" })

  // WEBSITE-SC-025: A browser that keeps no data shows the system theme with no error.
  test("follows the system when storage throws", async ({ page }) => {
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text())
    })
    await page.addInitScript(() => {
      Object.defineProperty(window, "localStorage", {
        get() {
          throw new Error("blocked")
        },
      })
    })
    await page.goto("/")

    expect(await page.evaluate(() => getComputedStyle(document.body).backgroundColor)).toBe("rgb(26, 22, 20)")
    expect(errors).toEqual([])
  })
})

// WEBSITE-SC-026: An unknown address answers 404 with links to both home pages.
test("answers an unknown address with the 404 page", async ({ page }) => {
  const response = await page.goto("/no-such-page/")

  expect(response?.status()).toBe(404)
  await expect(page.locator("main a[href='/']")).toHaveCount(1)
  await expect(page.locator("main [lang=ka]")).toHaveCount(0)
  await expect(page.locator("a.lang-switch")).toHaveAttribute("href", "/ka/")
  await expect(page.locator("meta[name=robots]")).toHaveAttribute("content", "noindex")
  await expect(page.locator("link[rel=canonical]")).toHaveCount(0)

  const bare = await page.goto("/no-such-page")
  expect(bare?.status()).toBe(404)
  await expect(page).toHaveTitle(/^This page does not exist/)
})

test.describe("weight", () => {
  test.use({ viewport: { width: 390, height: 844 } })

  // WEBSITE-SC-030: The first view of a home page transfers 500 KB at most.
  for (const path of ["/", "/ka/"]) {
    test(`${path} transfers 500 KB at most`, async ({ page }) => {
      const responses: Array<Promise<number>> = []
      page.on("requestfinished", (request) => {
        responses.push(request.sizes().then((sizes) => sizes.responseBodySize))
      })
      await page.goto(path, { waitUntil: "networkidle" })
      const total = (await Promise.all(responses)).reduce((sum, size) => sum + size, 0)

      expect(total).toBeLessThanOrEqual(500 * 1024)
    })
  }
})

test.describe("accessibility", () => {
  const pages = ["/", "/ka/", "/tools/fuel_get_prices/", "/ka/tools/transit_plan_trip/", "/no-such-page/"]

  // WEBSITE-SC-031: axe finds no violation of WCAG 2.2 AA, in each theme.
  for (const scheme of ["light", "dark"] as const) {
    for (const path of pages) {
      test(`${path} in ${scheme} has no violation`, async ({ browser }) => {
        const context = await browser.newContext({ colorScheme: scheme })
        const page = await context.newPage()
        await page.goto(path, { waitUntil: "networkidle" })
        await page.evaluate(() =>
          document
            .getAnimations()
            .filter((animation) => animation.effect?.getTiming().iterations !== Infinity)
            .forEach((animation) => animation.finish()),
        )
        const result = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze()

        expect(result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`)).toEqual([])
        await context.close()
      })
    }
  }
})

// WEBSITE-SC-032: Every control of the home page works from the keyboard, with a visible ring.
test("works from the keyboard alone", async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"])
  await page.goto("/")

  const focused = () =>
    page.evaluate(() => {
      const el = document.activeElement as HTMLElement
      return { role: el.getAttribute("role"), copy: el.hasAttribute("data-copy"), theme: el.hasAttribute("data-theme-toggle"), id: el.id, outline: getComputedStyle(el).outlineStyle }
    })
  const tabUntil = async (match: (f: Awaited<ReturnType<typeof focused>>) => boolean, key = "Tab") => {
    for (let i = 0; i < 80; i++) {
      await page.keyboard.press(key)
      const f = await focused()
      if (match(f)) return f
    }
    throw new Error(`no focus match after 80 presses of ${key}`)
  }

  const tab = await tabUntil((f) => f.role === "tab")
  expect(tab.outline).not.toBe("none")
  await page.keyboard.press("ArrowRight")
  expect((await focused()).id).toBe("tab-en-claudeApps")
  await page.keyboard.press("ArrowRight")
  expect((await focused()).id).toBe("tab-en-claudeCode")
  await expect(page.locator("#guide-en-claudeCode")).toBeVisible()

  const copy = await tabUntil((f) => f.copy)
  expect(copy.outline).not.toBe("none")
  await page.keyboard.press("Enter")
  await expect(page.locator("#guide-en-claudeCode .status")).toHaveText("Copied")

  const theme = await tabUntil((f) => f.theme, "Shift+Tab")
  expect(theme.outline).not.toBe("none")
  await page.keyboard.press("Enter")
  await expect(page.locator("[data-theme-name]")).toHaveText("Light")
})

// WEBSITE-SC-036: The tool list shows one context at a time, and the arrow keys move between them.
test("shows the tools of the selected context", async ({ page }) => {
  await page.goto("/")
  const tools = page.locator("#tools")
  const tablist = tools.getByRole("tablist")
  await expect(tablist).toHaveAttribute("aria-orientation", "vertical")

  await tools.getByRole("tab", { name: /Tbilisi transit/ }).click()
  const transit = tools.locator("[data-tools-panel=transit]")
  await expect(transit).toBeVisible()
  await expect(transit.locator("li")).toHaveCount(8)
  for (const key of ["rates", "fuel", "offers"]) await expect(tools.locator(`[data-tools-panel=${key}]`)).toBeHidden()

  await page.keyboard.press("ArrowDown")
  await expect(tools.getByRole("tab", { name: /Card offers/ })).toBeFocused()
  await expect(tools.locator("[data-tools-panel=offers]")).toBeVisible()
  await expect(transit).toBeHidden()
})

const chatWindow = (page: import("@playwright/test").Page) => page.locator("#examples .chat-window")

test.describe("chat player", () => {
  // The fake clock does not deliver an IntersectionObserver change after a scroll, so the window starts in view.
  test.use({ viewport: { width: 1280, height: 1600 } })

  test.beforeEach(async ({ page }) => {
    await page.clock.install()
  })

  // WEBSITE-SC-037: A random question types, the tool shows, the answer types, then another tool joins below.
  test("grows one conversation from random examples", async ({ page }) => {
    await page.goto("/")
    await page.locator("#examples").scrollIntoViewIfNeeded()
    const exchanges = chatWindow(page).locator(".exchange")
    const first = exchanges.first()
    const tool = await first.getAttribute("data-tool")

    await page.clock.runFor(200)
    const partial = (await first.locator(".question .typed").textContent()) ?? ""
    expect(partial.length).toBeGreaterThan(0)
    await page.clock.runFor(5000)
    expect(((await first.locator(".question .typed").textContent()) ?? "").length).toBeGreaterThan(partial.length)
    await expect(first.locator(".calling")).toContainText(tool ?? "")
    await page.clock.runFor(3000)
    await expect(first.locator(".answer .typed")).not.toBeEmpty()

    await page.clock.runFor(30000)
    expect(await exchanges.count()).toBeGreaterThanOrEqual(2)
    await expect(exchanges.nth(1)).not.toHaveAttribute("data-tool", tool ?? "")
    await expect(page.locator("#examples a")).toHaveCount(0)
    await expect(page.locator("#examples button")).toHaveCount(0)
  })
})

test.describe("chat player with reduced motion", () => {
  test.use({ reducedMotion: "reduce" })

  // WEBSITE-SC-039: With reduced motion, one example shows in full, does not change, and has no control.
  test("shows one full example with no control", async ({ page }) => {
    await page.clock.install()
    await page.goto("/")
    const exchanges = chatWindow(page).locator(".exchange")
    await expect(exchanges).toHaveCount(1)
    const key = await exchanges.first().getAttribute("data-key")
    expect(((await exchanges.first().locator(".answer .typed").textContent()) ?? "").length).toBeGreaterThan(10)

    await page.clock.runFor(8000)
    await expect(exchanges).toHaveCount(1)
    await expect(exchanges.first()).toHaveAttribute("data-key", key ?? "")
    await expect(page.locator("#examples button")).toHaveCount(0)
  })
})

test.describe("how it works", () => {
  test.use({ viewport: { width: 1280, height: 1600 } })

  // WEBSITE-SC-041: The step of the tool call lights up while the chat calls the tool.
  test("follows the state of the player", async ({ page }) => {
    await page.clock.install()
    await page.goto("/")
    const player = page.locator("#examples .chat-player")
    for (let i = 0; i < 200 && (await player.getAttribute("data-phase")) !== "calling"; i++) {
      await page.clock.runFor(100)
    }
    await expect(player).toHaveAttribute("data-phase", "calling")

    const fill = (step: string) =>
      page.locator(`#examples [data-step=${step}]`).evaluate((el) => getComputedStyle(el).backgroundColor)
    // The fill changes with a 200 ms transition, which runs on the real clock.
    await expect.poll(() => fill("call")).toBe("rgb(244, 207, 211)")
    expect(await fill("ask")).not.toBe("rgb(244, 207, 211)")
    expect(await fill("answer")).not.toBe("rgb(244, 207, 211)")
  })
})
