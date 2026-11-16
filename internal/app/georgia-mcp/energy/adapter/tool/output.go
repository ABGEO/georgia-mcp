package tool

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func toGridOutput(read app.GridRead) gridOutput {
	state := read.State
	net := state.NetImport()
	generation := state.Generation

	out := gridOutput{
		Source:         source,
		AsOf:           instant(read.ReadAt),
		Stale:          read.Stale,
		Time:           instant(state.Slot),
		AverageMinutes: int(domain.SlotLength / time.Minute),
		Generation: generationOutput{
			Hydro:   toValue(generation.Hydro),
			Thermal: toValue(generation.Thermal),
			Wind:    toValue(generation.Wind),
			Solar:   toValue(generation.Solar),
			Total:   toValue(generation.Total),
		},
		Consumption: toValue(state.Consumption),
		Flows:       make([]flowOutput, 0, len(state.Flows)),
		NetImport:   netOutput{Value: text(net), Direction: string(domain.DirectionOf(net))},
		Measured:    state.Measured(),
		Values:      len(state.Values()),
	}

	for _, flow := range state.Flows {
		out.Flows = append(out.Flows, toFlow(flow))
	}

	if read.Frequency != nil {
		value := read.Frequency.Value
		out.Frequency = &frequencyOutput{
			Value:   text(value.Number),
			Time:    instant(read.Frequency.Minute),
			Quality: string(value.Quality),
			Meaning: value.Quality.Meaning(),
		}
	}

	return out
}

func toValue(value domain.Value) valueOutput {
	return valueOutput{
		Value:   text(value.Number),
		Quality: string(value.Quality),
		Meaning: value.Quality.Meaning(),
	}
}

// toFlow gives a country with one line the value and the quality of that line, and a
// country with more lines their sum and each line.
func toFlow(flow domain.Flow) flowOutput {
	net := flow.Net()
	out := flowOutput{
		Country:   flow.Country,
		Value:     text(net),
		Direction: string(domain.DirectionOf(net)),
	}

	if len(flow.Lines) == 1 {
		quality := flow.Lines[0].Value.Quality
		out.Quality, out.Meaning = string(quality), quality.Meaning()

		return out
	}

	for _, line := range flow.Lines {
		number := line.Value.Number
		out.Lines = append(out.Lines, lineOutput{
			Name:      line.Name,
			Value:     text(number),
			Direction: string(domain.DirectionOf(number)),
			Quality:   string(line.Value.Quality),
			Meaning:   line.Value.Quality.Meaning(),
		})
	}

	return out
}

// text keeps the decimals that GSE publishes, which String would cut: 0.000000.
func text(number decimal.Decimal) string {
	if number.Exponent() < 0 {
		return number.StringFixed(-number.Exponent())
	}

	return number.String()
}

func instant(t time.Time) string {
	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
