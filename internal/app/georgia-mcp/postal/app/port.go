package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// AddressList is the addresses that match a text, with the facts of its read.
type AddressList struct {
	// Addresses are in the order of Georgian Post.
	Addresses []domain.Address
	ReadAt    time.Time
	Stale     bool
}

// PostalCodeSource searches the addresses and their postal codes.
type PostalCodeSource interface {
	Addresses(ctx context.Context, text string, language kernel.Language) (AddressList, error)
}
