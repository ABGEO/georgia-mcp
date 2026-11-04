package nea_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	neaadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/nea"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	neaclient "github.com/abgeo/georgia-mcp/pkg/nea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/nea/testdata/"
	tsrt     = "TSRT"
)

func clock() time.Time {
	return time.Date(2026, time.November, 5, 0, 28, 0, 0, kernel.TbilisiZone())
}

// serve answers the hourly method with the file and the municipalities with their
// recorded answer, and records each query of the hourly method.
func serve(t *testing.T, hourly string, status int) (*neaadapter.Adapter, *[]url.Values) {
	t.Helper()

	read := func(name string) []byte {
		body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	stations, municipalities := read(hourly), read("municipalities.json")

	var queries []url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)

		if strings.Contains(r.URL.Path, "municipalities") {
			_, _ = w.Write(municipalities)

			return
		}

		queries = append(queries, r.URL.Query())
		_, _ = w.Write(stations)
	}))
	t.Cleanup(server.Close)

	client := neaclient.New(vendorhttp.WithBaseURL(server.URL))

	return neaadapter.NewAdapter(client, clock), &queries
}

func code(t *testing.T, text string) domain.StationCode {
	t.Helper()

	parsed, err := domain.ParseStationCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// NEAAIR-SC-008: Current reads every station from 24 hours before the read hour to 24
// hours after it, and gives each time in Tbilisi.
func TestCurrent(t *testing.T) {
	t.Parallel()

	adapter, queries := serve(t, "hourly-all.json", http.StatusOK)

	network, err := adapter.Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	query := (*queries)[0]
	if query.Get("from_date_time") != "2026-11-04T00:00:00" ||
		query.Get("to_date_time") != "2026-11-06T00:00:00" || query.Get("station_code") != "all" {
		t.Errorf("query = %v", query)
	}

	if len(network.Stations) != 21 || !network.ReadAt.Equal(clock()) {
		t.Fatalf("network = %d stations at %v", len(network.Stations), network.ReadAt)
	}

	checkTSRT(t, network.Stations)
}

// find returns the station of the code, or fails the test.
func find(t *testing.T, stations []domain.Station, text string) domain.Station {
	t.Helper()

	index := slices.IndexFunc(stations, func(s domain.Station) bool {
		return s.Code.String() == text
	})
	if index < 0 {
		t.Fatalf("no station %s", text)
	}

	return stations[index]
}

// checkTSRT checks the station TSRT of the recorded answer.
func checkTSRT(t *testing.T, stations []domain.Station) {
	t.Helper()

	tsrt := find(t, stations, tsrt)

	if tsrt.CityID != 51 || tsrt.Address.English != "Tbilisi - Akaki Tsereteli ave. №105" ||
		tsrt.Address.Georgian != "ქ.თბილისი - წერეთლის გამზ., №105" {
		t.Errorf("TSRT = %+v", tsrt)
	}

	pollutants := make([]domain.Pollutant, 0, len(tsrt.Series))
	for _, series := range tsrt.Series {
		pollutants = append(pollutants, series.Pollutant)
	}

	if diff := cmp.Diff(domain.Pollutants(), pollutants); diff != "" {
		t.Errorf("pollutants mismatch (-want +got):\n%s", diff)
	}

	pm10 := tsrt.Series[0].Readings
	last := pm10[len(pm10)-1]

	want := time.Date(2026, time.November, 5, 5, 0, 0, 0, kernel.TbilisiZone())
	if last.Value.String() != "51.8833333333333" || !last.Time.Equal(want) ||
		last.Time.Location().String() != "Asia/Tbilisi" || last.Level != domain.LevelPoor {
		t.Errorf("last PM10 = %s at %v, %v", last.Value, last.Time, last.Level)
	}
}

// NEAAIR-SC-008: Recent asks for one station over the window of Current.
func TestRecent(t *testing.T) {
	t.Parallel()

	adapter, queries := serve(t, "hourly-tsrt-week.json", http.StatusOK)

	network, err := adapter.Recent(t.Context(), code(t, tsrt))
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}

	query := (*queries)[0]
	if query.Get("from_date_time") != "2026-11-04T00:00:00" ||
		query.Get("to_date_time") != "2026-11-06T00:00:00" || query.Get("station_code") != tsrt {
		t.Errorf("query = %v", query)
	}

	if len(network.Stations) != 1 {
		t.Errorf("network = %d stations, want 1", len(network.Stations))
	}

	empty, _ := serve(t, "hourly-empty.json", http.StatusOK)

	none, err := empty.Recent(t.Context(), code(t, "XXXX"))
	if err != nil || len(none.Stations) != 0 {
		t.Errorf("Recent(XXXX) = %d stations, %v, want none", len(none.Stations), err)
	}
}

// NEAAIR-SC-008: History asks for one station between two hours.
func TestHistory(t *testing.T) {
	t.Parallel()

	adapter, queries := serve(t, "hourly-tsrt-week.json", http.StatusOK)

	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, kernel.TbilisiZone())

	network, err := adapter.History(t.Context(), code(t, tsrt), from, from.Add(95*time.Hour))
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}

	query := (*queries)[0]
	if query.Get("from_date_time") != "2026-11-01T00:00:00" ||
		query.Get("to_date_time") != "2026-11-04T23:00:00" || query.Get("station_code") != tsrt {
		t.Errorf("query = %v", query)
	}

	if len(network.Stations) != 1 || len(network.Stations[0].Series) != 6 {
		t.Errorf("network = %+v", network.Stations)
	}

	empty, _ := serve(t, "hourly-empty.json", http.StatusOK)

	none, err := empty.History(t.Context(), code(t, "XXXX"), from, from)
	if err != nil || len(none.Stations) != 0 {
		t.Errorf("History(XXXX) = %d stations, %v, want none", len(none.Stations), err)
	}
}

// NEAAIR-SC-008: A broken time, broken bands, an answer with no station, and a client
// error each wrap kernel.ErrSourceUnavailable.
func TestFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, file string
		status     int
	}{
		{name: "time", file: "hourly-bad-time.json", status: http.StatusOK},
		{name: "bands", file: "hourly-bad-bands.json", status: http.StatusOK},
		{name: "empty", file: "hourly-empty.json", status: http.StatusOK},
		{name: "status", file: "hourly-empty.json", status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := serve(t, tt.file, tt.status)

			if _, err := adapter.Current(
				t.Context(),
			); !errors.Is(
				err,
				kernel.ErrSourceUnavailable,
			) {
				t.Errorf("Current() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}

	adapter, _ := serve(t, "hourly-empty.json", http.StatusInternalServerError)
	if _, err := adapter.Cities(t.Context()); !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Cities() error = %v, want ErrSourceUnavailable", err)
	}
}

// NEAAIR-SC-008: A pollutant outside the six is left out.
func TestSkipsAnUnknownPollutant(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(testdata + "hourly-tsrt-week.json")
	if err != nil {
		t.Fatal(err)
	}

	renamed := strings.Replace(string(body), `"name":"PM10"`, `"name":"NO"`, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(renamed))
	}))
	t.Cleanup(server.Close)

	adapter := neaadapter.NewAdapter(neaclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	network, err := adapter.Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if series := network.Stations[0].Series; len(series) != 5 ||
		series[0].Pollutant != domain.PM25 {
		t.Errorf("series = %d, first %q, want 5 from PM2.5", len(series), series[0].Pollutant)
	}
}

// reversed returns the recorded week of TSRT with each series in reverse order.
func reversed(t *testing.T) []byte {
	t.Helper()

	body, err := os.ReadFile(testdata + "hourly-tsrt-week.json")
	if err != nil {
		t.Fatal(err)
	}

	var answer []map[string]any
	if err := json.Unmarshal(body, &answer); err != nil {
		t.Fatal(err)
	}

	set, ok := answer[0]["stationequipment_set"].([]any)
	if !ok {
		t.Fatal("no stationequipment_set")
	}

	for _, item := range set {
		equipment, ok := item.(map[string]any)
		if !ok {
			t.Fatal("an equipment is not an object")
		}

		values, ok := equipment["data1hour_set"].([]any)
		if !ok {
			t.Fatal("no data1hour_set")
		}

		slices.Reverse(values)
	}

	body, err = json.Marshal(answer)
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// NEAAIR-SC-008: A series out of order comes back sorted by time.
func TestSortsEachSeries(t *testing.T) {
	t.Parallel()

	body := reversed(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	adapter := neaadapter.NewAdapter(neaclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	network, err := adapter.History(t.Context(), code(t, tsrt), clock(), clock())
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}

	for _, series := range network.Stations[0].Series {
		if !slices.IsSortedFunc(series.Readings, func(a, b domain.Reading) int {
			return a.Time.Compare(b.Time)
		}) {
			t.Errorf("%s readings out of order", series.Pollutant)
		}
	}
}

// NEAAIR-SC-002: Cities gives each municipality in Georgian and English.
func TestCities(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "hourly-empty.json", http.StatusOK)

	list, err := adapter.Cities(t.Context())
	if err != nil {
		t.Fatalf("Cities() error = %v", err)
	}

	index := slices.IndexFunc(list.Cities, func(c domain.City) bool { return c.ID == 51 })
	if len(list.Cities) != 73 || index < 0 ||
		list.Cities[index].Name != (domain.Name{Georgian: "თბილისი", English: "Tbilisi"}) {
		t.Errorf("cities = %d, Tbilisi at %d", len(list.Cities), index)
	}

	if !list.ReadAt.Equal(clock()) {
		t.Errorf("ReadAt = %v", list.ReadAt)
	}
}
