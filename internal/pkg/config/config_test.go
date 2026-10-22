package config_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
)

const apiKey = "API_KEY"

type inner struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Count   int           `env:"COUNT"   envDefault:"1"   validate:"min=1"`
}

type root struct {
	First  inner `envPrefix:"FIRST_"`
	Second inner `envPrefix:"SECOND_"`
}

type vendor struct {
	Enabled bool     `env:"ENABLED"  envDefault:"true"`
	APIKey  string   `env:"API_KEY"  validate:"required_if=Enabled true"`
	BaseURL url.URL  `env:"BASE_URL" envDefault:"https://example.com"    validate:"http_url"`
	Rate    float64  `env:"RATE"     envDefault:"1"                      validate:"gt=0"`
	Ranges  []string `env:"RANGES"   envSeparator:","                    validate:"dive,cidr"`
}

type vendorRoot struct {
	Vendor vendor `envPrefix:"VENDOR_"`
}

// PLATFORM-SC-014: A value that fails to parse names its variable.
func TestLoadNamesTheVariableOfAParseError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_SECOND_TIMEOUT", "soon")

	_, err := config.Load(root{})
	if err == nil || !strings.Contains(err.Error(), "GEOMCP_SECOND_TIMEOUT") {
		t.Fatalf("Load() error = %v, want an error that names GEOMCP_SECOND_TIMEOUT", err)
	}
}

// PLATFORM-SC-014: A value that fails its rule names its variable, under the prefix of
// its struct.
func TestLoadNamesTheVariableOfAValidationError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_SECOND_COUNT", "0")

	_, err := config.Load(root{})

	fieldErr, ok := errors.AsType[*config.FieldError](err)
	if !ok {
		t.Fatalf("Load() error = %v, want a *FieldError", err)
	}

	if fieldErr.Variable != "GEOMCP_SECOND_COUNT" {
		t.Errorf("Variable = %q, want GEOMCP_SECOND_COUNT", fieldErr.Variable)
	}
}

// PLATFORM-SC-012: A missing variable keeps its default.
func TestLoadAppliesTheDefaults(t *testing.T) {
	t.Chdir(t.TempDir())

	got, err := config.Load(inner{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Timeout != 10*time.Second || got.Count != 1 {
		t.Errorf("Load() = %+v, want the defaults", got)
	}
}

// PLATFORM-SC-036: Each rule gives a message that names the variable.
func TestLoadGivesTheMessageOfEachRule(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]string
		want     string
	}{
		{
			name:     "http_url",
			settings: map[string]string{"BASE_URL": "ftp://example.com", apiKey: "k"},
			want:     "GEOMCP_VENDOR_BASE_URL: must be an http or https URL",
		},
		{
			name:     "gt",
			settings: map[string]string{"RATE": "0", apiKey: "k"},
			want:     "GEOMCP_VENDOR_RATE: must be more than 0",
		},
		{
			name:     "required_if",
			settings: map[string]string{"ENABLED": "true"},
			want: "GEOMCP_VENDOR_API_KEY: must be set while GEOMCP_VENDOR_ENABLED is true. " +
				"Set it, or set GEOMCP_VENDOR_ENABLED=false",
		},
		{
			name:     "required_if, off",
			settings: map[string]string{"ENABLED": "false"},
		},
		{
			name:     "cidr",
			settings: map[string]string{"RANGES": "10.0.0.0/8,nope", apiKey: "k"},
			want:     "GEOMCP_VENDOR_RANGES: nope must be a CIDR range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			for key, value := range tt.settings {
				t.Setenv("GEOMCP_VENDOR_"+key, value)
			}

			_, err := config.Load(vendorRoot{})
			if tt.want == "" {
				if err != nil {
					t.Fatalf("Load() error = %v, want none", err)
				}

				return
			}

			fieldErr, ok := errors.AsType[*config.FieldError](err)
			if !ok {
				t.Fatalf("Load() error = %v, want a *FieldError", err)
			}

			if got := fieldErr.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// PLATFORM-SC-036: Two invalid values give two errors, and each one is a *FieldError.
func TestLoadGivesEachError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_VENDOR_RATE", "0")
	t.Setenv("GEOMCP_VENDOR_BASE_URL", "ftp://example.com")
	t.Setenv("GEOMCP_VENDOR_API_KEY", "k")

	_, err := config.Load(vendorRoot{})

	joined, ok := errors.AsType[interface {
		error
		Unwrap() []error
	}](err)
	if !ok {
		t.Fatalf("Load() error = %v, want a joined error", err)
	}

	var fieldErrs int

	for _, inner := range joined.Unwrap() {
		if _, ok := errors.AsType[*config.FieldError](inner); ok {
			fieldErrs++
		}
	}

	if fieldErrs != 2 {
		t.Errorf("Load() error = %v, want 2 field errors: the rate and the address", err)
	}
}
