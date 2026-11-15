---
id: TASPERM
title: The applications of the Tbilisi Architecture Service, the tools
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: permits
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [MCP]
requirements: features/tasperm/requirements.md
---

# Specification: The applications of the Tbilisi Architecture Service, the tools

The structs of `P/adapter/tool`. `spec.md` 4.3 holds the table of the tools.

`permits_search_applications`. Title: `Search the building applications of Tbilisi`. Description:

> Returns the applications to the Tbilisi Architecture Service for a parcel or a period,
> newest first: building permits, project approvals, signs, fences, and requests for public
> information. Give a cadastral code (01.18.04.020.005), a period of at most 31 days, or
> both. A period of more than 500 applications fails, and Tbilisi registers about 70 each
> working day, so ask for a week at most. Each application has its number, address,
> status, category, class, and stages. Read the decision with permits_get_application.
> Tbilisi only. Texts are in Georgian. Data from TAS.

`permits_get_application`. Title: `Get a building application of Tbilisi`. Description:

> Returns one application to the Tbilisi Architecture Service by its number (AR11168188):
> the address, the status, the category, the class, the stages, the deadline, the
> executor, the text of the decision, and the link to its page. Find a number with
> permits_search_applications. Texts are in Georgian. Data from TAS.

```go
type searchInput struct {
    Code  string `json:"code,omitempty"  jsonschema:"The cadastral code of the parcel: 4 or 5 groups of digits with dots, as 01.18.04.020.005."`
    From  string `json:"from,omitempty"  jsonschema:"The first date of registration, YYYY-MM-DD. Without to, the period is 7 days."`
    To    string `json:"to,omitempty"    jsonschema:"The last date of registration, YYYY-MM-DD. Without from, the period is 7 days."`
    Limit int    `json:"limit,omitempty" jsonschema:"The maximum count of applications, 1 to 50. The default is 10."`
}

type getInput struct {
    Number string `json:"number" jsonschema:"The application number, as AR11168188."`
}

type searchOutput struct {
    Source       string              `json:"source"       jsonschema:"The source of the data: TAS."`
    AsOf         string              `json:"as_of"        jsonschema:"The instant at which the server read the applications, RFC 3339."`
    Stale        bool                `json:"stale"        jsonschema:"True when TAS did not answer and the server returns applications it read before."`
    Total        int                 `json:"total"        jsonschema:"The count of all the applications that match."`
    Applications []applicationOutput `json:"applications" jsonschema:"The applications, newest first."`
}

type getOutput struct {
    applicationOutput

    Source       string `json:"source"                  jsonschema:"The source of the data: TAS."`
    AsOf         string `json:"as_of"                   jsonschema:"The instant at which the server read the application, RFC 3339."`
    Stale        bool   `json:"stale"                   jsonschema:"True when TAS did not answer and the server returns an application it read before."`
    DecisionText string `json:"decision_text,omitempty" jsonschema:"The text of the decision of TAS, in Georgian, with no HTML. Empty before a decision."`
}

type applicationOutput struct {
    Number         string     `json:"number"                    jsonschema:"The application number."`
    URL            string     `json:"url"                       jsonschema:"The page of the application on tas.ge."`
    Address        string     `json:"address,omitempty"         jsonschema:"The address, in Georgian, as TAS publishes it."`
    Registered     string     `json:"registered"                jsonschema:"The date of registration, YYYY-MM-DD."`
    Deadline       string     `json:"deadline,omitempty"        jsonschema:"The date by which TAS must answer, YYYY-MM-DD."`
    Status         codeOutput `json:"status"                    jsonschema:"approved, refused, deficient, conditional, not_reviewed, revoked, completed, awaiting_payment, or in_review."`
    Category       string     `json:"category,omitempty"        jsonschema:"The kind of object or work, in Georgian."`
    ParentCategory string     `json:"parent_category,omitempty" jsonschema:"The group of the category, in Georgian."`
    Class          string     `json:"class,omitempty"           jsonschema:"The class of risk, I to IV."`
    Stages         []string   `json:"stages,omitempty"          jsonschema:"The stages that the application covers, in Georgian."`
    Action         string     `json:"action,omitempty"          jsonschema:"The action, in Georgian, as a change of an approved project."`
    Executor       string     `json:"executor,omitempty"        jsonschema:"The employee of TAS who handles the application."`
}

type codeOutput struct {
    Code string `json:"code" jsonschema:"The code."`
    Name string `json:"name" jsonschema:"The name of the code at TAS, in Georgian."`
}
```

`source` is `TAS`. `as_of` is `ReadAt` in `kernel.TbilisiZone()`, written with
`time.RFC3339`. A `limit` of 0 asks the default. `Class` is `I` to `IV`: the adapter keeps
the first word of `className`. The tools take no `language`: TAS publishes Georgian only
(`MCP-007`). The names of the statuses:

| Code               | Name                    |
| ------------------ | ----------------------- |
| `approved`         | თანხმობა                |
| `refused`          | უარყოფა                 |
| `deficient`        | შუალედური               |
| `conditional`      | პირობადადებული          |
| `not_reviewed`     | განუხილველი             |
| `revoked`          | ძალადაკარგული           |
| `completed`        | დასრულებული             |
| `awaiting_payment` | გადახდის მოლოდინში      |
| `in_review`        | განხილვის პროცესში      |

The names are those of the page of TAS.

## Retired identifiers

This file has no retired identifier.
