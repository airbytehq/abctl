package build

import (
	"github.com/google/go-cmp/cmp"
	"runtime/debug"
	"testing"
)

func TestVersion(t *testing.T) {
	// struct definition
	// name: name of the sub-test
	// buildInfoFunc: test function to replace the default readBuildInfo function pointer
	// versionFunc: test function for updating the default Version value
	// exp: expected value that Version should be at the end
	tests := []struct {
		name          string
		versionFunc   func()
		buildInfoFunc buildInfoFunc
		exp           string
	}{
		{
			name: "default",
			exp:  "dev",
		},
		{
			name:        "no v prefix",
			versionFunc: func() { Version = "9.8.7" },
			exp:         "v9.8.7",
		},
		{
			name:        "v prefix",
			versionFunc: func() { Version = "v3.1.4" },
			exp:         "v3.1.4",
		},
		{
			name:          "non-ok BuildInfo",
			buildInfoFunc: func() (*debug.BuildInfo, bool) { return nil, false },
			exp:           "dev",
		},
		{
			name:        "version non-dev, BuildInfo ignored",
			versionFunc: func() { Version = "5.5.5" },
			buildInfoFunc: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{Main: debug.Module{
					Version: "v9.9.9",
				}}, true
			},
			exp: "v5.5.5",
		},
		{
			name: "version dev, BuildInfo honored",
			buildInfoFunc: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{Main: debug.Module{
					Version: "v9.9.9",
				}}, true
			},
			exp: "v9.9.9",
		},
		{
			name:        "invalid version defined",
			versionFunc: func() { Version = "bad.version" },
			exp:         "invalid (bad.version)",
		},
		{
			name: "invalid version from buildInfo",
			buildInfoFunc: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{Main: debug.Module{
					Version: "BAD_BUILD",
				}}, true
			},
			exp: "invalid (BAD_BUILD)",
		},
		{
			name:        "invalid version defined with v",
			versionFunc: func() { Version = "vbad.version" },
			exp:         "invalid (vbad.version)",
		},
		{
			name: "invalid version from buildInfo with v",
			buildInfoFunc: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{Main: debug.Module{
					Version: "VBAD_BUILD",
				}}, true
			},
			exp: "invalid (VBAD_BUILD)",
		},
	}

	origReadBuildInfo := readBuildInfo
	origVersion := Version
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() { readBuildInfo = origReadBuildInfo })
			if tt.buildInfoFunc != nil {
				readBuildInfo = tt.buildInfoFunc
				Version = origVersion
			}
			if tt.versionFunc != nil {
				tt.versionFunc()
			}
			setVersion()

			if d := cmp.Diff(tt.exp, Version); d != "" {
				t.Error("Version differed (-want, +got):", d)
			}
		})
	}
}
