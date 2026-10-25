package domain

import "time"

// Cinema is one open cinema of Cavea.
type Cinema struct {
	// ID is the identifier of Cavea. A session names its cinema with it.
	ID   string
	Code CinemaCode
	Name Text
	// City and Address are in English. Cavea gives no Georgian text.
	City    string
	Address string
}

// Movie is one movie that Cavea shows now or soon.
type Movie struct {
	// ID is the identifier of Cavea. A session names its movie with it.
	ID          string
	Code        MovieCode
	Status      MovieStatus
	Name        Text
	Genres      TextList
	Duration    time.Duration
	AgeRating   string
	Premiere    string // "2026-10-22", a year as "2026", or empty
	Directors   TextList
	Actors      TextList
	Description Text
}

// Session is one showing of a movie in one hall at one start time.
type Session struct {
	Code     SessionCode
	MovieID  string
	CinemaID string
	// Hall is the label on the door of the hall: "5" or "IMAX".
	Hall      string
	Audio     AudioLanguage
	Subtitles *AudioLanguage
	// Formats is in the order of the function Formats. A 2D session has none.
	Formats []Format
	// Prices holds the class Standard only when Cavea gives one price.
	Prices map[SeatClass]Price
}

// Seat is one seat of the hall of a session.
type Seat struct {
	Row          string
	Number       int
	Class        SeatClass
	SpecialNeeds bool
	Taken        bool
}
