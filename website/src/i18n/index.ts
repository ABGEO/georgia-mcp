import type { Lang } from "../lib/schemas"
import { en, type Dictionary } from "./en"
import { ka } from "./ka"

const dictionaries: Record<Lang, Dictionary> = { en, ka }

/** Gives the copy of one language. */
export function t(lang: Lang): Dictionary {
  return dictionaries[lang]
}

/** Gives the address of a page in one language. The path has no language prefix and starts with a slash. */
export function localize(path: string, lang: Lang): string {
  return lang === "en" ? path : `/ka${path}`
}

/** Gives the language of the other version of a page. */
export function other(lang: Lang): Lang {
  return lang === "en" ? "ka" : "en"
}

/** Gives the address of the same page in the other language. */
export function alternate(path: string, lang: Lang): string {
  return localize(path, other(lang))
}
