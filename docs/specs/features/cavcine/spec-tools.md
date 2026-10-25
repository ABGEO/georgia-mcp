---
id: CAVCINE
title: The cinemas, movies, sessions, and seats of Cavea, the tools
type: spec
status: approved
created: 2026-10-25
updated: 2026-10-25
context: cinema
approved_by: Temuri
approved_on: 2026-10-25
constrained_by: [MCP]
requirements: features/cavcine/requirements.md
---

# Specification: The cinemas, movies, sessions, and seats of Cavea, the tools

## 1. The tools

| Tool                    | Input (all optional unless marked)                                          | Output          | Realizes |
| ----------------------- | --------------------------------------------------------------------------- | --------------- | -------- |
| `cinema_list_cinemas`   | `language`                                                                  | `cinemasOutput` | `CAVCINE-FR-001`, `CAVCINE-FR-011` |
| `cinema_list_movies`    | `status`, `language`, `limit`                                               | `moviesOutput`  | `CAVCINE-FR-002`, `CAVCINE-FR-011` |
| `cinema_search_movies`  | `query` (required), `language`, `limit`                                     | `moviesOutput`  | `CAVCINE-FR-003` |
| `cinema_list_sessions`  | `movie`, `date`, `cinema`, `audio_language`, `format`, `language`, `limit`  | `sessionsOutput` | `CAVCINE-FR-004` to `CAVCINE-FR-008` |
| `cinema_get_seat_map`   | `session` (required), `language`                                            | `seatMapOutput` | `CAVCINE-FR-009` |

Every tool has the annotations of `MCP-004`. `source` is `Cavea`. `as_of` is `ReadAt` in
`kernel.TbilisiZone()`, written with `time.RFC3339`. A `limit` of 0 asks the default.
Each description ends with `Data from the web site of Cavea, cavea.ge.`

The shared input field:

```go
type languageField struct {
    Language string `json:"language,omitempty" jsonschema:"The language of the names and the texts: ka or en. The default is en."`
}
```

### 1.1 `cinema_list_cinemas`

Title: `List the cinemas of Cavea`. Description:

> Lists the open cinemas of Cavea, the largest cinema chain of Georgia, in Tbilisi and
> Batumi. Returns the code, the name, the city, and the address of each cinema. Use the
> code in cinema_list_sessions.

```go
type listCinemasInput struct{ languageField }

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
```

### 1.2 `cinema_list_movies`

Title: `List the movies of Cavea`. Description:

> Lists the movies that Cavea shows now, or the movies coming soon. Returns the code, the
> name, the genres, the duration in minutes, the age rating, the premiere, the directors,
> the actors, and a description of each movie. Use the code in cinema_list_sessions.

```go
type listMoviesInput struct {
    Status string `json:"status,omitempty" jsonschema:"now_showing or coming_soon. The default is now_showing."`
    Limit  int    `json:"limit,omitempty"  jsonschema:"The maximum count of movies, 1 to 100. The default is 50."`
    languageField
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
    Status          string   `json:"status"                jsonschema:"now_showing or coming_soon. A movie before its premiere date is coming_soon."`
    DurationMinutes int      `json:"duration_minutes"      jsonschema:"The duration of the movie, in minutes."`
    AgeRating       string   `json:"age_rating,omitempty"  jsonschema:"The age rating of the MPAA, for example PG-13."`
    Genres          []string `json:"genres,omitempty"      jsonschema:"The genres of the movie."`
    Premiere        string   `json:"premiere,omitempty"    jsonschema:"The premiere in Georgia: YYYY-MM-DD, or the year only."`
    Directors       []string `json:"directors,omitempty"   jsonschema:"The directors."`
    Actors          []string `json:"actors,omitempty"      jsonschema:"The main actors."`
    Description     string   `json:"description,omitempty" jsonschema:"A short description of the story."`
    URL             string   `json:"url"                   jsonschema:"The page of the movie on cavea.ge, where a person buys a ticket."`
}
```

`url` is `https://cavea.ge/movie/<code>`, the page that the site of Cavea links.

`cinema_search_movies` fills `code`, `name`, `status`, `duration_minutes`, `age_rating`,
and `url` only. Cavea gives no other field in a search.

### 1.3 `cinema_search_movies`

Title: `Search the movies of Cavea`. Description:

> Finds the movies of Cavea, now showing or coming soon, by a part of the English name.
> Give the text in English: Cavea does not match a Georgian name. Returns the code, the
> name, the status, the duration in minutes, and the age rating of each movie.

```go
type searchMoviesInput struct {
    Query string `json:"query"           jsonschema:"A part of the English name of the movie, 2 to 100 characters."`
    Limit int    `json:"limit,omitempty" jsonschema:"The maximum count of movies, 1 to 100. The default is 20."`
    languageField
}
```

### 1.4 `cinema_list_sessions`

Title: `List the sessions of Cavea`. Description:

> Lists the sessions of the movies of Cavea on one date that have not started, in the
> order of the start time. Narrow the list by a movie, a cinema, an audio language, or a
> format. Returns the cinema, the hall, the start time, the languages, the formats, the
> price in lari, and the occupancy level of each session. With a movie, each session has
> the price of each seat class. With no movie, it has the price of the class Standard.
> Use the session code in cinema_get_seat_map.

```go
type listSessionsInput struct {
    Movie         string `json:"movie,omitempty"          jsonschema:"The code of a movie from cinema_list_movies or cinema_search_movies."`
    Date          string `json:"date,omitempty"           jsonschema:"The date in Tbilisi, YYYY-MM-DD. The default is today. Cavea publishes about 3 days."`
    Cinema        string `json:"cinema,omitempty"         jsonschema:"The code of a cinema from cinema_list_cinemas."`
    AudioLanguage string `json:"audio_language,omitempty" jsonschema:"The language of the sound: ka or en."`
    Format        string `json:"format,omitempty"         jsonschema:"One format: 3d, imax, dolby_atmos, screenx, or infinity_vision."`
    Limit         int    `json:"limit,omitempty"          jsonschema:"The maximum count of sessions, 1 to 100. The default is 30."`
    languageField
}

type sessionsOutput struct {
    Source         string          `json:"source"                    jsonschema:"The source of the data: Cavea."`
    AsOf           string          `json:"as_of"                     jsonschema:"The instant at which the server read the sessions, RFC 3339."`
    Stale          bool            `json:"stale"                     jsonschema:"True when Cavea did not answer and the server returns sessions it read before."`
    OccupancyAsOf  string          `json:"occupancy_as_of,omitempty" jsonschema:"The instant at which the server read the occupancy levels, RFC 3339."`
    OccupancyNote  string          `json:"occupancy_note,omitempty"  jsonschema:"Why the sessions have no occupancy level."`
    Total          int             `json:"total"                     jsonschema:"The count of sessions that matched. More than the count of sessions when the limit cut the list."`
    Sessions       []sessionOutput `json:"sessions"                  jsonschema:"The sessions, in the order of the start time, then the cinema code."`
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
```

The prices follow the order of `SeatClass`: `economy`, `standard`, `vip`, `status`. When
`Occupancy` is false and the list holds a session, `occupancy_note` is `Cavea did not give
the occupancy levels.` An empty list has neither `occupancy_as_of` nor `occupancy_note`.

### 1.5 `cinema_get_seat_map`

Title: `Get the seat map of a session of Cavea`. Description:

> Returns the seats of the hall of one session of Cavea that has not started: each row,
> each seat with its class, a mark for a taken seat, and a mark for special needs. Gives
> the count of free seats and the price in lari of each seat class. The server reads the
> map at each call. It reserves no seat. A person buys the ticket on cavea.ge.

```go
type getSeatMapInput struct {
    Session string `json:"session" jsonschema:"The session code from cinema_list_sessions, for example cag_5/27-10-2026/22:15."`
    languageField
}

type seatMapOutput struct {
    Source     string        `json:"source"      jsonschema:"The source of the data: Cavea."`
    AsOf       string        `json:"as_of"       jsonschema:"The instant at which the server read the seats, RFC 3339."`
    Session    sessionOutput `json:"session"     jsonschema:"The session. It has no occupancy level."`
    Classes    []classOutput `json:"classes"     jsonschema:"Each seat class of the hall, with its price and its free seats."`
    Rows       []rowOutput   `json:"rows"        jsonschema:"The rows, in the order of Cavea."`
}

type classOutput struct {
    Class string `json:"class" jsonschema:"The seat class: economy, standard, vip, or status."`
    Price string `json:"price,omitempty" jsonschema:"The price of one ticket in lari, as a decimal string."`
    Seats int    `json:"seats" jsonschema:"The count of seats of the class."`
    Free  int    `json:"free"  jsonschema:"The count of seats of the class that are not taken."`
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
```

`classes` holds only the classes of the hall. A class with no price in the session has no
`price`.

## 2. Errors

| Condition                                  | Sentinel                        | Tool message |
| ------------------------------------------ | ------------------------------- | ------------ |
| A language other than `ka` and `en`        | `kernel.ErrUnknownLanguage`     | `language: "ru" is not a language of this tool. Use ka or en.` |
| An unknown status                          | `domain.ErrUnknownMovieStatus`  | `status: "archived" is not a status. Use now_showing or coming_soon.` |
| A short or long query                      | `app.ErrInvalidQuery`           | `query: give 2 to 100 characters, in English.` |
| A limit out of range                       | none, the tool checks it        | `limit: 500 is out of range. Use 1 to 100.` |
| A date that is not `YYYY-MM-DD`            | `kernel.ErrInvalidDate`         | `date: "27.10" is not a date. Use YYYY-MM-DD.` |
| A date before today                        | `app.ErrPastDate`               | `date: 2026-10-24 is before today. Use today or a later date.` |
| An unknown audio language                  | `domain.ErrUnknownAudioLanguage`| `audio_language: "ru" is not a language of Cavea. Use ka or en.` |
| An unknown format                          | `domain.ErrUnknownFormat`       | `format: "4dx" is not a format. Use 3d, imax, dolby_atmos, screenx, or infinity_vision.` |
| A movie that Cavea does not list           | `app.ErrUnknownMovie`           | `movie: "x" is not a movie of Cavea. Use cinema_list_movies or cinema_search_movies.` |
| A cinema that Cavea does not list          | `app.ErrUnknownCinema`          | `cinema: "x" is not a cinema of Cavea. Use cinema_list_cinemas.` |
| A session code of a wrong form             | `domain.ErrInvalidSessionCode`  | `session: "xyz" is not a session code. Use a code from cinema_list_sessions, for example cag_5/27-10-2026/22:15.` |
| A session that Cavea does not list         | `app.ErrUnknownSession`         | `session: "cag_5/29-10-2026/22:15" is not a session of Cavea. Use cinema_list_sessions.` |
| A session that started                     | `app.ErrSessionStarted`         | `session: "cag_5/24-10-2026/22:15" started. Use cinema_list_sessions for a later session.` |
| Cavea fails, with no stale list            | `kernel.ErrSourceUnavailable`   | `Cavea did not answer. Try again in a few minutes.` |
| Any other error                            | none                            | `The server failed to read the data of Cavea.` |

## Retired identifiers

This file has no retired identifier.
