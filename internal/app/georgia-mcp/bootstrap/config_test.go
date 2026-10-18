package bootstrap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/bootstrap"
)

// PLATFORM-SC-012: A variable of the environment sets a value. A missing one keeps its default.
func TestLoadConfigReadsTheEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_HTTP_RATE_LIMIT", "30")

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.HTTP.RateLimit != 30 {
		t.Errorf("HTTP.RateLimit = %d, want 30", cfg.HTTP.RateLimit)
	}

	if cfg.HTTP.Address != ":8080" {
		t.Errorf("HTTP.Address = %q, want %q", cfg.HTTP.Address, ":8080")
	}
}

// PLATFORM-SC-013: The local file sets a value, and the environment wins over it.
func TestLoadConfigReadsTheLocalFile(t *testing.T) {
	dir := t.TempDir()
	content := "GEOMCP_HTTP_RATE_LIMIT=10\nGEOMCP_LOG_FORMAT=text\n"

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)
	t.Setenv("GEOMCP_HTTP_RATE_LIMIT", "20")

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.HTTP.RateLimit != 20 {
		t.Errorf("HTTP.RateLimit = %d, want 20", cfg.HTTP.RateLimit)
	}

	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "text")
	}
}

// PLATFORM-SC-014: An invalid value stops the load, and the error names the variable.
func TestLoadConfigNamesTheInvalidVariable(t *testing.T) {
	tests := []struct {
		variable string
		value    string
	}{
		{variable: "", value: ""},
		{variable: "GEOMCP_HTTP_ADDRESS", value: "abc"},
		{variable: "GEOMCP_HTTP_RATE_LIMIT", value: "zero"},
		{variable: "GEOMCP_HTTP_RATE_LIMIT", value: "0"},
		{variable: "GEOMCP_LOG_FORMAT", value: "xml"},
		{variable: "GEOMCP_LOG_LEVEL", value: "loud"},
	}

	for _, tt := range tests {
		t.Run(tt.variable+"="+tt.value, func(t *testing.T) {
			t.Chdir(t.TempDir())

			if tt.variable != "" {
				t.Setenv(tt.variable, tt.value)
			}

			_, err := bootstrap.LoadConfig()

			if tt.variable == "" {
				if err != nil {
					t.Fatalf("LoadConfig() error = %v, want nil", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.variable) {
				t.Fatalf("LoadConfig() error = %v, want an error that names %s", err, tt.variable)
			}
		})
	}
}

// TBCRATE-SC-024: With the key only, the calls to TBC default to 1 each second, burst 1.
func TestLoadConfigGivesTheLimitOfTBC(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_RATES_TBC_API_KEY", "test-key")

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if http := cfg.Rates.TBC.HTTP; http.Rate != 1 || http.Burst != 1 {
		t.Errorf("TBC.HTTP = rate %v burst %d, want rate 1 burst 1", http.Rate, http.Burst)
	}
}
