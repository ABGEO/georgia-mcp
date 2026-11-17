package domain

import (
	"slices"

	"github.com/shopspring/decimal"
)

// Item is one item of the balance of GNERC.
type Item string

// The items of the balance, in million kWh.
const (
	GenerationTotal             Item = "generation_total"
	GenerationThermal           Item = "generation_thermal"
	GenerationWind              Item = "generation_wind"
	GenerationSolar             Item = "generation_solar"
	GenerationHydro             Item = "generation_hydro"
	StationOwnUse               Item = "station_own_use"
	ImportTotal                 Item = "import_total"
	ImportRussia                Item = "import_russia"
	ImportTurkey                Item = "import_turkey"
	ImportAzerbaijan            Item = "import_azerbaijan"
	ImportArmenia               Item = "import_armenia"
	ImportIntoAbkhazia          Item = "import_into_abkhazia"
	Supply                      Item = "supply"
	ConsumptionTotal            Item = "consumption_total"
	ConsumptionAbkhazia         Item = "consumption_abkhazia"
	ConsumptionSuppliers        Item = "consumption_suppliers"
	ConsumptionUniversalService Item = "consumption_universal_service"
	ConsumptionPublicService    Item = "consumption_public_service"
	ConsumptionFreeMarket       Item = "consumption_free_market"
	ConsumptionDirect           Item = "consumption_direct"
	ConsumptionStoppedStations  Item = "consumption_stopped_stations"
	ExportTotal                 Item = "export_total"
	ExportRussia                Item = "export_russia"
	ExportTurkey                Item = "export_turkey"
	ExportAzerbaijan            Item = "export_azerbaijan"
	ExportArmenia               Item = "export_armenia"
	LossesTotal                 Item = "losses_total"
	LossesTransmission          Item = "losses_transmission"
	LossesDistribution          Item = "losses_distribution"
	DomesticConsumption         Item = "domestic_consumption"
	TransitTotal                Item = "transit_total"
	TransitAzerbaijanTurkey     Item = "transit_azerbaijan_turkey"
	TransitRussiaTurkey         Item = "transit_russia_turkey"
	TransitArmeniaTurkey        Item = "transit_armenia_turkey"
)

// Items returns every item of the balance, in the order of the balance.
func Items() []Item {
	return []Item{
		GenerationTotal,
		GenerationThermal,
		GenerationWind,
		GenerationSolar,
		GenerationHydro,
		StationOwnUse,
		ImportTotal,
		ImportRussia,
		ImportTurkey,
		ImportAzerbaijan,
		ImportArmenia,
		ImportIntoAbkhazia,
		Supply,
		ConsumptionTotal,
		ConsumptionAbkhazia,
		ConsumptionSuppliers,
		ConsumptionUniversalService,
		ConsumptionPublicService,
		ConsumptionFreeMarket,
		ConsumptionDirect,
		ConsumptionStoppedStations,
		ExportTotal,
		ExportRussia,
		ExportTurkey,
		ExportAzerbaijan,
		ExportArmenia,
		LossesTotal,
		LossesTransmission,
		LossesDistribution,
		DomesticConsumption,
		TransitTotal,
		TransitAzerbaijanTurkey,
		TransitRussiaTurkey,
		TransitArmeniaTurkey,
	}
}

// Required tells whether each year of the balance must hold the item.
func (i Item) Required() bool {
	return slices.Contains(
		[]Item{GenerationTotal, ImportTotal, Supply, ConsumptionTotal, ExportTotal},
		i,
	)
}

// Figures holds the value of each item that GNERC gives, in million kWh. An item that
// GNERC does not give for the year has no key.
type Figures map[Item]decimal.Decimal

// MonthBalance is the balance of one published month.
type MonthBalance struct {
	Month   Month
	Figures Figures
}

// YearBalance is the balance of one year.
type YearBalance struct {
	Year int
	// Months are the published months, in order.
	Months []MonthBalance
	// Total is the column of the year total of GNERC.
	Total Figures
}

// MonthPrices holds the three prices of one month, in tetri per kWh. A price that GNERC
// does not give is nil.
type MonthPrices struct {
	Month       Month
	Balancing   *decimal.Decimal
	CapacityFee *decimal.Decimal
	Deregulated *decimal.Decimal
}
