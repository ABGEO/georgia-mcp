package buildinfo_test

import (
	"runtime"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
)

func TestGetFillsEveryField(t *testing.T) {
	t.Parallel()

	info := buildinfo.Get()

	if info.Version == "" || info.Commit == "" {
		t.Errorf("Get() = %+v, want no empty field", info)
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("Get().GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
}
