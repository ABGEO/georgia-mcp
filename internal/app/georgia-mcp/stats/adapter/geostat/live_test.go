//go:build live

package geostat_test

import (
	"testing"
	"time"

	geostatadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/geostat"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
)

// GEOSTAT-SC-026: Against Geostat, the table of salaries has the dimension Region, and
// the indices of 2026 have a published annual index for the country, the total, and
// January. The adapter finds each region, group, and month by its English text.
func TestLiveAdapter(t *testing.T) {
	t.Parallel()

	adapter := geostatadapter.NewAdapter(geostatclient.New(), time.Now, geostatadapter.Pause)

	wages, err := adapter.Table(t.Context(), code(t, wagesRegion), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	if first := wages.Table.Dimensions[0].Code; first != "Region" {
		t.Errorf("first dimension = %q, want Region", first)
	}

	indices, err := adapter.PriceIndices(t.Context(), 2026)
	if err != nil {
		t.Fatalf("PriceIndices() error = %v", err)
	}

	january := app.IndexKey{Region: domain.RegionGeorgia, Group: domain.GroupTotal, Month: 1}
	if !indices.Annual[january].Published() {
		t.Errorf("Annual of January = %q, want a published index", indices.Annual[january])
	}
}
