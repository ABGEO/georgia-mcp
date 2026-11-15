package app

import (
	"context"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
)

// VoteResult is one page of votes, with the notice of a text that is not Georgian.
type VoteResult struct {
	VotePage

	Notice string
}

// VoteDetail is one vote with the choices of the members grouped by faction.
type VoteDetail struct {
	VoteRead

	Factions []domain.FactionTally
}

// ListVotes returns the votes that match a filter, newest first.
type ListVotes struct {
	source VoteSource
}

// NewListVotes returns the use case over the source.
func NewListVotes(source VoteSource) *ListVotes {
	return &ListVotes{source: source}
}

// List refuses a filter with no criterion and reads one page.
func (u *ListVotes) List(
	ctx context.Context,
	filter VoteFilter,
	page domain.Page,
) (VoteResult, error) {
	if filter.IsZero() {
		return VoteResult{}, ErrNoCriterion
	}

	read, err := u.source.ListVotes(ctx, filter, page)
	if err != nil {
		return VoteResult{}, fmt.Errorf("listing the votes: %w", err)
	}

	return VoteResult{VotePage: read, Notice: notice(read.Total, filter.BillTitle)}, nil
}

// GetVote returns one vote with the choice of each member.
type GetVote struct {
	source VoteSource
}

// NewGetVote returns the use case over the source.
func NewGetVote(source VoteSource) *GetVote {
	return &GetVote{source: source}
}

// Get reads the vote with the identifier and groups its choices by faction.
func (u *GetVote) Get(ctx context.Context, id domain.ID) (VoteDetail, error) {
	read, err := u.source.Vote(ctx, id)
	if err != nil {
		return VoteDetail{}, fmt.Errorf("reading the vote %d: %w", id, err)
	}

	return VoteDetail{VoteRead: read, Factions: domain.Tally(read.Choices)}, nil
}
