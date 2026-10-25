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
	listSeparator  = ", "
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
			ID:          movie.ID,
			Code:        code,
			Status:      status,
			Name:        domain.Text{Georgian: movie.Name, English: movie.NameEnglish},
			Genres:      pairItems(movie.Genre, movie.GenreEng),
			Duration:    time.Duration(movie.Duration) * time.Minute,
			AgeRating:   movie.MPAARating,
			Premiere:    premiere(movie.PremiereEng, movie.Premiere),
			Directors:   pairItems(movie.Directors, movie.DirectorsEng),
			Actors:      pairItems(movie.Actors, movie.ActorsEng),
			Description: domain.Text{Georgian: movie.Description, English: movie.DescriptionEng},
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
			Name:      domain.Text{Georgian: movie.Name, English: movie.NameEnglish},
			Duration:  time.Duration(movie.Duration) * time.Minute,
			AgeRating: movie.MPAARating,
		})
	}

	return app.MovieList{Movies: movies, ReadAt: a.readAt()}, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

// pairItems splits two lists of one text and pairs their items by position.
func pairItems(georgian, english string) []domain.Text {
	kaItems, enItems := splitItems(georgian), splitItems(english)

	switch {
	case len(kaItems) == 0 && len(enItems) == 0:
		return nil
	case len(enItems) == 0:
		return textsOf(
			kaItems,
			func(item string) domain.Text { return domain.Text{Georgian: item} },
		)
	case len(kaItems) == 0:
		return textsOf(enItems, func(item string) domain.Text { return domain.Text{English: item} })
	case len(kaItems) != len(enItems):
		return []domain.Text{
			{Georgian: strings.TrimSpace(georgian), English: strings.TrimSpace(english)},
		}
	}

	texts := make([]domain.Text, len(kaItems))
	for index := range kaItems {
		texts[index] = domain.Text{Georgian: kaItems[index], English: enItems[index]}
	}

	return texts
}

func splitItems(text string) []string {
	items := []string{}

	for item := range strings.SplitSeq(text, listSeparator) {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

func textsOf(items []string, text func(string) domain.Text) []domain.Text {
	texts := make([]domain.Text, len(items))
	for index, item := range items {
		texts[index] = text(item)
	}

	return texts
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
