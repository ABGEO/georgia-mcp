//go:build live

package nea_test

import (
	"slices"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/nea"
)

// NEAAIR-SC-026: The hourly values of the last 24 hours hold the station TSRT with PM10,
// and the municipalities hold Tbilisi.
func TestLiveHourly(t *testing.T) {
	t.Parallel()

	client := nea.New()

	zone, err := time.LoadLocation("Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().In(zone)

	stations, err := client.Hourly(t.Context(), nea.HourlyQuery{
		From: now.Add(-24 * time.Hour).Format("2006-01-02T15:00:00"),
		To:   now.Add(24 * time.Hour).Format("2006-01-02T15:00:00"),
	})
	if err != nil {
		t.Fatalf("Hourly() error = %v", err)
	}

	index := slices.IndexFunc(stations, func(s nea.Station) bool { return s.Code == "TSRT" })
	if index < 0 {
		t.Fatal("no station TSRT")
	}

	if !slices.ContainsFunc(stations[index].Equipment, func(e nea.Equipment) bool {
		return e.Substance.Name == "PM10"
	}) {
		t.Error("TSRT has no PM10")
	}

	municipalities, err := client.Municipalities(t.Context())
	if err != nil {
		t.Fatalf("Municipalities() error = %v", err)
	}

	if !slices.ContainsFunc(municipalities, func(m nea.Municipality) bool {
		return m.ID == 51 && m.DistrictEN == "Tbilisi"
	}) {
		t.Error("no municipality 51 named Tbilisi")
	}
}
