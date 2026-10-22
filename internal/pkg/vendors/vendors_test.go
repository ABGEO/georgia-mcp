package vendors_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

var (
	errBuild = errors.New("the cache cannot start")
	errClose = errors.New("the cache cannot stop")
)

// source is a source that records its close.
type source struct {
	name     string
	closed   bool
	closeErr error
}

func (s *source) Close() error {
	s.closed = true

	return s.closeErr
}

func entry(name string, enabled bool, built *source, err error) vendors.Entry[*source] {
	return vendors.Entry[*source]{
		Name:     name,
		Enabled:  enabled,
		Variable: "GEOMCP_TEST_" + name + "_ENABLED",
		Build: func() (*source, error) {
			if err != nil {
				return nil, err
			}

			return built, nil
		},
	}
}

func namesOf(sources []*source) []string {
	names := make([]string, 0, len(sources))

	for _, built := range sources {
		names = append(names, built.name)
	}

	return names
}

// offRecords returns the message and the variable of each record of the log.
func offRecords(logs *bytes.Buffer) []string {
	var records []string

	for line := range strings.Lines(logs.String()) {
		records = append(records, strings.TrimSpace(line))
	}

	return records
}

// PLATFORM-SC-035: The registry builds each vendor that is on, in order, and logs each
// vendor that is off with its switch.
func TestBuildLeavesOutAVendorThatIsOff(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	registry := vendors.New[*source](slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
		ReplaceAttr: dropTime,
	})))
	registry.Add(entry("A", true, &source{name: "A"}, nil))
	registry.Add(entry("B", false, &source{name: "B"}, nil))
	registry.Add(entry("C", true, &source{name: "C"}, nil))

	built, err := registry.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if diff := cmp.Diff([]string{"A", "C"}, namesOf(built)); diff != "" {
		t.Errorf("sources (-want +got):\n%s", diff)
	}

	want := []string{`level=INFO msg="B is off" variable=GEOMCP_TEST_B_ENABLED`}
	if diff := cmp.Diff(want, offRecords(&logs)); diff != "" {
		t.Errorf("log (-want +got):\n%s", diff)
	}
}

// PLATFORM-SC-035: A vendor that cannot start stops the build, closes each source that
// the registry built, and names the vendor.
func TestBuildClosesTheBuiltSourcesOnAFailure(t *testing.T) {
	t.Parallel()

	first := &source{name: "A"}

	registry := vendors.New[*source](slog.New(slog.DiscardHandler))
	registry.Add(entry("A", true, first, nil))
	registry.Add(entry("B", true, nil, errBuild))

	built, err := registry.Build()
	if !errors.Is(err, errBuild) || !strings.Contains(err.Error(), "B") {
		t.Errorf("Build() error = %v, want one that wraps %v and names B", err, errBuild)
	}

	if built != nil {
		t.Errorf("Build() = %v, want no source", namesOf(built))
	}

	if !first.closed {
		t.Error("A is open, want it closed")
	}
}

// PLATFORM-SC-035: With every vendor off, the registry builds no source and logs each one.
func TestBuildWithEveryVendorOff(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	registry := vendors.New[*source](slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
		ReplaceAttr: dropTime,
	})))
	registry.Add(entry("A", false, &source{name: "A"}, nil))
	registry.Add(entry("B", false, &source{name: "B"}, nil))

	built, err := registry.Build()
	if err != nil || len(built) != 0 {
		t.Errorf("Build() = %v, %v, want no source and no error", namesOf(built), err)
	}

	if records := offRecords(&logs); len(records) != 2 {
		t.Errorf("log = %q, want 2 records", records)
	}
}

// PLATFORM-SC-035: CloseAll closes each source, and returns the failure of each one.
func TestCloseAllClosesEverySource(t *testing.T) {
	t.Parallel()

	failing := &source{name: "A", closeErr: errClose}
	second := &source{name: "B"}

	err := vendors.CloseAll([]*source{failing, second})
	if !errors.Is(err, errClose) {
		t.Errorf("CloseAll() error = %v, want %v", err, errClose)
	}

	if !failing.closed || !second.closed {
		t.Errorf("closed = %v, %v, want both", failing.closed, second.closed)
	}
}

func dropTime(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		return slog.Attr{}
	}

	return attr
}
