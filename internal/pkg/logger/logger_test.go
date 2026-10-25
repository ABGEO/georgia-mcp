package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/logger"
)

const (
	formatJSON = "json"
	formatText = "text"
)

// PLATFORM-SC-038: Every record names the service and its version.
func TestNewAddsTheService(t *testing.T) {
	t.Parallel()

	info := buildinfo.Info{Version: "v0.1.0"}
	want := map[string]string{"service.name": "georgia-mcp", "service.version": "v0.1.0"}

	record := func(l *slog.Logger) { l.Info("an event") }
	child := func(l *slog.Logger) { l.With(slog.String("component", "context")).Info("an event") }

	tests := []struct {
		name   string
		format string
		log    func(*slog.Logger)
	}{
		{name: "a json record", format: formatJSON, log: record},
		{name: "a text record", format: formatText, log: record},
		{name: "a json record of a child", format: formatJSON, log: child},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer

			cfg := logger.Config{Level: slog.LevelInfo, Format: test.format}
			test.log(logger.New(cfg, &output, info))

			got := serviceAttributes(t, test.format, output.String())
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("the service attributes (-want +got):\n%s", diff)
			}
		})
	}
}

func serviceAttributes(t *testing.T, format, record string) map[string]string {
	t.Helper()

	got := map[string]string{}

	if format == formatText {
		for field := range strings.FieldsSeq(record) {
			key, value, _ := strings.Cut(field, "=")
			if strings.HasPrefix(key, "service.") {
				got[key] = value
			}
		}

		return got
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(record), &fields); err != nil {
		t.Fatalf("parsing the record %q: %v", record, err)
	}

	for key, value := range fields {
		if strings.HasPrefix(key, "service.") {
			got[key], _ = value.(string)
		}
	}

	return got
}
