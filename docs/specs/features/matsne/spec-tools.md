---
id: MATSNE
title: The legal acts of the Legislative Herald, the tools
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: law
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [MCP]
requirements: features/matsne/requirements.md
---

# Specification: The legal acts of the Legislative Herald, the tools

## MCP tools

| Tool              | Input | Output | Errors | Realizes |
| ----------------- | ----- | ------ | ------ | -------- |
| `law_search_acts` | `query` (required), `target`, `type`, `status`, `issuer`, `number`, `signed_from`, `signed_to`, `order`, `page`, `language`, `limit` | `searchOutput` | Errors | `MATSNE-FR-001` to `MATSNE-FR-006`, `MATSNE-FR-012`, `MATSNE-FR-015` |
| `law_get_act`     | `id` (required), `version`, `date`, `articles`, `piece`, `language` | `actOutput` | Errors | `MATSNE-FR-007` to `MATSNE-FR-015` |

`law_search_acts`. Title: `Search the legal acts of Georgia`. Description:

> Searches the legal acts of Georgia in the Legislative Herald (matsne.gov.ge): laws,
> codes, resolutions, decrees, orders, treaties, and court decisions. Returns 20 acts on
> a page, each with its identifier, title, document type, issuer, number, signing date,
> and status, and the total count. The source has no order by relevance: give a type,
> a status, or an issuer to bring an act to the first page. A Georgian query that equals
> the full title of an active act puts that act first. Search in Georgian for the best
> match; an English query matches the titles of translated acts only. Call law_get_act
> with the identifier to read the act. Data from the Legislative Herald of Georgia.

`law_get_act`. Title: `Get a legal act of Georgia`. Description:

> Returns one legal act of the Legislative Herald (matsne.gov.ge) by its identifier from
> law_search_acts. With no article, returns the metadata, every consolidated version with
> its publication date, and the table of contents. With articles, returns the full text
> of at most 10 articles, for example 82 or 133^3 (article 133 with the superscript 3).
> Give a version number or a date to read the act as it stood then; the default is the
> newest version. A version date is its publication date, not the date it took effect.
> The Georgian text is the only official text; an English text is a translation, and
> not every act has one. Data from the Legislative Herald of Georgia.

```go
type searchActsInput struct {
    Query      string `json:"query"                 jsonschema:"The text to find, 1 to 128 characters. Georgian finds every act; English finds translated acts only."`
    Target     string `json:"target,omitempty"      jsonschema:"Where to search: title or text. The default is title."`
    Type       string `json:"type,omitempty"        jsonschema:"A document type: law, parliament_resolution, presidential_decree, government_resolution, government_decree, edict, order, international_treaty, constitutional_court_decision, court_act, self_government_ordinance, self_government_decision, revenue_service_order, individual_legal_act, eu_law, or draft. A code is a law."`
    Status     string `json:"status,omitempty"      jsonschema:"The status of the act: active, expired, or pending."`
    Issuer     string `json:"issuer,omitempty"      jsonschema:"The body that adopted the act, in the language of the request, for example Parliament of Georgia."`
    Number     string `json:"number,omitempty"      jsonschema:"The number of the act, for example 3591."`
    SignedFrom string `json:"signed_from,omitempty" jsonschema:"The first signing date, YYYY-MM-DD."`
    SignedTo   string `json:"signed_to,omitempty"   jsonschema:"The last signing date, YYYY-MM-DD."`
    Order      string `json:"order,omitempty"       jsonschema:"newest or oldest signing date first. The default is newest."`
    Page       int    `json:"page,omitempty"        jsonschema:"The page of 20 results, from 1. The default is 1."`
    Language   string `json:"language,omitempty"    jsonschema:"The language of the titles: ka or en. The default is en."`
    Limit      int    `json:"limit,omitempty"       jsonschema:"The maximum count of acts of the page, 1 to 20. The default is 20."`
}

type getActInput struct {
    ID       string   `json:"id"                 jsonschema:"The identifier of the act from law_search_acts, for example 1043717."`
    Version  *int     `json:"version,omitempty"  jsonschema:"A version number from the versions of the act, from 0. Give a version or a date, not both."`
    Date     string   `json:"date,omitempty"     jsonschema:"A date, YYYY-MM-DD. Reads the newest version published on or before it."`
    Articles []string `json:"articles,omitempty" jsonschema:"At most 10 article numbers, for example 82 or 133^3. Leave empty for the table of contents."`
    Piece    int      `json:"piece,omitempty"    jsonschema:"For an act with no articles: the piece of 50,000 characters of its text, from 1. The default is 1."`
    Language string   `json:"language,omitempty" jsonschema:"The language of the text: ka or en. The default is en. Only ka is official."`
}

type searchOutput struct {
    Source string       `json:"source" jsonschema:"The source of the data: Legislative Herald of Georgia."`
    AsOf   string       `json:"as_of"  jsonschema:"The instant at which the server read the results, RFC 3339."`
    Stale  bool         `json:"stale"  jsonschema:"True when the source did not answer and the server returns results it read before."`
    Total  int          `json:"total"  jsonschema:"The count of matching acts at the source."`
    Page   int          `json:"page"   jsonschema:"The page of 20 results."`
    Acts   []actSummary `json:"acts"   jsonschema:"The acts of the page."`
}

type actSummary struct {
    ID     string `json:"id"               jsonschema:"The identifier of the act."`
    Title  string `json:"title"            jsonschema:"The title of the act."`
    Type   string `json:"type,omitempty"   jsonschema:"The document type as the source names it."`
    Issuer string `json:"issuer,omitempty" jsonschema:"The body that adopted the act."`
    Number string `json:"number,omitempty" jsonschema:"The number of the act."`
    Signed string `json:"signed,omitempty" jsonschema:"The signing date, YYYY-MM-DD."`
    Status string `json:"status,omitempty" jsonschema:"active, expired, or pending."`
    URL    string `json:"url"              jsonschema:"The page of the act at the Legislative Herald."`
}

type actOutput struct {
    Source    string          `json:"source"              jsonschema:"The source of the data: Legislative Herald of Georgia."`
    AsOf      string          `json:"as_of"               jsonschema:"The instant at which the server read the act, RFC 3339."`
    Stale     bool            `json:"stale"               jsonschema:"True when the source did not answer and the server returns a text it read before."`
    URL       string          `json:"url"                 jsonschema:"The page of this version at the Legislative Herald."`
    Language  string          `json:"language"            jsonschema:"ka or en."`
    Official  bool            `json:"official"            jsonschema:"True for the Georgian text. False for a translation."`
    Notice    string          `json:"notice,omitempty"    jsonschema:"Set for a translation: only the Georgian text is official."`
    ID        string          `json:"id"                  jsonschema:"The identifier of the act."`
    Title     string          `json:"title"               jsonschema:"The title of the act."`
    Type      string          `json:"type,omitempty"      jsonschema:"The document type as the source names it."`
    Issuer    string          `json:"issuer,omitempty"    jsonschema:"The body that adopted the act."`
    Number    string          `json:"number,omitempty"    jsonschema:"The number of the act."`
    Signed    string          `json:"signed,omitempty"    jsonschema:"The signing date, YYYY-MM-DD."`
    Status    string          `json:"status,omitempty"    jsonschema:"active, expired, or pending. Empty when the source gives none."`
    Registration string       `json:"registration_code,omitempty" jsonschema:"The registration code of the act."`
    Version   versionOutput   `json:"version"             jsonschema:"The version of this text."`
    Versions  []versionOutput `json:"versions,omitempty"  jsonschema:"Every version, newest first. Only when no article is requested."`
    Contents  []headingOutput `json:"contents,omitempty"  jsonschema:"The sections, chapters, and articles, in order. Only when no article is requested."`
    Articles  []articleOutput `json:"articles,omitempty"  jsonschema:"The requested articles, in the order of the act."`
    Next      string          `json:"next_article,omitempty" jsonschema:"The first requested article left out by the size limit. Ask for it in a new call."`
    Truncated bool            `json:"truncated,omitempty" jsonschema:"True when one article passed the size limit and its last paragraphs are left out."`
    Text      string          `json:"text,omitempty"      jsonschema:"The text of an act that has no articles: one piece."`
    Piece     int             `json:"piece,omitempty"     jsonschema:"The piece of the text, from 1."`
    Pieces    int             `json:"pieces,omitempty"    jsonschema:"The count of pieces of the text."`
}

type versionOutput struct {
    Number int    `json:"number" jsonschema:"The version number, from 0."`
    Date   string `json:"date"   jsonschema:"The publication date of the version, YYYY-MM-DD."`
}

type headingOutput struct {
    Kind   string `json:"kind"   jsonschema:"section, chapter, or article."`
    Number string `json:"number" jsonschema:"The number, for example IV or 133^3."`
    Title  string `json:"title"  jsonschema:"The title of the part."`
}

type articleOutput struct {
    Number     string   `json:"number"     jsonschema:"The article number, for example 133^3."`
    Title      string   `json:"title"      jsonschema:"The title of the article."`
    Paragraphs []string `json:"paragraphs" jsonschema:"The paragraphs of the article as published, with the notes of each amendment."`
}
```

`source` is `Legislative Herald of Georgia`. `as_of` is `ReadAt` in `kernel.TbilisiZone()`,
written with `time.RFC3339`. `url` is `https://matsne.gov.ge/{language}/document/view/{id}`,
with `?publication={n}` in `actOutput`. A `limit` or a `page` of 0 asks the default.
`SearchActs` puts the exact title matches first, then applies `limit`.

## Errors

| Condition                                  | Sentinel                          | Tool message |
| ------------------------------------------ | --------------------------------- | ------------ |
| A language other than `ka` and `en`        | `kernel.ErrUnknownLanguage`       | `language: "ru" is not a language of this tool. Use ka or en.` |
| An empty query, or more than 128 characters | none, the tool checks it         | `query: give 1 to 128 characters.` |
| An unknown target or order                 | none, the tool checks it          | `target: "body" is not a target. Use title or text.` |
| An unknown document type                   | `domain.ErrUnknownDocumentType`   | `type: "statute" is not a document type. Use one of: law, parliament_resolution, ...` |
| An unknown status                          | `domain.ErrUnknownActStatus`      | `status: "valid" is not a status. Use active, expired, or pending.` |
| A date that does not parse                 | `kernel.ErrInvalidDate`           | `signed_from: "5 Nov" is not a date. Use YYYY-MM-DD.` |
| A limit or a page out of range             | none, the tool checks it          | `limit: 50 is out of range. Use 1 to 20.` |
| A malformed act identifier                 | `domain.ErrInvalidActID`          | `id: "abc" is not an act identifier. Call law_search_acts to find the identifier.` |
| An unknown act, or a refusal of the source (`MATSNE-DD-015`) | `app.ErrUnknownAct` | `id: the Legislative Herald returned no act 999999999. Check the identifier with law_search_acts, or try again in a few minutes.` |
| A malformed article number                 | `domain.ErrInvalidArticleNumber`  | `articles: "82a" is not an article number. Use digits, with ^ and digits for an inserted article, for example 133^3.` |
| More than 10 articles                      | `app.ErrTooManyArticles`          | `articles: 12 articles. Give at most 10.` |
| An article that the act does not have      | `*app.ArticleError`               | `articles: the act has no article 999. Its articles run from 1 to 309.` |
| A version and a date                       | `app.ErrBothVersions`             | `version, date: give a version or a date, not both.` |
| An unknown version or a date before version 0 | `*app.VersionError`            | `version: the act has versions 0 (2010-10-12) to 248 (2026-10-28).` |
| A piece out of range                       | `*app.PieceError`                 | `piece: the text has 2 pieces.` |
| No English translation                     | `app.ErrNoTranslation`            | `The act has no English translation at the Legislative Herald. Ask for language ka.` |
| The source fails, with no stale value (not `app.ErrUnknownAct`) | `kernel.ErrSourceUnavailable`     | `The Legislative Herald did not answer. Try again in a few minutes.` |
| Any other error                            | none                              | `The server failed to read the data of the Legislative Herald.` |

## Retired identifiers

This file has no retired identifier.
