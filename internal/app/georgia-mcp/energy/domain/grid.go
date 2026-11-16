package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Line is one line to a country, as GSE names it.
type Line struct {
	Name  string
	Value Value
}

// Flow is the power on the lines between Georgia and one country.
type Flow struct {
	Country string
	// Lines holds one line or more, in the order of GSE.
	Lines []Line
}

// Net returns the sum of the lines.
func (f Flow) Net() decimal.Decimal {
	net := decimal.Zero
	for _, line := range f.Lines {
		net = net.Add(line.Value.Number)
	}

	return net
}

// Generation is the generation of each type of power plant, and the total that GSE gives.
type Generation struct {
	Hydro, Thermal, Wind, Solar, Total Value
}

// GridState is the state of the grid in one slot.
type GridState struct {
	// Slot is the time that GSE stamps, in kernel.TbilisiZone().
	Slot        time.Time
	Generation  Generation
	Consumption Value
	// Flows holds the flow of each country, in the order of GSE.
	Flows []Flow
}

// NetImport returns the sum of every flow. A negative net import is an export.
func (g GridState) NetImport() decimal.Decimal {
	net := decimal.Zero
	for _, flow := range g.Flows {
		net = net.Add(flow.Net())
	}

	return net
}

// Values returns each value that GSE published: the generation, the consumption, and
// each line.
func (g GridState) Values() []Value {
	values := []Value{
		g.Generation.Hydro, g.Generation.Thermal, g.Generation.Wind, g.Generation.Solar,
		g.Generation.Total, g.Consumption,
	}

	for _, flow := range g.Flows {
		for _, line := range flow.Lines {
			values = append(values, line.Value)
		}
	}

	return values
}

// Measured returns the count of the values that are measurements.
func (g GridState) Measured() int {
	count := 0

	for _, value := range g.Values() {
		if value.Quality.Measured() {
			count++
		}
	}

	return count
}

// Frequency is the frequency of the grid in one minute, in Hz.
type Frequency struct {
	Minute time.Time
	Value  Value
}
