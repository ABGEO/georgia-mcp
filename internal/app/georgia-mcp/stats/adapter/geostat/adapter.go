// Package geostat implements the ports of the statistics on the PxWeb API of Geostat. It
// converts each answer of Geostat to a domain value.
package geostat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
)

const (
	updatedLayout = "2006-01-02T15:04:05"
	folderNode    = "l"
	tableNode     = "t"
	folderJoin    = " / "

	// A folder list of Geostat can hang and then answer at the next call.
	folderAttempts = 3
	folderPause    = 2 * time.Second

	// The codes of the values are the same in each language, so the cells are read in one.
	cellLanguage = "en"
)

// errMalformed tells that an answer of Geostat does not have the shape that the adapter
// reads.
var errMalformed = errors.New("geostat: malformed answer")

// Adapter reads Geostat for the ports of the context.
type Adapter struct {
	client *geostatclient.Client
	now    func() time.Time
	pause  func(context.Context, time.Duration) error
}

var (
	_ app.CatalogSource    = (*Adapter)(nil)
	_ app.TableSource      = (*Adapter)(nil)
	_ app.PriceIndexSource = (*Adapter)(nil)
)

// NewAdapter returns an adapter over the client. now gives the instant of a read. pause
// waits between two attempts of a folder list.
func NewAdapter(
	client *geostatclient.Client,
	now func() time.Time,
	pause func(context.Context, time.Duration) error,
) *Adapter {
	return &Adapter{client: client, now: now, pause: pause}
}

// Pause waits for the duration, or until the context ends.
func Pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// Catalog walks the folders of Geostat from the root, one at a time.
func (a *Adapter) Catalog(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	entries := []domain.CatalogEntry{}
	if err := a.walk(ctx, string(language), nil, &entries); err != nil {
		return app.Catalog{}, unavailable("reading the catalog of Geostat", err)
	}

	return app.Catalog{Entries: entries, ReadAt: a.readAt()}, nil
}

// Table reads the folder of the table for its update date, then the table.
func (a *Adapter) Table(
	ctx context.Context,
	code domain.TableCode,
	language kernel.Language,
) (app.TableDescription, error) {
	entry, err := a.entry(ctx, string(language), code)
	if err != nil {
		return app.TableDescription{}, err
	}

	metadata, err := a.client.Table(ctx, string(language), code.String())
	if err != nil {
		return app.TableDescription{}, unavailable("describing the table of Geostat", err)
	}

	dimensions, err := toDimensions(metadata.Variables)
	if err != nil {
		return app.TableDescription{}, unavailable("describing the table of Geostat", err)
	}

	return app.TableDescription{
		Table:  domain.Table{Entry: entry, Dimensions: dimensions},
		ReadAt: a.readAt(),
	}, nil
}

// Cells queries the selection in English.
func (a *Adapter) Cells(
	ctx context.Context,
	code domain.TableCode,
	selection []app.DimensionSelection,
) (app.CellSet, error) {
	query := make([]geostatclient.Selection, 0, len(selection))
	for _, item := range selection {
		query = append(query, geostatclient.Selection{Code: item.Dimension, Values: item.Values})
	}

	figures, err := a.query(ctx, code.String(), query)
	if err != nil {
		return app.CellSet{}, unavailable("reading the cells of Geostat", err)
	}

	return app.CellSet{Figures: figures, ReadAt: a.readAt()}, nil
}

// PriceIndices reads the four tables of the consumer price index for one year.
func (a *Adapter) PriceIndices(ctx context.Context, year int) (app.PriceIndexYear, error) {
	indices, err := a.priceIndices(ctx, year)
	if err != nil {
		return app.PriceIndexYear{}, unavailable("reading the consumer price index of Geostat", err)
	}

	return indices, nil
}

// walk reads each folder under ids. The folder of an entry is its path of IDs: the same
// text in both languages, and the same as in Table.
func (a *Adapter) walk(
	ctx context.Context,
	lang string,
	ids []string,
	entries *[]domain.CatalogEntry,
) error {
	nodes, err := a.folder(ctx, lang, strings.Join(ids, "/"))
	if err != nil {
		return err
	}

	for _, node := range nodes {
		path := append(slices.Clip(ids), node.ID)

		switch node.Type {
		case folderNode:
			if err := a.walk(ctx, lang, path, entries); err != nil {
				return err
			}
		case tableNode:
			entry, err := toEntry(strings.Join(path, "/"), strings.Join(ids, folderJoin), node)
			if err != nil {
				return err
			}

			*entries = append(*entries, entry)
		}
	}

	return nil
}

// folder sends a folder list up to 3 times.
func (a *Adapter) folder(ctx context.Context, lang, path string) ([]geostatclient.Node, error) {
	var err error

	for attempt := range folderAttempts {
		if attempt > 0 {
			if pauseErr := a.pause(ctx, folderPause); pauseErr != nil {
				return nil, errors.Join(err, pauseErr)
			}
		}

		var nodes []geostatclient.Node

		nodes, err = a.client.Folder(ctx, lang, path)
		if err == nil {
			return nodes, nil
		}
	}

	return nil, fmt.Errorf("listing the folder %q: %w", path, err)
}

// entry finds the table in the list of its folder. A folder that answers 404 holds no
// table.
func (a *Adapter) entry(
	ctx context.Context,
	lang string,
	code domain.TableCode,
) (domain.CatalogEntry, error) {
	nodes, err := a.client.Folder(ctx, lang, code.Folder())

	var apiErr *geostatclient.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return domain.CatalogEntry{}, fmt.Errorf("%w: %s", app.ErrUnknownTable, code)
	}

	if err != nil {
		return domain.CatalogEntry{}, unavailable("listing the folder of the table", err)
	}

	for _, node := range nodes {
		if node.Type == tableNode && node.ID == code.File() {
			folder := strings.ReplaceAll(code.Folder(), "/", folderJoin)

			entry, err := toEntry(code.String(), folder, node)
			if err != nil {
				return domain.CatalogEntry{}, unavailable("listing the folder of the table", err)
			}

			return entry, nil
		}
	}

	return domain.CatalogEntry{}, fmt.Errorf("%w: %s", app.ErrUnknownTable, code)
}

// query returns the figure of each cell under the key of its value codes.
func (a *Adapter) query(
	ctx context.Context,
	path string,
	selection []geostatclient.Selection,
) (map[string]domain.Figure, error) {
	dataset, err := a.client.Query(ctx, cellLanguage, path, selection)
	if err != nil {
		return nil, fmt.Errorf("querying the table: %w", err)
	}

	figures := make(map[string]domain.Figure, len(dataset.Data))

	for _, datum := range dataset.Data {
		if len(datum.Key) != len(selection) || len(datum.Values) == 0 {
			return nil, fmt.Errorf("%w: a cell has %d codes and %d figures for %d dimensions",
				errMalformed, len(datum.Key), len(datum.Values), len(selection))
		}

		figure, err := domain.ParseFigure(datum.Values[0])
		if err != nil {
			return nil, fmt.Errorf("reading a figure: %w", err)
		}

		figures[app.CellKey(datum.Key)] = figure
	}

	return figures, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

func unavailable(action string, err error) error {
	return fmt.Errorf("%s: %w: %w", action, kernel.ErrSourceUnavailable, err)
}

func toEntry(path, folder string, node geostatclient.Node) (domain.CatalogEntry, error) {
	code, err := domain.ParseTableCode(path)
	if err != nil {
		return domain.CatalogEntry{}, fmt.Errorf("reading the code of a table: %w", err)
	}

	updated, err := time.ParseInLocation(updatedLayout, node.Updated, kernel.TbilisiZone())
	if err != nil {
		return domain.CatalogEntry{}, fmt.Errorf("reading the update date of %q: %w", path, err)
	}

	return domain.CatalogEntry{
		Code: code, Title: clean(node.Text), Folder: folder, Updated: updated,
	}, nil
}

func toDimensions(variables []geostatclient.Variable) ([]domain.Dimension, error) {
	dimensions := make([]domain.Dimension, 0, len(variables))

	for _, variable := range variables {
		if err := checkLengths(variable); err != nil {
			return nil, err
		}

		values := make([]domain.DimensionValue, 0, len(variable.Values))
		for i, value := range variable.Values {
			values = append(values, domain.DimensionValue{
				Code: value, Text: clean(variable.ValueTexts[i]),
			})
		}

		dimensions = append(dimensions, domain.Dimension{
			Code: variable.Code, Text: clean(variable.Text), Values: values,
		})
	}

	return dimensions, nil
}

func checkLengths(variable geostatclient.Variable) error {
	if len(variable.Values) != len(variable.ValueTexts) {
		return fmt.Errorf("%w: the dimension %q has %d values and %d texts", errMalformed,
			variable.Code, len(variable.Values), len(variable.ValueTexts))
	}

	return nil
}

// clean removes the spaces at the ends of a text of Geostat, and writes each inner run of
// white space as one space: a text can hold a line break.
func clean(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
