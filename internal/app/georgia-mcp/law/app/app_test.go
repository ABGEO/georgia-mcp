package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	taxCode   = "საქართველოს საგადასახადო კოდექსი"
	taxCodeID = "1043717"
	newest    = "newest"
)

var errDown = fmt.Errorf("down: %w", kernel.ErrSourceUnavailable)

func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func readAt() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

func id(t *testing.T, text string) domain.ActID {
	t.Helper()

	parsed, err := domain.ParseActID(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func article(t *testing.T, text string) domain.ArticleNumber {
	t.Helper()

	parsed, err := domain.ParseArticleNumber(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// fakeSource serves fixed answers and records each call. Each act is keyed by its
// identifier, its version or "newest", and its language.
type fakeSource struct {
	page      app.SearchPage
	titles    app.TitleList
	titlesErr error
	acts      map[string]domain.Act
	actErrs   map[string]error
	status    domain.ActStatus
	statusErr error
	delay     time.Duration

	mu    sync.Mutex
	calls []string
}

func actKey(id domain.ActID, version *int, language kernel.Language) string {
	number := newest
	if version != nil {
		number = strconv.Itoa(*version)
	}

	return id.String() + "|" + number + "|" + string(language)
}

func (f *fakeSource) Search(ctx context.Context, _ domain.SearchQuery) (app.SearchPage, error) {
	f.record("search")
	f.wait(ctx)

	return f.page, nil
}

func (f *fakeSource) Titles(ctx context.Context, _ string) (app.TitleList, error) {
	f.record("titles")
	f.wait(ctx)

	return f.titles, f.titlesErr
}

func (f *fakeSource) Act(
	ctx context.Context,
	actID domain.ActID,
	version *int,
	language kernel.Language,
) (app.ActText, error) {
	key := actKey(actID, version, language)
	f.record("act " + key)
	f.wait(ctx)

	if err := f.actErrs[key]; err != nil {
		return app.ActText{}, err
	}

	act, ok := f.acts[key]
	if !ok {
		return app.ActText{}, app.ErrUnknownAct
	}

	return app.ActText{Act: act, ReadAt: readAt()}, nil
}

func (f *fakeSource) Status(ctx context.Context, _ string) (app.StatusResult, error) {
	f.record("status")
	f.wait(ctx)

	return app.StatusResult{Status: f.status, ReadAt: readAt()}, f.statusErr
}

func (f *fakeSource) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, call)
}

func (f *fakeSource) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.calls)
}

func (f *fakeSource) wait(ctx context.Context) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
		}
	}
}

func summaries(t *testing.T, count int) []domain.ActSummary {
	t.Helper()

	acts := make([]domain.ActSummary, 0, count)
	for i := range count {
		acts = append(acts, domain.ActSummary{
			ID: id(
				t,
				strconv.Itoa(5000000+i),
			),
			Title:  "act " + strconv.Itoa(i),
			Status: domain.StatusActive,
		})
	}

	return acts
}

func searchFake(t *testing.T) *fakeSource {
	t.Helper()

	return &fakeSource{
		page: app.SearchPage{Acts: summaries(t, 20), Total: 29, ReadAt: readAt()},
		titles: app.TitleList{Titles: []domain.ActSummary{
			{ID: id(t, "32842"), Title: taxCode, Status: domain.StatusExpired},
			{ID: id(t, "31690"), Title: taxCode, Status: domain.StatusExpired},
			{ID: id(t, taxCodeID), Title: taxCode, Status: domain.StatusActive},
		}},
		acts: map[string]domain.Act{
			taxCodeID + "|newest|en": {
				ID: id(t, taxCodeID), Title: "TAX CODE OF GEORGIA", Type: "Law of Georgia",
				Issuer: "Parliament of Georgia", Number: "3591",
			},
		},
	}
}

func ids(acts []domain.ActSummary) []string {
	out := make([]string, 0, len(acts))
	for _, act := range acts {
		out = append(out, act.ID.String())
	}

	return out
}

// MATSNE-SC-015: The exact Georgian title puts the active act first, with its metadata.
func TestSearchPutsTheExactTitleFirst(t *testing.T) {
	t.Parallel()

	fake := searchFake(t)

	result, err := app.NewSearchActs(fake, discard()).Search(t.Context(), domain.SearchQuery{
		Text: "„" + taxCode + "“", Page: 1, Language: kernel.LanguageEnglish,
	}, 20)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(result.Acts) != 20 || result.Total != 29 {
		t.Fatalf("acts = %d, total = %d, want 20 and 29", len(result.Acts), result.Total)
	}

	first := result.Acts[0]
	if first.ID.String() != taxCodeID || first.Type != "Law of Georgia" ||
		first.Issuer != "Parliament of Georgia" || first.Status != domain.StatusActive {
		t.Errorf("first = %+v, want 1043717 with its type and issuer", first)
	}

	if got := ids(result.Acts[1:]); !slices.Equal(got, ids(fake.page.Acts[:19])) {
		t.Errorf("rest = %v, want the first 19 of the page", got)
	}
}

// MATSNE-SC-016: The rule keeps the page in each case that it does not cover.
func TestSearchKeepsThePage(t *testing.T) {
	t.Parallel()

	law := domain.TypeLaw

	tests := []struct {
		name       string
		query      domain.SearchQuery
		wantTitles bool
	}{
		{
			name:       "not the title",
			query:      domain.SearchQuery{Text: "საგადასახადო კოდექსი"},
			wantTitles: true,
		},
		{name: "page 2", query: domain.SearchQuery{Text: taxCode, Page: 2}},
		{name: "in the text", query: domain.SearchQuery{Text: taxCode, InText: true}},
		{name: "a type", query: domain.SearchQuery{Text: taxCode, Type: &law}},
		{name: "English", query: domain.SearchQuery{Text: "Tax Code of Georgia"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := searchFake(t)
			tt.query.Language = kernel.LanguageEnglish

			result, err := app.NewSearchActs(fake, discard()).Search(t.Context(), tt.query, 20)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if !slices.Equal(ids(result.Acts), ids(fake.page.Acts)) {
				t.Errorf("acts = %v, want the page", ids(result.Acts))
			}

			if got := slices.Contains(fake.called(), "titles"); got != tt.wantTitles {
				t.Errorf("Titles called = %v, want %v", got, tt.wantTitles)
			}
		})
	}
}

// MATSNE-SC-016: A failure of Titles keeps the page, and an act of the page appears once.
func TestSearchSurvivesTitlesAndDuplicates(t *testing.T) {
	t.Parallel()

	failing := searchFake(t)
	failing.titlesErr = errDown

	query := domain.SearchQuery{Text: taxCode, Language: kernel.LanguageEnglish}

	result, err := app.NewSearchActs(failing, discard()).Search(t.Context(), query, 20)
	if err != nil || !slices.Equal(ids(result.Acts), ids(failing.page.Acts)) {
		t.Errorf("Search() = %v, %v, want the page", ids(result.Acts), err)
	}

	holding := searchFake(t)
	holding.page.Acts[7] = domain.ActSummary{ID: id(t, taxCodeID), Title: taxCode}

	result, err = app.NewSearchActs(holding, discard()).Search(t.Context(), query, 20)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	got := ids(result.Acts)
	if got[0] != taxCodeID || slices.Index(got[1:], taxCodeID) >= 0 || len(got) != 20 {
		t.Errorf("acts = %v, want 1043717 once, first", got)
	}
}

func actWithArticles(t *testing.T, numbers []string, runes int, paragraphs int) domain.Act {
	t.Helper()

	articles := make([]domain.Article, 0, len(numbers))
	contents := make([]domain.Heading, 0, len(numbers))

	for _, number := range numbers {
		texts := make([]string, paragraphs)
		for i := range texts {
			texts[i] = strings.Repeat("ა", runes/paragraphs)
		}

		articles = append(articles, domain.Article{Number: article(t, number), Paragraphs: texts})
		contents = append(contents, domain.Heading{Kind: domain.KindArticle, Number: number})
	}

	return domain.Act{
		ID: id(t, taxCodeID), Title: taxCode, RegistrationCode: "200000000.05.001.016012",
		Version:  domain.Version{Number: 4, Date: date(t, "2026-10-28")},
		Versions: versions(t),
		Body:     domain.Body{Contents: contents, Articles: articles},
	}
}

// versions returns the versions of MATSNE-SC-008, newest first.
func versions(t *testing.T) []domain.Version {
	t.Helper()

	return []domain.Version{
		{Number: 4, Date: date(t, "2026-10-28")},
		{Number: 3, Date: date(t, "2026-04-01")},
		{Number: 2, Date: date(t, "2026-04-01")},
		{Number: 1, Date: date(t, "2011-01-01")},
		{Number: 0, Date: date(t, "2010-10-12")},
	}
}

func actFake(act domain.Act) *fakeSource {
	return &fakeSource{
		acts:   map[string]domain.Act{act.ID.String() + "|newest|ka": act},
		status: domain.StatusActive,
	}
}

func numbers(articles []domain.Article) []string {
	out := make([]string, 0, len(articles))
	for _, item := range articles {
		out = append(out, item.Number.String())
	}

	return out
}

// MATSNE-SC-017: Articles come in the order of the act, and an unknown one names the range.
func TestGetActSelectsTheArticles(t *testing.T) {
	t.Parallel()

	all := make([]string, 0, 310)
	for i := 1; i <= 309; i++ {
		all = append(all, strconv.Itoa(i))
		if i == 133 {
			all = append(all, "133^3")
		}
	}

	fake := actFake(actWithArticles(t, all, 10, 1))
	get := app.NewGetAct(fake, discard())
	query := app.ActQuery{ID: id(t, taxCodeID), Language: kernel.LanguageGeorgian}

	query.Articles = []domain.ArticleNumber{article(t, "82"), article(t, "1")}

	view, err := get.Get(t.Context(), query)
	if err != nil || !slices.Equal(numbers(view.Articles), []string{"1", "82"}) {
		t.Errorf("Get(82, 1) = %v, %v, want 1 then 82", numbers(view.Articles), err)
	}

	query.Articles = []domain.ArticleNumber{article(t, "999")}

	var articleErr *app.ArticleError
	if _, err := get.Get(t.Context(), query); !errors.As(err, &articleErr) ||
		articleErr.First.String() != "1" || articleErr.Last.String() != "309" {
		t.Errorf("Get(999) error = %v, want an *ArticleError from 1 to 309", err)
	}

	calls := len(fake.called())
	query.Articles = make([]domain.ArticleNumber, 11)

	if _, err := get.Get(t.Context(), query); !errors.Is(err, app.ErrTooManyArticles) ||
		len(fake.called()) != calls {
		t.Errorf("Get(11 articles) error = %v, want ErrTooManyArticles and no call", err)
	}
}

// MATSNE-SC-018: The size keeps the articles that fit and names the next.
func TestGetActKeepsTheArticlesThatFit(t *testing.T) {
	t.Parallel()

	three := actFake(actWithArticles(t, []string{"1", "2", "3"}, 25_000, 1))
	query := app.ActQuery{
		ID: id(t, taxCodeID), Language: kernel.LanguageGeorgian,
		Articles: []domain.ArticleNumber{article(t, "1"), article(t, "2"), article(t, "3")},
	}

	view, err := app.NewGetAct(three, discard()).Get(t.Context(), query)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !slices.Equal(numbers(view.Articles), []string{"1", "2"}) ||
		view.Next == nil || view.Next.String() != "3" || view.Truncated {
		t.Errorf("Get() = %v, next %v, want 1 and 2, next 3", numbers(view.Articles), view.Next)
	}
}

// MATSNE-SC-018: One article that passes the size keeps the paragraphs that fit.
func TestGetActTruncatesALongArticle(t *testing.T) {
	t.Parallel()

	long := actFake(actWithArticles(t, []string{"1"}, 70_000, 10))
	query := app.ActQuery{
		ID: id(t, taxCodeID), Language: kernel.LanguageGeorgian,
		Articles: []domain.ArticleNumber{article(t, "1")},
	}

	view, err := app.NewGetAct(long, discard()).Get(t.Context(), query)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(view.Articles) != 1 || len(view.Articles[0].Paragraphs) != 8 ||
		!view.Truncated || view.Next != nil {
		t.Errorf("Get() = %d articles, truncated %v, next %v, want 8 paragraphs, truncated",
			len(view.Articles), view.Truncated, view.Next)
	}
}

// MATSNE-SC-019: A text with no article divides into pieces at a new line.
func TestGetActDividesTheText(t *testing.T) {
	t.Parallel()

	act := actWithArticles(t, nil, 0, 1)
	act.Body = domain.Body{Text: strings.Repeat(strings.Repeat("ა", 999)+"\n", 76)}

	get := app.NewGetAct(actFake(act), discard())
	query := app.ActQuery{ID: id(t, taxCodeID), Language: kernel.LanguageGeorgian}

	view, err := get.Get(t.Context(), query)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if runes := len([]rune(view.Text)); runes > 50_000 || !strings.HasSuffix(view.Text, "\n") ||
		view.Pieces != 2 || view.Piece != 1 {
		t.Errorf("piece 1 = %d runes, pieces %d, piece %d, want 50,000 or less, 2, 1",
			runes, view.Pieces, view.Piece)
	}

	query.Piece = 3

	var pieceErr *app.PieceError
	if _, err := get.Get(t.Context(), query); !errors.As(err, &pieceErr) || pieceErr.Pieces != 2 {
		t.Errorf("Get(piece 3) error = %v, want a *PieceError with 2", err)
	}
}

func versionFake(t *testing.T) *fakeSource {
	t.Helper()

	act := actWithArticles(t, []string{"1"}, 10, 1)
	fake := actFake(act)

	old := act
	old.Version = domain.Version{Number: 1, Date: date(t, "2011-01-01")}
	old.Versions = nil
	fake.acts[taxCodeID+"|1|ka"] = old

	return fake
}

func versionQuery(t *testing.T) app.ActQuery {
	t.Helper()

	return app.ActQuery{ID: id(t, taxCodeID), Language: kernel.LanguageGeorgian}
}

// MATSNE-SC-020: A date selects the newest version on or before it.
func TestGetActSelectsTheVersionOfADate(t *testing.T) {
	t.Parallel()

	fake := versionFake(t)
	query := versionQuery(t)
	on := date(t, "2025-01-01")
	query.Date = &on

	view, err := app.NewGetAct(fake, discard()).Get(t.Context(), query)
	if err != nil || view.Act.Version.Number != 1 || len(view.Act.Versions) != 5 {
		t.Errorf("Get() = version %d, %d versions, %v, want 1 and 5", view.Act.Version.Number,
			len(view.Act.Versions), err)
	}

	if calls := fake.called(); calls[0] != "act 1043717|newest|ka" ||
		!slices.Contains(calls, "act 1043717|1|ka") {
		t.Errorf("calls = %v, want the newest then version 1", calls)
	}
}

// MATSNE-SC-020: A date after the newest version reads the newest once.
func TestGetActReadsTheNewestOnce(t *testing.T) {
	t.Parallel()

	fake := versionFake(t)
	query := versionQuery(t)
	on := date(t, "2027-01-01")
	query.Date = &on

	if _, err := app.NewGetAct(fake, discard()).Get(t.Context(), query); err != nil ||
		slices.Contains(fake.called(), "act 1043717|4|ka") {
		t.Errorf("Get() error = %v, calls %v, want one read of the newest", err, fake.called())
	}
}

// MATSNE-SC-020: A version and a date call no source.
func TestGetActRefusesBothVersions(t *testing.T) {
	t.Parallel()

	nine, early := 9, date(t, "2010-10-11")
	query := versionQuery(t)
	query.Version, query.Date = &nine, &early

	fake := versionFake(t)
	if _, err := app.NewGetAct(fake, discard()).
		Get(t.Context(), query); !errors.Is(
		err,
		app.ErrBothVersions,
	) ||
		len(fake.called()) != 0 {
		t.Errorf("Get() error = %v, calls %v, want ErrBothVersions and no call", err, fake.called())
	}
}

// MATSNE-SC-020: An unknown version and a date before version 0 name the range.
func TestGetActNamesTheVersions(t *testing.T) {
	t.Parallel()

	nine, early := 9, date(t, "2010-10-11")
	version, before := versionQuery(t), versionQuery(t)
	version.Version, before.Date = &nine, &early

	for _, query := range []app.ActQuery{version, before} {
		fake := versionFake(t)

		var versionErr *app.VersionError
		if _, err := app.NewGetAct(fake, discard()).
			Get(t.Context(), query); !errors.As(err, &versionErr) ||
			versionErr.First.Number != 0 ||
			versionErr.Newest.Number != 4 ||
			!slices.Equal(fake.called(), []string{"act 1043717|newest|ka"}) {
			t.Errorf(
				"Get() error = %v, calls %v, want a *VersionError from 0 to 4",
				err,
				fake.called(),
			)
		}
	}
}

// MATSNE-SC-021: The overview holds the versions, the contents, and the status.
func TestGetActGivesTheOverview(t *testing.T) {
	t.Parallel()

	get := app.NewGetAct(actFake(actWithArticles(t, []string{"1", "2"}, 10, 1)), discard())

	view, err := get.Get(t.Context(), versionQuery(t))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if view.Act.Title != taxCode || len(view.Act.Versions) != 5 || len(view.Contents) != 2 ||
		view.Status != domain.StatusActive || len(view.Articles) != 0 {
		t.Errorf("overview = %+v", view)
	}
}

// MATSNE-SC-036: The overview of an act of 25 versions holds the newest 20 and the count 25.
func TestGetActBoundsTheVersions(t *testing.T) {
	t.Parallel()

	act := actWithArticles(t, []string{"1"}, 10, 1)
	act.Versions = nil

	for number := 24; number >= 0; number-- {
		act.Versions = append(
			act.Versions,
			domain.Version{Number: number, Date: date(t, "2026-10-28")},
		)
	}

	act.Version = act.Versions[0]

	view, err := app.NewGetAct(actFake(act), discard()).Get(t.Context(), versionQuery(t))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(view.Act.Versions) != app.MaxVersions || view.Act.Versions[0].Number != 24 ||
		view.Act.Versions[app.MaxVersions-1].Number != 5 || view.VersionCount != 25 {
		t.Errorf("versions = %d from %d, count %d, want 20 from 24 and 25",
			len(view.Act.Versions), view.Act.Versions[0].Number, view.VersionCount)
	}
}

// MATSNE-SC-021: A query of articles gives no versions and no contents.
func TestGetActGivesTheArticlesAlone(t *testing.T) {
	t.Parallel()

	get := app.NewGetAct(actFake(actWithArticles(t, []string{"1", "2"}, 10, 1)), discard())
	query := versionQuery(t)
	query.Articles = []domain.ArticleNumber{article(t, "2")}

	view, err := get.Get(t.Context(), query)
	if err != nil || len(view.Act.Versions) != 0 || len(view.Contents) != 0 ||
		len(view.Articles) != 1 {
		t.Errorf("articles = %+v, %v, want one article with no versions and no contents", view, err)
	}
}

// MATSNE-SC-033: An English text that trails names the newest version.
func TestGetActNamesTheNewestVersion(t *testing.T) {
	t.Parallel()

	act := actWithArticles(t, []string{"1"}, 10, 1)
	act.Versions = append(
		[]domain.Version{{Number: 5, Date: date(t, "2026-11-01")}},
		act.Versions...)

	georgian := act
	georgian.Version = act.Versions[0]

	fake := &fakeSource{
		acts: map[string]domain.Act{
			"1043717|newest|en": act,
			"1043717|newest|ka": georgian,
		},
		actErrs: map[string]error{"1043717|5|en": app.ErrNoTranslation},
	}
	get := app.NewGetAct(fake, discard())
	query := app.ActQuery{ID: id(t, taxCodeID), Language: kernel.LanguageEnglish}

	view, err := get.Get(t.Context(), query)
	if err != nil || view.Act.Version.Number != 4 || view.Newest == nil || view.Newest.Number != 5 {
		t.Errorf(
			"Get(en) = version %d, newest %v, %v, want 4 and 5",
			view.Act.Version.Number,
			view.Newest,
			err,
		)
	}

	query.Language = kernel.LanguageGeorgian

	view, err = get.Get(t.Context(), query)
	if err != nil || view.Newest != nil {
		t.Errorf("Get(ka) newest = %v, %v, want none", view.Newest, err)
	}

	five := 5
	query.Language, query.Version = kernel.LanguageEnglish, &five

	if _, err := get.Get(t.Context(), query); !errors.Is(err, app.ErrNoTranslation) {
		t.Errorf("Get(en, 5) error = %v, want ErrNoTranslation", err)
	}
}

// MATSNE-SC-022: The exact title path and a dated article each answer within 15 seconds.
func TestTheUseCasesAnswerInTime(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := searchFake(t)
		fake.delay = 4900 * time.Millisecond

		start := time.Now()

		_, err := app.NewSearchActs(fake, discard()).Search(t.Context(), domain.SearchQuery{
			Text: taxCode, Language: kernel.LanguageEnglish,
		}, 20)
		if elapsed := time.Since(start); err != nil || elapsed >= 15*time.Second {
			t.Errorf("Search() took %v, %v, want less than 15s", elapsed, err)
		}

		slow := versionFake(t)
		slow.delay = 4900 * time.Millisecond
		on := date(t, "2025-01-01")

		start = time.Now()

		_, err = app.NewGetAct(slow, discard()).Get(t.Context(), app.ActQuery{
			ID: id(t, taxCodeID), Date: &on, Language: kernel.LanguageGeorgian,
			Articles: []domain.ArticleNumber{article(t, "1")},
		})
		if elapsed := time.Since(start); err != nil || elapsed >= 15*time.Second {
			t.Errorf("Get() took %v, %v, want less than 15s", elapsed, err)
		}
	})
}

// MATSNE-SC-033: A version or a date asks an older text on purpose, so it names no newer
// version.
func TestGetActNamesNoNewestForAnAskedVersion(t *testing.T) {
	t.Parallel()

	fake := versionFake(t)
	on := date(t, "2025-01-01")
	query := versionQuery(t)
	query.Date = &on

	view, err := app.NewGetAct(fake, discard()).Get(t.Context(), query)
	if err != nil || view.Act.Version.Number != 1 || view.Newest != nil {
		t.Errorf("Get(ka, 2025-01-01) = version %d, newest %v, %v, want 1 and none",
			view.Act.Version.Number, view.Newest, err)
	}

	one := 1
	query.Date, query.Version = nil, &one

	view, err = app.NewGetAct(fake, discard()).Get(t.Context(), query)
	if err != nil || view.Newest != nil {
		t.Errorf("Get(version 1) newest = %v, %v, want none", view.Newest, err)
	}
}

// leftOut counts the records "source unavailable" at the level warn with an error.
func leftOut(t *testing.T, buffer *bytes.Buffer) int {
	t.Helper()

	count := 0

	for line := range strings.Lines(buffer.String()) {
		var record struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}

		if record.Level == "WARN" && record.Msg == "source unavailable" && record.Error != "" {
			count++
		}
	}

	return count
}

// MATSNE-SC-035: A failed read of the titles writes one record and keeps the page.
func TestSearchRecordsTheTitlesItLeavesOut(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	fake := searchFake(t)
	fake.titlesErr = errDown

	result, err := app.NewSearchActs(fake, slog.New(slog.NewJSONHandler(&buffer, nil))).
		Search(t.Context(), domain.SearchQuery{Text: taxCode, Language: kernel.LanguageEnglish}, 20)
	if err != nil || len(result.Acts) != 20 {
		t.Fatalf("Search() = %d acts, %v, want the page", len(result.Acts), err)
	}

	if count := leftOut(t, &buffer); count != 1 {
		t.Errorf("records = %d, want 1:\n%s", count, buffer.String())
	}
}

// MATSNE-SC-035: A failed read of the status writes one record and keeps the act.
func TestGetActRecordsTheStatusItLeavesOut(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	fake := actFake(actWithArticles(t, []string{"1"}, 10, 1))
	fake.statusErr = errDown

	view, err := app.NewGetAct(fake, slog.New(slog.NewJSONHandler(&buffer, nil))).
		Get(t.Context(), versionQuery(t))
	if err != nil || view.Status != "" || view.Act.Title != taxCode {
		t.Fatalf("Get() = status %q, %v, want the act with no status", view.Status, err)
	}

	if count := leftOut(t, &buffer); count != 1 {
		t.Errorf("records = %d, want 1:\n%s", count, buffer.String())
	}
}
