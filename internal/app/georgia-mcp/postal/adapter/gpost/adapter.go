// Package gpost reads the postal codes from the Georgian Post client and converts the
// rows of its search answer to the domain. It is the anti-corruption layer of the
// context for Georgian Post.
package gpost

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gpostclient "github.com/abgeo/georgia-mcp/pkg/gpost"
)

// Adapter implements app.PostalCodeSource with the Georgian Post client.
type Adapter struct {
	client *gpostclient.Client
	now    func() time.Time
}

var _ app.PostalCodeSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *gpostclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Addresses searches the text with one visit of the page and one search. Every failure
// wraps kernel.ErrSourceUnavailable. A postal code that is not valid fails the whole
// answer, since a partial list can hide the address that a person asks for.
func (a *Adapter) Addresses(
	ctx context.Context,
	text string,
	language kernel.Language,
) (app.AddressList, error) {
	rows, err := a.client.PostalCodes(ctx, text, string(language))
	if err != nil {
		return app.AddressList{}, fmt.Errorf(
			"reading the Georgian Post postal codes: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	addresses := make([]domain.Address, 0, len(rows))

	for _, row := range rows {
		code, err := domain.ParsePostalCode(row.Code)
		if err != nil {
			return app.AddressList{}, fmt.Errorf(
				"reading the Georgian Post postal codes: %w: %w", kernel.ErrSourceUnavailable, err,
			)
		}

		addresses = append(addresses, domain.Address{
			Text:     row.Address,
			District: row.District,
			Code:     code,
		})
	}

	return app.AddressList{Addresses: addresses, ReadAt: a.now().In(kernel.TbilisiZone())}, nil
}
