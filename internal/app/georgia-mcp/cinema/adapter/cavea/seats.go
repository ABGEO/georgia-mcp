package cavea

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	caveaclient "github.com/abgeo/georgia-mcp/pkg/cavea"
)

// Occupancy returns the level of each session that Cavea knows, with one call for each
// caveaclient.MaxAvailabilityIDs codes. "green" is many seats, "yellow" is filling up,
// and any other value is almost sold out, as the page of Cavea shows them.
func (a *Adapter) Occupancy(
	ctx context.Context,
	codes []domain.SessionCode,
) (map[domain.SessionCode]domain.OccupancyLevel, time.Time, error) {
	byText := make(map[string]domain.SessionCode, len(codes))
	levels := make(map[domain.SessionCode]domain.OccupancyLevel, len(codes))

	for start := 0; start < len(codes); start += caveaclient.MaxAvailabilityIDs {
		batch := codes[start:min(start+caveaclient.MaxAvailabilityIDs, len(codes))]

		ids := make([]string, 0, len(batch))
		for _, code := range batch {
			byText[code.String()] = code
			ids = append(ids, code.String())
		}

		colors, err := a.client.Availability(ctx, ids)
		if err != nil {
			return nil, time.Time{}, unavailable("reading the occupancy of Cavea", err)
		}

		for id, color := range colors {
			code, asked := byText[id]
			if !asked || color == "" {
				continue
			}

			levels[code] = levelOf(color)
		}
	}

	return levels, a.readAt(), nil
}

func levelOf(color string) domain.OccupancyLevel {
	switch color {
	case "green":
		return domain.ManySeats
	case "yellow":
		return domain.FillingUp
	default:
		return domain.AlmostSoldOut
	}
}

// SeatMap returns the seats of the hall of one session, in the order of Cavea. A seat of
// a class that the context does not know fails the whole map.
func (a *Adapter) SeatMap(
	ctx context.Context,
	code domain.SessionCode,
) ([]domain.Seat, time.Time, error) {
	found, err := a.client.SeatMap(ctx, code.String())
	if err != nil {
		return nil, time.Time{}, unavailable("reading the seats of Cavea", err)
	}

	seats := make([]domain.Seat, 0, len(found))

	for _, seat := range found {
		class, err := classOf(seat.Class)
		if err != nil {
			return nil, time.Time{}, unavailable("reading the seats of Cavea", err)
		}

		seats = append(seats, domain.Seat{
			Row:          seat.Row,
			Number:       seat.Seat,
			Class:        class,
			SpecialNeeds: seat.SpecialNeeds,
			Taken:        seat.IsTaken,
		})
	}

	return seats, a.readAt(), nil
}
