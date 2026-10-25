package cavea

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	caveaclient "github.com/abgeo/georgia-mcp/pkg/cavea"
)

// errOtherLanguage marks a session in a language that the context does not serve. The
// session stays out of the list, and the rest of the answer stays.
var errOtherLanguage = errors.New("session: other language")

// classOf maps a price key or a seat class of Cavea to a seat class.
func classOf(key string) (domain.SeatClass, error) {
	switch key {
	case "eco":
		return domain.Economy, nil
	case "std":
		return domain.Standard, nil
	case "vip":
		return domain.VIP, nil
	case "sta":
		return domain.Status, nil
	default:
		return "", fmt.Errorf("%w: %q", domain.ErrUnknownSeatClass, key)
	}
}

func audioOf(language string) (domain.AudioLanguage, error) {
	switch language {
	case "geo":
		return domain.AudioGeorgian, nil
	case "eng":
		return domain.AudioEnglish, nil
	default:
		return "", fmt.Errorf("%w: %q", errOtherLanguage, language)
	}
}

// vendorSession is the part that a page session and a session of a movie share.
type vendorSession struct {
	id, movieID, cinema, audio string
	subtitles                  *string
	formats                    caveaclient.Formats
	prices                     map[string]string
}

func (a *Adapter) halls(ctx context.Context) (map[string]string, error) {
	halls, err := a.client.HallMap(ctx)
	if err != nil {
		return nil, unavailable("reading the halls of Cavea", err)
	}

	return halls, nil
}

// Sessions returns the sessions of every movie from the page of sessions, each with the
// price of the class Standard.
func (a *Adapter) Sessions(ctx context.Context) (app.SessionList, error) {
	halls, err := a.halls(ctx)
	if err != nil {
		return app.SessionList{}, err
	}

	groups, err := a.client.Sessions(ctx)
	if err != nil {
		return app.SessionList{}, unavailable("reading the sessions of Cavea", err)
	}

	found := []vendorSession{}

	for _, group := range groups {
		for _, movie := range group.Movies {
			for _, session := range movie.Sessions {
				found = append(found, vendorSession{
					id: session.ID, movieID: movie.Movie.ID, cinema: session.Cinema,
					audio: session.SessionLanguage, subtitles: session.SubtitleLanguage,
					formats: session.Formats, prices: map[string]string{"std": session.Price},
				})
			}
		}
	}

	return a.sessionList(found, halls)
}

// MovieSessions returns the sessions of one movie, each with the price of each class.
func (a *Adapter) MovieSessions(ctx context.Context, movieID string) (app.SessionList, error) {
	halls, err := a.halls(ctx)
	if err != nil {
		return app.SessionList{}, err
	}

	sessions, err := a.client.MovieSessions(ctx, movieID)
	if err != nil {
		return app.SessionList{}, unavailable("reading the sessions of a movie of Cavea", err)
	}

	found := make([]vendorSession, 0, len(sessions))
	for _, session := range sessions {
		found = append(found, vendorSession{
			id: session.ID, movieID: movieID, cinema: session.Cinema,
			audio: session.SessionLanguage, subtitles: session.SubtitleLanguage,
			formats: session.Formats, prices: session.Prices,
		})
	}

	return a.sessionList(found, halls)
}

func (a *Adapter) sessionList(
	found []vendorSession,
	halls map[string]string,
) (app.SessionList, error) {
	sessions := make([]domain.Session, 0, len(found))

	for _, item := range found {
		session, err := convertSession(item, halls)
		if errors.Is(err, errOtherLanguage) {
			continue
		}

		if err != nil {
			return app.SessionList{}, unavailable("reading the sessions of Cavea", err)
		}

		sessions = append(sessions, session)
	}

	return app.SessionList{Sessions: sessions, ReadAt: a.readAt()}, nil
}

func convertSession(item vendorSession, halls map[string]string) (domain.Session, error) {
	code, err := domain.ParseSessionCode(item.id)
	if err != nil {
		return domain.Session{}, err
	}

	audio, err := audioOf(item.audio)
	if err != nil {
		return domain.Session{}, err
	}

	var subtitles *domain.AudioLanguage

	if item.subtitles != nil && *item.subtitles != "" {
		language, err := audioOf(*item.subtitles)
		if err != nil {
			return domain.Session{}, err
		}

		subtitles = &language
	}

	prices := make(map[domain.SeatClass]domain.Price, len(item.prices))

	for key, text := range item.prices {
		class, err := classOf(key)
		if err != nil {
			return domain.Session{}, err
		}

		price, err := domain.NewPrice(text)
		if err != nil {
			return domain.Session{}, fmt.Errorf("the price of %s: %w", code, err)
		}

		prices[class] = price
	}

	return domain.Session{
		Code:      code,
		MovieID:   item.movieID,
		CinemaID:  item.cinema,
		Hall:      hallOf(code, halls),
		Audio:     audio,
		Subtitles: subtitles,
		Formats:   formatsOf(item.formats),
		Prices:    prices,
	}, nil
}

// hallOf returns the label of the hall map, or the part of the hall key after _ in
// uppercase: "ccm_x" gives "X".
func hallOf(code domain.SessionCode, halls map[string]string) string {
	if hall, ok := halls[code.HallKey()]; ok && hall != "" {
		return hall
	}

	_, hall, _ := strings.Cut(code.HallKey(), "_")

	return strings.ToUpper(hall)
}

func formatsOf(marks caveaclient.Formats) []domain.Format {
	marked := map[domain.Format]bool{
		domain.Format3D:             marks.Is3D,
		domain.FormatIMAX:           marks.IsIMAX,
		domain.FormatDolbyAtmos:     marks.IsAtmos,
		domain.FormatScreenX:        marks.IsScreenX,
		domain.FormatInfinityVision: marks.IsInfinityVision,
	}

	formats := []domain.Format{}

	for _, format := range domain.Formats() {
		if marked[format] {
			formats = append(formats, format)
		}
	}

	return formats
}
