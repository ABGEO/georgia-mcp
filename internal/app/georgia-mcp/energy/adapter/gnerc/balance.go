package gnerc

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
)

const (
	labelColumn  = "B"
	headerLabel  = "დასახელება"
	totalHeader  = "სულ"
	sheetSuffix  = " GEO"
	monthsInYear = 12
)

var (
	errNoHeader     = errors.New("sheet: no header row")
	errNoColumn     = errors.New("sheet: no column")
	errTwice        = errors.New("sheet: an item twice")
	errNoItem       = errors.New("sheet: no required item")
	footnotePattern = regexp.MustCompile(`\s*\(\*+\)$`)
)

// monthNames returns the name of each month in Georgian, as GNERC writes it.
func monthNames() map[string]time.Month {
	return map[string]time.Month{
		"იანვარი": time.January, "თებერვალი": time.February, "მარტი": time.March,
		"აპრილი": time.April, "მაისი": time.May, "ივნისი": time.June,
		"ივლისი": time.July, "აგვისტო": time.August, "სექტემბერი": time.September,
		"ოქტომბერი": time.October, "ნოემბერი": time.November, "დეკემბერი": time.December,
	}
}

// labels returns the item of each label of the balance, with the spelling of GNERC.
func labels() map[string]domain.Item {
	return map[string]domain.Item{
		"ელექტროენერგიის ჯამური გამომუშავება":         domain.GenerationTotal,
		"თბოელექტროსადგურების ჯამური გამომუშავება":    domain.GenerationThermal,
		"ქარის ელექტროსადგურების ჯამური გამომუშავება": domain.GenerationWind,
		"მზის ელექტროსადგურების ჯამური გამომუშავება":  domain.GenerationSolar,
		"ჰიდროელექტროსადგურების ჯამური გამომუშავება":  domain.GenerationHydro,
		"ელექტროენერგიის საერთო სასადგურო დანახარჯი":  domain.StationOwnUse,
		"ელექტროენერგიის ჯამური იმპორტი":              domain.ImportTotal,
		"რუსეთიდან":      domain.ImportRussia,
		"თურქეთიდან":     domain.ImportTurkey,
		"აზერბაიჯანიდან": domain.ImportAzerbaijan,
		"სომხეთიდან":     domain.ImportArmenia,
		"მ.შ. აფხაზეთში": domain.ImportIntoAbkhazia,
		"ელექტროენერგიის ჯამური მიწოდება":  domain.Supply,
		"ელექტროენერგიის ჯამური მოხმარება": domain.ConsumptionTotal,
		"აფხაზეთი": domain.ConsumptionAbkhazia,
		"ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება":                              domain.ConsumptionSuppliers,
		"უნივერსალური მომსახურების მიმწოდებლების ჯამური მოხმარება":                    domain.ConsumptionUniversalService,
		"საჯარო მომსახურების სახით ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება":    domain.ConsumptionPublicService,
		"თავისუფალი მომხმარბლის სახით ელექტროენერგიის მიმწოდებლების ჯამური მოხმარება": domain.ConsumptionFreeMarket,
		"პირდაპირი მომხმარებლების ჯამური მოხმარება":                                   domain.ConsumptionDirect,
		"გაჩერებული ელექტროსადგურების საკუთარი მოხამარება":                            domain.ConsumptionStoppedStations,
		"ელექტროენერგიის ჯამური ექსპორტი":                                             domain.ExportTotal,
		"რუსეთში":      domain.ExportRussia,
		"თურქეთში":     domain.ExportTurkey,
		"აზერბაიჯანში": domain.ExportAzerbaijan,
		"სომხეთში":     domain.ExportArmenia,
		"ელექტროენერგიის გადამცემი და გამანაწილებელი ქსელის " +
			"ელექტროენერგიის ჯამური დანაკარგი": domain.LossesTotal,
		"გადამცემი სისტემის ოპერატორის მიერ ქსელში დანაკარგების " +
			"დაფარვის მიზნით შესყიდული ჯამური ელექტროენერგია": domain.LossesTransmission,
		"გამანაწილებელი სისტემის ოპერატორების მიერ ქსელში დანაკარგების " +
			"დაფარვის მიზნით შესყიდული ჯამური ელექტროენერგია": domain.LossesDistribution,
		"საქართველოს შიდა მოხმარება":     domain.DomesticConsumption,
		"ელექტროენეგიის ჯამური ტრანზიტი": domain.TransitTotal,
		"აზერბაიჯანიდან თურქეთში":        domain.TransitAzerbaijanTurkey,
		"რუსეთიდან თურქეთში":             domain.TransitRussiaTurkey,
		"სომხეთიდან თურქეთში":            domain.TransitArmeniaTurkey,
	}
}

// balanceOf maps each sheet "<year> GEO" from domain.FirstBalanceYear. A year with no
// published month is left out.
func balanceOf(book gnercclient.Workbook) ([]domain.YearBalance, error) {
	var years []domain.YearBalance

	for _, sheet := range book.Sheets {
		yearText, found := strings.CutSuffix(strings.TrimSpace(sheet.Name), sheetSuffix)
		if !found {
			continue
		}

		year, err := strconv.Atoi(yearText)
		if err != nil || year < domain.FirstBalanceYear {
			continue
		}

		balance, err := yearOf(year, sheet)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", sheet.Name, err)
		}

		if len(balance.Months) > 0 {
			years = append(years, balance)
		}
	}

	return years, nil
}

// columns names the column of each month and of the year total.
type columns struct {
	months map[string]time.Month
	total  string
}

func yearOf(year int, sheet gnercclient.Sheet) (domain.YearBalance, error) {
	header, layout, err := headerOf(sheet)
	if err != nil {
		return domain.YearBalance{}, err
	}

	months := make(map[time.Month]domain.Figures, monthsInYear)
	for _, m := range layout.months {
		months[m] = domain.Figures{}
	}

	total := domain.Figures{}
	known := labels()

	for _, row := range sheet.Rows[header+1:] {
		item, found := known[labelOf(row)]
		if !found {
			continue
		}

		if _, twice := total[item]; twice {
			return domain.YearBalance{}, fmt.Errorf("%w: %s", errTwice, item)
		}

		if err := fill(row, item, layout, months, total); err != nil {
			return domain.YearBalance{}, err
		}
	}

	if err := checkRequired(total); err != nil {
		return domain.YearBalance{}, err
	}

	return domain.YearBalance{Year: year, Total: total, Months: published(year, months)}, nil
}

func checkRequired(total domain.Figures) error {
	for _, item := range domain.Items() {
		if _, found := total[item]; item.Required() && !found {
			return fmt.Errorf("%w: %s", errNoItem, item)
		}
	}

	return nil
}

// published returns each month whose total generation is above zero, in order. GNERC
// fills the months to come with zero.
func published(year int, months map[time.Month]domain.Figures) []domain.MonthBalance {
	var balances []domain.MonthBalance

	for calendar := time.January; calendar <= time.December; calendar++ {
		if figures := months[calendar]; figures[domain.GenerationTotal].IsPositive() {
			balances = append(balances, domain.MonthBalance{
				Month: domain.Month{Year: year, Month: calendar}, Figures: figures,
			})
		}
	}

	return balances
}

// fill sets the value of item in each month and in the total. An empty cell is zero.
func fill(
	row gnercclient.Row,
	item domain.Item,
	layout columns,
	months map[time.Month]domain.Figures,
	total domain.Figures,
) error {
	cells := make(map[string]string, len(row.Cells))
	for _, cell := range row.Cells {
		cells[cell.Column] = cell.Text
	}

	read := func(column string) (decimal.Decimal, error) {
		text, found := cells[column]
		if !found || strings.TrimSpace(text) == "" {
			return decimal.Zero, nil
		}

		return valueOf(text)
	}

	for column, calendar := range layout.months {
		value, err := read(column)
		if err != nil {
			return fmt.Errorf("row %d: %w", row.Number, err)
		}

		months[calendar][item] = value
	}

	value, err := read(layout.total)
	if err != nil {
		return fmt.Errorf("row %d: %w", row.Number, err)
	}

	total[item] = value

	return nil
}

// headerOf finds the row whose label is headerLabel, and the column of each month and of
// the total in it.
func headerOf(sheet gnercclient.Sheet) (int, columns, error) {
	names := monthNames()

	for index, row := range sheet.Rows {
		if labelOf(row) != headerLabel {
			continue
		}

		layout := columns{months: map[string]time.Month{}}

		for _, cell := range row.Cells {
			text := collapse(cell.Text)
			if m, found := names[text]; found {
				layout.months[cell.Column] = m
			}

			if text == totalHeader {
				layout.total = cell.Column
			}
		}

		if len(layout.months) != monthsInYear || layout.total == "" {
			return 0, columns{}, fmt.Errorf(
				"%w: the header names %d months",
				errNoColumn,
				len(layout.months),
			)
		}

		return index, layout, nil
	}

	return 0, columns{}, errNoHeader
}

// labelOf returns the text of the cell of the label, with each run of spaces as one space
// and no footnote mark.
func labelOf(row gnercclient.Row) string {
	for _, cell := range row.Cells {
		if cell.Column == labelColumn {
			return footnotePattern.ReplaceAllString(collapse(cell.Text), "")
		}
	}

	return ""
}
