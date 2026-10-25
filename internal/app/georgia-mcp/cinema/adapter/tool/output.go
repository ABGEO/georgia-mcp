package tool

import (
	"cmp"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const occupancyNote = "Cavea did not give the occupancy levels."

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}

func toCinemasOutput(list app.CinemaList, language kernel.Language) cinemasOutput {
	cinemas := make([]cinemaOutput, 0, len(list.Cinemas))
	for _, cinema := range list.Cinemas {
		cinemas = append(cinemas, cinemaOutput{
			Code:    cinema.Code.String(),
			Name:    cinema.Name.In(language),
			City:    cinema.City,
			Address: cinema.Address,
		})
	}

	return cinemasOutput{
		Source:  source,
		AsOf:    instant(list.ReadAt),
		Stale:   list.Stale,
		Cinemas: cinemas,
	}
}

func toMoviesOutput(list app.Movies, language kernel.Language) moviesOutput {
	movies := make([]movieOutput, 0, len(list.Movies))
	for _, movie := range list.Movies {
		movies = append(movies, movieOutput{
			Code:            movie.Code.String(),
			Name:            movie.Name.In(language),
			Status:          string(movie.Status),
			DurationMinutes: int(movie.Duration / time.Minute),
			AgeRating:       movie.AgeRating,
			Genres:          movie.Genres.In(language),
			Premiere:        movie.Premiere,
			Directors:       movie.Directors.In(language),
			Actors:          movie.Actors.In(language),
			Description:     movie.Description.OnlyIn(language),
			URL:             movieAddress + movie.Code.String(),
		})
	}

	return moviesOutput{
		Source: source,
		AsOf:   instant(list.ReadAt),
		Stale:  list.Stale,
		Total:  list.Total,
		Movies: movies,
	}
}

func toSessionsOutput(list app.Sessions, language kernel.Language) sessionsOutput {
	sessions := make([]sessionOutput, 0, len(list.Sessions))
	for _, item := range list.Sessions {
		sessions = append(sessions, toSessionOutput(item, language))
	}

	out := sessionsOutput{
		Source:   source,
		AsOf:     instant(list.ReadAt),
		Stale:    list.Stale,
		Total:    list.Total,
		Sessions: sessions,
	}

	switch {
	case list.Occupancy:
		out.OccupancyAsOf = instant(list.OccupancyAt)
	case len(sessions) > 0:
		out.OccupancyNote = occupancyNote
	}

	return out
}

func toSessionOutput(item app.ScheduledSession, language kernel.Language) sessionOutput {
	session := item.Session

	formats := make([]string, 0, len(session.Formats))
	for _, format := range session.Formats {
		formats = append(formats, string(format))
	}

	prices := []priceOutput{}

	for _, class := range domain.SeatClasses() {
		if price, ok := session.Prices[class]; ok {
			prices = append(prices, priceOutput{Class: string(class), Price: price.String()})
		}
	}

	out := sessionOutput{
		Code:          session.Code.String(),
		MovieCode:     item.Movie.Code.String(),
		MovieName:     item.Movie.Name.In(language),
		CinemaCode:    item.Cinema.Code.String(),
		CinemaName:    item.Cinema.Name.In(language),
		Hall:          session.Hall,
		Start:         session.Code.Start().Format(time.RFC3339),
		AudioLanguage: string(session.Audio),
		Formats:       formats,
		Prices:        prices,
		Occupancy:     string(item.Occupancy),
	}

	if session.Subtitles != nil {
		out.SubtitleLanguage = string(*session.Subtitles)
	}

	return out
}

func toSeatMapOutput(seatMap app.SeatMap, language kernel.Language) seatMapOutput {
	session := toSessionOutput(seatMap.Session, language)
	session.Occupancy = ""

	classes := make([]classOutput, 0, len(seatMap.Classes))
	for _, count := range seatMap.Classes {
		class := classOutput{Class: string(count.Class), Seats: count.Seats, Free: count.Free}
		if count.Price != nil {
			class.Price = count.Price.String()
		}

		classes = append(classes, class)
	}

	return seatMapOutput{
		Source:  source,
		AsOf:    instant(seatMap.ReadAt),
		Session: session,
		Classes: classes,
		Rows:    rowsOf(seatMap.Seats),
	}
}

// rowsOf groups the seats by row, in the order in which each row first comes, and sorts
// the seats of each row by number.
func rowsOf(seats []domain.Seat) []rowOutput {
	rows := []rowOutput{}
	index := map[string]int{}

	for _, seat := range seats {
		position, ok := index[seat.Row]
		if !ok {
			position = len(rows)
			index[seat.Row] = position
			rows = append(rows, rowOutput{Row: seat.Row, Seats: []seatOutput{}})
		}

		rows[position].Seats = append(rows[position].Seats, seatOutput{
			Number:       seat.Number,
			Class:        string(seat.Class),
			Taken:        seat.Taken,
			SpecialNeeds: seat.SpecialNeeds,
		})
	}

	for _, row := range rows {
		slices.SortStableFunc(
			row.Seats,
			func(a, b seatOutput) int { return cmp.Compare(a.Number, b.Number) },
		)
	}

	return rows
}
