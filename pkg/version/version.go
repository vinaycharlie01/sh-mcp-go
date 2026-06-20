package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected by the linker.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Info carries all version metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current build info.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable version string.
func (i Info) String() string {
	return fmt.Sprintf("sh-mcp-go %s (commit=%s, built=%s, %s/%s, %s)",
		i.Version, i.Commit, i.BuildDate, i.OS, i.Arch, i.GoVersion)
}
