---
id: ADR-0005
title: A brand and a static website
type: adr
status: accepted
created: 2026-10-20
updated: 2026-10-20
decided: 2026-10-20
changes: [GIT-004, BLD-001, SPC-002]
supersedes:
superseded_by:
---

# A brand and a static website

## Context

Georgia MCP has no website and no brand. No guideline governs a page that a person
reads. The owner wants a site with its own character, in English and in Georgian,
with a page for each tool.

## Decision

Add the guideline `BRD` for the brand. Build the website in `website/` as a static
Astro site, with English at `/` and Georgian at `/ka/`.

## Rationale

- Astro writes the HTML at build time. A crawler and a link preview read the content
  without a script.
- An interactive part is an island. Only an island sends JavaScript to the browser.
- The i18n routing of Astro gives the two languages with no extra library.
- A content collection validates the tool data with a schema at build time, and
  gives one static page for each tool.
- An island can use React, so a component library for React stays available.

## Alternatives

| Alternative                   | Why we did not select it                                             |
| ----------------------------- | -------------------------------------------------------------------- |
| A React application with Vite | No HTML before the script runs. A second page needs a router.        |
| React and Vite with prerender | The HTML exists, but the browser still loads and runs the full tree. |

## Consequences

- Positive: The site has rules that a review can check.
- Positive: The browser loads JavaScript for the islands only.
- Negative: The repository gains a Node toolchain beside Go.
- Work that follows:
  - Add the scope `website` for `website/` to `GIT-004`.
  - Add a row for `website/` to `BLD-001`: `npm run lint`, then `npm test`, then `npm run build`.
  - Add the context value `website` to `SPC-002`.
  - Write the requirements of the feature `WEBSITE` with `/sdd-requirements`.
