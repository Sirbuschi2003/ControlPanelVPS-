package version

// Commit and Date are injected at build time via ldflags.
var (
	Commit = "dev"
	Date   = "unknown"
)
