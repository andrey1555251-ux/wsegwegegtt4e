// Package version contains the human-readable version string baked
// into the binary at build time.
package version

// Version is overridden via -ldflags "-X .../version.Version=..." at
// release time.  The default is the development tag.
var Version = "0.1.0-dev"
