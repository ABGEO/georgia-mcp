package cavea

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	caveaclient "github.com/abgeo/georgia-mcp/pkg/cavea"
)

const (
	brandPrefix    = "Cavea"
	premiereLayout = "02.01.2006"
	yearLength     = 4
	listSeparator  = ","
)

// errNoOpenCinema tells that Cavea marks every cinema closed, which means a change of its
// answer.
var errNoOpenCinema = errors.New("cinemas: none open")

// Adapter implements the four ports of the context with the Cavea client.
type Adapter struct {
	client *caveaclient.Client
	now    func() time.Time
}

var (
	_ app.CinemaSource  = (*Adapter)(nil)
	_ app.MovieSource   = (*Adapter)(nil)
	_ app.SessionSource = (*Adapter)(nil)
	_ app.SeatSource    = (*Adapter)(nil)
)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *caveaclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

func unavailable(action string, err error) error {
	return fmt.Errorf("%s: %w: %w", action, kernel.ErrSourceUnavailable, err)
}

// Cinemas returns the open cinemas. A cinema is open when Cavea does not mark it closed,
// and when it is a cinema that is not for a test.
func (a *Adapter) Cinemas(ctx context.Context) (app.CinemaList, error) {
	theaters, err := a.client.Theaters(ctx)
	if err != nil {
		return app.CinemaList{}, unavailable("reading the cinemas of Cavea", err)
	}

	cinemas := []domain.Cinema{}

	for _, theater := range theaters {
		if !theater.IsCinema || theater.TestingOnly ||
			(theater.IsActive != nil && !*theater.IsActive) {
			continue
		}

		code, err := domain.ParseCinemaCode(cinemaCode(theater.NameEng))
		if err != nil {
			return app.CinemaList{}, unavailable("reading the cinemas of Cavea", err)
		}

		cinemas = append(cinemas, domain.Cinema{
			ID:      theater.ID,
			Code:    code,
			Name:    domain.Text{Georgian: theater.Name, English: theater.NameEng},
			City:    theater.City,
			Address: theater.Address,
		})
	}

	if len(cinemas) == 0 {
		return app.CinemaList{}, unavailable("reading the cinemas of Cavea", errNoOpenCinema)
	}

	return app.CinemaList{Cinemas: cinemas, ReadAt: a.readAt()}, nil
}

// cinemaCode writes an English name with no word Cavea at its start in lowercase, with
// one _ for each run of other characters: "Cavea City Mall Saburtalo" gives
// "city_mall_saburtalo".
func cinemaCode(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), brandPrefix)

	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})

	return strings.Join(words, "_")
}

// Movies returns the movies of a status, in the order of the page.
func (a *Adapter) Movies(ctx context.Context, status domain.MovieStatus) (app.MovieList, error) {
	found, err := a.client.Movies(ctx, status == domain.ComingSoon)
	if err != nil {
		return app.MovieList{}, unavailable("reading the movies of Cavea", err)
	}

	movies := make([]domain.Movie, 0, len(found))

	for _, movie := range found {
		code, err := domain.ParseMovieCode(movie.Slug)
		if err != nil {
			return app.MovieList{}, unavailable("reading the movies of Cavea", err)
		}

		movies = append(movies, domain.Movie{
			ID:        movie.ID,
			Code:      code,
			Status:    status,
			Name:      domain.Text{Georgian: movie.Name, English: english(movie.NameEnglish)},
			Genres:    listOf(movie.Genre, movie.GenreEng),
			Duration:  time.Duration(movie.Duration) * time.Minute,
			AgeRating: movie.MPAARating,
			Premiere:  premiere(movie.PremiereEng, movie.Premiere),
			Directors: listOf(movie.Directors, movie.DirectorsEng),
			Actors:    listOf(movie.Actors, movie.ActorsEng),
			Description: domain.Text{
				Georgian: movie.Description,
				English:  english(movie.DescriptionEng),
			},
		})
	}

	return app.MovieList{Movies: movies, ReadAt: a.readAt()}, nil
}

// Search returns the movies whose English name matches the text, in the order of Cavea.
func (a *Adapter) Search(ctx context.Context, text string) (app.MovieList, error) {
	found, err := a.client.SearchMovies(ctx, text)
	if err != nil {
		return app.MovieList{}, unavailable("searching the movies of Cavea", err)
	}

	movies := make([]domain.Movie, 0, len(found))

	for _, movie := range found {
		code, err := domain.ParseMovieCode(movie.Slug)
		if err != nil {
			return app.MovieList{}, unavailable("searching the movies of Cavea", err)
		}

		status := domain.NowShowing
		if movie.ComingSoon {
			status = domain.ComingSoon
		}

		movies = append(movies, domain.Movie{
			ID:        movie.ID,
			Code:      code,
			Status:    status,
			Name:      domain.Text{Georgian: movie.Name, English: english(movie.NameEnglish)},
			Duration:  time.Duration(movie.Duration) * time.Minute,
			AgeRating: movie.MPAARating,
		})
	}

	return app.MovieList{Movies: movies, ReadAt: a.readAt()}, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

// listOf splits the Georgian list and the English list of one field, each on its own.
func listOf(georgian, englishText string) domain.TextList {
	return domain.TextList{
		Georgian: splitItems(georgian),
		English:  splitItems(english(englishText)),
	}
}

// splitItems splits a list at each comma, trims each item, and drops an empty item. It
// returns nil for a list with no item.
func splitItems(text string) []string {
	var items []string

	for item := range strings.SplitSeq(text, listSeparator) {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

// english returns the empty text for an English field of Cavea that holds a Georgian
// letter, since Cavea then gives no English text.
func english(text string) string {
	if strings.ContainsFunc(text, func(r rune) bool { return unicode.Is(unicode.Georgian, r) }) {
		return ""
	}

	return text
}

// premiere returns the first premiere that is a date or a year, as YYYY-MM-DD or YYYY.
func premiere(texts ...string) string {
	for _, text := range texts {
		text = strings.TrimSpace(text)

		if date, err := time.Parse(premiereLayout, text); err == nil {
			return date.Format(time.DateOnly)
		}

		if len(text) == yearLength &&
			strings.IndexFunc(text, func(r rune) bool { return !unicode.IsDigit(r) }) < 0 {
			return text
		}
	}

	return ""
}
