import { describe, expect, test } from "vitest"

import { contexts, definitions, page, recordings, scriptBytes, text, toolContent } from "./helpers"

describe("the tool list", () => {
  // WEBSITE-SC-011: Each tool appears once in its group, with its summary in the language of the page.
  test.each([
    ["/", "en"],
    ["/ka/", "ka"],
  ] as const)("%s lists every tool once", (path, lang) => {
    const tools = page(path).querySelector("#tools")
    const names = tools?.querySelectorAll("code").map((code) => text(code)) ?? []

    expect(tools?.querySelectorAll("section.group")).toHaveLength(4)
    expect(names.toSorted()).toEqual(Object.keys(definitions).toSorted())
    for (const name of names) expect(text(tools)).toContain(toolContent(name).summary[lang])
  })
})

describe("the tool page", () => {
  // WEBSITE-SC-015: Each ticket shows the question, the input, and the output, with no date of the record.
  test.each([
    ["/tools/fuel_get_prices/", "en"],
    ["/ka/tools/fuel_get_prices/", "ka"],
  ] as const)("%s shows each example with its record", (path, lang) => {
    const content = toolContent("fuel_get_prices")
    const tickets = page(path).querySelectorAll("article.ticket")

    expect(tickets).toHaveLength(content.examples.length)
    content.examples.forEach((example, i) => {
      const recording = recordings[`fuel_get_prices.${example.id}`]
      const ticket = tickets[i]
      expect(text(ticket)).toContain(example.question[lang])
      expect(ticket.querySelectorAll("pre")).toHaveLength(2)
      expect(ticket.querySelectorAll("pre")[1].text).toBe(JSON.stringify(recording.output, null, 2))
      expect(ticket.querySelector("time")).toBeNull()
      expect(text(ticket)).not.toContain(recording.recordedAt.slice(0, 4) + " ")
      expect(text(ticket.querySelector(".stub"))).toBe(lang === "en" ? "Fuel prices" : "საწვავის ფასები")
    })
  })

  // WEBSITE-SC-016: The Georgian page holds the Georgian texts, and the definition in English.
  test("shows the Georgian summary and questions, and the English definition", () => {
    const html = page("/ka/tools/fuel_get_prices/")
    const content = toolContent("fuel_get_prices")
    const english = html.querySelectorAll("[lang=en]").map((element) => text(element))

    expect(text(html.querySelector("main"))).toContain(content.summary.ka)
    for (const example of content.examples) expect(text(html.querySelector("main"))).toContain(example.question.ka)
    expect(english).toContain(definitions.fuel_get_prices.description.replace(/\s+/g, " ").trim())
  })
})

describe("without scripts", () => {
  // WEBSITE-SC-028: The HTML file holds every heading, every guide, every answer, and every output.
  test.each(["/", "/ka/", "/tools/transit_plan_trip/"])("%s holds all of its text", (path) => {
    const html = page(path)
    const main = html.querySelector("main")

    for (const heading of main?.querySelectorAll("section h2") ?? []) expect(text(heading)).not.toBe("")
    for (const panel of html.querySelectorAll("#install [data-panel]")) {
      expect(panel.hasAttribute("hidden")).toBe(false)
      expect(text(panel.querySelector("ol"))).not.toBe("")
    }
    const blocks = path.includes("/tools/")
      ? html.querySelectorAll("article.ticket pre.output")
      : html.querySelectorAll(".chat-list .answer")
    expect(blocks.length).toBeGreaterThan(0)
    for (const block of blocks) expect(block.text.length).toBeGreaterThan(2)
  })
})

describe("the weight of the scripts", () => {
  // WEBSITE-SC-029: 60 KB of script at most on the home page, 30 KB on a tool page.
  test("stays under the limits", () => {
    expect(scriptBytes("/")).toBeLessThanOrEqual(60 * 1024)
    expect(scriptBytes("/tools/fuel_get_prices/")).toBeLessThanOrEqual(30 * 1024)
  })
})

describe("the chat examples", () => {
  // WEBSITE-SC-035: The list holds every example of every tool, in the language of the page, with no link and no JSON.
  test.each([
    ["/", "en"],
    ["/ka/", "ka"],
  ] as const)("%s lists a question and an answer for each example", (path, lang) => {
    const examples = page(path).querySelector("#examples")
    const items = examples?.querySelectorAll(".chat-list > li") ?? []
    const expected = Object.keys(definitions).flatMap((name) =>
      toolContent(name).examples.map((example) => ({ name, example })),
    )

    expect(items).toHaveLength(expected.length)
    expect(examples?.querySelectorAll("pre")).toHaveLength(0)
    expect(examples?.querySelectorAll("a")).toHaveLength(0)
    for (const { name, example } of expected) {
      const item = items.find((li) => li.getAttribute("data-key") === `${name}.${example.id}`)
      expect(text(item?.querySelector(".question p") ?? null)).toBe(example.question[lang])
      expect(text(item?.querySelector(".answer p") ?? null)).toBe(example.answer[lang])
    }
  })
})

describe("how it works", () => {
  // WEBSITE-SC-040: Three steps in order, and the counts of the catalog.
  test.each(["/", "/ka/"])("%s explains the steps and gives the counts", (path) => {
    const examples = page(path).querySelector("#examples")
    const count = (key: string) => text(examples?.querySelector(`[data-count=${key}] .number`) ?? null)

    expect(examples?.querySelectorAll("[data-step]").map((step) => step.getAttribute("data-step"))).toEqual([
      "ask",
      "call",
      "answer",
    ])
    expect(count("tools")).toBe(String(Object.keys(definitions).length))
    const vendorCount = new Set(Object.values(contexts).flatMap((context) => context.vendors.map((vendor) => vendor.en))).size
    expect(count("contexts")).toBe(String(Object.keys(contexts).length))
    expect(count("vendors")).toBe(String(vendorCount))
  })
})

function vendors(path: string) {
  return page(path)
    .querySelectorAll("#vendors li")
    .map((item) => ({ name: text(item.querySelector(".name")), code: text(item.querySelector("code")) }))
}

describe("the vendors of a tool", () => {
  // WEBSITE-SC-045: A tool page lists its vendors, with a code when the tool takes one.
  test.each(["/", "/ka/"])("%s pages list the banks and the fuel companies", (prefix) => {
    const base = prefix === "/" ? "/tools" : "/ka/tools"

    const banks = vendors(`${base}/rates_get_commercial_rates/`)
    expect(banks.map((bank) => bank.code)).toEqual(toolContent("rates_get_commercial_rates").vendors)
    expect(banks[0].code).toBe("tbc")
    expect(banks.every((bank) => bank.name !== "" && bank.code !== "")).toBe(true)

    const companies = vendors(`${base}/fuel_get_prices/`)
    expect(companies.map((company) => company.code)).toEqual(toolContent("fuel_get_prices").vendors)

    const nbg = prefix === "/" ? "National Bank of Georgia" : "საქართველოს ეროვნული ბანკი"
    expect(vendors(`${base}/rates_get_official_rates/`)).toEqual([{ name: nbg, code: "" }])
  })
})

describe("the vendors of a context", () => {
  // WEBSITE-SC-046: A card lists up to three vendors, then the count of the rest.
  const rest = contexts.fuel.vendors.length - 3

  test.each([
    ["/", "Fuel prices", `+${rest} more`],
    ["/ka/", "საწვავის ფასები", `+${rest} სხვა`],
  ])("%s caps each card at three vendors", (path, name, more) => {
    const cards = page(path).querySelectorAll("#contexts .card")
    expect(cards.length).toBeGreaterThan(0)
    for (const card of cards) {
      expect(card.querySelectorAll(".sources li:not(.more)").length).toBeLessThanOrEqual(3)
    }

    const fuel = cards.find((card) => text(card.querySelector("h3")) === name)
    const items = fuel?.querySelectorAll(".sources li").map(text) ?? []
    expect(items).toHaveLength(4)
    expect(items[3]).toBe(more)
  })
})
