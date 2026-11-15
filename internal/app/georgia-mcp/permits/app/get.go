package app

import (
	"context"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
)

// GetApplication returns one application by its number.
type GetApplication struct {
	source ApplicationSource
}

// NewGetApplication returns the use case over the source.
func NewGetApplication(source ApplicationSource) *GetApplication {
	return &GetApplication{source: source}
}

// Get returns the application with the number.
func (u *GetApplication) Get(
	ctx context.Context,
	number domain.ApplicationNumber,
) (ApplicationRead, error) {
	read, err := u.source.Application(ctx, number)
	if err != nil {
		return ApplicationRead{}, fmt.Errorf("reading the application %s: %w", number, err)
	}

	return read, nil
}
