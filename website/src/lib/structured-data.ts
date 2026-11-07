import {
  AUTHOR_GITHUB_URL,
  AUTHOR_NAME,
  AUTHOR_URL,
  LICENSE_URL,
  REPO_URL,
  SITE_NAME,
  SITE_URL,
} from "../site"
import { releaseVersion } from "./release"
import type { Lang } from "./schemas"

/** A JSON-LD document of schema.org with a graph of nodes. */
export type StructuredData = { "@context": "https://schema.org"; "@graph": Array<Record<string, unknown>> }

/** The JSON-LD graph of a home page. */
export function structuredData(lang: Lang, version: string, description: string): StructuredData {
  const home = lang === "en" ? `${SITE_URL}/` : `${SITE_URL}/${lang}/`
  const author = { "@id": `${SITE_URL}/#author` }
  const app = { "@id": `${SITE_URL}/#app` }
  const softwareVersion = releaseVersion(version)

  return {
    "@context": "https://schema.org",
    "@graph": [
      { ...author, "@type": "Person", name: AUTHOR_NAME, url: AUTHOR_URL, sameAs: [AUTHOR_GITHUB_URL] },
      { "@id": `${home}#website`, "@type": "WebSite", name: SITE_NAME, url: home, inLanguage: lang, publisher: author },
      {
        ...app,
        "@type": "SoftwareApplication",
        name: SITE_NAME,
        description,
        inLanguage: lang,
        applicationCategory: "DeveloperApplication",
        url: home,
        license: LICENSE_URL,
        offers: { "@type": "Offer", price: "0", priceCurrency: "GEL" },
        author,
        ...(softwareVersion && { softwareVersion }),
      },
      {
        "@id": `${SITE_URL}/#source`,
        "@type": "SoftwareSourceCode",
        codeRepository: REPO_URL,
        programmingLanguage: "Go",
        license: LICENSE_URL,
        author,
        targetProduct: app,
      },
    ],
  }
}
