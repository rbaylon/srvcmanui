// Package version holds the build version, injected at link time from the
// latest git tag: go build -ldflags "-X github.com/rbaylon/srvcmanui/version.Version=$(git describe --tags --always --dirty)"
package version

// Version is "dev" unless set by -ldflags at build time.
var Version = "dev"
