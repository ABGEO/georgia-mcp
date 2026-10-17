package mcpserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, err := b.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("writing the log: %w", err)
	}

	return n, nil
}

func (b *syncBuffer) records(t *testing.T) []map[string]any {
	t.Helper()

	b.mu.Lock()
	defer b.mu.Unlock()

	var records []map[string]any

	for line := range strings.Lines(b.buf.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("a log line is not JSON: %q", line)
		}

		records = append(records, record)
	}

	return records
}

func newBufferLogger(level slog.Level) (*slog.Logger, *syncBuffer) {
	buf := &syncBuffer{}

	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})), buf
}

// PLATFORM-SC-017: A tool call writes one record. A tool error adds error.type. A list writes none.
func TestMiddlewareLogsEachToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tool          string
		wantRecords   int
		wantErrorType string
	}{
		{name: "success", tool: echoTool, wantRecords: 1},
		{name: "tool error", tool: failTool, wantRecords: 1, wantErrorType: "tool_error"},
		{name: "list", wantRecords: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, buf := newBufferLogger(slog.LevelInfo)
			session := connectInMemory(t, newRegistry(t, logger, echoModule{}, failModule{}))

			if tt.tool == "" {
				toolNames(t, session)
			} else {
				callToolWith(t, session, tt.tool, "hello")
			}

			records := buf.records(t)
			if len(records) != tt.wantRecords {
				t.Fatalf("got %d records, want %d: %v", len(records), tt.wantRecords, records)
			}

			if tt.wantRecords == 0 {
				return
			}

			assertToolCallRecord(t, records[0], tt.wantErrorType)
		})
	}
}

func assertToolCallRecord(t *testing.T, record map[string]any, wantErrorType string) {
	t.Helper()

	for _, key := range []string{"mcp.method.name", "gen_ai.tool.name", "event.duration"} {
		if _, ok := record[key]; !ok {
			t.Errorf("the record has no %q: %v", key, record)
		}
	}

	if got, _ := record["error.type"].(string); got != wantErrorType {
		t.Errorf("error.type = %q, want %q", got, wantErrorType)
	}
}

// PLATFORM-SC-033: The arguments of a tool call appear in the log at the debug level only.
func TestMiddlewareLogsTheArgumentsAtDebugOnly(t *testing.T) {
	t.Parallel()

	const secret = "secret-value"

	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		logger, buf := newBufferLogger(level)
		session := connectInMemory(t, newRegistry(t, logger, echoModule{}))

		callToolWith(t, session, echoTool, secret)

		records := buf.records(t)
		logged := strings.Contains(buf.buf.String(), secret)

		if level == slog.LevelInfo && logged {
			t.Errorf("an info record holds the arguments: %v", records)
		}

		if level == slog.LevelDebug && !logged {
			t.Errorf("no debug record holds the arguments: %v", records)
		}
	}
}

func callToolWith(t *testing.T, session *mcp.ClientSession, name, text string) {
	t.Helper()

	_, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: name, Arguments: map[string]any{"text": text}},
	)
	if err != nil {
		t.Fatalf("CallTool(%q) error = %v", name, err)
	}
}
