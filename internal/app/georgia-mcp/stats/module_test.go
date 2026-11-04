package stats_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

// failingGeostat answers each call with a 500, in memory, and counts the calls.
type failingGeostat struct {
	calls atomic.Int32
}

func (f *failingGeostat) RoundTrip(request *http.Request) (*http.Response, error) {
	f.calls.Add(1)

	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
		Request:    request,
	}, nil
}

func build(t *testing.T, enabled bool, source *failingGeostat) *stats.Module {
	t.Helper()

	base, err := url.Parse("https://pc-axis.geostat.ge/PXWeb/api/v1")
	if err != nil {
		t.Fatal(err)
	}

	cfg := stats.Config{Geostat: vendors.Config{
		Enabled: enabled,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}

	module, err := stats.NewWithTransport(cfg, slog.New(slog.DiscardHandler), source)
	if err != nil {
		t.Fatal(err)
	}

	return module
}

func toolNames(t *testing.T, module *stats.Module) []string {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := module.Register(registry); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(tools.Tools))

	for _, listed := range tools.Tools {
		if listed.Annotations == nil || !listed.Annotations.ReadOnlyHint {
			t.Errorf("%s is not read-only", listed.Name)
		}

		names = append(names, listed.Name)
	}

	slices.Sort(names)

	return names
}

// GEOSTAT-SC-025: The switch decides the tools: none with Geostat off, the four
// read-only tools with Geostat on.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
		want    []string
	}{
		{name: "off", want: []string{}},
		{name: "on", enabled: true, want: []string{
			"stats_get_cells", "stats_get_inflation", "stats_get_table", "stats_search_tables",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			module := build(t, tt.enabled, &failingGeostat{})
			t.Cleanup(func() { _ = module.Close() })

			if diff := cmp.Diff(tt.want, toolNames(t, module)); diff != "" {
				t.Errorf("tools mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// GEOSTAT-SC-025: With Geostat off, the module starts no read.
func TestGeostatOffSendsNoCall(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &failingGeostat{}
		module := build(t, false, source)

		time.Sleep(time.Hour)
		synctest.Wait()

		if err := module.Close(); err != nil || source.calls.Load() != 0 {
			t.Errorf("Close() = %v with %d calls, want no call", err, source.calls.Load())
		}
	})
}

// GEOSTAT-SC-025: With Geostat on, the module reads the catalog at once, and Close stops
// the job: no call follows it.
func TestCloseStopsTheJob(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &failingGeostat{}
		module := build(t, true, source)

		synctest.Wait()

		if source.calls.Load() == 0 {
			t.Fatal("the job sent no call to Geostat")
		}

		if err := module.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		after := source.calls.Load()

		time.Sleep(30 * 24 * time.Hour)
		synctest.Wait()

		if got := source.calls.Load(); got != after {
			t.Errorf("calls after Close = %d, want %d", got, after)
		}
	})
}
