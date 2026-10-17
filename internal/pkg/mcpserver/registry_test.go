package mcpserver_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// PLATFORM-SC-006: A module joins through the registry, and its tool gets the annotations of MCP-004.
func TestRegisterAddsTheToolWithTheAnnotations(t *testing.T) {
	t.Parallel()

	session := connectInMemory(t, newRegistry(t, discardLogger(), echoModule{}))

	result, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("ListTools() = %d tools, want 1", len(result.Tools))
	}

	got := result.Tools[0].Annotations
	if got == nil {
		t.Fatal("the tool has no annotations")
	}

	destructive, openWorld := false, true
	want := &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: &destructive,
		OpenWorldHint:   &openWorld,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("annotations mismatch (-want +got):\n%s", diff)
	}
}

// PLATFORM-SC-007: Two modules with one tool name stop the registration, and the error names the tool.
func TestRegisterRejectsADuplicateName(t *testing.T) {
	t.Parallel()

	registry := mcpserver.New(newInfo(), discardLogger())

	err := registry.Register(echoModule{}, echoModule{})
	if !errors.Is(err, mcpserver.ErrDuplicateTool) {
		t.Fatalf("Register() error = %v, want ErrDuplicateTool", err)
	}

	if !strings.Contains(err.Error(), echoTool) {
		t.Errorf("Register() error = %q, want the name %q", err, echoTool)
	}
}
