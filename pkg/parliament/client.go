package parliament

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// DefaultBaseURL is the production address of the data of the Parliament.
const DefaultBaseURL = "https://info.parliament.ge"

// Client reads the data of the Parliament. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the Parliament.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// Bills returns one page of the bills that match the query, newest first when the query
// has a BillName.
func (c *Client) Bills(ctx context.Context, query BillQuery) (BillList, error) {
	values := page(query.Start, query.Limit)
	set(values, "billName", query.BillName)
	set(values, "registrationNumber", query.RegistrationNumber)
	set(values, "registrationDate1", query.RegistrationDate1)
	set(values, "registrationDate2", query.RegistrationDate2)

	var bills BillList
	if err := c.get(ctx, values, &bills, "law", "1", "bill"); err != nil {
		return BillList{}, fmt.Errorf("reading the bills: %w", err)
	}

	return bills, nil
}

// Bill returns the bill with the identifier, or nil when the Parliament has none.
func (c *Client) Bill(ctx context.Context, id int) (*Bill, error) {
	var bills []*Bill
	if err := c.get(ctx, nil, &bills, "law", "1", "bill", strconv.Itoa(id)); err != nil {
		return nil, fmt.Errorf("reading the bill: %w", err)
	}

	return first(bills), nil
}

// BillReviews returns the review stages of a bill, in the order of the review.
func (c *Client) BillReviews(ctx context.Context, billID int) ([]BillReview, error) {
	var reviews []BillReview

	err := c.get(ctx, nil, &reviews, "law", "1", "bill", strconv.Itoa(billID), "billReview")
	if err != nil {
		return nil, fmt.Errorf("reading the review stages: %w", err)
	}

	return reviews, nil
}

// VoteSessions returns one page of the plenary votes that match the query, newest first.
func (c *Client) VoteSessions(
	ctx context.Context,
	query VoteSessionQuery,
) (VoteSessionList, error) {
	values := page(query.Start, query.Limit)
	set(values, "billRd", query.BillRd)
	set(values, "votingDate1", query.VotingDate1)
	set(values, "votingDate2", query.VotingDate2)

	if query.BillReviewID != 0 {
		values.Set("billReviewId", strconv.Itoa(query.BillReviewID))
	}

	var votes VoteSessionList
	if err := c.get(ctx, values, &votes, "law", "1", "voteSession"); err != nil {
		return VoteSessionList{}, fmt.Errorf("reading the votes: %w", err)
	}

	return votes, nil
}

// VoteSession returns the vote with the identifier, or nil when the Parliament has none.
func (c *Client) VoteSession(ctx context.Context, id int) (*VoteSession, error) {
	var votes []*VoteSession
	if err := c.get(
		ctx,
		nil,
		&votes,
		"law",
		"1",
		"voting",
		"session",
		strconv.Itoa(id),
	); err != nil {
		return nil, fmt.Errorf("reading the vote: %w", err)
	}

	return first(votes), nil
}

// Votings returns one page of the choices of the members in a vote, by family name.
func (c *Client) Votings(ctx context.Context, voteSessionID, start, limit int) (VotingList, error) {
	values := page(start, limit)
	values.Set("voteSessionId", strconv.Itoa(voteSessionID))
	values.Set("group", "false")

	var votings VotingList
	if err := c.get(ctx, values, &votings, "law", "1", "voting"); err != nil {
		return VotingList{}, fmt.Errorf("reading the choices of the vote: %w", err)
	}

	return votings, nil
}

// MPQuestions returns one page of the member questions that match the query, newest first.
func (c *Client) MPQuestions(ctx context.Context, query MPQuestionQuery) (MPQuestionList, error) {
	values := page(query.Start, query.Limit)
	set(values, "requestTitle", query.RequestTitle)
	set(values, "colleagueName", query.ColleagueName)
	set(values, "registrationDate1", query.RegistrationDate1)
	set(values, "registrationDate2", query.RegistrationDate2)

	var questions MPQuestionList
	if err := c.get(ctx, values, &questions, "mpQuestion", "1/"); err != nil {
		return MPQuestionList{}, fmt.Errorf("reading the member questions: %w", err)
	}

	return questions, nil
}

// get decodes the answer of the path. An HTML page or an empty body does not decode,
// so it is an error.
func (c *Client) get(ctx context.Context, query url.Values, target any, path ...string) error {
	answer, err := c.http.Fetch(ctx, vendorhttp.Request{Path: path, Query: query})
	if err != nil {
		return err
	}

	if err := json.Unmarshal(answer, target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}

func page(start, limit int) url.Values {
	return url.Values{"start": {strconv.Itoa(start)}, "limit": {strconv.Itoa(limit)}}
}

func set(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}

// first returns the first item. The Parliament answers [null] for an unknown identifier.
func first[T any](items []*T) *T {
	if len(items) == 0 {
		return nil
	}

	return items[0]
}
