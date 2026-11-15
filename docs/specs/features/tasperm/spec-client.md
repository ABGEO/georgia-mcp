---
id: TASPERM
title: The applications of the Tbilisi Architecture Service, the client
type: spec
status: approved
created: 2026-11-15
updated: 2026-11-15
context: permits
approved_by: Temuri
approved_on: 2026-11-15
constrained_by: [CLT, DOM]
requirements: features/tasperm/requirements.md
---

# Specification: The applications of the Tbilisi Architecture Service, the client

## 1. `pkg/tas`

The shape of `pkg/napr`. The package comment and the README say that the method serves the
public search page of TAS and has no documentation.

```go
const DefaultBaseURL = "https://docs.tbilisi.gov.ge"

// MaxBody holds 500 applications of about 11 KB each, with room.
const MaxBody = 16 << 20

var (
    ErrUnexpectedStatus = errors.New("tas: unexpected status")
    ErrInvalidFilter    = errors.New("tas: invalid filter")
    ErrInvalidReply     = errors.New("tas: invalid reply")
    ErrRemote           = errors.New("tas: remote exception")
)

type APIError = vendorhttp.StatusError

// New sets MaxBody unless an option sets another limit.
func New(opts ...vendorhttp.Option) *Client

// SearchDocuments returns one page of the applications that match the filter, and the
// count of all matches.
func (c *Client) SearchDocuments(ctx context.Context, filter Filter) (Page, error)

type Filter struct {
    CadastralCode string    // "01.18.04.020.005", or empty
    DocumentNo    string    // "AR11168188", a prefix, or empty
    From, To      time.Time // the zero time sends no bound
    StatusIDs     []int     // empty sends no status filter
    Start, Limit  int
}

type Page struct {
    Total     int        // sources[0]
    Documents []Document // source
}

type Document struct {
    DocumentID       int64      `json:"documentId"`
    DocumentNo       string     `json:"documentNo"`
    Address          string     `json:"address"`
    RegistrationDate *Millis    `json:"registrationDate"`
    DeadLineDate     *Millis    `json:"deadLineDate"`
    DocumentStatusID int        `json:"documentStatusId"`
    CachedInfo       CachedInfo `json:"cachedInfo"`   // XML in a string
    ResponseText     string     `json:"responseText"` // HTML, or empty
}

// Millis is an instant in milliseconds since the epoch, from new Date(ms).
type Millis int64
func (m Millis) Time() time.Time

type CachedInfo struct {
    StatusName   string      `xml:"documentStatusName"`
    ExecutorName string      `xml:"executorEmployeeName"`
    Nomenclature Nomenclature `xml:"nomenklature"`
}

type Nomenclature struct {
    ActionName         string   `xml:"actionName"`
    CategoryName       string   `xml:"categoryName"`
    ParentCategoryName string   `xml:"parentCategoryName"`
    ClassName          string   `xml:"className"` // "IV კლასი"
    Stadiums           []string `xml:"stadiums"`
}
```

`Document` has no field for `userId` or any other field of the applicant. The decoder
never reads them.

**The request.** `POST /architect/dwr/call/plaincall/DocumentManager.getDocsForPublicInfo.dwr`,
`Content-Type: text/plain`. The body has one `key=value` on each line:

```
callCount=1
windowName=
nextReverseAjaxIndex=0
c0-scriptName=DocumentManager
c0-methodName=getDocsForPublicInfo
c0-id=0
c0-e1=number:2
c0-e2=number:0
c0-e3=number:500
c0-e4=string:01.18.04.020.005
c0-e5=Array:[reference:c0-e6,reference:c0-e7]
c0-e6=number:1
c0-e7=number:2
c0-param0=Object_Object:{applicationId:reference:c0-e1, start:reference:c0-e2, limit:reference:c0-e3, naprCadCode:reference:c0-e4, docStatusIds:reference:c0-e5}
batchId=1
instanceId=0
page=%2Farchitect%2FpublicInformation.html
httpSessionId=
scriptSessionId=00000000000000000000000000000000/georgia-mcp
```

- `applicationId` is always 2. `start` and `limit` are always sent.
- `CadastralCode` is `naprCadCode`, `DocumentNo` is `documentNo`, `From` is `fromDate`,
  `To` is `toDate`, `StatusIDs` is `docStatusIds`. A field that is empty is not sent.
- A time is `Date:<milliseconds>`. A number of the array is `number:<n>`.
- A code must match `^[0-9.]+$` and a number `^[A-Z0-9]+$`. Any other text gives
  `ErrInvalidFilter` and no call, since the value goes into the body with no encoding.
  A `Limit` below 1 gives `ErrInvalidFilter`.

**The reply.** JavaScript, `text/javascript`. The client:

1. Finds the line that starts with `dwr.engine.remote.handleCallback("1","0",`. The value
   runs from that prefix to the last `);` of the line.
2. Finds `handleBatchException(` or `handleException(` when the callback is absent. It
   gives `ErrRemote` with the `name` and the `message`.
3. Gives `ErrInvalidReply` when it finds neither.
4. Converts the value to JSON with a hand scanner: a key with no quotation marks gets them,
   `new Date(n)` becomes `n`, a string keeps its escapes, and `\u0000` is dropped. Every
   other token passes as it is.
5. Decodes the JSON. `isSuccess` false gives `ErrRemote` with `errorMessage`.
6. Decodes each `cachedInfo` with `encoding/xml`. An empty text gives the zero value.
   An XML text that does not decode gives `ErrInvalidReply`.

The scanner is a function with a fuzz test: it never panics, and a value with no `new Date`
converts to the same JSON that `encoding/json` reads.

## 2. Recorded answers

`pkg/tas/testdata/` holds the answers of 2026-11-15. Each `userId` is `0`.

| File               | Filter                                                       | Content |
| ------------------ | ------------------------------------------------------------ | ------- |
| `search-code.js`   | `CadastralCode` `01.18.04.020.005`, the statuses of the page, limit 500 | 5 applications. |
| `search-number.js` | `DocumentNo` `AR11168188`, the statuses of the page, limit 50 | 1 application, status 1, with a decision text. |
| `search-month.js`  | 2026-10-01 to 2026-10-31, the statuses of the page, limit 5  | 5 applications of 2128. |
| `search-empty.js`  | `CadastralCode` `99.99.99.999.999`                            | `source:[]`, `sources:[0]`. |
| `exception.js`     | A body with no `scriptSessionId`                             | `handleBatchException`, `IllegalArgumentException`. |

Real answers, cut:

```
POST https://docs.tbilisi.gov.ge/architect/dwr/call/plaincall/DocumentManager.getDocsForPublicInfo.dwr
200 text/javascript (3881 bytes)
throw 'allowScriptTagRemoting is false.';
//#DWR-INSERT
dwr.engine.remote.handleNewScriptSession("4BD5103F86C7BA952EBD596DF3EE0EA5");
//#DWR-REPLY
dwr.engine.remote.handleCallback("1","0",{errodCode:null,errorMessage:null,isSuccess:true,
  source:[{address:"ქ. თბი...",amountToPay:0,applicationId:2,
  cachedInfo:"<?xml version=\"1.0\" ...><documentCachedInfo><documentStatusName>-</documentStatusName>
    <executorEmployeeName>ეკა ...</executorEmployeeName><nomenklature><caseId>3</caseId>
    <categoryName>...</categoryName><parentCategoryName>...</parentCategoryName></nomenklature></documentCachedInfo>",
  createDate:new Date(1793781976000),createDateStr:"04/11/2026",deadLineDate:new Date(1797451200000),
  documentId:1168188,documentNo:"AR11168188",documentStatusId:1,
  registrationDate:new Date(1793781976000),responseText:"<p>...</p>",userId:0, ...}],sources:[1]});

if (window.dwr) dwr.engine.remote.handleBatchException({ name:'java.lang.IllegalArgumentException',
  message:'Failed to find parameter: scriptSessionId' });
```

A construction application carries more in `nomenklature`:

```
<actionName>შეთანხმებული პროექტის ცვლილება</actionName><caseId>1</caseId>
<categoryName>მრავალბინიანი საცხოვრებელი სახლი</categoryName><classId>4</classId>
<className>IV კლასი</className><stadiums>პროექტის შეთანხმება</stadiums>
<stadiums>მშენებლობის ნებართვა</stadiums>
```

Other answers seen:

- `registrationDate` is the date of the filters `fromDate` and `toDate`. `createDate` is
  the date of the draft and can be months earlier.
- `className` can be `" "`, with the quotation marks. `executorEmployeeName` and
  `documentStatusName` can be `-`.
- A page of 87 applications with decision texts is 950 KB.
- A `sort` or `dir` field changes nothing.
- A number of a draft gives no application with the statuses of the page.

## Retired identifiers

This file has no retired identifier.
