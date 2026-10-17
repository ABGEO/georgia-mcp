package mcpserver_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
)

func newInfo() buildinfo.Info {
	return buildinfo.Info{Version: testVersion, Commit: "abc", GoVersion: "go"}
}

// PLATFORM-SC-002: A client over stdio lists the tool of the test module.
func TestStdioListsTheTools(t *testing.T) {
	t.Parallel()

	session := connectStdio(t, newRegistry(t, discardLogger(), echoModule{}))

	if diff := cmp.Diff([]string{echoTool}, toolNames(t, session)); diff != "" {
		t.Errorf("tools mismatch (-want +got):\n%s", diff)
	}
}

// PLATFORM-SC-003: The HTTP transport and the stdio transport list the same tools.
func TestBothTransportsListTheSameTools(t *testing.T) {
	t.Parallel()

	registry := newRegistry(t, discardLogger(), echoModule{})

	overHTTP := toolNames(t, connectHTTP(t, registry))
	overStdio := toolNames(t, connectStdio(t, registry))

	if diff := cmp.Diff(overHTTP, overStdio); diff != "" {
		t.Errorf("tools mismatch (-http +stdio):\n%s", diff)
	}
}

// PLATFORM-SC-004: With no module, both transports list zero tools with no error.
func TestNoModuleListsZeroTools(t *testing.T) {
	t.Parallel()

	registry := newRegistry(t, discardLogger())

	for name, session := range map[string]*mcp.ClientSession{
		"http":  connectHTTP(t, registry),
		"stdio": connectStdio(t, registry),
	} {
		if got := toolNames(t, session); len(got) != 0 {
			t.Errorf("%s: ListTools() = %v, want no tool", name, got)
		}
	}
}

// PLATFORM-SC-005: The server gives its name and its version to a client.
func TestServerGivesItsNameAndVersion(t *testing.T) {
	t.Parallel()

	session := connectInMemory(t, newRegistry(t, discardLogger()))

	got := session.InitializeResult().ServerInfo
	want := &mcp.Implementation{Name: "georgia-mcp", Version: testVersion}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("server info mismatch (-want +got):\n%s", diff)
	}
}
