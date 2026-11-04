package geostat

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
)

const (
	cpiFolder     = "Price Statistics/CPI/National Consumer Price Index"
	cpiFolderName = "Price Statistics / CPI / National Consumer Price Index"

	annualFile  = "CPI_in_Georgia(same_month_of_the_previous_year=100_orenovani2.px"
	monthlyFile = "CPI_in_Georgia_(previous_month=100)_orenovani2.px"
	averageFile = "CPI_in_Georgia_the_previous_12_month_average=100_orenovani2.px"
	coreFile    = "Core_Inflation-(to_the-same-month-of-the-previous-year).px"

	coreIndicator = "Core Inflation"
)

// romanMonths holds the text of Geostat for each month, from January.
func romanMonths() []string {
	return []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
}

// indexTable is one table of the consumer price index. Its dimensions come in this order:
// the region or the core indicator, the group when it has one, the year, then the month.
type indexTable struct {
	file     string
	hasGroup bool
	isCore   bool
}

// dimensions counts the region, the year, the month, and the group when the table has one.
func (t indexTable) dimensions() int {
	const withoutGroup = 3

	if t.hasGroup {
		return withoutGroup + 1
	}

	return withoutGroup
}

// axis maps the code of each value of a dimension to its meaning in the domain.
type axis[T comparable] struct {
	code   string
	values map[string]T
}

func (x axis[T]) codes() []string {
	codes := make([]string, 0, len(x.values))
	for code := range x.values {
		codes = append(codes, code)
	}

	slices.Sort(codes)

	return codes
}

func (a *Adapter) priceIndices(ctx context.Context, year int) (app.PriceIndexYear, error) {
	tables, err := a.indexEntries(ctx)
	if err != nil {
		return app.PriceIndexYear{}, err
	}

	indices := app.PriceIndexYear{
		Year:    year,
		Annual:  map[app.IndexKey]domain.Figure{},
		Monthly: map[app.IndexKey]domain.Figure{},
		Average: map[app.IndexKey]domain.Figure{},
		Core:    map[int]domain.Figure{},
		Tables:  tables,
		ReadAt:  a.readAt(),
	}
	core := map[app.IndexKey]domain.Figure{}

	targets := []struct {
		table  indexTable
		target map[app.IndexKey]domain.Figure
	}{
		{indexTable{file: annualFile, hasGroup: true}, indices.Annual},
		{indexTable{file: monthlyFile, hasGroup: true}, indices.Monthly},
		{indexTable{file: averageFile}, indices.Average},
		{indexTable{file: coreFile, isCore: true}, core},
	}

	for _, target := range targets {
		if err := a.readIndex(ctx, target.table, year, target.target); err != nil {
			return app.PriceIndexYear{}, err
		}
	}

	for key, figure := range core {
		indices.Core[key.Month] = figure
	}

	return indices, nil
}

// indexEntries finds the four tables in the list of their folder.
func (a *Adapter) indexEntries(ctx context.Context) ([]domain.CatalogEntry, error) {
	nodes, err := a.client.Folder(ctx, cellLanguage, cpiFolder)
	if err != nil {
		return nil, fmt.Errorf("listing the folder of the index: %w", err)
	}

	files := []string{annualFile, monthlyFile, averageFile, coreFile}
	entries := make([]domain.CatalogEntry, 0, len(files))

	for _, file := range files {
		index := slices.IndexFunc(nodes, func(node geostatclient.Node) bool {
			return node.Type == tableNode && node.ID == file
		})
		if index < 0 {
			return nil, fmt.Errorf("%w: the folder of the index lists no %q", errMalformed, file)
		}

		entry, err := toEntry(cpiFolder+"/"+file, cpiFolderName, nodes[index])
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// readIndex reads one table for one year into target. A table that does not list the year
// adds nothing.
func (a *Adapter) readIndex(
	ctx context.Context,
	table indexTable,
	year int,
	target map[app.IndexKey]domain.Figure,
) error {
	path := cpiFolder + "/" + table.file

	metadata, err := a.client.Table(ctx, cellLanguage, path)
	if err != nil {
		return fmt.Errorf("describing %q: %w", table.file, err)
	}

	selection, found, err := newIndexSelection(metadata.Variables, table, year)
	if err != nil || !found {
		return err
	}

	figures, err := a.query(ctx, path, selection.query)
	if err != nil {
		return fmt.Errorf("reading %q: %w", table.file, err)
	}

	for cellKey, figure := range figures {
		key, known := selection.keys[cellKey]
		if !known {
			return fmt.Errorf("%w: %q gives a cell that the query did not select",
				errMalformed, table.file)
		}

		target[key] = figure
	}

	return nil
}

// indexSelection holds the query of a table, and the index key of each cell.
type indexSelection struct {
	query []geostatclient.Selection
	keys  map[string]app.IndexKey
}

func newIndexSelection(
	variables []geostatclient.Variable,
	table indexTable,
	year int,
) (indexSelection, bool, error) {
	if err := checkIndexTable(variables, table); err != nil {
		return indexSelection{}, false, err
	}

	yearVariable := variables[len(variables)-2]

	yearIndex := slices.Index(yearVariable.ValueTexts, strconv.Itoa(year))
	if yearIndex < 0 {
		return indexSelection{}, false, nil
	}

	regions, err := regionAxis(variables[0], table.isCore)
	if err != nil {
		return indexSelection{}, false, err
	}

	groups := axis[domain.PriceGroup]{values: map[string]domain.PriceGroup{"": domain.GroupTotal}}
	if table.hasGroup {
		groups, err = byText(variables[1], domain.PriceGroups(), domain.PriceGroup.Name)
		if err != nil {
			return indexSelection{}, false, err
		}
	}

	months, err := byText(variables[len(variables)-1], monthNumbers(), romanMonth)
	if err != nil {
		return indexSelection{}, false, err
	}

	yearAxis := axis[int]{
		code:   yearVariable.Code,
		values: map[string]int{yearVariable.Values[yearIndex]: year},
	}

	return combine(regions, groups, table.hasGroup, yearAxis, months), true, nil
}

func checkIndexTable(variables []geostatclient.Variable, table indexTable) error {
	if len(variables) != table.dimensions() {
		return fmt.Errorf("%w: %q has %d dimensions, want %d",
			errMalformed, table.file, len(variables), table.dimensions())
	}

	for _, variable := range variables {
		if err := checkLengths(variable); err != nil {
			return err
		}
	}

	return nil
}

// combine builds the query and the index key of each cell. A table with no group gives
// every cell the group total, and sends no group in the query.
func combine(
	regions axis[domain.Region],
	groups axis[domain.PriceGroup],
	hasGroup bool,
	year axis[int],
	months axis[int],
) indexSelection {
	yearCode := year.codes()[0]
	selection := indexSelection{keys: map[string]app.IndexKey{}}

	selection.query = append(selection.query,
		geostatclient.Selection{Code: regions.code, Values: regions.codes()})
	if hasGroup {
		selection.query = append(selection.query,
			geostatclient.Selection{Code: groups.code, Values: groups.codes()})
	}

	selection.query = append(selection.query,
		geostatclient.Selection{Code: year.code, Values: []string{yearCode}},
		geostatclient.Selection{Code: months.code, Values: months.codes()},
	)

	for regionCode, region := range regions.values {
		for groupCode, group := range groups.values {
			for monthCode, month := range months.values {
				codes := []string{regionCode}
				if hasGroup {
					codes = append(codes, groupCode)
				}

				codes = append(codes, yearCode, monthCode)
				selection.keys[app.CellKey(codes)] = app.IndexKey{
					Region: region, Group: group, Month: month,
				}
			}
		}
	}

	return selection
}

// regionAxis maps the regions of a table. The table of the core inflation has an
// indicator in place of the region: its core indicator stands for the country.
func regionAxis(variable geostatclient.Variable, isCore bool) (axis[domain.Region], error) {
	if !isCore {
		return byText(variable, domain.Regions(), domain.Region.Name)
	}

	code, err := codeOf(variable, coreIndicator)
	if err != nil {
		return axis[domain.Region]{}, err
	}

	return axis[domain.Region]{
		code: variable.Code, values: map[string]domain.Region{code: domain.RegionGeorgia},
	}, nil
}

func romanMonth(number int, _ kernel.Language) string {
	return romanMonths()[number-1]
}

func monthNumbers() []int {
	numbers := make([]int, 0, len(romanMonths()))
	for number := range len(romanMonths()) {
		numbers = append(numbers, number+1)
	}

	return numbers
}

// byText finds the code of each wanted value by its English text.
func byText[T comparable](
	variable geostatclient.Variable,
	wanted []T,
	text func(T, kernel.Language) string,
) (axis[T], error) {
	found := axis[T]{code: variable.Code, values: map[string]T{}}

	for _, value := range wanted {
		code, err := codeOf(variable, text(value, kernel.LanguageEnglish))
		if err != nil {
			return axis[T]{}, err
		}

		found.values[code] = value
	}

	return found, nil
}

func codeOf(variable geostatclient.Variable, text string) (string, error) {
	for i, valueText := range variable.ValueTexts {
		if clean(valueText) == text {
			return variable.Values[i], nil
		}
	}

	return "", fmt.Errorf("%w: the dimension %q has no value %q", errMalformed, variable.Code, text)
}
