# tas

A Go client of the public search of the Tbilisi Architecture Service (TAS), at
`docs.tbilisi.gov.ge/architect`. It finds the applications to TAS by cadastral code, by
number, and by date of registration: building permits, project approvals, and the rest.

The method serves the public search page of TAS over DWR and has no documentation.

```
go get github.com/abgeo/georgia-mcp/pkg/tas
```

```go
client := tas.New()

page, err := client.SearchDocuments(ctx, tas.Filter{
    CadastralCode: "01.18.04.020.005",
    StatusIDs:     []int{1, 2, 5, 6, 7, 9, 11, 12, 20, 21, 25, 26, 27, 28},
    Limit:         50,
})
if err != nil {
    return err
}

fmt.Println(page.Total, page.Documents[0].DocumentNo)
```

## The API

| Method            | Request                                                                    | Answer                         |
| ----------------- | -------------------------------------------------------------------------- | ------------------------------ |
| `SearchDocuments` | `POST /architect/dwr/call/plaincall/DocumentManager.getDocsForPublicInfo.dwr` | One page of applications, and the total |

The client mirrors what the page answers today:

- The reply is JavaScript, not JSON. The client converts the value of the callback to JSON
  before the decode, and reads the summary of each application from its XML.
- The order is not by date, and a sort field changes nothing.
- With no status filter, the answer holds drafts. The statuses above are those of the page.
- A number is a prefix: `AR1116818` matches `AR11168188` too.
- Every text is Georgian.
- An application holds the identifier of its applicant. `Document` has no field for it.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
