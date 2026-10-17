package config_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
)

type inner struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Count   int           `env:"COUNT"   envDefault:"1"`
}

func (i inner) Validate() error {
	if i.Count < 1 {
		return &config.FieldError{Field: "Count", Problem: "must be 1 or more"}
	}

	return nil
}

type root struct {
	First  inner `envPrefix:"FIRST_"`
	Second inner `envPrefix:"SECOND_"`
}

func TestLoadNamesTheVariableOfAParseError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_SECOND_TIMEOUT", "soon")

	_, err := config.Load[root]()
	if err == nil || !strings.Contains(err.Error(), "GEOMCP_SECOND_TIMEOUT") {
		t.Fatalf("Load() error = %v, want an error that names GEOMCP_SECOND_TIMEOUT", err)
	}
}

func TestLoadNamesTheVariableOfAValidationError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GEOMCP_FIRST_COUNT", "0")

	_, err := config.Load[root]()

	if _, ok := errors.AsType[*config.FieldError](err); !ok {
		t.Fatalf("Load() error = %v, want a *FieldError", err)
	}

	if !strings.Contains(err.Error(), "GEOMCP_FIRST_COUNT") {
		t.Errorf("Load() error = %v, want an error that names GEOMCP_FIRST_COUNT", err)
	}
}

func TestLoadAppliesTheDefaults(t *testing.T) {
	t.Chdir(t.TempDir())

	got, err := config.Load[inner]()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Timeout != 10*time.Second || got.Count != 1 {
		t.Errorf("Load() = %+v, want the defaults", got)
	}
}
