package gse_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	gseadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/gse"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gseclient "github.com/abgeo/georgia-mcp/pkg/gse"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata  = "../../../../../../pkg/gse/testdata/"
	gridFile  = "grid-16112026-1200.xml"
	frequency = "frequency-16112026-1200.xml"
)

type answer struct {
	status int
	file   string
}

// at returns a time of 2026-11-16 in the hour 12 of Tbilisi.
func at(minute, second int) time.Time {
	return time.Date(2026, time.November, 16, 12, minute, second, 0, kernel.TbilisiZone())
}

func clock() time.Time {
	return at(1, 30)
}

func grid(stamp string) string {
	return "GSE_EXPORT_NEW_PERIOD_03M_16112026_" + stamp + ".XML"
}

func minute(stamp string) string {
	return "GSE_EXPORT_PERIOD_01M_16112026_" + stamp + ".XML"
}

// serve answers each file name of files, and any other with the 404 page of nginx. It
// records the name of each file that the adapter asks.
func serve(t *testing.T, files map[string]answer) (*gseadapter.Adapter, *[]string) {
	t.Helper()

	read := func(name string) []byte {
		body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	notFound := read("not-found.html")
	bodies := map[string][]byte{}

	for name, a := range files {
		bodies[name] = read(a.file)
	}

	var asked []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Base(r.URL.Path)
		asked = append(asked, name)

		reply, ok := files[name]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(notFound)

			return
		}

		w.WriteHeader(reply.status)
		_, _ = w.Write(bodies[name])
	}))
	t.Cleanup(server.Close)

	client := gseclient.New(vendorhttp.WithBaseURL(server.URL))

	return gseadapter.NewAdapter(client, clock), &asked
}

func value(number string, quality domain.Quality) domain.Value {
	return domain.Value{Number: decimal.RequireFromString(number), Quality: quality}
}

func line(name, number string, quality domain.Quality) domain.Line {
	return domain.Line{Name: name, Value: value(number, quality)}
}

func recordedState() domain.GridState {
	return domain.GridState{
		Slot: at(0, 0),
		Generation: domain.Generation{
			Hydro:   value("1331.362305", domain.QualitySubstituted),
			Thermal: value("170.777710", domain.QualitySubstituted),
			Wind:    value("-0.036000", domain.QualityNotReceived),
			Solar:   value("6.000000", domain.QualitySubstituted),
			Total:   value("1511.088379", domain.QualityNotReceived),
		},
		Consumption: value("1843.806030", domain.QualityNotReceived),
		Flows: []domain.Flow{
			{Country: "Azerbaijan", Lines: []domain.Line{
				line("Azerbaij", "20.631090", domain.QualityMeasured),
			}},
			{Country: "Armenia", Lines: []domain.Line{
				line("Armenia", "0.000000", domain.QualitySubstituted),
			}},
			{Country: "Russia", Lines: []domain.Line{
				line("Russia", "0.000000", domain.QualityMeasured),
				line("Salkhino", "-0.000000", domain.QualityMeasured),
				line("Java", "18.299999", domain.QualityBLS),
				line("Nakaduli", "14.191387", domain.QualityMeasured),
			}},
			{Country: "Turkey", Lines: []domain.Line{
				line("Turkey", "279.235687", domain.QualityMeasured),
			}},
		},
	}
}

func options() cmp.Options {
	return cmp.Options{
		cmp.Comparer(func(a, b decimal.Decimal) bool {
			return a.Equal(b) && a.Exponent() == b.Exponent()
		}),
		cmp.Comparer(func(a, b time.Time) bool {
			return a.Equal(b) && a.Location().String() == b.Location().String()
		}),
	}
}

// GSEGRID-SC-009: The adapter maps the grid file to the grid state, leaves out a block of
// another name, and fails a file it cannot read.
func TestAdapterMapsTheGrid(t *testing.T) {
	t.Parallel()

	for _, file := range []string{gridFile, "grid-extra-block.xml"} {
		adapter, _ := serve(t, map[string]answer{grid("1200"): {http.StatusOK, file}})

		read, err := adapter.At(t.Context(), at(0, 0))
		if err != nil {
			t.Fatalf("At() with %s error = %v", file, err)
		}

		if diff := cmp.Diff(recordedState(), read.State, options()); diff != "" {
			t.Errorf("At() with %s state mismatch (-want +got):\n%s", file, diff)
		}

		if !read.ReadAt.Equal(clock()) ||
			read.ReadAt.Location().String() != kernel.TbilisiZone().String() {
			t.Errorf("ReadAt = %v, want %v in Tbilisi", read.ReadAt, clock())
		}
	}
}

// GSEGRID-SC-009: A file the adapter cannot read is an outage of GSE.
func TestAdapterRefusesABrokenGrid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files map[string]answer
		slot  time.Time
	}{
		{
			name:  "bad value",
			files: map[string]answer{grid("1200"): {http.StatusOK, "grid-bad-value.xml"}},
			slot:  at(0, 0),
		},
		{
			name:  "no consumption",
			files: map[string]answer{grid("1200"): {http.StatusOK, "grid-no-consumption.xml"}},
			slot:  at(0, 0),
		},
		{
			name:  "wrong time",
			files: map[string]answer{grid("1203"): {http.StatusOK, gridFile}},
			slot:  at(3, 0),
		},
		{
			name:  "frequency as grid",
			files: map[string]answer{grid("1200"): {http.StatusOK, frequency}},
			slot:  at(0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := serve(t, tt.files)

			_, err := adapter.At(t.Context(), tt.slot)
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("At() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}

// GSEGRID-SC-010: A 404 moves the search to the slot before. No file in 30 minutes gives
// ErrNoGrid. Another failure ends the search.
func TestAdapterSearchesBack(t *testing.T) {
	t.Parallel()

	adapter, asked := serve(t, map[string]answer{grid("1200"): {http.StatusOK, gridFile}})

	read, err := adapter.At(t.Context(), at(3, 0))
	if err != nil {
		t.Fatalf("At() error = %v", err)
	}

	if !read.State.Slot.Equal(at(0, 0)) {
		t.Errorf("Slot = %v, want 12:00", read.State.Slot)
	}

	if diff := cmp.Diff([]string{grid("1203"), grid("1200"), minute("1200")}, *asked); diff != "" {
		t.Errorf("asked mismatch (-want +got):\n%s", diff)
	}

	adapter, asked = serve(t, nil)

	if _, err := adapter.At(t.Context(), at(3, 0)); !errors.Is(err, app.ErrNoGrid) {
		t.Errorf("At() with no file error = %v, want app.ErrNoGrid", err)
	}

	if len(*asked) != 11 || (*asked)[10] != grid("1133") {
		t.Errorf("asked = %v, want 11 grid files down to 11:33", *asked)
	}

	adapter, asked = serve(t, map[string]answer{
		grid("1203"): {http.StatusInternalServerError, "not-found.html"},
		grid("1200"): {http.StatusOK, gridFile},
	})

	if _, err := adapter.At(
		t.Context(),
		at(3, 0),
	); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("At() with a 500 error = %v, want kernel.ErrSourceUnavailable", err)
	}

	if len(*asked) != 1 {
		t.Errorf("asked = %v, want 1 call", *asked)
	}
}

// GSEGRID-SC-010: Newest searches from the slot of the read.
func TestAdapterNewestStartsAtTheSlotOfNow(t *testing.T) {
	t.Parallel()

	adapter, asked := serve(t, map[string]answer{grid("1200"): {http.StatusOK, gridFile}})

	read, err := adapter.Newest(t.Context())
	if err != nil {
		t.Fatalf("Newest() error = %v", err)
	}

	if !read.State.Slot.Equal(at(0, 0)) || (*asked)[0] != grid("1200") {
		t.Errorf("Slot = %v, asked = %v, want 12:00 first", read.State.Slot, *asked)
	}
}

// GSEGRID-SC-011: The adapter reads the frequency of the minute of the slot. A failure of
// that read gives no frequency and keeps the grid state.
func TestAdapterReadsTheFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		answer *answer
		want   *domain.Frequency
	}{
		{
			name:   "found",
			answer: &answer{http.StatusOK, frequency},
			want: &domain.Frequency{
				Minute: at(0, 0), Value: value("49.990002", domain.QualityMeasured),
			},
		},
		{name: "404"},
		{name: "500", answer: &answer{http.StatusInternalServerError, "not-found.html"}},
		{name: "grid as frequency", answer: &answer{http.StatusOK, gridFile}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]answer{grid("1200"): {http.StatusOK, gridFile}}
			if tt.answer != nil {
				files[minute("1200")] = *tt.answer
			}

			adapter, _ := serve(t, files)

			read, err := adapter.At(t.Context(), at(0, 0))
			if err != nil {
				t.Fatalf("At() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, read.Frequency, options()); diff != "" {
				t.Errorf("Frequency mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
