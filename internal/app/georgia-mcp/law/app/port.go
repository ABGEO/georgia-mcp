package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// SearchPage is one page of a search, and the instant of the read.
type SearchPage struct {
	Acts   []domain.ActSummary
	Total  int
	ReadAt time.Time
	Stale  bool
}

// TitleList is the titles that match a Georgian text. Each summary holds the identifier,
// the title, and the status only.
type TitleList struct {
	Titles []domain.ActSummary
	ReadAt time.Time
	Stale  bool
}

// ActText is one version of an act, and the instant of the read.
type ActText struct {
	Act    domain.Act
	ReadAt time.Time
	Stale  bool
}

// StatusResult is the status of an act. The zero status tells that no act has the code.
type StatusResult struct {
	Status domain.ActStatus
	ReadAt time.Time
	Stale  bool
}

// ActSource gives the legal acts.
type ActSource interface {
	// Search returns one page of the acts that match the query.
	Search(ctx context.Context, query domain.SearchQuery) (SearchPage, error)
	// Titles returns the titles that match a Georgian text.
	Titles(ctx context.Context, text string) (TitleList, error)
	// Act returns one version of an act. A nil version asks the newest. An unknown
	// identifier gives ErrUnknownAct. An act in en with no translation gives
	// ErrNoTranslation.
	Act(
		ctx context.Context,
		id domain.ActID,
		version *int,
		language kernel.Language,
	) (ActText, error)
	// Status returns the status of the act with the registration code.
	Status(ctx context.Context, registrationCode string) (StatusResult, error)
}
