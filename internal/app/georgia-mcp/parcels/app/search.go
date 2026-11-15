package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
)

// AddressQuery asks for the addresses that match a text, in a municipality when it is
// not empty.
type AddressQuery struct {
	Text         string
	Municipality string
	Limit        int
}

// AddressList is the addresses of a search with their centres.
type AddressList struct {
	Addresses []domain.Address
	// ReadAt is the oldest read used.
	ReadAt time.Time
	// Stale is true when any read used is stale.
	Stale bool
}

// SearchAddresses finds addresses and reads the centre of each.
type SearchAddresses struct {
	source ParcelSource
}

// NewSearchAddresses returns the use case over the source.
func NewSearchAddresses(source ParcelSource) *SearchAddresses {
	return &SearchAddresses{source: source}
}

// Search keeps the matches of the municipality, takes the first Limit, and reads their
// centres concurrently. A failed centre fails the search.
func (u *SearchAddresses) Search(ctx context.Context, query AddressQuery) (AddressList, error) {
	read, err := u.source.MatchAddresses(ctx, query.Text)
	if err != nil {
		return AddressList{}, fmt.Errorf("searching the addresses: %w", err)
	}

	matches := inMunicipality(read.Matches, strings.TrimSpace(query.Municipality))
	matches = matches[:min(len(matches), query.Limit)]

	centres := make([]CentreRead, len(matches))
	failures := make([]error, len(matches))

	var group sync.WaitGroup

	for i, match := range matches {
		group.Go(func() {
			centres[i], failures[i] = u.source.AddressCentre(ctx, match.Ref)
		})
	}

	group.Wait()

	if err := errors.Join(failures...); err != nil {
		return AddressList{}, fmt.Errorf("reading the centres of the addresses: %w", err)
	}

	list := AddressList{
		Addresses: make([]domain.Address, 0, len(matches)),
		ReadAt:    read.ReadAt,
		Stale:     read.Stale,
	}

	for i, match := range matches {
		list.Addresses = append(list.Addresses, domain.Address{
			Text:         match.Text,
			Municipality: match.Municipality,
			Centre:       centres[i].Centre,
		})

		if centres[i].ReadAt.Before(list.ReadAt) {
			list.ReadAt = centres[i].ReadAt
		}

		list.Stale = list.Stale || centres[i].Stale
	}

	return list, nil
}

func inMunicipality(matches []AddressMatch, municipality string) []AddressMatch {
	if municipality == "" {
		return matches
	}

	kept := make([]AddressMatch, 0, len(matches))

	for _, match := range matches {
		if strings.Contains(match.Municipality, municipality) {
			kept = append(kept, match)
		}
	}

	return kept
}
