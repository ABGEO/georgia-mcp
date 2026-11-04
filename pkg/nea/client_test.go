package nea_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/nea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	from      = "2026-11-04T00:00:00"
	all       = "all"
	station   = "TSRT"
	substance = "PM10"
)

type recorded struct {
	path   string
	query  url.Values
	header http.Header
}

// serve answers each request with the file and the status, and records the request.
func serve(t *testing.T, file string, status int) (*nea.Client, *[]recorded) {
	t.Helper()

	body := []byte("not json")

	if file != "" {
		read, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		body = read
	}

	var requests []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, recorded{
			path: r.URL.Path, query: r.URL.Query(), header: r.Header.Clone(),
		})

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return nea.New(vendorhttp.WithBaseURL(server.URL)), &requests
}

// NEAAIR-SC-001: Hourly sends the eight parameters, "all" for each empty field, and the
// User-Agent of the server.
func TestHourlySendsTheQuery(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "hourly-empty.json", http.StatusOK)

	_, err := client.Hourly(t.Context(), nea.HourlyQuery{
		From: from, To: "2026-11-05T13:00:00", Substance: substance,
	})
	if err != nil {
		t.Fatalf("Hourly() error = %v", err)
	}

	if _, err := client.Municipalities(t.Context()); err != nil {
		t.Fatalf("Municipalities() error = %v", err)
	}

	got := (*requests)[0]
	if got.path != "/api/get_data_1hour/" {
		t.Errorf("path = %q, want /api/get_data_1hour/", got.path)
	}

	want := url.Values{
		"from_date_time":  {from},
		"to_date_time":    {"2026-11-05T13:00:00"},
		"station_code":    {all},
		"municipality_id": {all},
		"substance":       {"PM10"},
		"last_data":       {"false"},
		"chart":           {"false"},
		"format":          {"json"},
	}
	if !reflect.DeepEqual(got.query, want) {
		t.Errorf("query = %v, want %v", got.query, want)
	}

	if (*requests)[1].path != "/api/municipalities/" {
		t.Errorf("path = %q, want /api/municipalities/", (*requests)[1].path)
	}

	for _, request := range *requests {
		agent := request.header.Get("User-Agent")
		if !strings.HasPrefix(agent, "georgia-mcp/") ||
			!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
			t.Errorf("User-Agent = %q, want the agent of Georgia MCP", agent)
		}

		if request.header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", request.header.Get("Accept"))
		}
	}
}

// NEAAIR-SC-001: A station code and a municipality pass unchanged.
func TestHourlySendsTheStation(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "hourly-empty.json", http.StatusOK)

	_, err := client.Hourly(t.Context(), nea.HourlyQuery{
		From: "a", To: "b", Station: station, Municipality: "51",
	})
	if err != nil {
		t.Fatalf("Hourly() error = %v", err)
	}

	query := (*requests)[0].query
	if query.Get("station_code") != station || query.Get("municipality_id") != "51" ||
		query.Get("substance") != all {
		t.Errorf("query = %v, want TSRT, 51, and all", query)
	}
}

// NEAAIR-SC-002: The recorded answer gives 21 stations, KUTS first.
func TestHourlyReadsTheStations(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "hourly-all.json", http.StatusOK)

	stations, err := client.Hourly(t.Context(), nea.HourlyQuery{})
	if err != nil {
		t.Fatalf("Hourly() error = %v", err)
	}

	if len(stations) != 21 {
		t.Fatalf("len(stations) = %d, want 21", len(stations))
	}

	kutaisi := stations[0]
	head := kutaisi
	head.Equipment = nil

	wantHead := nea.Station{
		Code:           "KUTS",
		MunicipalityID: 52,
		AddressGE:      "ქ.ქუთაისი - ლადო ასათიანის ქ., №98",
		AddressEN:      "Kutaisi - Lado Asatiani St.№98",
		Lat:            42.249081,
		Long:           42.693868,
	}
	if !reflect.DeepEqual(head, wantHead) {
		t.Errorf("first station = %+v, want %+v", head, wantHead)
	}

	pm10 := kutaisi.Equipment[0]
	if pm10.Substance.Name != substance {
		t.Errorf("first substance = %q, want PM10", pm10.Substance.Name)
	}

	wantLevel := nea.IndexLevel{GoodTo: "20.0", FairTo: "35.0", ModerateTo: "50.0", PoorTo: "100.0"}
	if !reflect.DeepEqual(pm10.Substance.Levels, []nea.IndexLevel{wantLevel}) {
		t.Errorf("levels = %+v, want %+v", pm10.Substance.Levels, wantLevel)
	}

	want := nea.HourlyValue{Value: "36.6176862651041", DateTime: from}
	if pm10.Values[0] != want {
		t.Errorf("first value = %+v, want %+v", pm10.Values[0], want)
	}
}

// NEAAIR-SC-002: The recorded answer gives 73 municipalities, with Tbilisi as 51.
func TestMunicipalities(t *testing.T) {
	t.Parallel()

	cities, _ := serve(t, "municipalities.json", http.StatusOK)

	municipalities, err := cities.Municipalities(t.Context())
	if err != nil {
		t.Fatalf("Municipalities() error = %v", err)
	}

	if len(municipalities) != 73 {
		t.Fatalf("len(municipalities) = %d, want 73", len(municipalities))
	}

	for _, municipality := range municipalities {
		if municipality.ID == 51 &&
			(municipality.DistrictEN != "Tbilisi" || municipality.DistrictGE != "თბილისი") {
			t.Errorf("municipality 51 = %+v, want Tbilisi", municipality)
		}
	}
}

// NEAAIR-SC-003: A status that is not 200 is an APIError, and a body that is not JSON
// is an error.
func TestFailures(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "hourly-empty.json", http.StatusInternalServerError)

	_, err := client.Hourly(t.Context(), nea.HourlyQuery{})

	var apiError *nea.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError {
		t.Errorf("Hourly() error = %v, want an APIError with 500", err)
	}

	if !errors.Is(err, nea.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
	}

	broken, _ := serve(t, "", http.StatusOK)

	if _, err := broken.Hourly(t.Context(), nea.HourlyQuery{}); err == nil {
		t.Error("Hourly() with a body that is not JSON gave no error")
	}

	if _, err := broken.Municipalities(t.Context()); err == nil {
		t.Error("Municipalities() with a body that is not JSON gave no error")
	}
}
