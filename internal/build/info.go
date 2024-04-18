package build

import (
	"fmt"
	"golang.org/x/mod/semver"
	"runtime/debug"
	"strings"
)

// Version is the build of this tool.
// The expectation is that this will be set during build time via ldflags or via the BuildInfo if "go install"ed.
// Supported values: "dev", any semver (with or without a 'v', if no 'v' exists, one will be added).
// Any invalid version will be replaced with "invalid (BAD_VERSION)".
var Version = "dev"

// buildInfoFunc matches the debug.ReadBuildInfo method, redefined here for testing purposes.
type buildInfoFunc func() (*debug.BuildInfo, bool)

// readBuildInfo is a function pointer to debug.ReadBuildInfo, defined here for testing purposes.
var readBuildInfo buildInfoFunc = debug.ReadBuildInfo

// setVersion sets the Version variable correctly.
// This method is only separated out from the init method for testing purposes and should only
// be called by init and the unit tests.
func setVersion() {
	if Version == "dev" {
		buildInfo, ok := readBuildInfo()
		if ok {
			if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
				Version = buildInfo.Main.Version
			}
		}
	}

	if Version != "dev" {
		origVersion := Version
		if !strings.HasPrefix(Version, "v") {
			Version = "v" + Version
		}
		if !semver.IsValid(Version) {
			Version = fmt.Sprintf("invalid (%s)", origVersion)
		}
	}
}

func init() {
	setVersion()
}
