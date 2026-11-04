package domain

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	monthTextLength = len("2006-01")
	monthsInYear    = 12
)

// Month is one month of a year: 2026-10.
type Month struct {
	year, month int
}

// ParseMonth returns the month of a text in the form YYYY-MM.
func ParseMonth(text string) (Month, error) {
	if len(text) != monthTextLength || text[4] != '-' {
		return Month{}, fmt.Errorf("%w: %q", ErrInvalidMonth, text)
	}

	year, yearErr := digits(text[:4])
	month, monthErr := digits(text[5:])

	if yearErr != nil || monthErr != nil || month < 1 || month > monthsInYear {
		return Month{}, fmt.Errorf("%w: %q", ErrInvalidMonth, text)
	}

	return Month{year: year, month: month}, nil
}

// NewMonth returns the month of a year and a month number from 1 to 12.
func NewMonth(year, month int) (Month, error) {
	return ParseMonth(fmt.Sprintf("%04d-%02d", year, month))
}

func digits(text string) (int, error) {
	for _, r := range text {
		if r < '0' || r > '9' {
			return 0, ErrInvalidMonth
		}
	}

	number, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidMonth, err)
	}

	return number, nil
}

// Year returns the year: 2026.
func (m Month) Year() int {
	return m.year
}

// Number returns the month of the year, from 1 to 12.
func (m Month) Number() int {
	return m.month
}

// IsZero tells whether the month is the zero value.
func (m Month) IsZero() bool {
	return m.month == 0
}

// String returns the month in the form YYYY-MM: "2026-10".
func (m Month) String() string {
	return fmt.Sprintf("%04d-%02d", m.year, m.month)
}

// names holds the English text of Geostat and the Georgian name of a code.
type names struct {
	english, georgian string
}

func (n names) in(language kernel.Language) string {
	if language == kernel.LanguageGeorgian {
		return n.georgian
	}

	return n.english
}

// Region is a region of the consumer price index of Geostat: the country or a city.
type Region string

// The regions of the consumer price index.
const (
	RegionGeorgia Region = "georgia"
	RegionTbilisi Region = "tbilisi"
	RegionKutaisi Region = "kutaisi"
	RegionBatumi  Region = "batumi"
	RegionGori    Region = "gori"
	RegionTelavi  Region = "telavi"
	RegionZugdidi Region = "zugdidi"
)

func regionNames() map[Region]names {
	return map[Region]names{
		RegionGeorgia: {"Georgia", "საქართველო"},
		RegionTbilisi: {"Tbilisi", "თბილისი"},
		RegionKutaisi: {"Kutaisi", "ქუთაისი"},
		RegionBatumi:  {"Batumi", "ბათუმი"},
		RegionGori:    {"Gori", "გორი"},
		RegionTelavi:  {"Telavi", "თელავი"},
		RegionZugdidi: {"Zugdidi", "ზუგდიდი"},
	}
}

// Regions returns every region, the country first.
func Regions() []Region {
	return []Region{
		RegionGeorgia, RegionTbilisi, RegionKutaisi, RegionBatumi,
		RegionGori, RegionTelavi, RegionZugdidi,
	}
}

// ParseRegion returns the region of a code: "tbilisi".
func ParseRegion(code string) (Region, error) {
	region := Region(code)
	if !slices.Contains(Regions(), region) {
		return "", fmt.Errorf("%w: %q", ErrUnknownRegion, code)
	}

	return region, nil
}

// Name returns the name of the region. The English name is the text of Geostat.
func (r Region) Name(language kernel.Language) string {
	return regionNames()[r].in(language)
}

// PriceGroup is a group of the consumer price index of Geostat.
type PriceGroup string

// The price groups of the consumer price index.
const (
	GroupTotal          PriceGroup = "total"
	GroupFood           PriceGroup = "food"
	GroupAlcoholTobacco PriceGroup = "alcohol_tobacco"
	GroupClothing       PriceGroup = "clothing"
	GroupHousing        PriceGroup = "housing"
	GroupFurnishings    PriceGroup = "furnishings"
	GroupHealth         PriceGroup = "health"
	GroupTransport      PriceGroup = "transport"
	GroupCommunication  PriceGroup = "communication"
	GroupRecreation     PriceGroup = "recreation"
	GroupEducation      PriceGroup = "education"
	GroupRestaurants    PriceGroup = "restaurants"
	GroupMiscellaneous  PriceGroup = "miscellaneous"
)

func groupNames() map[PriceGroup]names {
	return map[PriceGroup]names{
		GroupTotal: {"Total", "სულ"},
		GroupFood:  {"Food and non-alcoholic beverages", "სურსათი და უალკოჰოლო სასმელები"},
		GroupAlcoholTobacco: {
			"Alcoholic beverages and tobacco", "ალკოჰოლური სასმელები და თამბაქო",
		},
		GroupClothing: {"Clothing and footwear", "ტანსაცმელი და ფეხსაცმელი"},
		GroupHousing: {
			"Housing, water, electricity, gas and other fuels",
			"საცხოვრებელი სახლი, წყალი, ელექტროენერგია, აირი და სათბობის სხვა სახეები",
		},
		GroupFurnishings: {
			"Furnishings, household equipment and routine maintenance of the house",
			"ავეჯი, საოჯახო ნივთები და მორთულობა, სახლის მოვლა-შეკეთება",
		},
		GroupHealth:        {"Health", "ჯანმრთელობის დაცვა"},
		GroupTransport:     {"Transport", "ტრანსპორტი"},
		GroupCommunication: {"Communication", "კავშირგაბმულობა"},
		GroupRecreation:    {"Recreation and culture", "დასვენება, გართობა და კულტურა"},
		GroupEducation:     {"Education", "განათლება"},
		GroupRestaurants:   {"Restaurants and hotels", "სასტუმროები, კაფეები და რესტორნები"},
		GroupMiscellaneous: {
			"Miscellaneous goods and services", "სხვადასხვა საქონელი და მომსახურება",
		},
	}
}

// PriceGroups returns every price group, the total first.
func PriceGroups() []PriceGroup {
	return []PriceGroup{
		GroupTotal, GroupFood, GroupAlcoholTobacco, GroupClothing, GroupHousing,
		GroupFurnishings, GroupHealth, GroupTransport, GroupCommunication,
		GroupRecreation, GroupEducation, GroupRestaurants, GroupMiscellaneous,
	}
}

// ParsePriceGroup returns the price group of a code: "food".
func ParsePriceGroup(code string) (PriceGroup, error) {
	group := PriceGroup(code)
	if !slices.Contains(PriceGroups(), group) {
		return "", fmt.Errorf("%w: %q", ErrUnknownPriceGroup, code)
	}

	return group, nil
}

// Name returns the name of the price group. The English name is the text of Geostat.
func (g PriceGroup) Name(language kernel.Language) string {
	return groupNames()[g].in(language)
}

// Inflation is a read model: the inflation of one month, region, and price group, in
// percent.
type Inflation struct {
	Month  Month
	Region Region
	Group  PriceGroup
	// Annual is the change since the same month of the previous year.
	Annual Figure
	// Monthly is the change since the previous month.
	Monthly Figure
	// Average is the change of the average of the last 12 months over the 12 months
	// before. Geostat publishes it for the group total only.
	Average Figure
	// Core is the core inflation. Geostat publishes it for the country and the group
	// total only.
	Core Figure
}
