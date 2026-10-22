package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//nolint:gochecknoglobals // TestMain builds the binary one time, for every test of the package.
var binary string

const (
	mainPackage = "github.com/abgeo/georgia-mcp/cmd/georgia-mcp"

	initializeRequest = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"v0"}}}`
	initializedNotification = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	listToolsRequest        = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	listToolsID             = "2"
	// ttcKey and tbcKey let the server start: the TTC and TBC are on by default and need
	// a key.
	ttcKey = "GEOMCP_TRANSIT_TTC_API_KEY=test-key"
	tbcKey = "GEOMCP_RATES_TBC_API_KEY=test-key"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "georgia-mcp-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	binary = filepath.Join(dir, "georgia-mcp")

	//nolint:gosec // The output path is a directory that this test made.
	build := exec.CommandContext(context.Background(), "go", "build", "-o", binary, mainPackage)
	build.Stderr = os.Stderr

	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building the binary:", err)
		os.Exit(1)
	}

	// The first start of a new binary includes the scan of the operating system.
	// PLATFORM-SC-029 measures the server, so this start takes that cost.
	if err := exec.CommandContext(context.Background(), binary, "version").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "starting the binary:", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = os.RemoveAll(dir)

	os.Exit(code)
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	address := listener.Addr().String()
	_ = listener.Close()

	return address
}

func startServe(t *testing.T, dir, address string) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), binary, "serve")
	cmd.Dir = dir

	cmd.Env = append(os.Environ(), "GEOMCP_HTTP_ADDRESS="+address, ttcKey, tbcKey)

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting serve: %v", err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill() })

	return cmd
}

func waitReady(t *testing.T, address string, limit time.Duration) time.Duration {
	t.Helper()

	start := time.Now()
	client := &http.Client{Timeout: 100 * time.Millisecond}

	for time.Since(start) < limit {
		req, _ := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://"+address+"/readyz",
			http.NoBody,
		)

		if resp, err := client.Do(req); err == nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return time.Since(start)
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("/readyz did not answer 200 within %v", limit)

	return 0
}

func stop(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("serve exited with %v, want 0", err)
	}
}

// PLATFORM-SC-029: The server reports ready less than 1 second after the process starts.
//
//nolint:paralleltest // A parallel test competes for the CPU and changes the measured time.
func TestServeIsReadyWithinOneSecond(t *testing.T) {
	address := freeAddress(t)
	cmd := startServe(t, t.TempDir(), address)

	if elapsed := waitReady(t, address, 5*time.Second); elapsed >= time.Second {
		t.Errorf("ready after %v, want less than 1s", elapsed)
	}

	stop(t, cmd)
}

// PLATFORM-SC-031: The server writes no file in its working directory.
func TestServeWritesNoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	address := freeAddress(t)
	cmd := startServe(t, dir, address)

	waitReady(t, address, 5*time.Second)

	transport := &mcp.StreamableClientTransport{
		Endpoint:             "http://" + address + "/mcp",
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	session, err := client.Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if _, err := session.ListTools(t.Context(), nil); err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	_ = session.Close()

	stop(t, cmd)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Errorf("the working directory holds %d entries, want 0", len(entries))
	}
}

// PLATFORM-SC-032: In stdio mode, stdout carries JSON-RPC messages only, and stderr holds the log.
func TestStdioWritesOnlyTheProtocolToStdout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "stdio")
	cmd.Dir = t.TempDir()

	cmd.Env = append(os.Environ(), "GEOMCP_LOG_LEVEL=debug", ttcKey, tbcKey)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	stdin, stdout := pipes(t, cmd)

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	for _, message := range []string{initializeRequest, initializedNotification, listToolsRequest} {
		if _, err := io.WriteString(stdin, message+"\n"); err != nil {
			t.Fatal(err)
		}
	}

	messages := readUntilID(t, stdout, listToolsID)

	_ = stdin.Close()

	if rest, _ := io.ReadAll(stdout); len(bytes.TrimSpace(rest)) != 0 {
		t.Errorf("stdout holds more after the answer: %q", rest)
	}

	if err := cmd.Wait(); err != nil {
		t.Errorf("stdio exited with %v, want 0", err)
	}

	if messages < 2 {
		t.Errorf("stdout held %d messages, want 2 or more", messages)
	}

	if !strings.Contains(stderr.String(), `"msg"`) {
		t.Errorf("stderr holds no log record: %q", stderr.String())
	}
}

func pipes(t *testing.T, cmd *exec.Cmd) (io.WriteCloser, io.Reader) {
	t.Helper()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	return stdin, stdout
}

// readUntilID reads stdout up to the answer with the identifier, and fails on a
// line that is not a JSON-RPC message. It returns the count of the messages.
func readUntilID(t *testing.T, stdout io.Reader, id string) int {
	t.Helper()

	scanner := bufio.NewScanner(stdout)
	count := 0

	for scanner.Scan() {
		var message struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
		}

		if err := json.Unmarshal(
			scanner.Bytes(),
			&message,
		); err != nil ||
			message.JSONRPC != "2.0" {
			t.Fatalf("a stdout line is not a JSON-RPC message: %q", scanner.Text())
		}

		count++

		if string(message.ID) == id {
			return count
		}
	}

	t.Fatalf("stdout ended before the answer %s", id)

	return count
}
