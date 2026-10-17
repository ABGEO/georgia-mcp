package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func testConfig() httpserver.Config {
	return httpserver.Config{Address: "127.0.0.1:0", RateLimit: 60, RateWindow: time.Minute}
}

const (
	allowedOrigin = "https://a.example"
	echoTool      = "test_get_echo"
	slowTool      = "test_get_slow"
)

func okHandler() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) },
	)
}

func slowCall() *mcp.CallToolParams {
	return &mcp.CallToolParams{Name: slowTool, Arguments: map[string]any{"text": "wait"}}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func serve(
	t *testing.T,
	handler http.Handler,
	method, path string,
	header http.Header,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader("{}"))
	maps.Copy(req.Header, header)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// PLATFORM-SC-008: The liveness answers 200 while the process runs, ready or not.
func TestLivezAnswersOK(t *testing.T) {
	t.Parallel()

	srv := httpserver.New(testConfig(), discardLogger(), okHandler())

	rec := serve(t, srv.Handler(), http.MethodGet, "/livez", nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("GET /livez = %d %q, want 200 %q", rec.Code, rec.Body.String(), "ok")
	}
}

// PLATFORM-SC-010: A request with an origin outside the list fails. A request with no origin passes.
func TestOriginCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		allowed   []string
		origin    string
		forbidden bool
	}{
		{name: "no origin", allowed: []string{allowedOrigin}},
		{name: "allowed origin", allowed: []string{allowedOrigin}, origin: allowedOrigin},
		{
			name:      "other origin",
			allowed:   []string{allowedOrigin},
			origin:    "https://b.example",
			forbidden: true,
		},
		{name: "empty list with origin", origin: allowedOrigin, forbidden: true},
		{name: "empty list without origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig()
			cfg.AllowedOrigins = tt.allowed
			srv := httpserver.New(cfg, discardLogger(), okHandler())

			header := http.Header{}
			if tt.origin != "" {
				header.Set("Origin", tt.origin)
			}

			rec := serve(t, srv.Handler(), http.MethodPost, "/mcp", header)
			if got := rec.Code == http.StatusForbidden; got != tt.forbidden {
				t.Errorf("POST /mcp = %d, want forbidden %v", rec.Code, tt.forbidden)
			}
		})
	}
}

// PLATFORM-SC-011: The request after the limit answers 429 with Retry-After.
func TestRateLimit(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.RateLimit = 2
	srv := httpserver.New(cfg, discardLogger(), okHandler())

	for i := range 3 {
		rec := serve(t, srv.Handler(), http.MethodPost, "/mcp", nil)

		limited := rec.Code == http.StatusTooManyRequests
		if want := i == 2; limited != want {
			t.Fatalf("request %d = %d, want limited %v", i+1, rec.Code, want)
		}

		if limited && rec.Header().Get("Retry-After") == "" {
			t.Error("the 429 answer has no Retry-After header")
		}
	}
}

// PLATFORM-SC-034: A forged X-Forwarded-For does not change the bucket of the rate limit.
func TestRateLimitIgnoresAForgedHeader(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.RateLimit = 1
	srv := httpserver.New(cfg, discardLogger(), okHandler())

	first := serve(
		t,
		srv.Handler(),
		http.MethodPost,
		"/mcp",
		http.Header{"X-Forwarded-For": {"198.51.100.1"}},
	)
	second := serve(
		t,
		srv.Handler(),
		http.MethodPost,
		"/mcp",
		http.Header{"X-Forwarded-For": {"198.51.100.2"}},
	)

	if first.Code == http.StatusTooManyRequests || second.Code != http.StatusTooManyRequests {
		t.Errorf("statuses = %d, %d, want not 429, then 429", first.Code, second.Code)
	}
}

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

func (b *syncBuffer) count(t *testing.T) int {
	t.Helper()

	b.mu.Lock()
	defer b.mu.Unlock()

	count := 0

	for line := range strings.Lines(b.buf.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("a log line is not JSON: %q", line)
		}

		count++
	}

	return count
}

// PLATFORM-SC-018: Each request writes one record, except a probe that answers 200.
func TestAccessLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantRecords int
	}{
		{
			name:        "mcp",
			method:      http.MethodPost,
			path:        "/mcp",
			wantStatus:  http.StatusOK,
			wantRecords: 1,
		},
		{
			name:        "livez",
			method:      http.MethodGet,
			path:        "/livez",
			wantStatus:  http.StatusOK,
			wantRecords: 0,
		},
		{
			name:        "readyz not ready",
			method:      http.MethodGet,
			path:        "/readyz",
			wantStatus:  http.StatusServiceUnavailable,
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := &syncBuffer{}
			logger := slog.New(
				slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
			)
			srv := httpserver.New(testConfig(), logger, okHandler())

			rec := serve(t, srv.Handler(), tt.method, tt.path, nil)
			if rec.Code != tt.wantStatus {
				t.Fatalf("%s %s = %d, want %d", tt.method, tt.path, rec.Code, tt.wantStatus)
			}

			if got := buf.count(t); got != tt.wantRecords {
				t.Errorf("got %d records, want %d", got, tt.wantRecords)
			}
		})
	}
}

type echoInput struct {
	Text string `json:"text" jsonschema:"the text to return"`
}

type echoOutput struct {
	Text string `json:"text" jsonschema:"the text of the input"`
}

// slowModule registers a tool that tells when it starts, then waits 200 ms.
type slowModule struct{ started chan struct{} }

func (m slowModule) Register(registry *mcpserver.Registry) error {
	return mcpserver.AddTool(
		registry,
		&mcp.Tool{Name: slowTool, Description: "Waits."},
		func(context.Context, *mcp.CallToolRequest, echoInput) (*mcp.CallToolResult, echoOutput, error) {
			close(m.started)
			time.Sleep(200 * time.Millisecond)

			return nil, echoOutput{Text: "done"}, nil
		},
	)
}

type echoModule struct{}

func (echoModule) Register(registry *mcpserver.Registry) error {
	return mcpserver.AddTool(
		registry,
		&mcp.Tool{Name: echoTool, Description: "Returns its input."},
		func(_ context.Context, _ *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, echoOutput, error) {
			return nil, echoOutput(in), nil
		},
	)
}

type running struct {
	srv  *httpserver.Server
	stop context.CancelFunc
	done chan error
}

func start(t *testing.T, module mcpserver.Module) running {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0.1.0"}, discardLogger())
	if err := registry.Register(module); err != nil {
		t.Fatal(err)
	}

	srv := httpserver.New(testConfig(), discardLogger(), registry.Handler())
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() { done <- srv.Run(ctx) }()

	t.Cleanup(stop)

	return running{srv: srv, stop: stop, done: done}
}

func connect(t *testing.T, srv *httpserver.Server) *mcp.ClientSession {
	t.Helper()

	transport := &mcp.StreamableClientTransport{
		Endpoint:             "http://" + srv.Addr().String() + "/mcp",
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// PLATFORM-SC-001: A client over streamable HTTP lists the tool and calls it.
func TestHTTPServesTheTools(t *testing.T) {
	t.Parallel()

	server := start(t, echoModule{})
	session := connect(t, server.srv)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != echoTool {
		t.Fatalf("ListTools() = %v, %v, want test_get_echo", tools, err)
	}

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      echoTool,
		Arguments: map[string]any{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	got, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling the structured content: %v", err)
	}

	if string(got) != `{"text":"hello"}` {
		t.Errorf("structured content = %s, want {\"text\":\"hello\"}", got)
	}
}

// PLATFORM-SC-009: The readiness follows SetReady, and turns off at the shutdown.
func TestReadyz(t *testing.T) {
	t.Parallel()

	module := slowModule{started: make(chan struct{})}
	server := start(t, module)

	if rec := serve(
		t,
		server.srv.Handler(),
		http.MethodGet,
		"/readyz",
		nil,
	); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("before SetReady: GET /readyz = %d, want 503", rec.Code)
	}

	server.srv.SetReady(true)

	if rec := serve(
		t,
		server.srv.Handler(),
		http.MethodGet,
		"/readyz",
		nil,
	); rec.Code != http.StatusOK {
		t.Errorf("after SetReady: GET /readyz = %d, want 200", rec.Code)
	}

	session := connect(t, server.srv)

	go func() { _, _ = session.CallTool(context.Background(), slowCall()) }()

	<-module.started
	server.stop()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if rec := serve(
			t,
			server.srv.Handler(),
			http.MethodGet,
			"/readyz",
			nil,
		); rec.Code == http.StatusServiceUnavailable {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Error("at the shutdown: GET /readyz did not answer 503")
}

// PLATFORM-SC-016: A call in progress ends with its result, and Run returns nil within 10 seconds.
func TestShutdownLetsTheCallEnd(t *testing.T) {
	t.Parallel()

	module := slowModule{started: make(chan struct{})}
	server := start(t, module)
	session := connect(t, server.srv)

	type outcome struct {
		result *mcp.CallToolResult
		err    error
	}

	called := make(chan outcome, 1)

	go func() {
		result, err := session.CallTool(context.Background(), slowCall())
		called <- outcome{result: result, err: err}
	}()

	<-module.started
	server.stop()

	stopped := time.Now()

	select {
	case err := <-server.done:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}

		if elapsed := time.Since(stopped); elapsed >= 10*time.Second {
			t.Errorf("Run() returned after %v, want less than 10s", elapsed)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Run() did not return")
	}

	got := <-called
	if got.err != nil || got.result == nil || got.result.IsError {
		t.Errorf("CallTool() = %+v, %v, want a result", got.result, got.err)
	}
}
