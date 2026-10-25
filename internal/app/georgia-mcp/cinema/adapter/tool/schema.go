package tool

type languageField struct {
	Language string `json:"language,omitempty" jsonschema:"The language of the names and the texts: ka or en. The default is en."`
}

type listCinemasInput struct {
	languageField
}

type cinemasOutput struct {
	Source  string         `json:"source"  jsonschema:"The source of the data: Cavea."`
	AsOf    string         `json:"as_of"   jsonschema:"The instant at which the server read the cinemas, RFC 3339."`
	Stale   bool           `json:"stale"   jsonschema:"True when Cavea did not answer and the server returns data it read before."`
	Cinemas []cinemaOutput `json:"cinemas" jsonschema:"The open cinemas, in the order of Cavea."`
}

type cinemaOutput struct {
	Code    string `json:"code"    jsonschema:"The code of the cinema, for example east_point."`
	Name    string `json:"name"    jsonschema:"The name of the cinema."`
	City    string `json:"city"    jsonschema:"The city of the cinema, in English."`
	Address string `json:"address" jsonschema:"The street address of the cinema, in English."`
}

type listMoviesInput struct {
	languageField

	Status string `json:"status,omitempty" jsonschema:"now_showing or coming_soon. The default is now_showing."`
	Limit  int    `json:"limit,omitempty"  jsonschema:"The maximum count of movies, 1 to 100. The default is 50."`
}

type searchMoviesInput struct {
	languageField

	Query string `json:"query"           jsonschema:"A part of the English name of the movie, 2 to 100 characters."`
	Limit int    `json:"limit,omitempty" jsonschema:"The maximum count of movies, 1 to 100. The default is 20."`
}

type moviesOutput struct {
	Source string        `json:"source" jsonschema:"The source of the data: Cavea."`
	AsOf   string        `json:"as_of"  jsonschema:"The instant at which the server read the movies, RFC 3339."`
	Stale  bool          `json:"stale"  jsonschema:"True when Cavea did not answer and the server returns data it read before."`
	Total  int           `json:"total"  jsonschema:"The count of movies that matched. More than the count of movies when the limit cut the list."`
	Movies []movieOutput `json:"movies" jsonschema:"The movies, in the order of Cavea."`
}

type movieOutput struct {
	Code            string   `json:"code"                  jsonschema:"The code of the movie, for example street-fighter."`
	Name            string   `json:"name"                  jsonschema:"The name of the movie."`
	Status          string   `json:"status"                jsonschema:"now_showing or coming_soon."`
	DurationMinutes int      `json:"duration_minutes"      jsonschema:"The duration of the movie, in minutes."`
	AgeRating       string   `json:"age_rating,omitempty"  jsonschema:"The age rating of the MPAA, for example PG-13."`
	Genres          []string `json:"genres,omitempty"      jsonschema:"The genres of the movie."`
	Premiere        string   `json:"premiere,omitempty"    jsonschema:"The premiere in Georgia: YYYY-MM-DD, or the year only."`
	Directors       []string `json:"directors,omitempty"   jsonschema:"The directors."`
	Actors          []string `json:"actors,omitempty"      jsonschema:"The main actors."`
	Description     string   `json:"description,omitempty" jsonschema:"A short description of the story."`
	URL             string   `json:"url"                   jsonschema:"The page of the movie on cavea.ge, where a person buys a ticket."`
}

type listSessionsInput struct {
	languageField

	Movie         string `json:"movie,omitempty"          jsonschema:"The code of a movie from cinema_list_movies or cinema_search_movies."`
	Date          string `json:"date,omitempty"           jsonschema:"The date in Tbilisi, YYYY-MM-DD. The default is today. Cavea publishes about 3 days."`
	Cinema        string `json:"cinema,omitempty"         jsonschema:"The code of a cinema from cinema_list_cinemas."`
	AudioLanguage string `json:"audio_language,omitempty" jsonschema:"The language of the sound: ka or en."`
	Format        string `json:"format,omitempty"         jsonschema:"One format: 3d, imax, dolby_atmos, screenx, or infinity_vision."`
	Limit         int    `json:"limit,omitempty"          jsonschema:"The maximum count of sessions, 1 to 100. The default is 30."`
}

type sessionsOutput struct {
	Source        string          `json:"source"                    jsonschema:"The source of the data: Cavea."`
	AsOf          string          `json:"as_of"                     jsonschema:"The instant at which the server read the sessions, RFC 3339."`
	Stale         bool            `json:"stale"                     jsonschema:"True when Cavea did not answer and the server returns sessions it read before."`
	OccupancyAsOf string          `json:"occupancy_as_of,omitempty" jsonschema:"The instant at which the server read the occupancy levels, RFC 3339."`
	OccupancyNote string          `json:"occupancy_note,omitempty"  jsonschema:"Why the sessions have no occupancy level."`
	Total         int             `json:"total"                     jsonschema:"The count of sessions that matched. More than the count of sessions when the limit cut the list."`
	Sessions      []sessionOutput `json:"sessions"                  jsonschema:"The sessions, in the order of the start time, then the cinema code."`
}

type sessionOutput struct {
	Code             string        `json:"code"                        jsonschema:"The session code, for cinema_get_seat_map."`
	MovieCode        string        `json:"movie_code"                  jsonschema:"The code of the movie."`
	MovieName        string        `json:"movie_name"                  jsonschema:"The name of the movie."`
	CinemaCode       string        `json:"cinema_code"                 jsonschema:"The code of the cinema."`
	CinemaName       string        `json:"cinema_name"                 jsonschema:"The name of the cinema."`
	Hall             string        `json:"hall"                        jsonschema:"The hall: a number or a name, for example 5 or IMAX."`
	Start            string        `json:"start"                       jsonschema:"The start of the session, RFC 3339 with the offset of Tbilisi."`
	AudioLanguage    string        `json:"audio_language"              jsonschema:"The language of the sound: ka or en."`
	SubtitleLanguage string        `json:"subtitle_language,omitempty" jsonschema:"The language of the subtitles: ka or en. Absent when the session has none."`
	Formats          []string      `json:"formats"                     jsonschema:"The formats of the session. Empty for a 2D session."`
	Prices           []priceOutput `json:"prices"                      jsonschema:"The price of each seat class. Standard only when the request has no movie."`
	Occupancy        string        `json:"occupancy,omitempty"         jsonschema:"many_seats, filling_up, or almost_sold_out."`
}

type priceOutput struct {
	Class string `json:"class" jsonschema:"The seat class: economy, standard, vip, or status."`
	Price string `json:"price" jsonschema:"The price of one ticket in lari, as a decimal string."`
}

type getSeatMapInput struct {
	languageField

	Session string `json:"session" jsonschema:"The session code from cinema_list_sessions, for example cag_5/27-10-2026/22:15."`
}

type seatMapOutput struct {
	Source  string        `json:"source"  jsonschema:"The source of the data: Cavea."`
	AsOf    string        `json:"as_of"   jsonschema:"The instant at which the server read the seats, RFC 3339."`
	Session sessionOutput `json:"session" jsonschema:"The session. It has no occupancy level."`
	Classes []classOutput `json:"classes" jsonschema:"Each seat class of the hall, with its price and its free seats."`
	Rows    []rowOutput   `json:"rows"    jsonschema:"The rows, in the order of Cavea."`
}

type classOutput struct {
	Class string `json:"class"           jsonschema:"The seat class: economy, standard, vip, or status."`
	Price string `json:"price,omitempty" jsonschema:"The price of one ticket in lari, as a decimal string."`
	Seats int    `json:"seats"           jsonschema:"The count of seats of the class."`
	Free  int    `json:"free"            jsonschema:"The count of seats of the class that are not taken."`
}

type rowOutput struct {
	Row   string       `json:"row"   jsonschema:"The row label."`
	Seats []seatOutput `json:"seats" jsonschema:"The seats of the row, by number."`
}

type seatOutput struct {
	Number       int    `json:"number"                  jsonschema:"The seat number."`
	Class        string `json:"class"                   jsonschema:"The seat class."`
	Taken        bool   `json:"taken"                   jsonschema:"True when Cavea marks the seat as taken. A person cannot buy it."`
	SpecialNeeds bool   `json:"special_needs,omitempty" jsonschema:"True for a seat for a person with special needs."`
}
