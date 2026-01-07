package version

// Version information - updated during build
var (
	// Version is the current version of gwt
	Version = "0.1.0"

	// Commit is the git commit hash (set via ldflags)
	Commit = "dev"

	// BuildDate is the build date (set via ldflags)
	BuildDate = "unknown"
)

// String returns the full version string
func String() string {
	if Commit != "dev" {
		return Version + " (" + Commit + ")"
	}
	return Version
}
