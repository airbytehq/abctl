package build

import (
	"fmt"
	"golang.org/x/mod/semver"
	"strings"
)

// Version is the build of this tool.
// The expectation is that this will be set during build time via ldflags.
// Supported values: "dev", any semver (with or without a 'v', if no 'v' exists, one will be added).
// Any invalid versions will be replaced with "invalid (BAD_VERSION)".
var Version = "dev"

func init() {
	if Version != "dev" {
		if !strings.HasPrefix(Version, "v") {
			Version = "v" + Version
		}
		if !semver.IsValid(Version) {
			Version = fmt.Sprintf("invalid (%s)", Version)
		}
	}
}
