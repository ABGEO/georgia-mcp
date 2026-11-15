---
id: MATSNE
title: The legal acts of the Legislative Herald, the scenarios
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: law
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [TST]
requirements: features/matsne/requirements.md
---

# Specification: The legal acts of the Legislative Herald, the scenarios

`pkg/matsne/testdata/` holds the pages that the tests serve. Step 1 of the build plan
records each one with the client. Each page has its inline scripts removed.

| File                       | Content |
| -------------------------- | ------- |
| `search-tax-code.html`     | The real page of `query=საგადასახადო კოდექსი&query_target=title&type=main&status=active`: total 29, 20 items. |
| `search-labour-law.html`   | The real page of `query=შრომის კოდექსი&type=main&status=active&group=1000003`. |
| `search-registration.html` | The real page of `registration_code=200000000.05.001.016012&type=main`: one item, `panel-success`, 1043717. |
| `search-en.html`           | The real page of `/en/document/search?query=Labour Code&type=main&status=active`: total 1, 1155567, with the links of its languages. |
| `search-empty.html`        | The real page of `query=zzqxv&type=main`: total 0. |
| `search-bad-date.html`     | A copy of `search-labour-law.html`. The first date is `2021-01-28`. |
| `suggest-tax-code.json`    | The real answer of Suggest for `საგადასახადო კოდექსი`: 45 items. |
| `act-1043717-ka.html`      | The real page of the Tax Code in `ka`, version 248, cut to sections I and II and to articles 133 to 134 with each heading above them. At the end: two real paragraphs of article 82 (a sup of a zero width space before a sup `1`, and the sup `1&nbsp;`), and the real paragraphs of chapter X (an empty `tavixml`) and of article 288^3 (a zero width space), recorded on 2026-11-15. Every option of `#publication-switcher` stays. |
| `act-1043717-en.html`      | The real page `/en/document/view/1043717` with no version: version 247, cut in the same way. At the end: the real paragraph of `SECTION X`, a soft hyphen, and `– (Deleted)`. |
| `act-6811279-ka.html`      | The real page of 6811279 in `ka`: an order with no `#publication-switcher`, published on 13/03/2026. |
| `act-1043717-en-248.html`  | The real page of version 248 in `en`: the consolidation is underway, no paragraph in `#maindoc`. |
| `act-5441237-ka.html`      | The real page of 5441237 in `ka`. No part class. |
| `act-5441237-en.html`      | The real page of 5441237 in `en`: a Georgian title under the English page. |
| `blocked.html`             | The real page of the filter: 840 bytes, `<title>Access Denied</title>`. |

The tests of the use cases take `now` as 2026-11-15T12:00:00+04:00. A fake `ActSource`
serves the tests of `app`.

## The client

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-001` | `MATSNE-NFR-005` | unit | A server records each request. `Search` with page 2 sends `GET /ka/document/search` with `page=1` and no empty parameter. `Suggest` sends `POST /document/suggest` with the form field `text`. `Document` with publication 200 sends `?publication=200`, and with -1 sends no query. Each call carries a User-Agent that starts with `georgia-mcp/` and holds `https://github.com/abgeo/georgia-mcp`, and `Accept: text/html`. The default transport sends the TLS hello of Chrome over HTTP/1.1 to a test server. |
| `MATSNE-SC-002` | `MATSNE-FR-001` | unit | `search-tax-code.html` gives `Total` `29` and 20 items. Each item has an ID of digits, a title, a class, and three or four fields with no `●`. `search-en.html` gives `Total` `1`, and the item 1155567 has the 4 fields `Organic Law of Georgia`, `Parliament of Georgia`, `N4113-რს`, `17/12/2010`, with no language link. `search-empty.html` gives `Total` `0` and no item. In `search-tax-code.html`, the title of 4740176, whose page holds `<b>საგადასახადო კოდექსი</b>ს`, holds the word `კოდექსის` whole. |
| `MATSNE-SC-003` | `MATSNE-FR-016` | unit | `blocked.html` with 200 gives `ErrBlocked` for each method. A 500 gives an `*APIError` with 500 that wraps `ErrUnexpectedStatus`. A page of 200 with no metadata table gives `ErrMalformedPage`. |
| `MATSNE-SC-004` | `MATSNE-FR-007`, `MATSNE-FR-009` | unit | `act-1043717-ka.html` gives the title `საქართველოს საგადასახადო კოდექსი`, 6 or more meta rows with `დოკუმენტის ნომერი` and `3591`, 249 publications from `248` `28/10/2026` to `0` `12/10/2010`, `Selected` `248`. A paragraph of class `muxlixml` has the text `მუხლი 133^3. ...`. No paragraph holds CSS text. One paragraph holds `ბ^1) იმ პირის მიერ`, and one holds `კანონის 34^1 მუხლის შესაბამისად`. `act-1043717-en.html` gives `Selected` `247` and a bold paragraph that starts with `Article 133^3 – `. `act-1043717-en-248.html` gives `Selected` `248` and no paragraph. |
| `MATSNE-SC-005` | `MATSNE-FR-001`, `MATSNE-FR-016` | live | The client with its default transport against `matsne.gov.ge`: `Document` of 1043717 in `ka` has 249 or more publications. `Search` for `შრომის კოდექსი` with the group 1000003 has 1 or more items. `Suggest` for `საგადასახადო კოდექსი` holds 1043717. No call gives `ErrBlocked`. |

## The domain

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-006` | `MATSNE-FR-009`, `MATSNE-FR-015` | unit | `ParseArticleNumber`: `82` gives `82`. `133^3` and `133³` give `133^3`. `82a`, `0`, `082`, `133^`, `^3`, `12345`, and the empty text give `ErrInvalidArticleNumber`. A fuzz test holds that `String` of a result parses to the same value. |
| `MATSNE-SC-007` | `MATSNE-FR-015` | unit | `ParseActID`: `1043717` parses. `abc`, `0`, `01`, and 11 digits give `ErrInvalidActID`. `ParseDocumentType`: each of the 16 codes parses, `statute` gives `ErrUnknownDocumentType`. `ParseActStatus`: `active`, `expired`, `pending` parse, `toBeActivated` gives `ErrUnknownActStatus`. A fuzz test of each parser holds that a result is a known value. |
| `MATSNE-SC-008` | `MATSNE-FR-011` | unit | Versions 0 (2010-10-12), 1 (2011-01-01), 2 (2026-04-01), 3 (2026-04-01), 4 (2026-10-28). `VersionOn` 2025-01-01 gives 1. 2026-04-01 gives 3. 2027-01-01 gives 4. 2010-10-11 gives false. |

## The adapter

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-009` | `MATSNE-FR-001`, `MATSNE-FR-002`, `MATSNE-FR-003`, `MATSNE-FR-005` | unit | `Search` with `InText` sends `query_target=text`, and without it `query_target=title`. `Search` with the type `law`, the status `active`, the dates 2010-01-01 to 2026-11-15, and `OldestFirst` sends `type=main`, `group=1000003`, `status=active`, `signing_date_fr[date]=01/01/2010`, `signing_date_to[date]=15/11/2026`, `sort=signingDate_asc`. With `search-labour-law.html`, the first summary has the type `საქართველოს ორგანული კანონი`, the issuer `საქართველოს პარლამენტი`, the number `N101-IIIრს-Xმპ`, the date 2021-01-28, and `active`. |
| `MATSNE-SC-010` | `MATSNE-FR-008`, `MATSNE-FR-009`, `MATSNE-INV-001` | unit | `Act` with `act-1043717-ka.html` gives the version 248 of 2026-10-28, 249 versions newest first, and contents that start with section `I` `ზოგადი დებულებანი`, chapter `I` `საქართველოს საგადასახადო სისტემა`, article `1`. Article `133^3` has the title `ცალკეული საქონლის მიმწოდებელი ფიზიკური პირის შემოსავლის დაბეგვრა` and 5 paragraphs: three numbered paragraphs and two notes of an amendment, each as on the page. A page with an empty `tavixml` and the title paragraph `თავი X. (ამოღებულია)` gives chapter `X`. A heading `მუხლი 288`, a zero width space, and a sup `3` gives article `288^3`. |
| `MATSNE-SC-011` | `MATSNE-FR-012`, `MATSNE-FR-008` | unit | `Act` in `en` with `act-1043717-en.html` gives the version 247 of 2026-10-15, the first of the versions 248, the title `TAX CODE OF GEORGIA`, section `I` `GENERAL PROVISIONS`, chapter `I` `Georgian Tax System`, and article `133^3` that starts with `Taxation of the income of a natural person`. A paragraph that cites "Article 281 of this Code" opens no article. A bold `SECTION X`, a soft hyphen, and `– (Deleted)` give section `X` with the title `(Deleted)`. |
| `MATSNE-SC-012` | `MATSNE-FR-010` | unit | `Act` with `act-5441237-ka.html` gives no article, no contents, and a `Text` of more than 20,000 runes that starts with `შემოსავლების სამსახურის უფროსის`. |
| `MATSNE-SC-013` | `MATSNE-FR-013` | unit | `Act` in `en` with `act-5441237-en.html` gives `ErrNoTranslation`. In `ka` with `act-5441237-ka.html` it gives the act. In `en` with `act-1043717-en.html` it gives the act. In `en` with `act-1043717-en-248.html` it gives `ErrNoTranslation`. |
| `MATSNE-SC-014` | `MATSNE-FR-016`, `MATSNE-FR-007` | unit | `blocked.html`, a 500, and `search-bad-date.html` each give an error that wraps `kernel.ErrSourceUnavailable`. `Act` with `blocked.html` gives an error that wraps `app.ErrUnknownAct` and `kernel.ErrSourceUnavailable`. `Search` with `blocked.html` gives one that does not wrap `app.ErrUnknownAct`. `Status` with `search-registration.html` gives `active`, with `search-empty.html` the zero status. |
| `MATSNE-SC-034` | `MATSNE-FR-007`, `MATSNE-FR-008` | unit | `Act` with `act-6811279-ka.html` gives the number `38`, the signing date 2026-03-12, and one version 0 of 2026-03-13 as the version and as the list. |

## The use cases

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-015` | `MATSNE-FR-004` | unit | A fake source whose page holds 20 acts without 1043717, and whose `Titles` give 32842 expired, 31690 expired, and 1043717 active, each titled `საქართველოს საგადასახადო კოდექსი`. `Search` with `„საქართველოს საგადასახადო კოდექსი“` gives 1043717 first, with the type and the issuer of its `Act`, then 19 acts of the page under the limit 20. `Total` is the total of the page. |
| `MATSNE-SC-016` | `MATSNE-FR-004` | unit | The same fake. `Search` with `საგადასახადო კოდექსი` keeps the order of the page. Page 2, the target `text`, the type `law`, or an English query calls no `Titles`. When `Titles` fails, the page of the source comes back with no error. When the page holds 1043717, it appears once, first. |
| `MATSNE-SC-017` | `MATSNE-FR-009`, `MATSNE-FR-015` | unit | An act with articles 1 to 309 and `133^3`. `Get` with `82` and `1` gives `1` then `82`. `999` gives an `*ArticleError` with 1 and 309. 11 articles give `ErrTooManyArticles` and call no source. |
| `MATSNE-SC-018` | `MATSNE-NFR-001` | unit | Articles 1, 2, and 3 of 25,000 runes each. `Get` with the three gives 1 and 2, and `Next` 3. One article of 70,000 runes in 10 paragraphs gives the paragraphs that fit in 60,000, `Truncated` true, and no `Next`. |
| `MATSNE-SC-019` | `MATSNE-FR-010` | unit | A text of 76,000 runes with a new line each 1,000. Piece 1 has 50,000 runes or less and ends after a new line. `Pieces` is 2. Piece 3 gives a `*PieceError` with 2. |
| `MATSNE-SC-020` | `MATSNE-FR-011` | unit | The versions of `MATSNE-SC-008`. `Get` with 2025-01-01 reads the newest, then version 1. With 2027-01-01 it reads the newest once. A version and a date give `ErrBothVersions` and call no source. Version 9 and 2010-10-11 each give a `*VersionError` with 0 and 4, after one read of the newest version only. |
| `MATSNE-SC-021` | `MATSNE-FR-007`, `MATSNE-FR-008` | unit | `Get` with no article gives the metadata, the 5 versions, the contents, and the status of `Status`. `Get` with an article gives no versions and no contents. |
| `MATSNE-SC-036` | `MATSNE-FR-007` | unit | A fake act of 25 versions, 24 to 0. `Get` with no article gives the versions 24 to 5 and `VersionCount` 25. |
| `MATSNE-SC-033` | `MATSNE-FR-018` | unit | A fake act in `en` whose text is version 4 of 5 versions. `Get` with no version gives the text of version 4 and `Newest` version 5. `Get` in `ka` with the text of version 5 gives no `Newest`. `Get` in `en` with the version 5, whose read gives `ErrNoTranslation`, returns that error. `Get` in `ka` with the date 2025-01-01 or the version 1 gives no `Newest`. |
| `MATSNE-SC-035` | `MATSNE-FR-002`, `MATSNE-FR-007` | unit | A source whose `Titles` fails gives the page of `Search` and one record `source unavailable` at `warn` with the error. A source whose `Status` fails gives the act with no status and one such record. |
| `MATSNE-SC-022` | `MATSNE-NFR-004` | unit | In `synctest`, each method of a fake source waits 4.9 seconds. `Search` on the exact title path returns in less than 15 seconds. `Get` with a date and an article returns in less than 15 seconds. |

## The cache

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-023` | `MATSNE-NFR-002` | unit | In `synctest`: `Search`, `Titles`, `Status`, and `Act` with no version each send one load at 59 minutes and a second at 1 hour. `Act` with a version sends one load at 23 hours 59 minutes and a second at 24 hours. With a source that fails with `kernel.ErrSourceUnavailable`, each returns the value with `Stale` true at 6 days 23 hours 59 minutes, and the error at 7 days and 1 second. |
| `MATSNE-SC-024` | `MATSNE-NFR-002` | unit | `ErrUnknownAct` and `ErrNoTranslation` pass through. A second call with the same key loads again. |

## The tools

An integration test serves the recorded pages from `httptest` and calls each tool through
the in-memory MCP transport.

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-025` | `MATSNE-FR-001`, `MATSNE-FR-006`, `MATSNE-FR-012` | integration | `law_search_acts` with `შრომის კოდექსი`, the type `law`, and `ka`: `source` is `Legislative Herald of Georgia`, `as_of` has `+04:00`, `page` is 1, `total` is `3`, the total of the page, with 3 acts, each act has a `url` of `https://matsne.gov.ge/ka/document/view/{id}`. |
| `MATSNE-SC-026` | `MATSNE-FR-007`, `MATSNE-FR-008`, `MATSNE-INV-002` | integration | `law_get_act` with `1043717` and `ka`: `id`, `title`, `number` `3591`, `signed` `2010-09-17`, `status` `active`, `version` 248 of `2026-10-28`, 20 `versions` from 248, `version_count` 249, `contents` that start with the section `I`, `official` true, no `notice`, and a `url` with `?publication=248`. `law_get_act` with `6811279` and `ka`: `version` 0 of `2026-03-13`, that one version in `versions`, `version_count` 1, and a `url` with `?publication=0`. |
| `MATSNE-SC-027` | `MATSNE-FR-009`, `MATSNE-FR-014`, `MATSNE-INV-001` | integration | `law_get_act` with `1043717`, the articles `133^3`, and no language: one article `133^3` whose paragraphs equal the paragraphs of the page, `version` 247, `newest_version` 248 of `2026-10-28`, `official` false, and the `notice` of `spec-tools.md` with both sentences. |
| `MATSNE-SC-028` | `MATSNE-FR-013`, `MATSNE-FR-015` | integration | Each input of the errors of `spec-tools.md` returns `IsError` with its message: `ru`, an empty query, 129 characters, `body`, `statute`, `valid`, `5 Nov`, `limit` 50, page -1, the order `x`, `abc`, `82a`, 12 articles, piece -1, article 1 of 5441237, `999`, a version and a date, version 300, piece 3, 5441237 in `en`, and an act that the vendor answers with `blocked.html`. |
| `MATSNE-SC-029` | `MATSNE-FR-016` | integration | The vendor answers 500, then `blocked.html`, with an empty cache. `law_search_acts` returns `IsError` with `The Legislative Herald did not answer. Try again in a few minutes.` A source that answers stale, as the cache does in `MATSNE-SC-023`, gives `stale` true in the output of each tool. |

## The module

| ID              | Verifies | Layer | Given, when, then |
| --------------- | -------- | ----- | ----------------- |
| `MATSNE-SC-030` | `MATSNE-NFR-003` | unit | In `bootstrap`, `LoadConfig` with no variable of law gives the Legislative Herald on, at `https://matsne.gov.ge`, with a timeout of 10 seconds, the rate 5, and the burst 5. |
| `MATSNE-SC-031` | `MATSNE-FR-017` | unit | `law.New` with `Enabled` false registers no tool. With `Enabled` true it registers `law_search_acts` and `law_get_act`. |
| `MATSNE-SC-032` | `MATSNE-FR-009` | manual | The owner asks a client "what does article 82 of the Tax Code of Georgia say". The model calls `law_search_acts`, then `law_get_act` with `82`, and quotes the article with its version date. |

## Retired identifiers

This file has no retired identifier.
