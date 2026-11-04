package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// QualityGetter gives the newest readings of the stations.
type QualityGetter interface {
	Get(ctx context.Context, query app.QualityQuery) (app.Quality, error)
}

// HistoryLister gives the hourly readings of one station over a period.
type HistoryLister interface {
	List(ctx context.Context, query app.HistoryQuery) (app.History, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	GetQuality  QualityGetter
	ListHistory HistoryLister
}

const (
	source = "National Environmental Agency"

	defaultStations = 25
	defaultReadings = 24
	maxItems        = 100

	getQualityDescription = "Returns the newest air quality readings of the automatic " +
		"stations of the National Environmental Agency of Georgia (NEA), for a city, one " +
		"station, or every station. Each reading gives PM10, PM2.5, NO2, O3, SO2, or CO, its " +
		"concentration (µg/m³, CO in mg/m³), its hour, and its level: good, fair, moderate, " +
		"poor, or very_poor. A station level is its worst reading. Stations come worst first. " +
		"A reading older than 24 hours is left out. The NEA stamps some readings a few hours " +
		"ahead of the clock: the time is as published. Data from the NEA."
	listHistoryDescription = "Returns the hourly readings of one station of the National " +
		"Environmental Agency of Georgia (NEA) over at most 7 days of the last 7, newest first, " +
		"with the level of each reading. The default is the last 24 hours. Call air_get_quality " +
		"first to find the station code. The NEA stamps some readings a few hours ahead of the " +
		"clock: the time is as published. Data from the NEA."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "air_get_quality", Title: "Get the air quality in Georgia",
		Description: getQualityDescription,
	}, handlers.getQuality)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "air_list_history", Title: "List the air quality history of a station",
			Description: listHistoryDescription,
		}, handlers.listHistory)
	}

	if err != nil {
		return fmt.Errorf("registering the air tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) getQuality(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getQualityInput,
) (*mcp.CallToolResult, qualityOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, qualityOutput{}, err
	}

	limit, err := withinRange("limit", in.Limit, defaultStations, maxItems)
	if err != nil {
		return nil, qualityOutput{}, err
	}

	query := app.QualityQuery{City: strings.TrimSpace(in.City), Limit: limit}

	if in.Station != "" {
		code, err := parseStation(in.Station)
		if err != nil {
			return nil, qualityOutput{}, err
		}

		query.Station = &code
	}

	quality, err := h.uses.GetQuality.Get(ctx, query)
	if err != nil {
		return nil, qualityOutput{}, toolError(err, in.City, in.Station)
	}

	return nil, toQualityOutput(quality, language), nil
}

func (h handler) listHistory(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listHistoryInput,
) (*mcp.CallToolResult, historyOutput, error) {
	query, err := historyQuery(in)
	if err != nil {
		return nil, historyOutput{}, err
	}

	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, historyOutput{}, err
	}

	history, err := h.uses.ListHistory.List(ctx, query)
	if err != nil {
		return nil, historyOutput{}, toolError(err, "", in.Station)
	}

	return nil, toHistoryOutput(history, language), nil
}

func historyQuery(in listHistoryInput) (app.HistoryQuery, error) {
	code, err := parseStation(in.Station)
	if err != nil {
		return app.HistoryQuery{}, err
	}

	from, err := parseDate("from", in.From)
	if err != nil {
		return app.HistoryQuery{}, err
	}

	to, err := parseDate("to", in.To)
	if err != nil {
		return app.HistoryQuery{}, err
	}

	pollutant, err := parsePollutant(in.Pollutant)
	if err != nil {
		return app.HistoryQuery{}, err
	}

	limit, err := withinRange("limit", in.Limit, defaultReadings, maxItems)
	if err != nil {
		return app.HistoryQuery{}, err
	}

	return app.HistoryQuery{
		Station: code, From: from, To: to, Pollutant: pollutant, Limit: limit,
	}, nil
}
