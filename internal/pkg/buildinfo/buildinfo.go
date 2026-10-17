// Package buildinfo reads the version that the Go toolchain writes into the binary.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

const unknown = "unknown"

// Info is the version of the binary.
type Info struct {
	Version   string
	Commit    string
	GoVersion string
}

// Get returns the version of the running binary. An absent value is "unknown".
func Get() Info {
	info := Info{Version: unknown, Commit: unknown, GoVersion: runtime.Version()}

	build, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	if build.Main.Version != "" {
		info.Version = build.Main.Version
	}

	for _, setting := range build.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			info.Commit = setting.Value
		}
	}

	return info
}
