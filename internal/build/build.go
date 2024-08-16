package build

import (
	"fmt"
	"runtime/debug"
	"strings"

	"golang.org/x/mod/semver"
)

// Version is the build of this tool.
// The expectation is that this will be set during build time via ldflags or via the BuildInfo if "go install"ed.
// Supported values: "dev", any semver (with or without a 'v', if no 'v' exists, one will be added).
// Any invalid version will be replaced with "invalid (BAD_VERSION)".
var Version = "dev"

// Revision is the git hash which built this tool.
// This value is automatically set if the buildInfoFunc function returns the "vcs.revision" setting.
var Revision string

// Modified is true if there are local code modifications when this binary was built.
// This value is automatically set if the buildInfoFunc function returns the "vcs.modified" setting.
var Modified bool

// ModificationTime is the time, in RFC3339 format, of this binary.
// This value is automatically set fi the buildInfoFunc function returns the "vcs.time" settings.
var ModificationTime string

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
			for _, kv := range buildInfo.Settings {
				switch kv.Key {
				case "vcs.modified":
					Modified = kv.Value == "true"
				case "vcs.time":
					ModificationTime = kv.Value
				case "vcs.revision":
					Revision = kv.Value
				}
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
