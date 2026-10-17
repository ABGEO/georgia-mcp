package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cli"
)

// PLATFORM-SC-015: The version command prints three lines and loads no configuration.
func TestVersionNeedsNoConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_HTTP_ADDRESS", "abc")

	var out bytes.Buffer

	root := cli.NewRoot()
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("version error = %v, want nil", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	prefixes := []string{"version ", "commit ", "go "}

	if len(lines) != len(prefixes) {
		t.Fatalf("version printed %q, want %d lines", out.String(), len(prefixes))
	}

	for i, prefix := range prefixes {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d = %q, want the prefix %q", i+1, lines[i], prefix)
		}
	}
}
