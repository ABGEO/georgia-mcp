import sitemap from "@astrojs/sitemap"
import { defineConfig, fontProviders } from "astro/config"

export default defineConfig({
  site: "https://geo-mcp.abgeo.cloud",
  output: "static",
  trailingSlash: "ignore",
  i18n: {
    locales: ["en", "ka"],
    defaultLocale: "en",
    routing: {
      prefixDefaultLocale: false,
      redirectToDefaultLocale: false,
    },
  },
  fonts: [
    {
      provider: fontProviders.fontsource(),
      name: "Fraunces",
      cssVariable: "--font-display",
      weights: ["600"],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["Georgia", "serif"],
    },
    {
      provider: fontProviders.fontsource(),
      name: "Geist",
      cssVariable: "--font-body",
      weights: ["400 700"],
      styles: ["normal"],
      subsets: ["latin", "latin-ext"],
      fallbacks: ["system-ui", "sans-serif"],
    },
    {
      provider: fontProviders.fontsource(),
      name: "JetBrains Mono",
      cssVariable: "--font-mono",
      weights: ["400 600"],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["ui-monospace", "monospace"],
    },
    {
      provider: fontProviders.fontsource(),
      name: "Noto Sans Georgian",
      cssVariable: "--font-georgian",
      weights: ["400 700"],
      styles: ["normal"],
      subsets: ["georgian"],
      fallbacks: ["sans-serif"],
    },
  ],
  integrations: [
    sitemap({
      i18n: {
        defaultLocale: "en",
        locales: { en: "en", ka: "ka" },
      },
    }),
  ],
})
