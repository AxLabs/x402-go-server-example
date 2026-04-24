// Package version provides build-time version information.
package version

// Build-time variables set via ldflags.
var (
	// Version is the semantic version of the application.
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// BuildTime is the build timestamp.
	BuildTime = "unknown"
)

// Info returns version information as a struct.
func Info() VersionInfo {
	return VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}

// VersionInfo holds version metadata.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}
