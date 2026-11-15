package app

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// MaxArticles is the count of articles that one query can ask.
	MaxArticles = 10
	// MaxVersions is the count of the newest versions that the overview holds.
	MaxVersions = 20
	// maxRunes bounds the titles and the paragraphs of the articles of one answer.
	maxRunes = 60_000
	// pieceRunes bounds one piece of the text of an act with no article.
	pieceRunes = 50_000
)

// ActQuery asks one act: its overview with no article, or the articles.
type ActQuery struct {
	ID       domain.ActID
	Version  *int
	Date     *kernel.Date
	Articles []domain.ArticleNumber
	// Piece is the piece of the text, from 1. 0 asks 1.
	Piece    int
	Language kernel.Language
}

// ActView is one version of an act as a query asks it.
type ActView struct {
	// Act holds the metadata. Its body is empty, and its versions are empty unless the
	// query asks the overview.
	Act    domain.Act
	Status domain.ActStatus
	// Contents holds the headings of the overview.
	Contents []domain.Heading
	Articles []domain.Article
	// Next is the first requested article that the size left out.
	Next *domain.ArticleNumber
	// Truncated tells that the one article passed the size and lost paragraphs.
	Truncated bool
	// Newest is the newest version, when it is newer than the version of the text.
	Newest *domain.Version
	// VersionCount is the count of every version of the act, of which the overview
	// holds the newest MaxVersions.
	VersionCount int
	Text         string
	Piece        int
	Pieces       int
	ReadAt       time.Time
	Stale        bool
}

// GetAct reads one act.
type GetAct struct {
	source ActSource
	logger *slog.Logger
}

// NewGetAct returns the use case. The logger names the vendor of the source.
func NewGetAct(source ActSource, logger *slog.Logger) *GetAct {
	return &GetAct{source: source, logger: logger}
}

// Get reads the newest version of the act first, as only a page of the act lists its
// versions. It then reads the version that the query selects, unless it holds it, and the
// status of the act in parallel.
func (g *GetAct) Get(ctx context.Context, query ActQuery) (ActView, error) {
	if query.Version != nil && query.Date != nil {
		return ActView{}, ErrBothVersions
	}

	if len(query.Articles) > MaxArticles {
		return ActView{}, ErrTooManyArticles
	}

	newest, err := g.source.Act(ctx, query.ID, nil, query.Language)
	if err != nil {
		return ActView{}, fmt.Errorf("reading the newest version: %w", err)
	}

	target, err := selectVersion(newest.Act, query)
	if err != nil {
		return ActView{}, err
	}

	text, status, err := g.readVersion(ctx, query, newest, target)
	if err != nil {
		return ActView{}, err
	}

	return view(text, status, query)
}

func selectVersion(act domain.Act, query ActQuery) (domain.Version, error) {
	versionError := func() error {
		if len(act.Versions) == 0 {
			return &VersionError{First: act.Version, Newest: act.Version}
		}

		return &VersionError{First: act.Versions[len(act.Versions)-1], Newest: act.Versions[0]}
	}

	switch {
	case query.Version != nil:
		index := slices.IndexFunc(act.Versions, func(v domain.Version) bool {
			return v.Number == *query.Version
		})
		if index < 0 {
			return domain.Version{}, versionError()
		}

		return act.Versions[index], nil
	case query.Date != nil:
		version, ok := act.VersionOn(*query.Date)
		if !ok {
			return domain.Version{}, versionError()
		}

		return version, nil
	default:
		return act.Version, nil
	}
}

func (g *GetAct) readVersion(
	ctx context.Context,
	query ActQuery,
	newest ActText,
	target domain.Version,
) (ActText, StatusResult, error) {
	var (
		text    = newest
		status  StatusResult
		textErr error
		group   sync.WaitGroup
	)

	if target.Number != newest.Act.Version.Number {
		number := target.Number

		group.Go(func() {
			text, textErr = g.source.Act(ctx, query.ID, &number, query.Language)
			if textErr == nil {
				text.Act.Versions = newest.Act.Versions
			}
		})
	}

	if code := newest.Act.RegistrationCode; code != "" {
		group.Go(func() {
			result, err := g.source.Status(ctx, code)
			if err != nil {
				logLeftOut(ctx, g.logger, err)

				return
			}

			status = result
		})
	}

	group.Wait()

	if textErr != nil {
		return ActText{}, StatusResult{}, fmt.Errorf(
			"reading version %d: %w",
			target.Number,
			textErr,
		)
	}

	return text, status, nil
}

func view(text ActText, status StatusResult, query ActQuery) (ActView, error) {
	act := text.Act
	body := act.Body
	act.Body = domain.Body{}

	result := ActView{
		Act:          act,
		Status:       status.Status,
		VersionCount: len(act.Versions),
		ReadAt:       text.ReadAt,
		Stale:        text.Stale || status.Stale,
	}

	// Only the read with no version can trail: a translation that waits for the newest
	// version. A version or a date asks an older text on purpose.
	defaultRead := query.Version == nil && query.Date == nil
	if defaultRead && len(act.Versions) > 0 && act.Versions[0].Number > act.Version.Number {
		newest := act.Versions[0]
		result.Newest = &newest
	}

	if len(query.Articles) > 0 {
		result.Act.Versions = nil

		return withArticles(result, body, query.Articles)
	}

	result.Act.Versions = act.Versions[:min(len(act.Versions), MaxVersions)]
	result.Contents = body.Contents

	if body.Text == "" {
		return result, nil
	}

	pieces := splitText(body.Text)

	piece := max(query.Piece, 1)
	if piece > len(pieces) {
		return ActView{}, &PieceError{Piece: piece, Pieces: len(pieces)}
	}

	result.Text, result.Piece, result.Pieces = pieces[piece-1], piece, len(pieces)

	return result, nil
}

// withArticles adds the requested articles in the order of the act while the size allows.
func withArticles(
	result ActView,
	body domain.Body,
	numbers []domain.ArticleNumber,
) (ActView, error) {
	indexes := make([]int, 0, len(numbers))

	for _, number := range numbers {
		index := slices.IndexFunc(
			body.Articles,
			func(a domain.Article) bool { return a.Number == number },
		)
		if index < 0 {
			articleError := &ArticleError{Number: number}
			if len(body.Articles) > 0 {
				articleError.First = body.Articles[0].Number
				articleError.Last = body.Articles[len(body.Articles)-1].Number
			}

			return ActView{}, articleError
		}

		if !slices.Contains(indexes, index) {
			indexes = append(indexes, index)
		}
	}

	slices.Sort(indexes)

	used := 0

	for i, index := range indexes {
		article := body.Articles[index]

		size := articleRunes(article)
		if used+size <= maxRunes {
			result.Articles = append(result.Articles, article)
			used += size

			continue
		}

		next := i

		if i == 0 {
			result.Articles = []domain.Article{truncate(article, maxRunes)}
			result.Truncated = true
			next++
		}

		if next < len(indexes) {
			number := body.Articles[indexes[next]].Number
			result.Next = &number
		}

		break
	}

	return result, nil
}

func articleRunes(article domain.Article) int {
	size := utf8.RuneCountInString(article.Title)
	for _, paragraph := range article.Paragraphs {
		size += utf8.RuneCountInString(paragraph)
	}

	return size
}

// truncate keeps the paragraphs of an article while they fit in the limit.
func truncate(article domain.Article, limit int) domain.Article {
	used := utf8.RuneCountInString(article.Title)
	kept := make([]string, 0, len(article.Paragraphs))

	for _, paragraph := range article.Paragraphs {
		size := utf8.RuneCountInString(paragraph)
		if used+size > limit {
			break
		}

		kept = append(kept, paragraph)
		used += size
	}

	article.Paragraphs = kept

	return article
}

// splitText divides a text into pieces of at most pieceRunes runes. Each piece ends after
// the last new line before the limit, or at the limit when it holds no new line.
func splitText(text string) []string {
	var pieces []string

	for text != "" {
		if utf8.RuneCountInString(text) <= pieceRunes {
			pieces = append(pieces, text)

			break
		}

		limit := byteOffset(text, pieceRunes)

		end := strings.LastIndexByte(text[:limit], '\n') + 1
		if end == 0 {
			end = limit
		}

		pieces = append(pieces, text[:end])
		text = text[end:]
	}

	return pieces
}

// byteOffset returns the byte offset of the rune at the index.
func byteOffset(text string, index int) int {
	count := 0

	for offset := range text {
		if count == index {
			return offset
		}

		count++
	}

	return len(text)
}
