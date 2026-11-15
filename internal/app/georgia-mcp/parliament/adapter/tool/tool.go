package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// BillSearcher gives one page of the bills of a filter.
type BillSearcher interface {
	Search(ctx context.Context, filter app.BillFilter, page domain.Page) (app.BillResult, error)
}

// BillGetter gives one bill by its identifier.
type BillGetter interface {
	Get(ctx context.Context, id domain.ID) (app.BillRead, error)
}

// VoteLister gives one page of the votes of a filter.
type VoteLister interface {
	List(ctx context.Context, filter app.VoteFilter, page domain.Page) (app.VoteResult, error)
}

// VoteGetter gives one vote by its identifier.
type VoteGetter interface {
	Get(ctx context.Context, id domain.ID) (app.VoteDetail, error)
}

// QuestionSearcher gives one page of the member questions of a filter.
type QuestionSearcher interface {
	Search(
		ctx context.Context,
		filter app.QuestionFilter,
		page domain.Page,
	) (app.QuestionResult, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchBills     BillSearcher
	GetBill         BillGetter
	ListVotes       VoteLister
	GetVote         VoteGetter
	SearchQuestions QuestionSearcher
}

const (
	commonEnd = " Texts are in Georgian. Search in Georgian. Data from the Parliament of Georgia."

	searchBillsDescription = "Returns the bills whose title holds a text, or that the " +
		"Parliament registered in a period, or that carry a registration number " +
		"(372/2-XIმპ), newest first. Each bill has its identifier, title, type, " +
		"registration number and date, initiator, author, and newest review stage. Read " +
		"the stages and votes with parliament_get_bill." + commonEnd
	getBillDescription = "Returns one bill by its identifier (32449): its description, its " +
		"package, and every review stage in order, committee hearings and readings, with " +
		"dates, yes and no counts, and the identifiers of its votes. Read a vote with " +
		"parliament_get_vote." + commonEnd
	listVotesDescription = "Returns the plenary votes in a period, or on bills whose title " +
		"holds a text, newest first. Each vote has its identifier, date, bill, reading, " +
		"outcome (adopted or rejected), quorum, and the counts of yes, no, and absent. Read " +
		"the choice of each member with parliament_get_vote." + commonEnd
	getVoteDescription = "Returns one vote by its identifier (31510), with the choice of " +
		"each member (yes, no, or absent), grouped by faction, with the count of each " +
		"choice in each faction." + commonEnd
	searchQuestionsDescription = "Returns the written questions of members to the bodies " +
		"of the state, newest first: by a text of the request, by the member (a family " +
		"name, ხორგუანი), or by a period of registration. Each question has its addressee, " +
		"deadline, response date, and status: answered, pending, or overdue. The texts of " +
		"a question and an answer are PDF files on its page." + commonEnd

	billInputs     = "query, number, from, or to"
	voteInputs     = "query, from, or to"
	questionInputs = "query, member, from, or to"
)

// Register adds the five tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name:        "parliament_search_bills",
		Title:       "Search the bills of the Parliament of Georgia",
		Description: searchBillsDescription,
	}, handlers.searchBills)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name:        "parliament_get_bill",
			Title:       "Get a bill of the Parliament of Georgia",
			Description: getBillDescription,
		}, handlers.getBill)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name:        "parliament_list_votes",
			Title:       "List the plenary votes of the Parliament of Georgia",
			Description: listVotesDescription,
		}, handlers.listVotes)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name:        "parliament_get_vote",
			Title:       "Get a plenary vote of the Parliament of Georgia",
			Description: getVoteDescription,
		}, handlers.getVote)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name:        "parliament_search_questions",
			Title:       "Search the questions of the members of the Parliament of Georgia",
			Description: searchQuestionsDescription,
		}, handlers.searchQuestions)
	}

	if err != nil {
		return fmt.Errorf("registering the parliament tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) searchBills(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchBillsInput,
) (*mcp.CallToolResult, billsOutput, error) {
	page, err := parsePage(in.Page, in.Limit)
	if err != nil {
		return nil, billsOutput{}, err
	}

	text, err := parseText("query", in.Query)
	if err != nil {
		return nil, billsOutput{}, err
	}

	period, err := parsePeriod(in.From, in.To)
	if err != nil {
		return nil, billsOutput{}, err
	}

	filter := app.BillFilter{
		Text: text, Number: strings.TrimSpace(in.Number), Registered: period,
	}

	result, err := h.uses.SearchBills.Search(ctx, filter, page)
	if err != nil {
		return nil, billsOutput{}, toolError(err, 0, billInputs)
	}

	return nil, toBillsOutput(result, page), nil
}

func (h handler) getBill(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getInput,
) (*mcp.CallToolResult, billOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, billOutput{}, err
	}

	read, err := h.uses.GetBill.Get(ctx, id)
	if err != nil {
		return nil, billOutput{}, toolError(err, id, "")
	}

	return nil, toBillOutput(read), nil
}

func (h handler) listVotes(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listVotesInput,
) (*mcp.CallToolResult, votesOutput, error) {
	page, err := parsePage(in.Page, in.Limit)
	if err != nil {
		return nil, votesOutput{}, err
	}

	text, err := parseText("query", in.Query)
	if err != nil {
		return nil, votesOutput{}, err
	}

	period, err := parsePeriod(in.From, in.To)
	if err != nil {
		return nil, votesOutput{}, err
	}

	result, err := h.uses.ListVotes.List(ctx, app.VoteFilter{BillTitle: text, Voted: period}, page)
	if err != nil {
		return nil, votesOutput{}, toolError(err, 0, voteInputs)
	}

	return nil, toVotesOutput(result, page), nil
}

func (h handler) getVote(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getInput,
) (*mcp.CallToolResult, voteOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, voteOutput{}, err
	}

	detail, err := h.uses.GetVote.Get(ctx, id)
	if err != nil {
		return nil, voteOutput{}, toolError(err, id, "")
	}

	return nil, toVoteOutput(detail), nil
}

func (h handler) searchQuestions(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchQuestionsInput,
) (*mcp.CallToolResult, questionsOutput, error) {
	page, err := parsePage(in.Page, in.Limit)
	if err != nil {
		return nil, questionsOutput{}, err
	}

	text, err := parseText("query", in.Query)
	if err != nil {
		return nil, questionsOutput{}, err
	}

	member, err := parseText("member", in.Member)
	if err != nil {
		return nil, questionsOutput{}, err
	}

	period, err := parsePeriod(in.From, in.To)
	if err != nil {
		return nil, questionsOutput{}, err
	}

	filter := app.QuestionFilter{Text: text, Member: member, Registered: period}

	result, err := h.uses.SearchQuestions.Search(ctx, filter, page)
	if err != nil {
		return nil, questionsOutput{}, toolError(err, 0, questionInputs)
	}

	return nil, toQuestionsOutput(result, page), nil
}
