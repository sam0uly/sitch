// Package version reports the sitch build version and commit.
package version

import (
	"runtime/debug"
)

// Build-time parameters set via -ldflags in release builds.
var (
	Version = "devel"
	Commit  = "unknown"
)

// A user may install sitch using `go install samouly.fun/sitch@latest`
// without -ldflags, in which case the variables above are unset. As a
// workaround we use the embedded build version that *is* set when using
// `go install` (and is only set for `go install` and not for `go build`).
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		Version = v
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
			Commit = setting.Value[:7]
		}
	}
}
