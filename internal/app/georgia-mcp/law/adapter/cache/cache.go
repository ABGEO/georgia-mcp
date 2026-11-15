// Package cache keeps the searches, the titles, the acts, and the statuses of the
// Legislative Herald in memory, in front of the port of the context.
package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// newestTTL bounds the age of an answer that a new version can change.
	newestTTL = time.Hour
	// versionTTL bounds the age of one version, which does not change.
	versionTTL = 24 * time.Hour
	// maxAge bounds the age of a stale answer.
	maxAge = 7 * 24 * time.Hour

	queryEntries = 256
	actEntries   = 16
	keySeparator = "|"
	newestKey    = "newest"
)

func newStore[V any](ttl time.Duration, entries int) (*cache.Cache[V], error) {
	store, err := cache.New[V](cache.Config{TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: entries})
	if err != nil {
		return nil, fmt.Errorf("building a store of the Legislative Herald: %w", err)
	}

	return store, nil
}

// ActCache implements app.ActSource in front of another source.
type ActCache struct {
	source   app.ActSource
	searches *cache.Cache[app.SearchPage]
	titles   *cache.Cache[app.TitleList]
	newest   *cache.Cache[app.ActText]
	versions *cache.Cache[app.ActText]
	statuses *cache.Cache[app.StatusResult]
}

var _ app.ActSource = (*ActCache)(nil)

// NewActCache returns a cache in front of the source.
func NewActCache(source app.ActSource) (*ActCache, error) {
	acts := &ActCache{source: source}

	err := buildStore(&acts.searches, newestTTL, queryEntries)
	if err == nil {
		err = buildStore(&acts.titles, newestTTL, queryEntries)
	}

	if err == nil {
		err = buildStore(&acts.newest, newestTTL, actEntries)
	}

	if err == nil {
		err = buildStore(&acts.versions, versionTTL, actEntries)
	}

	if err == nil {
		err = buildStore(&acts.statuses, newestTTL, queryEntries)
	}

	if err != nil {
		_ = acts.Close()

		return nil, err
	}

	return acts, nil
}

func buildStore[V any](store **cache.Cache[V], ttl time.Duration, entries int) error {
	built, err := newStore[V](ttl, entries)
	*store = built

	return err
}

// Search returns the cached page of the query, or reads it.
func (c *ActCache) Search(ctx context.Context, query domain.SearchQuery) (app.SearchPage, error) {
	result, err := c.searches.Get(
		ctx,
		searchKey(query),
		func(ctx context.Context) (app.SearchPage, error) {
			return c.source.Search(ctx, query)
		},
	)
	if err != nil {
		return app.SearchPage{}, fmt.Errorf("searching the acts: %w", err)
	}

	page := result.Value
	page.Stale = result.Stale

	return page, nil
}

// Titles returns the cached titles of the text, or reads them.
func (c *ActCache) Titles(ctx context.Context, text string) (app.TitleList, error) {
	result, err := c.titles.Get(
		ctx,
		strings.TrimSpace(text),
		func(ctx context.Context) (app.TitleList, error) {
			return c.source.Titles(ctx, text)
		},
	)
	if err != nil {
		return app.TitleList{}, fmt.Errorf("reading the titles: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Act returns the cached version of the act, or reads it. The newest version and a given
// version live in two stores with two times to live.
func (c *ActCache) Act(
	ctx context.Context,
	id domain.ActID,
	version *int,
	language kernel.Language,
) (app.ActText, error) {
	store, number := c.newest, newestKey
	if version != nil {
		store, number = c.versions, strconv.Itoa(*version)
	}

	key := id.String() + keySeparator + number + keySeparator + string(language)

	result, err := store.Get(ctx, key, func(ctx context.Context) (app.ActText, error) {
		return c.source.Act(ctx, id, version, language)
	})
	if err != nil {
		return app.ActText{}, fmt.Errorf("reading the act: %w", err)
	}

	text := result.Value
	text.Stale = result.Stale

	return text, nil
}

// Status returns the cached status of the registration code, or reads it.
func (c *ActCache) Status(ctx context.Context, registrationCode string) (app.StatusResult, error) {
	result, err := c.statuses.Get(
		ctx,
		registrationCode,
		func(ctx context.Context) (app.StatusResult, error) {
			return c.source.Status(ctx, registrationCode)
		},
	)
	if err != nil {
		return app.StatusResult{}, fmt.Errorf("reading the status: %w", err)
	}

	status := result.Value
	status.Stale = result.Stale

	return status, nil
}

// Close stops the goroutines of the stores.
func (c *ActCache) Close() error {
	closeStore(c.searches)
	closeStore(c.titles)
	closeStore(c.newest)
	closeStore(c.versions)
	closeStore(c.statuses)

	return nil
}

func closeStore[V any](store *cache.Cache[V]) {
	if store != nil {
		store.Close()
	}
}

// searchKey joins every field of the query, after the trim of the texts.
func searchKey(query domain.SearchQuery) string {
	var docType, status string
	if query.Type != nil {
		docType = string(*query.Type)
	}

	if query.Status != nil {
		status = string(*query.Status)
	}

	return strings.Join([]string{
		strings.TrimSpace(query.Text),
		strconv.FormatBool(query.InText),
		docType,
		status,
		strings.TrimSpace(query.Issuer),
		strings.TrimSpace(query.Number),
		dateKey(query.From),
		dateKey(query.To),
		strconv.Itoa(int(query.Order)),
		strconv.Itoa(max(query.Page, 1)),
		string(query.Language),
	}, keySeparator)
}

func dateKey(date *kernel.Date) string {
	if date == nil {
		return ""
	}

	return date.String()
}
