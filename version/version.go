// Package version exposes the build information of the application.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Version is the version of the application. It is set at build time:
//
//	go build -ldflags "-X github.com/torilabs/mqtt-prometheus-exporter/version.Version=v1.2.3"
var Version = "dev"

// Commit returns the VCS revision the binary was built from, or "unknown" when it is not embedded.
func Commit() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return commitFrom(info)
	}
	return "unknown"
}

func commitFrom(info *debug.BuildInfo) string {
	revision, modified := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return "unknown"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified {
		revision += "-dirty"
	}
	return revision
}

// String describes the running application.
func String() string {
	return fmt.Sprintf("%s (commit %s, %s, %s/%s)", Version, Commit(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
