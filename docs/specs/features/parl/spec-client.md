---
id: PARL
title: The bills, votes, and questions of the Parliament of Georgia, the client
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: parliament
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [CLT, DOM]
requirements: features/parl/requirements.md
---

# Specification: The bills, votes, and questions of the Parliament of Georgia, the client

## 1. `pkg/parliament`

The shape of `pkg/nea`. Each list method sends `start` and `limit`.

```go
const DefaultBaseURL = "https://info.parliament.ge"

var ErrUnexpectedStatus = errors.New("parliament: unexpected status")

type APIError = vendorhttp.StatusError

// A date of a query is dd-MM-yyyy. An empty field is not sent.
type BillQuery struct {
    Start, Limit       int
    BillName           string // billName
    RegistrationNumber string // registrationNumber
    RegistrationDate1  string // registrationDate1
    RegistrationDate2  string // registrationDate2
}

type VoteSessionQuery struct {
    Start, Limit  int
    BillRd        string // billRd: the title of the bill
    BillReviewID  int    // billReviewId, 0 is not sent
    VotingDate1   string // votingDate1
    VotingDate2   string // votingDate2
}

type MPQuestionQuery struct {
    Start, Limit      int
    RequestTitle      string // requestTitle: the text of the request
    ColleagueName     string // colleagueName
    RegistrationDate1 string
    RegistrationDate2 string
}

func (c *Client) Bills(ctx context.Context, query BillQuery) (BillList, error)             // GET /law/1/bill
func (c *Client) Bill(ctx context.Context, id int) (*Bill, error)                          // GET /law/1/bill/{id}
func (c *Client) BillReviews(ctx context.Context, billID int) ([]BillReview, error)        // GET /law/1/bill/{id}/billReview
func (c *Client) VoteSessions(ctx context.Context, query VoteSessionQuery) (VoteSessionList, error) // GET /law/1/voteSession
func (c *Client) VoteSession(ctx context.Context, id int) (*VoteSession, error)            // GET /law/1/voting/session/{id}
func (c *Client) Votings(ctx context.Context, voteSessionID, start, limit int) (VotingList, error) // GET /law/1/voting, group=false
func (c *Client) MPQuestions(ctx context.Context, query MPQuestionQuery) (MPQuestionList, error)    // GET /mpQuestion/1/
```

`Bill` and `VoteSession` return nil and no error for the answer `[null]` or `[]`. The types
mirror the fields that the adapter reads, with the JSON names of the source:

```go
type BillList struct { Total int `json:"total"`; List []Bill `json:"list"` }

type Bill struct {
    ID              int           `json:"id"`
    BillName        string        `json:"billName"`
    BillDescription *string       `json:"billDescription"`
    BillPackage     BillPackage   `json:"billPackage"`
    BillType        Library       `json:"billType"`
    BillReviewTypes []Library     `json:"billReviewTypes"`
}

type BillPackage struct {
    ID                 int    `json:"id"`
    RegistrationNumber string `json:"registrationNumber"`
    PackageName        string `json:"packageName"`
    Initiator          string `json:"initiator"`
    Author             string `json:"author"`
    RegistrationDate   string `json:"registrationDate"` // "30-09-2026"
    BillCount          int    `json:"billCount"`
}

type Library struct { ID int `json:"id"`; Name string `json:"name"` }

type BillReview struct {
    ID                 int     `json:"id"`
    Index              int     `json:"index"`
    ReviewTypeName     string  `json:"reviewTypeName"`
    SessionNumber      *string `json:"sessionNumber"`
    BureauNumber       *string `json:"bureauNumber"`
    SessionReviewDate  *string `json:"sessionReviewDate"`
    BureauReviewDate   *string `json:"bureauReviewDate"`
    BillReviewFromDate *string `json:"billReviewFromDate"`
    BillReviewToDate   *string `json:"billReviewToDate"`
    VoteY              *int    `json:"voteY"`
    VoteN              *int    `json:"voteN"`
}

type VoteSessionList struct { Total int `json:"total"`; List []VoteSession `json:"list"` }

type VoteSession struct {
    ID                    int             `json:"id"`
    Quorum                *int            `json:"quorum"`
    VotingConclusionType  *Library        `json:"votingConclusionType"`
    ImportConferenceDate  string          `json:"importConferenceDate"` // "11-11-2026"
    BillReviewTypeRd      string          `json:"billReviewTypeRd"`
    BillIDRd              int             `json:"billIdRd"`
    BillRd                string          `json:"billRd"`
    RegistrationNumberRd  string          `json:"registrationNumberRd"`
    VotingResultSet       VotingResultSet `json:"votingResultSet"`
}

type VotingResultSet struct { Yes int `json:"yes"`; No int `json:"no"`; WithoutInteraction int `json:"withoutInteraction"` }

type VotingList struct { Total int `json:"total"`; List []Voting `json:"list"` }

type Voting struct {
    VotingResult *Library `json:"votingResult"`
    ColleagueRd  string   `json:"colleagueRd"`
    FactionRd    *string  `json:"factionRd"`
}

type MPQuestionList struct { Total int `json:"total"`; List []MPQuestion `json:"list"` }

type MPQuestion struct {
    ID                 int      `json:"id"`
    Type               *Library `json:"type"`
    RegistrationNumber string   `json:"registrationNumber"`
    RegistrationDate   string   `json:"registrationDate"`
    Request            string   `json:"request"`
    Target             *string  `json:"target"`
    TargetOrganisation *string  `json:"targetOrganisation"`
    TargetPerson       *string  `json:"targetPerson"`
    Deadline1          *string  `json:"deadline1"`
    ResponseDate       *string  `json:"responseDate"`
    ColleagueName      *string  `json:"colleagueName"`
    NodeName           *string  `json:"nodeName"`
}
```

The body limit stays the default 1 MiB. The largest answer is a page of 50 bills (193 KB)
or the choices of 151 members (about 270 KB): each choice repeats the whole vote.

## 2. The recorded answers

Real answers, recorded on 2026-11-15, cut:

```
GET /law/1/bill?start=0&limit=1&billName=ბიუჯეტ
200 application/json;charset=UTF-8
{"total":1088,"list":[{"id":32559,"billName":"საქართველოს პარლამენტის საფინანსო-საბიუჯეტო კომიტეტის დასკვნა ...",
  "billPackage":{"id":23209,"registrationNumber":"2-15416/26/11","initiator":"...","registrationDate":"11-11-2026",...},
  "billType":{"id":1,"name":"საქართველოს კანონის პროექტი"},"billReviewTypes":[...]}]}

GET /law/1/bill/99999999                       200 [null]
GET /law/1/bill?limit=51&start=0               200, empty body
GET /law/1/bill?searchItem=abc&limit=1         404 text/html (no start)

GET /law/1/voteSession?start=0&limit=1&billReviewId=114284
{"total":1,"list":[{"id":31510,"quorum":50,"votingConclusionType":{"id":1,"name":"საკითხი მიღებულია"},
  "importConferenceDate":"11-11-2026","billReviewTypeRd":"I მოსმენა","billIdRd":32449,
  "billRd":"„საქართველოს საბიუჯეტო კოდექსში ცვლილების შეტანის შესახებ“",
  "registrationNumberRd":"372/2-XIმპ","votingResultSet":{"yes":77,"no":0,"withoutInteraction":24}}]}

GET /law/1/voting?voteSessionId=31507&start=0&limit=200&group=false
{"total":101,"list":[{"votingResult":{"id":1,"name":"დიახ"},"colleagueRd":"აბესაძე გრიგოლ",
  "factionRd":"ფრაქცია „ქართული ოცნება“"}, ...]}   79 დიახ, 10 არა, 12 არ ესწრება

GET /mpQuestion/1/?start=0&limit=1
{"total":14183,"list":[{"id":14583,"registrationNumber":"6368/3-79/26","registrationDate":"13-11-2026",
  "request":"ითხოვს ინფორმაციას ...","targetOrganisation":"სსიპ აღსრულების ეროვნული ბიურო",
  "targetPerson":"მოძღვრიშვილი ლაშა","deadline1":"28-11-2026","responseDate":null,
  "colleagueName":"ხორგუანი სოფიო","nodeName":"ფრაქცია „საქართველოსთვის“"}]}
```


## 3. `adapter/parliament`

`NewAdapter(client *parliamentclient.Client, now func() time.Time)`.
It implements the three ports. `ReadAt` is `now()` in `kernel.TbilisiZone()` after the
last answer.

| Vendor field                          | Domain value                                          |
| ------------------------------------- | ----------------------------------------------------- |
| A date `dd-MM-yyyy`                   | `kernel.Date`. A text that does not parse fails (`PARL-DD-009`). |
| `billName`, `packageName`, the names  | As published, trimmed at the ends.                    |
| `billReviewTypes`                     | The last name is `Bill.NewestStage` (`PARL-DD-004`).  |
| `reviewTypeName`, `billReviewTypeRd`  | `domain.ReadingOf`.                                   |
| `votingConclusionType.name`           | `domain.OutcomeOf`.                                   |
| `votingResult.name`                   | `domain.ChoiceOf`.                                    |
| `withoutInteraction`                  | `Vote.Absent`.                                        |
| `factionRd` null                      | An empty faction.                                     |
| `nodeName`                            | `Question.Faction`.                                   |
| `target`, `targetOrganisation`, `targetPerson` | `Question.Addressee`.                        |
| `deadline1`                           | `Question.Deadline`.                                  |

The page of a bill is `https://info.parliament.ge/#law-drafting/{id}`. The page of a
question is `https://info.parliament.ge/#mpqs/{id}`. The adapter builds both from
`DefaultBaseURL`, never from the configured address.
