package tool

import (
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func toListOutput(list app.HolidayList, language kernel.Language) listOutput {
	holidays := make([]holidayOutput, 0, len(list.Holidays))

	for _, holiday := range list.Holidays {
		holidays = append(holidays, holidayOutput{
			Date:    holiday.Date.String(),
			Weekday: weekday(holiday.Weekday()),
			Name:    holiday.Name.In(language),
		})
	}

	return listOutput{
		Source:   source,
		AsOf:     instant(list.ReadAt),
		Stale:    list.Stale,
		Holidays: holidays,
	}
}

func toDayOutput(day app.Day, language kernel.Language) dayOutput {
	output := dayOutput{
		Source:  source,
		AsOf:    instant(day.ReadAt),
		Stale:   day.Stale,
		Date:    day.Date.String(),
		Weekday: weekday(domain.Holiday{Date: day.Date}.Weekday()),
	}

	if day.Holiday != nil {
		output.Holiday = true
		output.Name = day.Holiday.Name.In(language)
	}

	return output
}

func weekday(day time.Weekday) string {
	return strings.ToLower(day.String())
}

func instant(at time.Time) string {
	return at.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
