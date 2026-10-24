package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func toOutput(codes app.Codes) codesOutput {
	addresses := make([]addressOutput, 0, len(codes.Addresses))
	for _, address := range codes.Addresses {
		addresses = append(addresses, addressOutput{
			PostalCode: address.Code.String(),
			Address:    address.Text,
			District:   address.District,
		})
	}

	out := codesOutput{
		Source:    source,
		AsOf:      instant(codes.ReadAt),
		Stale:     codes.Stale,
		Total:     codes.Total,
		Addresses: addresses,
	}
	if len(addresses) == 0 {
		out.Note = emptyNote
	}

	return out
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
