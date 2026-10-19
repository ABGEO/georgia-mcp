package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"

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

// BOGRATE-SC-009: With no variable of Bank of Georgia, the bank is on, at its production
// address, with 0.8 calls each second and a burst of 10.
func TestLoadConfigGivesTheDefaultsOfBankOfGeorgia(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bog := cfg.Rates.BOG
	if !bog.Enabled {
		t.Error("BOG.Enabled = false, want true")
	}

	if got := bog.BaseURL.String(); got != "https://api.businessonline.ge" {
		t.Errorf("BOG.BaseURL = %q, want %q", got, "https://api.businessonline.ge")
	}

	if bog.HTTP.Rate != 0.8 || bog.HTTP.Burst != 10 {
		t.Errorf(
			"BOG.HTTP = rate %v burst %d, want rate 0.8 burst 10",
			bog.HTTP.Rate,
			bog.HTTP.Burst,
		)
	}
}

// BOGRATE-SC-010: The default limit of Bank of Georgia lets at most 58 calls pass in
// the first 60 seconds, under the 60 calls in a minute that the bank allows.
func TestTheDefaultLimitOfBankOfGeorgiaStaysUnderItsQuota(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	synctest.Test(t, func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Limit(cfg.Rates.BOG.HTTP.Rate), cfg.Rates.BOG.HTTP.Burst)
		ctx, cancel := context.WithCancel(t.Context())

		var passed atomic.Int32

		for range 100 {
			go func() {
				if limiter.Wait(ctx) == nil {
					passed.Add(1)
				}
			}()
		}

		time.Sleep(time.Minute)
		synctest.Wait()

		if got := passed.Load(); got < 10 || got > 58 {
			t.Errorf("calls in the first 60 seconds = %d, want 10 to 58", got)
		}

		cancel()
		synctest.Wait()
	})
}

// WISFUEL-SC-020: With no variable of fuel, Wissol is on, at 1 call each second and a
// burst of 1.
func TestLoadConfigGivesTheDefaultsOfWissol(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	wissol := cfg.Fuel.Wissol
	if !wissol.Enabled {
		t.Error("Wissol.Enabled = false, want true")
	}

	if got := wissol.BaseURL.String(); got != "https://api.wissol.ge" {
		t.Errorf("Wissol.BaseURL = %q, want https://api.wissol.ge", got)
	}

	if http := wissol.HTTP; http.Rate != 1 || http.Burst != 1 {
		t.Errorf("Wissol.HTTP = rate %v burst %d, want rate 1 burst 1", http.Rate, http.Burst)
	}
}
