package tool

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// ActSearcher searches the acts.
type ActSearcher interface {
	Search(ctx context.Context, query domain.SearchQuery, limit int) (app.SearchResult, error)
}

// ActGetter reads one act.
type ActGetter interface {
	Get(ctx context.Context, query app.ActQuery) (app.ActView, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchActs ActSearcher
	GetAct     ActGetter
}

const (
	source  = "Legislative Herald of Georgia"
	siteURL = "https://matsne.gov.ge"

	maxQueryRunes = 128
	pageSize      = 20

	searchActsDescription = "Searches the legal acts of Georgia in the Legislative Herald " +
		"(matsne.gov.ge): laws, codes, resolutions, decrees, orders, treaties, and court " +
		"decisions. Returns 20 acts on a page, each with its identifier, title, document type, " +
		"issuer, number, signing date, and status, and the total count. The source has no " +
		"order by relevance: give a type, a status, or an issuer to bring an act to the first " +
		"page. A Georgian query that equals the full title of an active act puts that act " +
		"first. Search in Georgian for the best match; an English query matches the titles of " +
		"translated acts only. Call law_get_act with the identifier to read the act. Data from " +
		"the Legislative Herald of Georgia."
	getActDescription = "Returns one legal act of the Legislative Herald (matsne.gov.ge) by " +
		"its identifier from law_search_acts. With no article, returns the metadata, the newest " +
		"20 consolidated versions with their publication dates, the count of every version, " +
		"and the table of contents. With " +
		"articles, returns the full text of at most 10 articles, for example 82 or 133^3 " +
		"(article 133 with the superscript 3). Give a version number or a date to read the act " +
		"as it stood then; the default is the newest version, or in English the newest " +
		"translated version. A version date is its publication date, not the date it took " +
		"effect. The Georgian text is the only official text; an English text is a " +
		"translation, and not every act has one. Data from the Legislative Herald of Georgia."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "law_search_acts", Title: "Search the legal acts of Georgia",
		Description: searchActsDescription,
	}, handlers.searchActs)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "law_get_act", Title: "Get a legal act of Georgia",
			Description: getActDescription,
		}, handlers.getAct)
	}

	if err != nil {
		return fmt.Errorf("registering the law tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) searchActs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchActsInput,
) (*mcp.CallToolResult, searchOutput, error) {
	query, err := searchQuery(in)
	if err != nil {
		return nil, searchOutput{}, err
	}

	limit, err := withinRange("limit", in.Limit, pageSize, pageSize)
	if err != nil {
		return nil, searchOutput{}, err
	}

	result, err := h.uses.SearchActs.Search(ctx, query, limit)
	if err != nil {
		return nil, searchOutput{}, toolError(err, "", 0)
	}

	return nil, toSearchOutput(result, query), nil
}

func (h handler) getAct(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getActInput,
) (*mcp.CallToolResult, actOutput, error) {
	query, err := actQuery(in)
	if err != nil {
		return nil, actOutput{}, err
	}

	view, err := h.uses.GetAct.Get(ctx, query)
	if err != nil {
		return nil, actOutput{}, toolError(err, query.ID.String(), len(in.Articles))
	}

	return nil, toActOutput(view, query.Language), nil
}

func searchQuery(in searchActsInput) (domain.SearchQuery, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return domain.SearchQuery{}, err
	}

	text := strings.TrimSpace(in.Query)
	if count := utf8.RuneCountInString(text); count < 1 || count > maxQueryRunes {
		return domain.SearchQuery{}, &mcpserver.ToolError{
			Message: "query: give 1 to 128 characters.",
		}
	}

	query := domain.SearchQuery{
		Text: text, Issuer: strings.TrimSpace(in.Issuer), Number: strings.TrimSpace(in.Number),
		Language: language,
	}

	if query.InText, err = parseTarget(in.Target); err != nil {
		return domain.SearchQuery{}, err
	}

	if query.Order, err = parseOrder(in.Order); err != nil {
		return domain.SearchQuery{}, err
	}

	if query.Page, err = parsePage(in.Page); err != nil {
		return domain.SearchQuery{}, err
	}

	if err := parseFilters(in, &query); err != nil {
		return domain.SearchQuery{}, err
	}

	return query, nil
}

func parseFilters(in searchActsInput, query *domain.SearchQuery) error {
	var err error

	if query.Type, err = parseType(in.Type); err != nil {
		return err
	}

	if query.Status, err = parseStatus(in.Status); err != nil {
		return err
	}

	if query.From, err = parseDate("signed_from", in.SignedFrom); err != nil {
		return err
	}

	query.To, err = parseDate("signed_to", in.SignedTo)

	return err
}

func actQuery(in getActInput) (app.ActQuery, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return app.ActQuery{}, err
	}

	id, err := domain.ParseActID(in.ID)
	if err != nil {
		return app.ActQuery{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"id: %q is not an act identifier. Call law_search_acts to find the identifier.",
				in.ID,
			),
			Cause: err,
		}
	}

	date, err := parseDate("date", in.Date)
	if err != nil {
		return app.ActQuery{}, err
	}

	articles, err := parseArticles(in.Articles)
	if err != nil {
		return app.ActQuery{}, err
	}

	if in.Piece < 0 {
		return app.ActQuery{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("piece: %d is out of range. Use 1 or more.", in.Piece),
		}
	}

	return app.ActQuery{
		ID: id, Version: in.Version, Date: date, Articles: articles, Piece: in.Piece,
		Language: language,
	}, nil
}
