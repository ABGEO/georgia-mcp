package domain

import (
	"fmt"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Text is a text of Cavea in Georgian and in English.
type Text struct {
	Georgian string
	English  string
}

// In returns the text in a language, or the text in the other language when Cavea gives
// none.
func (t Text) In(language kernel.Language) string {
	first, second := t.English, t.Georgian
	if language == kernel.LanguageGeorgian {
		first, second = second, first
	}

	if first != "" {
		return first
	}

	return second
}

// Price is the price in lari of one ticket. It is greater than zero.
type Price struct {
	amount decimal.Decimal
}

// NewPrice returns the price of a decimal text greater than zero: "20".
func NewPrice(text string) (Price, error) {
	amount, err := decimal.NewFromString(text)
	if err != nil || !amount.IsPositive() {
		return Price{}, fmt.Errorf("%w: %q", ErrInvalidPrice, text)
	}

	return Price{amount: amount}, nil
}

// Amount returns the amount in lari.
func (p Price) Amount() decimal.Decimal {
	return p.amount
}

// String returns the amount as a decimal text: "20".
func (p Price) String() string {
	return p.amount.String()
}

// MovieStatus tells whether Cavea shows a movie now or soon.
type MovieStatus string

const (
	// NowShowing is a movie that Cavea shows now.
	NowShowing MovieStatus = "now_showing"
	// ComingSoon is a movie that Cavea shows soon.
	ComingSoon MovieStatus = "coming_soon"
)

// MovieStatuses returns every movie status.
func MovieStatuses() []MovieStatus {
	return []MovieStatus{NowShowing, ComingSoon}
}

// ParseMovieStatus returns the movie status of a text. The empty text gives NowShowing.
func ParseMovieStatus(text string) (MovieStatus, error) {
	if text == "" {
		return NowShowing, nil
	}

	return parseOne(MovieStatuses(), text, ErrUnknownMovieStatus)
}

// AudioLanguage is the language of the sound of a session.
type AudioLanguage string

const (
	// AudioGeorgian is a session in Georgian.
	AudioGeorgian AudioLanguage = "ka"
	// AudioEnglish is a session in English.
	AudioEnglish AudioLanguage = "en"
)

// AudioLanguages returns every audio language.
func AudioLanguages() []AudioLanguage {
	return []AudioLanguage{AudioGeorgian, AudioEnglish}
}

// ParseAudioLanguage returns the audio language of a text: "ka" or "en".
func ParseAudioLanguage(text string) (AudioLanguage, error) {
	return parseOne(AudioLanguages(), text, ErrUnknownAudioLanguage)
}

// Format is a screen or sound technology of a session.
type Format string

const (
	// Format3D is a session in 3D.
	Format3D Format = "3d"
	// FormatIMAX is a session in an IMAX hall.
	FormatIMAX Format = "imax"
	// FormatDolbyAtmos is a session with Dolby Atmos sound.
	FormatDolbyAtmos Format = "dolby_atmos"
	// FormatScreenX is a session in a ScreenX hall.
	FormatScreenX Format = "screenx"
	// FormatInfinityVision is a session in an Infinity Vision hall.
	FormatInfinityVision Format = "infinity_vision"
)

// Formats returns every format, in the order of an answer.
func Formats() []Format {
	return []Format{
		Format3D, FormatIMAX, FormatDolbyAtmos, FormatScreenX, FormatInfinityVision,
	}
}

// ParseFormat returns the format of a text.
func ParseFormat(text string) (Format, error) {
	return parseOne(Formats(), text, ErrUnknownFormat)
}

// SeatClass is the class of a seat with its own price.
type SeatClass string

const (
	// Economy is the class Economy.
	Economy SeatClass = "economy"
	// Standard is the class Standard.
	Standard SeatClass = "standard"
	// VIP is the class VIP.
	VIP SeatClass = "vip"
	// Status is the class Status.
	Status SeatClass = "status"
)

// SeatClasses returns every seat class, in the order of an answer.
func SeatClasses() []SeatClass {
	return []SeatClass{Economy, Standard, VIP, Status}
}

// ParseSeatClass returns the seat class of a text.
func ParseSeatClass(text string) (SeatClass, error) {
	return parseOne(SeatClasses(), text, ErrUnknownSeatClass)
}

// OccupancyLevel is the share of sold seats of a session, as Cavea shows it.
type OccupancyLevel string

const (
	// ManySeats is a session with many free seats.
	ManySeats OccupancyLevel = "many_seats"
	// FillingUp is a session that fills up.
	FillingUp OccupancyLevel = "filling_up"
	// AlmostSoldOut is a session with few free seats.
	AlmostSoldOut OccupancyLevel = "almost_sold_out"
)

func parseOne[T ~string](values []T, text string, unknown error) (T, error) {
	if !slices.Contains(values, T(text)) {
		var zero T

		return zero, fmt.Errorf("%w: %q", unknown, text)
	}

	return T(text), nil
}
