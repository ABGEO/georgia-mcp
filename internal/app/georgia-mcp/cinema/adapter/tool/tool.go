package tool

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// CinemaLister lists the open cinemas.
type CinemaLister interface {
	List(ctx context.Context) (app.CinemaList, error)
}

// MovieLister lists the movies of a status.
type MovieLister interface {
	List(ctx context.Context, query app.MovieQuery) (app.Movies, error)
}

// MovieSearcher finds the movies whose English name matches a text.
type MovieSearcher interface {
	Search(ctx context.Context, query app.SearchQuery) (app.Movies, error)
}

// SessionLister lists the sessions of one date that have not started.
type SessionLister interface {
	List(ctx context.Context, query app.SessionQuery) (app.Sessions, error)
}

// SeatMapGetter returns the seats of the hall of one session.
type SeatMapGetter interface {
	Get(ctx context.Context, code domain.SessionCode) (app.SeatMap, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	ListCinemas  CinemaLister
	ListMovies   MovieLister
	SearchMovies MovieSearcher
	ListSessions SessionLister
	GetSeatMap   SeatMapGetter
}

const (
	source       = "Cavea"
	movieAddress = "https://cavea.ge/movie/"

	defaultMovies   = 50
	defaultFound    = 20
	defaultSessions = 30
	maxItems        = 100

	dataNote = " Data from the web site of Cavea, cavea.ge."

	listCinemasDescription = "Lists the open cinemas of Cavea, the largest cinema chain of " +
		"Georgia, in Tbilisi and Batumi. Returns the code, the name, the city, and the address " +
		"of each cinema. Use the code in cinema_list_sessions." + dataNote
	listMoviesDescription = "Lists the movies that Cavea shows now, or the movies coming soon. " +
		"Returns the code, the name, the genres, the duration in minutes, the age rating, the " +
		"premiere, the directors, the actors, and a description of each movie. Use the code in " +
		"cinema_list_sessions." + dataNote
	searchMoviesDescription = "Finds the movies of Cavea, now showing or coming soon, by a part " +
		"of the English name. Give the text in English: Cavea does not match a Georgian name. " +
		"Returns the code, the name, the status, the duration in minutes, and the age rating of " +
		"each movie." + dataNote
	listSessionsDescription = "Lists the sessions of the movies of Cavea on one date that have " +
		"not started, in the order of the start time. Narrow the list by a movie, a cinema, an " +
		"audio language, or a format. Returns the cinema, the hall, the start time, the " +
		"languages, the formats, the price in lari, and the occupancy level of each session. " +
		"With a movie, each session has the price of each seat class. With no movie, it has the " +
		"price of the class Standard. Use the session code in cinema_get_seat_map." + dataNote
	getSeatMapDescription = "Returns the seats of the hall of one session of Cavea that has not " +
		"started: each row, each seat with its class, a mark for a taken seat, and a mark for " +
		"special needs. Gives the count of free seats and the price in lari of each seat class. " +
		"The server reads the map at each call. It reserves no seat. A person buys the ticket " +
		"on cavea.ge." + dataNote
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := errors.Join(
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "cinema_list_cinemas", Title: "List the cinemas of Cavea",
			Description: listCinemasDescription,
		}, handlers.listCinemas),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "cinema_list_movies", Title: "List the movies of Cavea",
			Description: listMoviesDescription,
		}, handlers.listMovies),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "cinema_search_movies", Title: "Search the movies of Cavea",
			Description: searchMoviesDescription,
		}, handlers.searchMovies),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "cinema_list_sessions", Title: "List the sessions of Cavea",
			Description: listSessionsDescription,
		}, handlers.listSessions),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "cinema_get_seat_map", Title: "Get the seat map of a session of Cavea",
			Description: getSeatMapDescription,
		}, handlers.getSeatMap),
	)
	if err != nil {
		return fmt.Errorf("registering the cinema tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) listCinemas(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listCinemasInput,
) (*mcp.CallToolResult, cinemasOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, cinemasOutput{}, err
	}

	cinemas, err := h.uses.ListCinemas.List(ctx)
	if err != nil {
		return nil, cinemasOutput{}, toolError(err)
	}

	return nil, toCinemasOutput(cinemas, language), nil
}

func (h handler) listMovies(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listMoviesInput,
) (*mcp.CallToolResult, moviesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, moviesOutput{}, err
	}

	status, err := domain.ParseMovieStatus(in.Status)
	if err != nil {
		return nil, moviesOutput{}, invalid(err,
			"status: %q is not a status. Use now_showing or coming_soon.", in.Status)
	}

	limit, err := withinRange("limit", in.Limit, defaultMovies, maxItems)
	if err != nil {
		return nil, moviesOutput{}, err
	}

	movies, err := h.uses.ListMovies.List(ctx, app.MovieQuery{Status: status, Limit: limit})
	if err != nil {
		return nil, moviesOutput{}, toolError(err)
	}

	return nil, toMoviesOutput(movies, language), nil
}

func (h handler) searchMovies(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchMoviesInput,
) (*mcp.CallToolResult, moviesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, moviesOutput{}, err
	}

	limit, err := withinRange("limit", in.Limit, defaultFound, maxItems)
	if err != nil {
		return nil, moviesOutput{}, err
	}

	movies, err := h.uses.SearchMovies.Search(ctx, app.SearchQuery{Text: in.Query, Limit: limit})
	if err != nil {
		return nil, moviesOutput{}, toolError(err)
	}

	return nil, toMoviesOutput(movies, language), nil
}

func (h handler) listSessions(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listSessionsInput,
) (*mcp.CallToolResult, sessionsOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, sessionsOutput{}, err
	}

	query, err := toSessionQuery(in)
	if err != nil {
		return nil, sessionsOutput{}, err
	}

	sessions, err := h.uses.ListSessions.List(ctx, query)
	if err != nil {
		return nil, sessionsOutput{}, sessionsError(err, in)
	}

	return nil, toSessionsOutput(sessions, language), nil
}

func (h handler) getSeatMap(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getSeatMapInput,
) (*mcp.CallToolResult, seatMapOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, seatMapOutput{}, err
	}

	code, err := domain.ParseSessionCode(in.Session)
	if err != nil {
		return nil, seatMapOutput{}, invalid(err, "session: %q is not a session code. Use a code "+
			"from cinema_list_sessions, for example cag_5/27-10-2026/22:15.", in.Session)
	}

	seatMap, err := h.uses.GetSeatMap.Get(ctx, code)
	if err != nil {
		return nil, seatMapOutput{}, seatMapError(err, in.Session)
	}

	return nil, toSeatMapOutput(seatMap, language), nil
}
