// Package nbg reads the official rates from the NBG client and converts its answer
// to the domain. It is the anti-corruption layer of the context for the NBG.
package nbg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
)

const (
	dateLength = len(time.DateOnly)
	// publishedLayout reads the wall time without the suffix Z.
	publishedLayout = "2006-01-02T15:04:05.999"
)

var errMalformed = errors.New("nbg: malformed answer")

// Adapter implements app.OfficialRateSource with the NBG client.
type Adapter struct {
	client *nbgclient.Client
	now    func() time.Time
}

var _ app.OfficialRateSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *nbgclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// RateSet reads the rate set in force on the date. Every failure of the NBG wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) RateSet(
	ctx context.Context,
	date kernel.Date,
	language kernel.Language,
) (app.RateSetResult, error) {
	sets, err := a.client.RateSets(
		ctx,
		nbgclient.Language(language),
		nbgclient.RateSetsQuery{Date: date.Time()},
	)
	if err != nil {
		return app.RateSetResult{}, fmt.Errorf(
			"reading the NBG rates: %w: %w",
			kernel.ErrSourceUnavailable,
			err,
		)
	}

	readAt := a.now()

	if len(sets) == 0 {
		return app.RateSetResult{ReadAt: readAt}, nil
	}

	set, err := toRateSet(sets[0])
	if err != nil {
		return app.RateSetResult{}, fmt.Errorf(
			"reading the NBG rates: %w: %w",
			kernel.ErrSourceUnavailable,
			err,
		)
	}

	return app.RateSetResult{Set: set, ReadAt: readAt}, nil
}

func toRateSet(answer nbgclient.RateSet) (domain.RateSet, error) {
	if len(answer.Date) < dateLength {
		return domain.RateSet{}, fmt.Errorf("%w: valid date %q", errMalformed, answer.Date)
	}

	validFrom, err := kernel.ParseDate(answer.Date[:dateLength])
	if err != nil {
		return domain.RateSet{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	rates := make([]domain.OfficialRate, 0, len(answer.Currencies))

	var publishedAt time.Time

	for _, currency := range answer.Currencies {
		rate, err := toOfficialRate(currency)
		if err != nil {
			return domain.RateSet{}, err
		}

		rates = append(rates, rate)

		if publishedAt.IsZero() {
			publishedAt, err = toPublishedAt(currency.Date)
			if err != nil {
				return domain.RateSet{}, err
			}
		}
	}

	return domain.NewRateSet(validFrom, publishedAt, rates), nil
}

func toOfficialRate(currency nbgclient.Currency) (domain.OfficialRate, error) {
	code, err := domain.ParseCurrencyCode(currency.Code)
	if err != nil {
		return domain.OfficialRate{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	rate, err := decimal.NewFromString(currency.RateFormated)
	if err != nil {
		return domain.OfficialRate{}, fmt.Errorf("%w: rate of %s: %w", errMalformed, code, err)
	}

	change, err := decimal.NewFromString(currency.DiffFormated)
	if err != nil {
		return domain.OfficialRate{}, fmt.Errorf("%w: change of %s: %w", errMalformed, code, err)
	}

	// diffFormated has no sign. diff carries it.
	if strings.HasPrefix(currency.Diff.String(), "-") {
		change = change.Neg()
	}

	officialRate, err := domain.NewOfficialRate(
		code,
		currency.Name,
		int64(currency.Quantity),
		rate,
		change,
	)
	if err != nil {
		return domain.OfficialRate{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return officialRate, nil
}

// The NBG writes the wall time of Tbilisi with the suffix Z. Article 3.1 of its rules
// sets publication no later than 17:00 Tbilisi time.
func toPublishedAt(text string) (time.Time, error) {
	published, err := time.ParseInLocation(
		publishedLayout,
		strings.TrimSuffix(text, "Z"),
		kernel.TbilisiZone(),
	)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: publication instant %q: %w", errMalformed, text, err)
	}

	return published, nil
}
