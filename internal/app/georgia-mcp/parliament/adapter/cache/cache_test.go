package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingSource counts the reads of each method, and fails with an outage once failing
// is set.
type countingSource struct {
	bills, bill, votes, vote, questions atomic.Int32
	failing                             atomic.Bool
}

func (c *countingSource) SearchBills(
	context.Context,
	app.BillFilter,
	domain.Page,
) (app.BillPage, error) {
	c.bills.Add(1)

	return app.BillPage{Total: 1, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) Bill(_ context.Context, id domain.ID) (app.BillRead, error) {
	c.bill.Add(1)

	return app.BillRead{Bill: domain.Bill{ID: id}, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) ListVotes(
	context.Context,
	app.VoteFilter,
	domain.Page,
) (app.VotePage, error) {
	c.votes.Add(1)

	return app.VotePage{Total: 1, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) Vote(_ context.Context, id domain.ID) (app.VoteRead, error) {
	c.vote.Add(1)

	return app.VoteRead{Vote: domain.Vote{ID: id}, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) SearchQuestions(
	context.Context,
	app.QuestionFilter,
	domain.Page,
) (app.QuestionPage, error) {
	c.questions.Add(1)

	return app.QuestionPage{Total: 1, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) fail() error {
	if c.failing.Load() {
		return fmt.Errorf("calling the Parliament: %w", kernel.ErrSourceUnavailable)
	}

	return nil
}

func (c *countingSource) counts() [5]int32 {
	return [5]int32{
		c.bills.Load(),
		c.bill.Load(),
		c.votes.Load(),
		c.vote.Load(),
		c.questions.Load(),
	}
}

func newCache(t *testing.T, source *countingSource) *cache.SourceCache {
	t.Helper()

	sources, err := cache.NewSourceCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	return sources
}

func text(t *testing.T, value string) domain.SearchText {
	t.Helper()

	parsed, err := domain.NewSearchText(value)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func firstPage() domain.Page {
	return domain.Page{Number: 1, Size: 10}
}

// stale reads each method once, and returns the stale flag of each.
func stale(t *testing.T, sources *cache.SourceCache) ([5]bool, error) {
	t.Helper()

	ctx := t.Context()
	budget := text(t, "ბიუჯეტ")

	bills, billsErr := sources.SearchBills(ctx, app.BillFilter{Text: budget}, firstPage())
	bill, billErr := sources.Bill(ctx, 32449)
	votes, votesErr := sources.ListVotes(ctx, app.VoteFilter{BillTitle: budget}, firstPage())
	vote, voteErr := sources.Vote(ctx, 31507)
	questions, questionsErr := sources.SearchQuestions(
		ctx,
		app.QuestionFilter{Text: budget},
		firstPage(),
	)

	flags := [5]bool{bills.Stale, bill.Stale, votes.Stale, vote.Stale, questions.Stale}

	return flags, errors.Join(billsErr, billErr, votesErr, voteErr, questionsErr)
}

// PARL-SC-018: A search or a list lives 15 minutes. A bill and a vote live 1 hour.
func TestTimesToLive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)

		steps := []struct {
			sleep time.Duration
			want  [5]int32
		}{
			{0, [5]int32{1, 1, 1, 1, 1}},
			{14*time.Minute + 59*time.Second, [5]int32{1, 1, 1, 1, 1}},
			{time.Second, [5]int32{2, 1, 2, 1, 2}},
			{44*time.Minute + 59*time.Second, [5]int32{3, 1, 3, 1, 3}},
			{time.Second, [5]int32{3, 2, 3, 2, 3}},
		}

		for _, step := range steps {
			time.Sleep(step.sleep)

			if _, err := stale(t, sources); err != nil {
				t.Fatal(err)
			}

			if got := source.counts(); got != step.want {
				t.Fatalf("after %v: reads = %v, want %v", step.sleep, got, step.want)
			}
		}
	})
}

// PARL-SC-018: Another page, another text, and another identifier are other entries.
func TestKeys(t *testing.T) {
	t.Parallel()

	source := &countingSource{}
	sources := newCache(t, source)
	ctx := t.Context()

	filter := app.BillFilter{Text: text(t, "ბიუჯეტ")}
	calls := []error{
		ignore(sources.SearchBills(ctx, filter, firstPage())),
		ignore(sources.SearchBills(ctx, filter, domain.Page{Number: 2, Size: 10})),
		ignore(sources.SearchBills(ctx, app.BillFilter{Text: text(t, "კოდექს")}, firstPage())),
		ignore(sources.SearchBills(ctx, filter, firstPage())),
		ignore(sources.Bill(ctx, 1)),
		ignore(sources.Bill(ctx, 2)),
		ignore(sources.Bill(ctx, 1)),
	}

	if err := errors.Join(calls...); err != nil {
		t.Fatal(err)
	}

	if got := source.counts(); got[0] != 3 || got[1] != 2 {
		t.Errorf("reads = %v, want 3 searches and 2 bills", got)
	}
}

func ignore[T any](_ T, err error) error {
	return err
}

// PARL-SC-019: After an outage, each method returns its stale value for 24 hours from its
// read, then the error.
func TestStaleValues(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)

		if _, err := stale(t, sources); err != nil {
			t.Fatal(err)
		}

		source.failing.Store(true)
		time.Sleep(23*time.Hour + 59*time.Minute)

		flags, err := stale(t, sources)
		if err != nil || flags != [5]bool{true, true, true, true, true} {
			t.Fatalf("at 23 h 59 min: stale = %v, %v, want every value stale", flags, err)
		}

		time.Sleep(time.Minute + time.Second)

		if _, err := sources.Bill(
			t.Context(),
			32449,
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf("after 24 hours: Bill() error = %v, want ErrSourceUnavailable", err)
		}

		if _, err := sources.SearchBills(
			t.Context(), app.BillFilter{Text: text(t, "ბიუჯეტ")}, firstPage(),
		); !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("after 24 hours: SearchBills() error = %v, want ErrSourceUnavailable", err)
		}
	})
}
