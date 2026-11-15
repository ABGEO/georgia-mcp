package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// HolidaysLister gives the holidays of a year, or the next holidays from a date.
type HolidaysLister interface {
	List(ctx context.Context, query app.ListQuery) (app.HolidayList, error)
}

// DayGetter tells whether a date is a public holiday.
type DayGetter interface {
	Get(ctx context.Context, query app.DayQuery) (app.Day, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	ListHolidays HolidaysLister
	GetDay       DayGetter
}

const (
	source = "Nager.Date"

	defaultNext = 1
	maxNext     = 20

	listHolidaysDescription = "Returns the public holidays of Georgia: every holiday of " +
		"one year, or the next holidays on or after a date. Each holiday gives its date " +
		"(YYYY-MM-DD), its weekday, and its name in Georgian or English. With no input, it " +
		"returns every holiday of the current year in Georgia. It supports the years from " +
		"2025 to the next year. The list holds no one-off day off that the government " +
		"declares by decree. Data from Nager.Date."
	getDayDescription = "Tells whether one date is a public holiday in Georgia, with its " +
		"weekday and the holiday name in Georgian or English. The default date is today in " +
		"Georgia. It supports the years from 2025 to the next year. A one-off day off that " +
		"the government declares by decree is not in the data: a false answer does not rule " +
		"one out. Data from Nager.Date."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "holidays_list_holidays", Title: "List the public holidays of Georgia",
		Description: listHolidaysDescription,
	}, handlers.listHolidays)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "holidays_get_day", Title: "Check a date for a public holiday in Georgia",
			Description: getDayDescription,
		}, handlers.getDay)
	}

	if err != nil {
		return fmt.Errorf("registering the holiday tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) listHolidays(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listHolidaysInput,
) (*mcp.CallToolResult, listOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, listOutput{}, err
	}

	query, err := listQuery(in)
	if err != nil {
		return nil, listOutput{}, err
	}

	field := "from"
	if in.From == "" {
		field = "year"
	}

	list, err := h.uses.ListHolidays.List(ctx, query)
	if err != nil {
		return nil, listOutput{}, toolError(err, field, in.From)
	}

	return nil, toListOutput(list, language), nil
}

func listQuery(in listHolidaysInput) (app.ListQuery, error) {
	var query app.ListQuery

	if in.Year != 0 {
		query.Year = &in.Year
	}

	from, err := parseDate("from", in.From)
	if err != nil || from == nil {
		return query, err
	}

	query.From = from

	query.Limit, err = withinRange("limit", in.Limit, defaultNext, maxNext)
	if err != nil {
		return app.ListQuery{}, err
	}

	return query, nil
}

func (h handler) getDay(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getDayInput,
) (*mcp.CallToolResult, dayOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, dayOutput{}, err
	}

	date, err := parseDate("date", in.Date)
	if err != nil {
		return nil, dayOutput{}, err
	}

	day, err := h.uses.GetDay.Get(ctx, app.DayQuery{Date: date})
	if err != nil {
		return nil, dayOutput{}, toolError(err, "date", in.Date)
	}

	return nil, toDayOutput(day, language), nil
}
