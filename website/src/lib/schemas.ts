import { z } from "astro/zod"

export const langs = ["en", "ka"] as const

/** The language of a page. */
export type Lang = (typeof langs)[number]

const localized = z.object({ en: z.string().min(1), ka: z.string().min(1) })

/** A text in each language of the site. */
export type Localized = z.infer<typeof localized>

const jsonSchema = z.record(z.string(), z.unknown())

/** The fields of one tool in the answer of tools/list. */
export const definitionSchema = z.object({
  name: z.string(),
  title: z.string(),
  description: z.string(),
  inputSchema: jsonSchema,
  outputSchema: jsonSchema,
  annotations: z.record(z.string(), z.unknown()),
})

/** One tool as the server lists it. */
export type Definition = z.infer<typeof definitionSchema>

/** One output of the server, recorded for one tool example. */
export const recordingSchema = z.object({
  tool: z.string(),
  example: z.string(),
  recordedAt: z.iso.datetime({ offset: true }),
  output: z.unknown(),
})

/** One recorded output. */
export type Recording = z.infer<typeof recordingSchema>

export const contextKeys = ["rates", "fuel", "transit", "offers", "postal", "cinema"] as const

/** The name of a context. */
export type ContextKey = (typeof contextKeys)[number]

/** The hand-written content of one tool. */
export const toolContentSchema = z.object({
  context: z.enum(contextKeys),
  summary: localized.partial(),
  vendors: z.array(z.string()),
  examples: z.array(
    z.object({
      id: z.string().regex(/^[a-z0-9-]+$/),
      question: localized,
      answer: localized.partial(),
      input: z.record(z.string(), z.unknown()),
    }),
  ),
})

/** The content of one tool, written for the site. */
export type ToolContent = z.infer<typeof toolContentSchema>

const vendor = localized.extend({ code: z.string().regex(/^[a-z]+$/) })

/** A vendor of a context: its name in each language, and its code. */
export type Vendor = z.infer<typeof vendor>

/** The hand-written content of one context. */
export const contextContentSchema = z.object({
  order: z.number().int(),
  hue: z.enum(["butter", "peach", "sky", "lilac", "sage", "teal"]),
  icon: z.string(),
  name: localized,
  vendors: z.array(vendor).min(1),
})

/** The content of one context, written for the site. */
export type ContextContent = z.infer<typeof contextContentSchema>
