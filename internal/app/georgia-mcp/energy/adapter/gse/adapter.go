// Package gse reads the grid and the frequency from the GSE client and converts them to
// the domain. It is the anti-corruption layer of the context for GSE.
package gse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gseclient "github.com/abgeo/georgia-mcp/pkg/gse"
)

const (
	timeLayout      = "02.01.2006 15:04:05"
	gridPeriod      = "03M"
	frequencyPeriod = "01M"
	frequencyBlock  = "FREQUENCY"
	flowPrefix      = "TOTAL "
	flowSuffix      = " TIE LINES"
)

var (
	errWrongTime   = errors.New("the time is not the time of the file")
	errWrongPeriod = errors.New("the period is not the period of the file")
	errBadBlock    = errors.New("a block does not hold one entry")
	errNoBlock     = errors.New("a block is missing")
	errNoFlow      = errors.New("no flow in the file")
)

// Adapter implements app.GridSource with the GSE client.
type Adapter struct {
	client *gseclient.Client
	now    func() time.Time
}

var _ app.GridSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *gseclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Newest searches from the slot of the read.
func (a *Adapter) Newest(ctx context.Context) (app.GridRead, error) {
	return a.search(ctx, domain.SlotOf(a.now()))
}

// At searches from the slot.
func (a *Adapter) At(ctx context.Context, slot time.Time) (app.GridRead, error) {
	return a.search(ctx, domain.SlotOf(slot))
}

// search reads the grid file of each slot from first back domain.SearchWindow, until a
// file answers. Only a 404 tells that GSE has no file for a slot.
func (a *Adapter) search(ctx context.Context, first time.Time) (app.GridRead, error) {
	last := first.Add(-domain.SearchWindow)

	for slot := first; !slot.Before(last); slot = slot.Add(-domain.SlotLength) {
		export, err := a.client.Grid(ctx, slot)
		if notFound(err) {
			continue
		}

		if err != nil {
			return app.GridRead{}, unavailable(err)
		}

		readAt := a.now().In(kernel.TbilisiZone())

		state, err := gridState(export, slot)
		if err != nil {
			return app.GridRead{}, unavailable(err)
		}

		return app.GridRead{State: state, Frequency: a.frequency(ctx, slot), ReadAt: readAt}, nil
	}

	return app.GridRead{}, app.ErrNoGrid
}

// frequency gives nil for any failure: the grid state stands without it.
func (a *Adapter) frequency(ctx context.Context, minute time.Time) *domain.Frequency {
	export, err := a.client.Frequency(ctx, minute)
	if err != nil || checkHeader(export.Header, minute, frequencyPeriod) != nil {
		return nil
	}

	for _, block := range export.Blocks {
		if strings.TrimSpace(block.Name) != frequencyBlock {
			continue
		}

		value, err := single(block)
		if err != nil {
			return nil
		}

		return &domain.Frequency{Minute: minute, Value: value}
	}

	return nil
}

func gridState(export gseclient.Export, slot time.Time) (domain.GridState, error) {
	if err := checkHeader(export.Header, slot, gridPeriod); err != nil {
		return domain.GridState{}, err
	}

	state := domain.GridState{Slot: slot}
	fields := map[string]*domain.Value{
		"TOTAL HYDRO STATIONS":                   &state.Generation.Hydro,
		"TOTAL TERMAL STATIONS":                  &state.Generation.Thermal,
		"TOTAL WIND PLANTS":                      &state.Generation.Wind,
		"TOTAL SUN PLANTS":                       &state.Generation.Solar,
		"TOTAL GENERATION (HYDRO+TERMAL+WIND+SU": &state.Generation.Total,
		"TOTAL CONSUMPTION":                      &state.Consumption,
	}
	seen := map[string]bool{}

	for _, block := range export.Blocks {
		name := strings.TrimSpace(block.Name)

		if field, ok := fields[name]; ok {
			if seen[name] {
				return domain.GridState{}, fmt.Errorf("%s: %w", name, errBadBlock)
			}

			value, err := single(block)
			if err != nil {
				return domain.GridState{}, fmt.Errorf("%s: %w", name, err)
			}

			*field, seen[name] = value, true

			continue
		}

		if country, ok := countryOf(name); ok {
			flow, err := flowOf(country, block)
			if err != nil {
				return domain.GridState{}, fmt.Errorf("%s: %w", name, err)
			}

			state.Flows = append(state.Flows, flow)
		}
	}

	if len(seen) != len(fields) {
		return domain.GridState{}, errNoBlock
	}

	if len(state.Flows) == 0 {
		return domain.GridState{}, errNoFlow
	}

	return state, nil
}

func checkHeader(header gseclient.Header, at time.Time, period string) error {
	stamped, err := time.ParseInLocation(timeLayout, header.Time, kernel.TbilisiZone())
	if err != nil {
		return fmt.Errorf("parsing the time: %w", err)
	}

	if !stamped.Equal(at) {
		return fmt.Errorf("%s: %w", header.Time, errWrongTime)
	}

	if header.Period != period {
		return fmt.Errorf("%s: %w", header.Period, errWrongPeriod)
	}

	return nil
}

// countryOf reads "TOTAL AZERBAIJAN TIE LINES" as "Azerbaijan".
func countryOf(name string) (string, bool) {
	upper, ok := strings.CutPrefix(name, flowPrefix)
	if !ok {
		return "", false
	}

	upper, ok = strings.CutSuffix(upper, flowSuffix)
	if !ok || upper == "" {
		return "", false
	}

	return upper[:1] + strings.ToLower(upper[1:]), true
}

func flowOf(country string, block gseclient.Block) (domain.Flow, error) {
	if len(block.Entries) == 0 {
		return domain.Flow{}, errBadBlock
	}

	flow := domain.Flow{Country: country}

	for _, entry := range block.Entries {
		value, err := valueOf(entry)
		if err != nil {
			return domain.Flow{}, err
		}

		flow.Lines = append(
			flow.Lines,
			domain.Line{Name: strings.TrimSpace(entry.B3), Value: value},
		)
	}

	return flow, nil
}

func single(block gseclient.Block) (domain.Value, error) {
	if len(block.Entries) != 1 {
		return domain.Value{}, errBadBlock
	}

	return valueOf(block.Entries[0])
}

func valueOf(entry gseclient.Entry) (domain.Value, error) {
	number, err := decimal.NewFromString(strings.TrimSpace(entry.Value))
	if err != nil {
		return domain.Value{}, fmt.Errorf("parsing the value: %w", err)
	}

	return domain.Value{
		Number:  number,
		Quality: domain.Quality(strings.TrimSpace(entry.Quality)),
	}, nil
}

func notFound(err error) bool {
	var apiErr *gseclient.APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func unavailable(err error) error {
	return fmt.Errorf("reading GSE: %w: %w", kernel.ErrSourceUnavailable, err)
}
