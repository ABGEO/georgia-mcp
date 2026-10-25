package cavea

import (
	"encoding/json"
	"fmt"
)

// Theater is one cinema of the answer of /api/theaters.
type Theater struct {
	ID      string `json:"_id"`
	Name    string `json:"name"`
	NameEng string `json:"name_eng"`
	City    string `json:"city"`
	Address string `json:"address"`
	// IsActive is false for a closed cinema. An open cinema can have no field.
	IsActive    *bool `json:"is_active"`
	TestingOnly bool  `json:"testing_only"`
	IsCinema    bool  `json:"is_cinema"`
}

// Movie is one movie of the grid of the page /movies.
type Movie struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	NameEnglish string `json:"nameEnglish"`
	// Duration is in minutes.
	Duration   int    `json:"duration"`
	MPAARating string `json:"mpaaRating"`
	// Genre, Directors, and Actors are lists in one text, with ", " between two items.
	Genre          string `json:"genre"`
	GenreEng       string `json:"genreEng"`
	Description    string `json:"description"`
	DescriptionEng string `json:"descriptionEng"`
	Directors      string `json:"directors"`
	DirectorsEng   string `json:"directorsEng"`
	Actors         string `json:"actors"`
	ActorsEng      string `json:"actorsEng"`
	// Premiere is "22.10.2026", a year as "2026", or empty.
	Premiere    string `json:"premiere"`
	PremiereEng string `json:"premiereEng"`
}

// FoundMovie is one movie of the answer of /api/movies/search.
type FoundMovie struct {
	ID          string `json:"_id"`
	Slug        string `json:"website_slug"`
	Name        string `json:"name"`
	NameEnglish string `json:"name_english"`
	Duration    int    `json:"duration"`
	MPAARating  string `json:"mpaa_rating"`
	ComingSoon  bool   `json:"coming_soon"`
}

// DateGroup is the sessions of one date of the page /sessions.
type DateGroup struct {
	// Date is "25-10-2026".
	Date   string      `json:"date"`
	Movies []DateMovie `json:"movies"`
}

// DateMovie is one movie of a date group, with its sessions.
type DateMovie struct {
	Movie    PageMovie     `json:"movie"`
	Sessions []PageSession `json:"sessions"`
}

// PageMovie is the movie of a date group.
type PageMovie struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Formats holds the marks of the formats of a session.
type Formats struct {
	Is3D             bool `json:"is_3d"`
	IsIMAX           bool `json:"is_imax"`
	IsAtmos          bool `json:"is_atmos"`
	IsScreenX        bool `json:"is_screenx"`
	IsInfinityVision bool `json:"is_infinity_vision"`
}

// PageSession is one session of the page /sessions.
type PageSession struct {
	Formats

	// ID is the code of the session: "cav_5/25-10-2026/22:00".
	ID string `json:"_id"`
	// Cinema is the ID of a Theater.
	Cinema string `json:"cinema"`
	// SessionLanguage is "geo" or "eng".
	SessionLanguage  string  `json:"session_language"`
	SubtitleLanguage *string `json:"subtitle_language"`
	// Price is the price of the class Standard, in GEL, as text.
	Price string `json:"price"`
}

// MovieSession is one session of the answer of /api/booking/sessions/all.
type MovieSession struct {
	Formats

	ID               string  `json:"_id"`
	Cinema           string  `json:"cinema"`
	SessionLanguage  string  `json:"session_language"`
	SubtitleLanguage *string `json:"subtitle_language"`
	// Prices holds the price of each class, in GEL, as text: "eco", "std", "vip", "sta".
	Prices map[string]string `json:"-"`
}

// UnmarshalJSON decodes a session and the JSON text of its prices.
func (s *MovieSession) UnmarshalJSON(data []byte) error {
	type plain MovieSession

	var raw struct {
		plain

		Prices string `json:"prices"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decoding a session: %w", err)
	}

	*s = MovieSession(raw.plain)

	if raw.Prices == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(raw.Prices), &s.Prices); err != nil {
		return fmt.Errorf("decoding the prices of %s: %w", s.ID, err)
	}

	return nil
}

// Seat is one seat of the answer of /api/booking/seats.
type Seat struct {
	ID   string `json:"_id"`
	Row  string `json:"row"`
	Seat int    `json:"seat"`
	// Class is "eco", "std", "vip", or "sta".
	Class        string `json:"ttype"`
	SpecialNeeds bool   `json:"special_needs"`
	IsTaken      bool   `json:"isTaken"`
}
