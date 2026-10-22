import { defineCollection } from "astro:content"
import { file, glob } from "astro/loaders"

import { contextContentSchema, definitionSchema, recordingSchema, toolContentSchema } from "./lib/schemas"

const definitions = defineCollection({
  loader: file("src/data/definitions.json"),
  schema: definitionSchema,
})

const recordings = defineCollection({
  loader: file("src/data/recordings.json"),
  schema: recordingSchema,
})

const tools = defineCollection({
  loader: glob({
    pattern: "*.yaml",
    base: "src/content/tools",
    generateId: ({ entry }) => entry.replace(/\.yaml$/, ""),
  }),
  schema: toolContentSchema,
})

const contexts = defineCollection({
  loader: file("src/content/contexts.yaml"),
  schema: contextContentSchema,
})

export const collections = { definitions, recordings, tools, contexts }
