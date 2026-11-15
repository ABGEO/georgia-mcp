// Package cache keeps the bills, the votes, and the member questions of the Parliament in
// memory, in front of the ports of the context.
package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// listTTL bounds the age of a search or a list: a sitting adds votes during the day.
	listTTL = 15 * time.Minute
	// recordTTL bounds the age of one bill or one vote.
	recordTTL = time.Hour
	// maxAge bounds the age of a stale value.
	maxAge = 24 * time.Hour

	listEntries   = 256
	recordEntries = 64
	keySeparator  = "|"
)

// SourceCache implements the three ports in front of another source.
type SourceCache struct {
	bills       app.BillSource
	votes       app.VoteSource
	questions   app.QuestionSource
	billPages   *cache.Cache[app.BillPage]
	billReads   *cache.Cache[app.BillRead]
	votePages   *cache.Cache[app.VotePage]
	voteReads   *cache.Cache[app.VoteRead]
	questionPgs *cache.Cache[app.QuestionPage]
}

var (
	_ app.BillSource     = (*SourceCache)(nil)
	_ app.VoteSource     = (*SourceCache)(nil)
	_ app.QuestionSource = (*SourceCache)(nil)
)

// Source is a source of the three ports, as the adapter of the Parliament is.
type Source interface {
	app.BillSource
	app.VoteSource
	app.QuestionSource
}

// NewSourceCache returns a cache in front of the source.
func NewSourceCache(source Source) (*SourceCache, error) {
	sources := &SourceCache{bills: source, votes: source, questions: source}

	err := buildStore(&sources.billPages, listTTL, listEntries)
	if err == nil {
		err = buildStore(&sources.billReads, recordTTL, recordEntries)
	}

	if err == nil {
		err = buildStore(&sources.votePages, listTTL, listEntries)
	}

	if err == nil {
		err = buildStore(&sources.voteReads, recordTTL, recordEntries)
	}

	if err == nil {
		err = buildStore(&sources.questionPgs, listTTL, listEntries)
	}

	if err != nil {
		_ = sources.Close()

		return nil, err
	}

	return sources, nil
}

func buildStore[V any](store **cache.Cache[V], ttl time.Duration, entries int) error {
	built, err := cache.New[V](cache.Config{TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: entries})
	if err != nil {
		return fmt.Errorf("building a store of the Parliament: %w", err)
	}

	*store = built

	return nil
}

// SearchBills returns the cached page of the filter, or reads it.
func (c *SourceCache) SearchBills(
	ctx context.Context,
	filter app.BillFilter,
	page domain.Page,
) (app.BillPage, error) {
	key := join(filter.Text.String(), filter.Number, periodKey(filter.Registered), pageKey(page))

	result, err := c.billPages.Get(ctx, key, func(ctx context.Context) (app.BillPage, error) {
		return c.bills.SearchBills(ctx, filter, page)
	})
	if err != nil {
		return app.BillPage{}, fmt.Errorf("searching the bills: %w", err)
	}

	value := result.Value
	value.Stale = result.Stale

	return value, nil
}

// Bill returns the cached bill, or reads it.
func (c *SourceCache) Bill(ctx context.Context, id domain.ID) (app.BillRead, error) {
	result, err := c.billReads.Get(ctx, idKey(id), func(ctx context.Context) (app.BillRead, error) {
		return c.bills.Bill(ctx, id)
	})
	if err != nil {
		return app.BillRead{}, fmt.Errorf("reading the bill: %w", err)
	}

	value := result.Value
	value.Stale = result.Stale

	return value, nil
}

// ListVotes returns the cached page of the filter, or reads it.
func (c *SourceCache) ListVotes(
	ctx context.Context,
	filter app.VoteFilter,
	page domain.Page,
) (app.VotePage, error) {
	key := join(filter.BillTitle.String(), periodKey(filter.Voted), pageKey(page))

	result, err := c.votePages.Get(ctx, key, func(ctx context.Context) (app.VotePage, error) {
		return c.votes.ListVotes(ctx, filter, page)
	})
	if err != nil {
		return app.VotePage{}, fmt.Errorf("listing the votes: %w", err)
	}

	value := result.Value
	value.Stale = result.Stale

	return value, nil
}

// Vote returns the cached vote, or reads it.
func (c *SourceCache) Vote(ctx context.Context, id domain.ID) (app.VoteRead, error) {
	result, err := c.voteReads.Get(ctx, idKey(id), func(ctx context.Context) (app.VoteRead, error) {
		return c.votes.Vote(ctx, id)
	})
	if err != nil {
		return app.VoteRead{}, fmt.Errorf("reading the vote: %w", err)
	}

	value := result.Value
	value.Stale = result.Stale

	return value, nil
}

// SearchQuestions returns the cached page of the filter, or reads it.
func (c *SourceCache) SearchQuestions(
	ctx context.Context,
	filter app.QuestionFilter,
	page domain.Page,
) (app.QuestionPage, error) {
	key := join(
		filter.Text.String(), filter.Member.String(), periodKey(filter.Registered), pageKey(page),
	)

	result, err := c.questionPgs.Get(ctx, key, func(ctx context.Context) (app.QuestionPage, error) {
		return c.questions.SearchQuestions(ctx, filter, page)
	})
	if err != nil {
		return app.QuestionPage{}, fmt.Errorf("searching the member questions: %w", err)
	}

	value := result.Value
	value.Stale = result.Stale

	return value, nil
}

// Close stops the goroutines of the stores.
func (c *SourceCache) Close() error {
	closeStore(c.billPages)
	closeStore(c.billReads)
	closeStore(c.votePages)
	closeStore(c.voteReads)
	closeStore(c.questionPgs)

	return nil
}

func closeStore[V any](store *cache.Cache[V]) {
	if store != nil {
		store.Close()
	}
}

func join(parts ...string) string {
	return strings.Join(parts, keySeparator)
}

func idKey(id domain.ID) string {
	return strconv.Itoa(int(id))
}

func periodKey(period domain.Period) string {
	return join(dateKey(period.From), dateKey(period.To))
}

func dateKey(date *kernel.Date) string {
	if date == nil {
		return ""
	}

	return date.String()
}

func pageKey(page domain.Page) string {
	return join(strconv.Itoa(page.Number), strconv.Itoa(page.Size))
}
