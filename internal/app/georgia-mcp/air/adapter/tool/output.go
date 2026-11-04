package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func toQualityOutput(quality app.Quality, language kernel.Language) qualityOutput {
	stations := make([]stationOutput, 0, len(quality.Stations))

	for _, item := range quality.Stations {
		readings := make([]readingOutput, 0, len(item.Readings))
		for _, reading := range item.Readings {
			readings = append(readings, readingOutput{
				Pollutant: string(reading.Pollutant),
				Value:     reading.Reading.Value.String(),
				Unit:      reading.Pollutant.Unit(),
				Time:      instant(reading.Reading.Time),
				Level:     reading.Reading.Level.String(),
			})
		}

		stations = append(stations, stationOutput{
			Code:     item.Station.Code.String(),
			Address:  item.Station.Address.In(language),
			City:     item.City.Name.In(language),
			Lat:      item.Station.Lat,
			Lon:      item.Station.Lon,
			Level:    item.Level.String(),
			Readings: readings,
		})
	}

	return qualityOutput{
		Source:   source,
		AsOf:     instant(quality.ReadAt),
		Stale:    quality.Stale,
		Total:    quality.Total,
		Stations: stations,
	}
}

func toHistoryOutput(history app.History, language kernel.Language) historyOutput {
	series := make([]seriesOutput, 0, len(history.Station.Series))

	for _, item := range history.Station.Series {
		series = append(series, seriesOutput{
			Pollutant: string(item.Pollutant),
			Unit:      item.Pollutant.Unit(),
			Total:     history.Total[item.Pollutant],
			Readings:  toPoints(item.Readings),
		})
	}

	return historyOutput{
		Source:  source,
		AsOf:    instant(history.ReadAt),
		Stale:   history.Stale,
		Code:    history.Station.Code.String(),
		Address: history.Station.Address.In(language),
		City:    history.City.Name.In(language),
		From:    instant(history.From),
		To:      instant(history.To),
		Series:  series,
	}
}

func toPoints(readings []domain.Reading) []pointOutput {
	points := make([]pointOutput, 0, len(readings))
	for _, reading := range readings {
		points = append(points, pointOutput{
			Value: reading.Value.String(),
			Time:  instant(reading.Time),
			Level: reading.Level.String(),
		})
	}

	return points
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
