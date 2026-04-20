// Package version holds the dbguard version for CLI and build injection.
package version

// Version is the semantic version of dbguard.
// It can be overridden at build time with:
// -ldflags "-X github.com/ChimdumebiNebolisa/DBwall/internal/version.Version=vX.Y.Z"
var Version = "0.2.0-dev"
