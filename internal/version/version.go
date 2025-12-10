package version

import "fmt"

var (
	// Version is the semantic version of the binary. Overridden via -ldflags "-X".
	Version = "0.1.0"
	// Commit is the git commit hash injected at build time.
	Commit = "dev"
	// BuildDate is the build timestamp injected at build time.
	BuildDate = "unknown"
)

// Full returns a human-friendly version string.
func Full() string {
	return fmt.Sprintf("%s (commit:%s, built:%s)", Version, Commit, BuildDate)
}
