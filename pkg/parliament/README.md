# parliament

A Go client of the bills, the plenary votes, and the member questions of the Parliament
of Georgia, as `info.parliament.ge` publishes them.

The methods are official. The IT department of the Parliament describes them at
`https://info.parliament.ge/swagger-ui.html`. They need no key.

```
go get github.com/abgeo/georgia-mcp/pkg/parliament
```

```go
client := parliament.New()

bills, err := client.Bills(ctx, parliament.BillQuery{Limit: 10, BillName: "ბიუჯეტ"})
if err != nil {
    return err
}

fmt.Println(bills.Total, bills.List[0].BillName)
```

## The API

| Method         | Request                              | Answer                                          |
| -------------- | ------------------------------------ | ----------------------------------------------- |
| `Bills`        | `GET /law/1/bill`                    | A page of bills, each with its package          |
| `Bill`         | `GET /law/1/bill/{id}`               | One bill                                        |
| `BillReviews`  | `GET /law/1/bill/{id}/billReview`    | The review stages of a bill                     |
| `VoteSessions` | `GET /law/1/voteSession`             | A page of plenary votes, with their counts      |
| `VoteSession`  | `GET /law/1/voting/session/{id}`     | One plenary vote                                |
| `Votings`      | `GET /law/1/voting`                  | The choice of each member in a vote             |
| `MPQuestions`  | `GET /mpQuestion/1/`                 | A page of member questions                      |

The client mirrors what the API answers today:

- **A list call needs `start` and `limit`.** Without them, or with an unknown parameter,
  the API answers an HTML page with the status 404. The client always sends both.
- A page of more than 50 bills answers an empty body with the status 200.
- A date is `dd-MM-yyyy`. A date in another form gives an empty list, with no error.
- An unknown bill or vote answers `[null]`. `Bill` and `VoteSession` return nil.
- The filters `billName`, `billRd`, and `requestTitle` give the newest first. The filter
  `searchItem` gives an order with no date, and the client does not send it.
- Every text is Georgian. A text in English finds nothing.
- Each choice of `Votings` repeats the whole vote: 101 choices are 176 KB.

The client does not retry and does not cache. Pass an `http.Client` with
`vendorhttp.WithHTTPClient` to set a timeout or a rate limit.
